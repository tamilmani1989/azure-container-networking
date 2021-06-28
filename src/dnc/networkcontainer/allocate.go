package networkcontainer

import (
	"context"
	"dnc/aiwrapper"
	"dnc/allocationhelper"
	"dnc/api"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/retry"
	"dnc/utils"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Azure/azure-container-networking/aitelemetry"

	"github.com/Azure/azure-container-networking/cns"
)

const (
	reservedAddress  = "Azure Reserved Address"
	slash16HostCount = 256 * 256
	prepopulateOp    = "Prepopulate"
	depopulateOp     = "Depopulate"
)

// TODO: we should rename this as ReserveIpManager or something else instead of just IpAllocator
// IPAllocator can process different types of IP allocations for network containers
type IPAllocator struct {
	Nodes              NodeRepo
	NCs                NCRepo
	Subnets            SubnetRepo
	Reservations       IPReservationRepo
	Allocations        IPAllocationRepo
	PendingCleanupRepo *datamodel.PendingResourceCleanupRepo
	Logger             loggershim.LoggerInterface

	// TODO: These should not be exported and initialized only during startup
	MaxBatchSize                int
	DisableResMapInSubnetConfig bool
	PubsubProxyEnabled          bool
	UsePrepopulatedAllocations  bool
}

// ProcessIPReservationAllocationWithKey processes IP allocation for a given reservation key
// returns whether allocation was completed and any errors in processing
func (a *IPAllocator) ProcessIPReservationAllocationWithKey(reservationKey string) (bool, *api.JsonErr) {
	reservation, err := a.Reservations.TryQueryIPReservationByKey(reservationKey)
	if err != nil {
		return false, err
	}

	// if the parent reservation entry doesn't exist then it must have
	// been deleted via separate delete reservation/container request,
	// or if the subnet has already been allocated
	if reservation.Configuration == nil || reservation.Configuration.AllocatedSubnet != "" {
		return true, nil // don't add new pending work queue item, error
	}

	return a.ProcessIPReservationAllocation(reservation)
}

// ProcessIPReservationAllocation processes IP allocation for the given reservation
// returns whether ip allocation is complete, and any errors in processing
func (a *IPAllocator) ProcessIPReservationAllocation(reservation datamodel.ReservationRow) (bool, *api.JsonErr) {
	subnetConfig, err := a.Subnets.TryQuerySubnetConfiguration(reservation.Configuration.NetworkID, reservation.Configuration.AllocationRequest.SubnetName)
	if err != nil {
		return false, err
	}
	if subnetConfig == nil {
		err := api.NotFound(api.SubnetNotFound, fmt.Sprintf("Failed to find subnet for allocation request %s",
			reservation.Configuration.AllocationRequest.SubnetName))
		return a.markReservationAsFailed(reservation, err)
	}

	allocationState, err := a.getAllocationsFromSubnetTable(reservation.Configuration.NetworkID, reservation.Configuration.AllocationRequest.SubnetName)
	if err != nil {
		if err.HTTPStatus == http.StatusNotFound {
			return a.markReservationAsFailed(reservation, err)
		}
		return false, nil
	}

	isComplete := reconcileReservation(&reservation, allocationState)
	if isComplete {
		return a.completeAllocation(reservation, subnetConfig)
	}

	baseSubnet, parseErr := utils.ParseIPNet(subnetConfig.IPSubnet)
	if parseErr != nil {
		err := api.InternalServerError(api.StatusInternalServerError, parseErr.Error())
		return a.markReservationAsFailed(reservation, err)
	}

	addressesInUse, err := a.getAllAddressesInUse(baseSubnet, allocationState, reservation.Configuration)
	if err != nil {
		if err.HTTPStatus == http.StatusNotFound {
			return a.markReservationAsFailed(reservation, err)
		}
		return false, nil
	}

	var primaryIP *net.IPNet
	if reservation.Configuration.AllocatedSubnet != "" {
		_, primaryIP, parseErr = net.ParseCIDR(reservation.Configuration.AllocatedSubnet)
		if parseErr != nil {
			return a.markReservationAsFailed(reservation, api.InternalServerError(api.StatusInternalServerError, parseErr.Error()))
		}
	} else {
		primaryIP, err = determinePrimaryIP(reservation.Configuration, baseSubnet, addressesInUse)
		if err != nil {
			return a.markReservationAsFailed(reservation, err)
		}
	}

	if primaryIP == nil {
		err := api.BadRequest(api.AllocationFailed, fmt.Sprintf("Failed to allocate address for request %s. Subnet is full",
			reservation.Configuration.AllocationRequest.String()))
		return a.markReservationAsFailed(reservation, err)
	}

	// whether it's a single record or multiple, we can still use a batch for a single transaction
	// reservation key is the same for all rows of the batch
	batch := make(map[string]map[string]interface{})
	reservationKey := datamodel.GetReservationKey(reservation.Configuration.NetworkID, reservation.Configuration.ReservationID)

	// we only add the primary ip for the reservation if it hadn't been found previously
	if reservation.Configuration.AllocatedSubnet == "" {
		rowKey := datamodel.EncodeIPAllocationKey(primaryIP.String())
		batch[rowKey] = map[string]interface{}{
			datamodel.IPAllocationProperty: reservationKey,
			datamodel.AllocationIDProperty: reservation.Configuration.ReservationID}
	}

	// we keep track of all secondary allocations that have not been assigned an IP yet
	// we only find IPs for these, and as the IPAllocation record is built and added to the batch, the value will be populated
	// so we can merge back into the original reservation only after successful allocation
	unallocated := make(map[string]string)
	for allocationID, ipAddress := range reservation.Configuration.SecondaryAllocations {
		if ipAddress == "" {
			unallocated[allocationID] = ""
		}
	}

	if len(unallocated) > 0 {
		// adding primaryIP to allocatedIPs for no conflicts on secondary IP resolution
		addressesInUse[primaryIP.String()] = primaryIP
		// these are the unique IP addresses that should be skipped when determining free space for secondary allocations
		var ipsInUse []net.IP
		for _, subnet := range addressesInUse {
			ipsInUse = append(ipsInUse, subnet.IP)
		}

		availableIPs, allocErr := findAllocatableIPs(baseSubnet, ipsInUse, len(unallocated))
		if allocErr != nil {
			if allocErr == errCapacityExceeded {
				return false, api.BadRequest(api.AllocationFailed,
					fmt.Sprintf("Failed to allocate address for request %s. Subnet is full", reservation.Configuration.AllocationRequest.String()))
			}
			return false, api.InternalServerError(api.AllocationFailed, allocErr.Error())
		}
		i := 0
		for allocationID := range unallocated {
			encodedSubnet := availableIPs[i].String() + utils.PrefixDelimiter + utils.IPV4DefaultPrefix
			unallocated[allocationID] = encodedSubnet
			rowKey := datamodel.EncodeIPAllocationKey(encodedSubnet)
			batch[rowKey] = map[string]interface{}{
				datamodel.IPAllocationProperty: reservationKey,
				datamodel.AllocationIDProperty: allocationID}
			i++
		}
	}

	if err := a.Allocations.AddIPAllocations(reservation.Configuration.NetworkID, reservation.Configuration.AllocationRequest.SubnetName, batch); err != nil {
		// whether it's a conflict or a network error, we should retry
		return false, err
	}

	// once successful, mutate the original reservation with new state
	if reservation.Configuration.AllocatedSubnet == "" {
		reservation.Configuration.AllocatedSubnet = primaryIP.String()
	}
	// no-op if unallocated is empty
	for allocationID, ipAddress := range unallocated {
		reservation.Configuration.SecondaryAllocations[allocationID] = ipAddress
	}

	return a.completeAllocation(reservation, subnetConfig)
}

func (a *IPAllocator) getAllAddressesInUse(baseSubnet *net.IPNet, allocationState map[string]*net.IPNet, req *datamodel.ReservationConfiguration) (map[string]*net.IPNet, *api.JsonErr) {
	addressesInUse := make(map[string]*net.IPNet) // 10.0.0.4/32 to subnet
	for _, subnet := range allocationState {
		addressesInUse[subnet.String()] = subnet
	}

	azureReserved, resErr := allocationhelper.GetReservedAddressesInSubnet(baseSubnet)
	if resErr != nil {
		return nil, api.InternalServerError(api.StatusInternalServerError, resErr.Error())
	}
	for reservedSubnetString, reservedIPSubnet := range azureReserved {
		addressesInUse[reservedSubnetString] = reservedIPSubnet
	}

	if req.AssociatedNCID != "" {
		nc, err := a.NCs.TryQueryNetworkContainer(req.NetworkID, req.AssociatedNCID)
		if err != nil {
			return nil, err
		}

		if nc.RowKey == "" {
			return nil, api.NotFound(api.NetworkContainerNotFound, fmt.Sprintf("Failed to find configuration for network container %s", req.AssociatedNCID))
		}

		nodeID := nc.Config.GetAssociationInfo().NodeID
		// This needs to be fixed for scenarios where we are using the reserve IP
		// API to dump allocations on a node and orchestrator is WebApps.
		if nodeID != "" && nc.Config.GetAssociationInfo().ContainerType == cns.WebApps {
			allocatedAddressesOnNode, err := a.getAllocationsOnNode(nodeID)
			if err != nil {
				return nil, err
			}
			for allocatedNodeSubnetString, allocatedNodeSubnet := range allocatedAddressesOnNode {
				addressesInUse[allocatedNodeSubnetString] = allocatedNodeSubnet
			}
		}
	}
	return addressesInUse, nil
}

func determinePrimaryIP(req *datamodel.ReservationConfiguration, baseSubnet *net.IPNet, addressesInUse map[string]*net.IPNet) (*net.IPNet, *api.JsonErr) {
	if req.AllocationRequest.IPConstraint == "" { // can allocate from anywhere in subnet
		allocatedSubnet, err := allocationhelper.AllocateSubnetFromNetworkSpace(baseSubnet, addressesInUse, 32)
		if err != nil {
			return nil, api.BadRequest(api.AllocationFailed, fmt.Sprintf("Failed to allocate subnet for request %s. Error:[%s]", req.String(), err.Error()))
		}
		return allocatedSubnet, nil
	}

	subnetConstraint, err := allocationhelper.GetIPSubnetForIPAllocationConstraint(req.AllocationRequest.IPConstraint) // IP constraint in request can be a single address or a subnet
	if err != nil {
		return nil, err
	}
	if _, err := allocationhelper.IsRequestedRangeAllocatable(baseSubnet, addressesInUse, subnetConstraint); err != nil {
		return nil, err
	}

	return subnetConstraint, nil
}

func (a *IPAllocator) completeAllocation(request datamodel.ReservationRow, subnetConfig *datamodel.SubnetConfiguration) (bool, *api.JsonErr) {
	config := request.Configuration
	config.Status = datamodel.Completed

	if config.AllocationRequest.NodeConstraint != "" {
		err := a.Nodes.UpdateNodeConstrainedReservationSetByNodeID(config.AllocationRequest.NodeConstraint, config.NetworkID, config.ReservationID, false)
		if err != nil {
			err.Text = fmt.Sprintf("Failed to associate reservation request %s to node %s due to error %s",
				config.String(), config.AllocationRequest.NodeConstraint, err.String())
			return false, err
		}
	}

	if err := a.Subnets.UpdateSubnetReservationSet(config.ReservationID, config.AllocatedSubnet, false, subnetConfig); err != nil {
		err.Text = fmt.Sprintf("Failed to persist updating reservation request %s in subnet configuration due to error %s", config.String(), err.String())
		if err.Code == api.PropertyValueTooLargeStorageError {
			err := api.BadRequest(api.FailedToUpdateSubnetWithReservationInfo, err.Text)
			return a.markReservationAsFailed(request, err)
		}

		return false, err
	}

	if err := a.Reservations.PersistIPReservation(request); err != nil {
		err.Text = fmt.Sprintf("Failed to mark reservation request %s as succeded due to error %s", config.String(), err.String())
		return false, err
	}

	return true, nil
}

func (a *IPAllocator) getAllocationsOnNode(nodeID string) (map[string]*net.IPNet, *api.JsonErr) {
	nodeConfiguration, err := a.Nodes.QueryNodeConfiguration(nodeID)
	if err != nil {
		return nil, err
	}

	allocatedSubnets := make(map[string]*net.IPNet)

	for reservationID, networkID := range nodeConfiguration.ConstrainedReservations {
		allocatedIPSubnet, err := a.getSubnetFromReservation(reservationID, networkID)
		if err != nil {
			return nil, err
		}
		if allocatedIPSubnet == nil {
			continue
		}
		allocatedSubnets[allocatedIPSubnet.String()] = allocatedIPSubnet
	}

	for networkID, ncMap := range nodeConfiguration.DeployedNetworkContainerMap {
		for _, reservationID := range ncMap {
			allocatedIPSubnet, err := a.getSubnetFromReservation(reservationID, networkID)
			if err != nil {
				return nil, err
			}
			if allocatedIPSubnet == nil {
				continue
			}
			allocatedSubnets[allocatedIPSubnet.String()] = allocatedIPSubnet
		}
	}

	for _, ipAddress := range nodeConfiguration.NodeInfo.IPAddresses {
		nodeIPSubnetString := ipAddress + utils.PrefixDelimiter + utils.IPV4DefaultPrefix
		nodeIPSubnet, err := utils.ParseIPNet(nodeIPSubnetString)
		if err != nil {
			return nil, api.InternalServerError(api.StatusInternalServerError, err.Error())
		}
		allocatedSubnets[nodeIPSubnetString] = nodeIPSubnet
	}

	return allocatedSubnets, nil
}

func (a *IPAllocator) getSubnetFromReservation(networkID, reservationID string) (*net.IPNet, *api.JsonErr) {
	reservation, err := a.Reservations.TryQueryIPReservation(networkID, reservationID)
	if err != nil {
		return nil, err
	}
	if reservation.Configuration == nil || reservation.Configuration.AllocatedSubnet == "" {
		return nil, nil
	}

	allocatedIPSubnet, parseErr := utils.ParseIPNet(reservation.Configuration.AllocatedSubnet)
	if parseErr != nil {
		return nil, api.InternalServerError(api.StatusInternalServerError, parseErr.Error())
	}
	return allocatedIPSubnet, nil
}

func (a *IPAllocator) getAllocationsFromSubnetTable(networkID, subnetName string) (map[string]*net.IPNet, *api.JsonErr) {
	allocations := make(map[string]*net.IPNet)

	rows, err := a.Allocations.TryQueryAllIPAllocations(networkID, subnetName)
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		ipAddressString := datamodel.DecodeIPAllocationKey(row.RowKey)
		allocatedIPSubnet, err := utils.ParseIPNet(ipAddressString)
		if err != nil {
			return nil, api.InternalServerError(api.StatusInternalServerError, err.Error())
		}

		switch {
		case row.AllocationID != "":
			allocations[row.AllocationID] = allocatedIPSubnet
		case row.IPAllocation != "":
			resID := datamodel.GetReservationIDFromKey(row.IPAllocation)
			allocations[resID] = allocatedIPSubnet
		}
	}

	return allocations, nil
}

func (a *IPAllocator) getAllocationsForReservationIDs(networkID, subnetName string, reservationIDs []string) (map[string]*net.IPNet, *api.JsonErr) {
	allocations := make(map[string]*net.IPNet)

	// we can only query the ip allocations table based on reservation key. since these are uuids, the encoded key might be redundant.
	// we should consider only storing the reservation uuid.
	// note: this path is currently only used by the subnet handler, which means batches of at most 100 (default config) ncs (1:1 w/ reservations)
	// the resulting filters built to perform this query will work for this batch size, but can start breaking after around 130 (15k chars in filter)
	reservationKeys := make([]string, len(reservationIDs))
	for i, reservationID := range reservationIDs {
		reservationKeys[i] = datamodel.GetReservationKey(networkID, reservationID)
	}

	entities, err := a.Allocations.QueryIPAllocationsByReservationKeys(networkID, subnetName, reservationKeys)
	if err != nil {
		return nil, err
	}

	for _, entity := range entities {
		ipAddressString := datamodel.DecodeIPAllocationKey(entity.RowKey)
		allocatedIPSubnet, err := utils.ParseIPNet(ipAddressString)
		if err != nil {
			return nil, api.InternalServerError(api.StatusInternalServerError, err.Error())
		}
		allocations[entity.AllocationID] = allocatedIPSubnet
	}

	return allocations, nil
}

func (a *IPAllocator) markReservationAsFailed(req datamodel.ReservationRow, err *api.JsonErr) (bool, *api.JsonErr) {
	req.Configuration.Status = datamodel.Failed
	req.Configuration.FailureDetail = *err
	a.Logger.Errorf("Failed to process reservation request {%s} due to error {%s}", req.Configuration.String(), err.String())
	if err := a.Reservations.PersistIPReservation(req); err != nil {
		return false, err
	}
	return true, nil
}

func (a *IPAllocator) DeallocateIPs(res datamodel.ReservationConfiguration, allocationIDs []string) *api.JsonErr {
	allocIDsToDelete := make(map[string]struct{})
	for _, allocID := range allocationIDs {
		allocIDsToDelete[allocID] = struct{}{}
	}

	var rowKeys []string
	for allocID := range allocIDsToDelete {
		if ipAddress, found := res.SecondaryAllocations[allocID]; found {
			rowKey := datamodel.EncodeIPAllocationKey(ipAddress)
			rowKeys = append(rowKeys, rowKey)
		}
	}

	return a.DeleteAllocations(res.NetworkID, res.AllocationRequest.SubnetName, rowKeys)
}

func (a *IPAllocator) ClearReservationAllocations(networkID, subnetName, reservationKey string) *api.JsonErr {
	associatedAllocations, err := a.Allocations.QueryIPAllocationsByReservationKey(networkID, subnetName, reservationKey)
	if err != nil {
		return err
	}

	if len(associatedAllocations) == 0 {
		a.Logger.Debugf("no allocations found for vnet %s subnet %s reservation key %s", networkID, subnetName, reservationKey)
		return nil
	}

	for i, allocation := range associatedAllocations {
		allocation.IPAllocation = ""
		allocation.AllocationID = ""
		allocation.ToBeDeletedOwner = ""
		associatedAllocations[i] = allocation
	}

	return a.UpdateAllocations(networkID, subnetName, associatedAllocations)
}

func (a *IPAllocator) DeleteReservationAllocations(networkID, subnetName, reservationKey string) *api.JsonErr {
	associatedAllocations, err := a.Allocations.QueryIPAllocationsByReservationKey(networkID, subnetName, reservationKey)
	if err != nil {
		return err
	}

	if len(associatedAllocations) == 0 {
		a.Logger.Debugf("no allocations found for vnet %s subnet %s reservation key %s", networkID, subnetName, reservationKey)
		return nil
	}

	keys := make([]string, len(associatedAllocations))
	for i, allocation := range associatedAllocations {
		keys[i] = allocation.RowKey
	}

	return a.DeleteAllocations(networkID, subnetName, keys)
}

func (a *IPAllocator) DeleteAllocations(networkID, subnetName string, keys []string) *api.JsonErr {
	if len(keys) == 0 {
		a.Logger.Debugf("DeleteAllocations: no allocations to delete")
		return nil
	}
	var batches [][]string
	for len(keys) > a.MaxBatchSize {
		keys, batches = keys[a.MaxBatchSize:], append(batches, keys[0:a.MaxBatchSize])
	}
	batches = append(batches, keys)

	for i := 0; i < len(batches); i++ {
		if err := a.Allocations.DeleteIPAllocations(networkID, subnetName, batches[i]); err != nil {
			return err
		}
	}
	return nil
}

func (a *IPAllocator) DeleteAllocationsWithEtag(partitionKey string, entities []datamodel.IPAllocationEntity) *api.JsonErr {
	if len(entities) == 0 {
		a.Logger.Debugf("DeleteAllocations: no allocations to delete for partitionKey %s", partitionKey)
		return nil
	}

	// todo: Make it batch delete
	for _, entity := range entities {
		err := a.Allocations.DeleteIPAllocationEntityWithEtag(partitionKey, entity)
		if err != nil {
			a.Logger.Errorf("%s: %s", utils.TracePrefix(2), err)
			return err
		}
	}

	return nil
}

// todo: handle batch updates directly in the Repo* functions instead of creating a wrapper
func (a *IPAllocator) UpdateAllocations(networkID, subnetName string, entities []datamodel.IPAllocationEntity) *api.JsonErr {
	var batches [][]datamodel.IPAllocationEntity
	for len(entities) > a.MaxBatchSize {
		entities, batches = entities[a.MaxBatchSize:], append(batches, entities[0:a.MaxBatchSize])
	}

	batches = append(batches, entities)
	for i := 0; i < len(batches); i++ {
		if err := a.Allocations.UpdateIPAllocationEntities(networkID, subnetName, batches[i]); err != nil {
			return err
		}
	}
	return nil
}

func (a *IPAllocator) GetIPAllocationsWithToBeDeletedOwner(partitionKey string, nckey string) ([]datamodel.IPAllocationEntity, *api.JsonErr) {
	return a.Allocations.QueryIPAllocationsByToBeDeletedOwner(partitionKey, nckey)
}

func (a *IPAllocator) saveAllocationBatch(vnetID, subnetName string, batchMap map[string]map[string]interface{}) *api.JsonErr {
	var batches []map[string]map[string]interface{}

	batch := make(map[string]map[string]interface{})
	for k, v := range batchMap {
		batch[k] = v
		if len(batch) == a.MaxBatchSize {
			batches = append(batches, batch)
			batch = make(map[string]map[string]interface{})
		}
	}
	if len(batch) > 0 {
		batches = append(batches, batch)
	}

	for i := 0; i < len(batches); i++ {
		if err := a.Allocations.AddIPAllocations(vnetID, subnetName, batches[i]); err != nil {
			return err
		}
	}
	return nil
}

func (a *IPAllocator) SetToBeDeleteOwnerToDeletingIps(networkId, subnetName, reservationKey, ncKey string, allocIDsToDelete map[string]string) *api.JsonErr {
	if len(allocIDsToDelete) == 0 {
		a.Logger.Debugf("SetToBeDeleteOwnerToDeletingIps: Skip update as there are no allocations to delete for Reservation %s, ncKey %s, NetworkId %s, Subnet %s",
			reservationKey,
			ncKey,
			networkId,
			subnetName)
		return nil
	}

	// Query the IpAllocations reserved for a given reservationKey (aka IpAllocationId)
	ipAllocationEntities, err := a.Allocations.QueryIPAllocationsByReservationKey(networkId, subnetName, reservationKey)
	if err != nil {
		return err
	}

	// Build a map for faster search
	var ipEntities = make(map[string]datamodel.IPAllocationEntity)
	for _, entity := range ipAllocationEntities {
		ipEntities[entity.AllocationID] = entity
	}

	var ipEntitiesMarkedForDelete []datamodel.IPAllocationEntity
	for allocationId := range allocIDsToDelete {
		if entity, found := ipEntities[allocationId]; found {
			entity.ToBeDeletedOwner = ncKey
			ipEntitiesMarkedForDelete = append(ipEntitiesMarkedForDelete, entity)
		}
	}

	// whether it's a conflict or a network error, background monitor will retry
	err = a.UpdateAllocations(networkId, subnetName, ipEntitiesMarkedForDelete)
	if err != nil {
		return err
	}

	a.Logger.Debugf("SetToBeDeleteOwnerToDeletingIps: Marked ToBeDeletedOwner for Ips, ReservationKey %s, ncKey (ToBeDeletedOwner) %s,ipEntitiesMarkedForDelete %+v",
		reservationKey, ncKey, ipEntitiesMarkedForDelete)

	return nil
}

// todo: make configurable
const prepopulateDefaultWorkerCount = 10

// PrepopulateSubnet will persist all rows for a given partition and subnet CIDR range.
// If the subnet is large enough, the operation may not be atomic since rows will need to be persisted in batches.
// Given the batch slicing strategy (batches for a given subnet should always contain the same elements, conflicts
// are handled), this operation is safe to retry.
func (a *IPAllocator) PrepopulateSubnet(vnetID, subnetName, subnetCIDR string) *api.JsonErr {
	logger := loggershim.WithPrefixf(a.Logger, "[prepopulate subnet] vnet %s subnet %s", vnetID, subnetName)

	_, subnet, err := net.ParseCIDR(subnetCIDR)
	if err != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError, "vnet ID %s subnet %s: could not parse subnet CIDR: %v", vnetID, subnetName, err)
	}

	subnetSize, _ := subnet.Mask.Size()

	allAllocs := GenerateIPAllocationEntitiesForSubnet(subnet)
	logger.Debugf("persisting %d rows for cidr %s", len(allAllocs), subnetCIDR)
	ch := a.rowBatchProducer(allAllocs)

	start := time.Now()
	rowsPersisted, err := a.rowBatchConsumer(ch, vnetID, subnetName, prepopulateDefaultWorkerCount)
	took := time.Since(start)
	if err != nil {
		go sendSubnetPrepopulationMetric(vnetID, subnetName, subnetSize, aiwrapper.Failed, took, rowsPersisted)
		return api.InternalServerError(api.StatusInternalServerError, err.Error())
	}

	go sendSubnetPrepopulationMetric(vnetID, subnetName, subnetSize, aiwrapper.Succeeded, took, rowsPersisted)
	return nil
}

func (a *IPAllocator) InsertAllocations(vnetID, subnetName string, allocations []datamodel.IPAllocationEntity, workerCount int) *api.JsonErr {
	ch := a.rowBatchProducer(allocations)

	if _, err := a.rowBatchConsumer(ch, vnetID, subnetName, workerCount); err != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError, "vnet ID %s subnet %s: could not persist subnet rows: %v", vnetID, subnetName, err)
	}

	return nil
}

func (a *IPAllocator) rowBatchProducer(rows []datamodel.IPAllocationEntity) chan []datamodel.IPAllocationEntity {
	out := make(chan []datamodel.IPAllocationEntity)

	go func() {
		defer close(out)
		var batch []datamodel.IPAllocationEntity
		for i, row := range rows {
			batch = append(batch, row)

			// send batch when it reaches max size or is the last one
			if len(batch) == a.MaxBatchSize || i == len(rows)-1 {
				out <- batch
				batch = nil
			}
		}
	}()

	return out
}

// todo: retrier needs a jitter mode, otherwise multiple goroutines using the same one will retry all at the same time
//  which can keep triggering throttling
var cosmosThrottleRetrier = retry.Retrier{
	Attempts:   5,
	Delay:      time.Second,
	ExpBackoff: true,
}

func (a *IPAllocator) rowBatchConsumer(in chan []datamodel.IPAllocationEntity, vnetID, subnetName string, workerCount int) (int, error) {
	var g sync.WaitGroup
	g.Add(workerCount)

	logger := loggershim.WithPrefixf(a.Logger, "[prepopulate subnet] vnet %s subnet %s", vnetID, subnetName)

	var (
		batchError    rowBatchError
		rowsPersisted int64
	)

	for i := 0; i < workerCount; i++ {
		go func() {
			defer g.Done()
			for entities := range in {
				firstKey := entities[0].RowKey // batches are always len > 0

				// errors returned by this closure will be retried
				retryableFunc := func() error {
					// first, cheap check to see if this batch has already been processed.
					alloc, err := a.Allocations.TryQueryIPAllocation(vnetID, subnetName, firstKey)
					if err != nil {
						return err
					}

					// if we get a valid result (row key is populated), then the batch has already been processed.
					if alloc.RowKey != "" {
						logger.Debugf("batch starting with key %s already persisted", firstKey)
						return nil
					}

					// attempt to persist, but handle conflict since it is still possible.
					if err := a.Allocations.InsertIPAllocationEntities(vnetID, subnetName, entities); err != nil {
						if err.HTTPStatus == http.StatusConflict {
							logger.Debugf("batch starting with key %s already exists", firstKey)
							return nil
						}
						return err
					}

					atomic.AddInt64(&rowsPersisted, int64(len(entities)))
					logger.Debugf("batch starting with key %s successfully persisted", firstKey)
					return nil
				}

				if err := cosmosThrottleRetrier.Do(context.TODO(), retryableFunc); err != nil {
					logger.Debugf("could not persist batch starting with key %s: %v", firstKey, err)
					batchError.addKey(firstKey)
				}
			}
		}()
	}

	g.Wait()

	if len(batchError.keys) > 0 {
		return int(rowsPersisted), &batchError
	}

	return int(rowsPersisted), nil
}

type rowBatchError struct {
	keys []string
	m    sync.Mutex
}

func (r *rowBatchError) Error() string {
	return fmt.Sprintf("could not process batches starting with keys: %+v", r.keys)
}

func (r *rowBatchError) addKey(key string) {
	r.m.Lock()
	r.keys = append(r.keys, key)
	r.m.Unlock()
}

// GenerateIPAllocationEntitiesForSubnet generates allocation rows for the given subnet.
// The maximum supported subnet mask is /16. For any larger subnets, at most 65536 (/16
// max host count) rows will be produced, including those reserved by Azure.
func GenerateIPAllocationEntitiesForSubnet(subnet *net.IPNet) []datamodel.IPAllocationEntity {
	ipCount := subnetMaxHostCount(subnet)
	if ipCount > slash16HostCount {
		ipCount = slash16HostCount
	}

	rows := make([]datamodel.IPAllocationEntity, ipCount)
	offset := int(utils.IP2Int(subnet.IP))

	lastIdx := ipCount - 1
	for i := 0; i < ipCount; i++ {
		ip := utils.Int2IP(uint32(i + offset))
		if i == lastIdx {
			ip = allocationhelper.GetLastAddressInSubnet(subnet)
		}

		row := datamodel.IPAllocationEntity{
			RowKey: ip.String() + "_32",
		}

		// first 4 addresses and last address in a subnet are reserved
		if i < allocationhelper.ReservedIPCount || i == lastIdx {
			row.IPAllocation = reservedAddress
			row.AllocationID = reservedAddress
		}

		rows[i] = row
	}

	return rows
}

func (a *IPAllocator) DepopulateSubnet(vnetID, subnetName string) *api.JsonErr {
	logger := loggershim.WithPrefixf(a.Logger, "[depopulate subnet] vnet %s subnet %s", vnetID, subnetName)

	start := time.Now()
	// todo: consider processing each page of results as soon as fetched, since this function call can be slow for large subnets
	remainingAllocs, err := a.Allocations.TryQueryAllIPAllocations(vnetID, subnetName)
	if err != nil {
		return err
	}

	if len(remainingAllocs) == 0 {
		logger.Debugf("no allocations to delete")
		return nil
	}

	logger.Debugf("deleting %d rows", len(remainingAllocs))

	inputCh := a.rowBatchProducer(remainingAllocs)

	var (
		errCount    int64
		wg          sync.WaitGroup
		deletedRows int64
	)

	workerCount := 5
	wg.Add(workerCount)

	for i := 0; i < workerCount; i++ {
		go func() {
			defer wg.Done()
			for rows := range inputCh {
				keys := getKeys(rows)
				lenKeys := len(keys)

				if err := a.Allocations.DeleteIPAllocations(vnetID, subnetName, keys); err != nil {
					logger.Errorf("could not delete batch size %d: %v", lenKeys, err)
					atomic.AddInt64(&errCount, int64(lenKeys))
					continue
				}

				atomic.AddInt64(&deletedRows, int64(lenKeys))
				logger.Debugf("deleted batch size %d", lenKeys)
			}
		}()
	}

	wg.Wait()
	took := time.Since(start)

	if errCount > 0 {
		msg := fmt.Sprintf("could not delete %d out of %d allocations remaining in vnet %s subnet %s", errCount, len(remainingAllocs), vnetID, subnetName)
		go sendSubnetDepopulationMetric(vnetID, subnetName, aiwrapper.Failed, took, int(deletedRows))
		return api.InternalServerError(api.StatusInternalServerError, msg)
	}

	go sendSubnetDepopulationMetric(vnetID, subnetName, aiwrapper.Succeeded, took, int(deletedRows))
	return nil
}

func getKeys(rows []datamodel.IPAllocationEntity) []string {
	keys := make([]string, len(rows))
	for i, row := range rows {
		keys[i] = row.RowKey
	}
	return keys
}

// DeleteIPReservation - Delete an IP Reservation.
func (a *IPAllocator) DeleteIPReservation(reservation datamodel.ReservationRow, force bool) *api.JsonErr {
	var err *api.JsonErr
	if reservation.Configuration == nil {
		return nil
	}

	reservationKey := reservation.RowKey
	reservationID := datamodel.GetReservationIDFromKey(reservationKey)
	if reservation.Configuration.AssociatedNCID != "" && !force {
		return api.BadRequestf(api.ReservationInUse,
			"Reservation for key: %s is in use with container: %s", reservationKey, reservation.Configuration.AssociatedNCID)
	}

	if a.PubsubProxyEnabled && len(reservation.Configuration.AssociatedNCID) != 0 {
		a.Logger.Debugf("DeleteIPReservation: Set ToBeDeletedOwner for Primary and Secondary Ips (if any), ReservationKey %s, NCID %s",
			reservationKey, reservation.Configuration.AssociatedNCID)

		// Update IP Allocations Table with ToBeDeletedOwner ID, where ID is the NCID
		// Note: don't delete the ToBeDeletedIP's from the NC here, only updating the IPAllocations table
		var ipsPendingCleanup = make(map[string]string)

		// It is okay if the associated ips are empty, it will still get cleaned up by the background cleanup monitor
		if len(reservation.Configuration.SecondaryAllocations) > 0 {
			ipsPendingCleanup = reservation.Configuration.SecondaryAllocations
		}

		// Also mark ToBeDeletedOwner for primary IP.
		ipsPendingCleanup[reservationID] = reservation.Configuration.AllocatedSubnet

		ncKey := datamodel.GetNCKey(reservation.Configuration.NetworkID, reservation.Configuration.AssociatedNCID)
		err = a.SetToBeDeleteOwnerToDeletingIps(
			reservation.Configuration.NetworkID,
			reservation.Configuration.AllocationRequest.SubnetName,
			reservationKey,
			ncKey,
			ipsPendingCleanup)
		if err != nil {
			a.Logger.Errorf("DeleteIPReservation: Failed to mark ToBeDeletedOwner for associated ips, reservationKey %s, err %+v",
				reservationKey, err)
			return err
		}

		err = a.CreateIpCleanupRequest(
			reservation.Configuration.NetworkID,
			reservation.Configuration.AllocationRequest.SubnetName,
			reservation.Configuration.AssociatedNCID)
		if err != nil {
			return err
		}
	} else {
		// Delete IPs from IpAllocations table and clean the Reservation table
		// No need to wait for Pubsub job
		vnetID := reservation.Configuration.NetworkID
		subnetName := reservation.Configuration.AllocationRequest.SubnetName

		if a.UsePrepopulatedAllocations {
			if err := a.ClearReservationAllocations(vnetID, subnetName, reservationKey); err != nil {
				a.Logger.Errorf("could not clear allocations for vnet %s subnet %s reservation key %s: %v", vnetID, subnetName, reservationKey, err)
				return err
			}
		} else {
			if err := a.DeleteReservationAllocations(vnetID, subnetName, reservationKey); err != nil {
				a.Logger.Errorf("could not delete allocations for vnet %s subnet %s reservation key %s: %v", vnetID, subnetName, reservationKey, err)
				return err
			}
		}
	}

	// Delete the reservation from subnetConfig if the config flag to disable reservationMap is not set
	if !a.DisableResMapInSubnetConfig {
		subnetConfiguration, err := a.Subnets.TryQuerySubnetConfiguration(
			reservation.Configuration.NetworkID,
			reservation.Configuration.AllocationRequest.SubnetName)
		if err != nil {
			return err
		}

		if subnetConfiguration != nil && subnetConfiguration.Reservations != nil {
			delete(subnetConfiguration.Reservations, reservationID)
			if err = a.Subnets.PutSubnetConfiguration(subnetConfiguration); err != nil {
				return err
			}
		}
	}

	if len(reservation.Configuration.AllocationRequest.NodeConstraint) != 0 {
		nodeConfiguration, err := a.Nodes.TryQueryNodeConfiguration(reservation.Configuration.AllocationRequest.NodeConstraint)
		if err != nil {
			return err
		}
		if nodeConfiguration != nil {
			err := a.Nodes.UpdateNodeConstrainedReservationSet(
				reservation.Configuration.NetworkID,
				reservation.Configuration.ReservationID,
				true,
				nodeConfiguration)
			if err != nil {
				return err
			}
		}
	}

	return a.Reservations.DeleteReservation(reservationKey)
}

// Creates the PendingCleanup Request for ips which are marked deleted for a given NcId (ToBeDeletedOwner)
func (a *IPAllocator) CreateIpCleanupRequest(networkId, subnetName, ncId string) *api.JsonErr {
	// This API is called during Unreserve or Secondary IP scale down state
	// Always fetch again all the ips which are marked with ToBeDeletedOwner as there can be a race if a delete
	// NC is received while ScaleDown is getting processed. During scale down, we could have set TBD for a deleting IP
	// and removed it from Reservation but missed creating the CleanupRequest
	// ToBeDeletedOwner is always NCKey
	ncKey := datamodel.GetNCKey(networkId, ncId)
	subnetKey := datamodel.GetSubnetKey(networkId, subnetName)
	ipEntitiesMarkedForDelete, err := a.Allocations.QueryIPAllocationsByToBeDeletedOwner(subnetKey, ncKey)
	if err != nil {
		a.Logger.Errorf("CreateIpCleanupRequest, failed to QueryIPAllocationsByToBeDeletedOwner for networkId %s, subnetName %s, tobeDeletedOwner %s, err %+v",
			networkId, subnetName, ncKey, err)
		return err
	}

	// Create the BG Job to delete AllocatedIPs only after the associated unpublish
	// task (Created in the unpublish state) is completed
	if len(ipEntitiesMarkedForDelete) > 0 {
		err = a.PendingCleanupRepo.CreateOrMergeIPResourceCleanupRequestFromIPAllocations(
			networkId,
			subnetName,
			ncId,
			ipEntitiesMarkedForDelete)
		if err != nil {
			a.Logger.Errorf("CreateIpCleanupRequest: failed to create IPCleanupRequest for networkId %s, subnetName %s, ncId %s, err %+v",
				networkId, subnetName, ncId, err)
			return err
		}
	}

	return nil
}

// todo: these metrics are not representative of total processing time as seen by client. once they become
//  background tasks, multiple retries can be taken into account.
func sendSubnetPrepopulationMetric(vnetID, subnetName string, maskSize int, status string, duration time.Duration, persisted int) {
	metric := baseSubnetAllocMetric(vnetID, subnetName, status, duration)
	metric.CustomDimensions[aiwrapper.SubnetAllocationOpStr] = prepopulateOp
	metric.CustomDimensions[aiwrapper.AllocationsPersistedStr] = strconv.Itoa(persisted)
	metric.CustomDimensions[aiwrapper.SubnetSizeStr] = strconv.Itoa(maskSize)

	aiwrapper.SendMetric(metric)
}

func sendSubnetDepopulationMetric(vnetID, subnetName string, status string, duration time.Duration, removed int) {
	metric := baseSubnetAllocMetric(vnetID, subnetName, status, duration)
	metric.CustomDimensions[aiwrapper.SubnetAllocationOpStr] = depopulateOp
	metric.CustomDimensions[aiwrapper.AllocationsRemovedStr] = strconv.Itoa(removed)

	aiwrapper.SendMetric(metric)
}

func baseSubnetAllocMetric(vnetID, subnetName string, status string, duration time.Duration) aitelemetry.Metric {
	return aitelemetry.Metric{
		Name:  aiwrapper.SubnetAllocationsProcessTimeMetric,
		Value: float64(duration.Milliseconds()),
		CustomDimensions: map[string]string{
			aiwrapper.NetworkIDStr: vnetID,
			aiwrapper.SubNetStr:    subnetName,
			aiwrapper.StatusStr:    status,
		},
	}
}
