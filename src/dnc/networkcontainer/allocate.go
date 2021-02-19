package networkcontainer

import (
	"context"
	"dnc/allocationhelper"
	"dnc/api"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/retry"
	"dnc/utils"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Azure/azure-container-networking/cns"
)

const (
	reservedAddress = "Azure Reserved Address"
)

// IPAllocator can process different types of IP allocations for network containers
type IPAllocator struct {
	Nodes        NodeRepo
	NCs          NCRepo
	Subnets      SubnetRepo
	Reservations IPReservationRepo
	Allocations  IPAllocationRepo
	Logger       loggershim.LoggerInterface
	MaxBatchSize int
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

	return a.ProcessIPReservationAllocation(reservation.Configuration)
}

// ProcessIPReservationAllocation processes IP allocation for the given reservation
// returns whether ip allocation is complete, and any errors in processing
func (a *IPAllocator) ProcessIPReservationAllocation(req *datamodel.ReservationConfiguration) (bool, *api.JsonErr) {
	subnetConfig, err := a.Subnets.TryQuerySubnetConfiguration(req.NetworkID, req.AllocationRequest.SubnetName)
	if err != nil {
		return false, err
	}
	if subnetConfig == nil {
		err := api.NotFound(api.SubnetNotFound, fmt.Sprintf("Failed to find subnet for allocation request %s", req.AllocationRequest.SubnetName))
		return a.markReservationAsFailed(req, err)
	}

	allocationState, err := a.getAllocationsFromSubnetTable(req.NetworkID, req.AllocationRequest.SubnetName)
	if err != nil {
		if err.HTTPStatus == http.StatusNotFound {
			return a.markReservationAsFailed(req, err)
		}
		return false, nil
	}

	isComplete := reconcileReservation(req, allocationState)
	if isComplete {
		return a.completeAllocation(req, subnetConfig)
	}

	baseSubnet, parseErr := utils.ParseIPNet(subnetConfig.IPSubnet)
	if parseErr != nil {
		err := api.InternalServerError(api.StatusInternalServerError, parseErr.Error())
		return a.markReservationAsFailed(req, err)
	}

	addressesInUse, err := a.getAllAddressesInUse(baseSubnet, allocationState, req)
	if err != nil {
		if err.HTTPStatus == http.StatusNotFound {
			return a.markReservationAsFailed(req, err)
		}
		return false, nil
	}

	var primaryIP *net.IPNet
	if req.AllocatedSubnet != "" {
		_, primaryIP, parseErr = net.ParseCIDR(req.AllocatedSubnet)
		if parseErr != nil {
			return a.markReservationAsFailed(req, api.InternalServerError(api.StatusInternalServerError, parseErr.Error()))
		}
	} else {
		primaryIP, err = determinePrimaryIP(req, baseSubnet, addressesInUse)
		if err != nil {
			return a.markReservationAsFailed(req, err)
		}
	}

	if primaryIP == nil {
		err := api.BadRequest(api.AllocationFailed, fmt.Sprintf("Failed to allocate address for request %s. Subnet is full", req.AllocationRequest.String()))
		return a.markReservationAsFailed(req, err)
	}

	// whether it's a single record or multiple, we can still use a batch for a single transaction
	// reservation key is the same for all rows of the batch
	batch := make(map[string]map[string]interface{})
	reservationKey := datamodel.GetReservationKey(req.NetworkID, req.ReservationID)

	// we only add the primary ip for the reservation if it hadn't been found previously
	if req.AllocatedSubnet == "" {
		rowKey := datamodel.EncodeIPAllocationKey(primaryIP.String())
		batch[rowKey] = map[string]interface{}{datamodel.IPAllocationProperty: reservationKey, datamodel.AllocationIDProperty: req.ReservationID}
	}

	// we keep track of all secondary allocations that have not been assigned an IP yet
	// we only find IPs for these, and as the IPAllocation record is built and added to the batch, the value will be populated
	// so we can merge back into the original reservation only after successful allocation
	unallocated := make(map[string]string)
	for allocationID, ipAddress := range req.SecondaryAllocations {
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
				return false, api.BadRequest(api.AllocationFailed, fmt.Sprintf("Failed to allocate address for request %s. Subnet is full", req.AllocationRequest.String()))
			}
			return false, api.InternalServerError(api.AllocationFailed, allocErr.Error())
		}
		i := 0
		for allocationID := range unallocated {
			encodedSubnet := availableIPs[i].String() + utils.PrefixDelimiter + utils.IPV4DefaultPrefix
			unallocated[allocationID] = encodedSubnet
			rowKey := datamodel.EncodeIPAllocationKey(encodedSubnet)
			batch[rowKey] = map[string]interface{}{datamodel.IPAllocationProperty: reservationKey, datamodel.AllocationIDProperty: allocationID}
			i++
		}
	}

	if err := a.Allocations.AddIPAllocations(req.NetworkID, req.AllocationRequest.SubnetName, batch); err != nil {
		// whether it's a conflict or a network error, we should retry
		return false, err
	}

	// once successful, mutate the original reservation with new state
	if req.AllocatedSubnet == "" {
		req.AllocatedSubnet = primaryIP.String()
	}
	// no-op if unallocated is empty
	for allocationID, ipAddress := range unallocated {
		req.SecondaryAllocations[allocationID] = ipAddress
	}

	return a.completeAllocation(req, subnetConfig)
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
		ncConfig, err := a.NCs.TryQueryNetworkContainer(req.NetworkID, req.AssociatedNCID)
		if err != nil {
			return nil, err
		}
		if ncConfig == nil {
			return nil, api.NotFound(api.NetworkContainerNotFound, fmt.Sprintf("Failed to find configuration for network container %s", req.AssociatedNCID))
		}

		nodeID := ncConfig.GetAssociationInfo().NodeID
		// This needs to be fixed for scenarios where we are using the reserve IP
		// API to dump allocations on a node and orchestrator is WebApps.
		if nodeID != "" && ncConfig.GetAssociationInfo().ContainerType == cns.WebApps {
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

func (a *IPAllocator) completeAllocation(request *datamodel.ReservationConfiguration, subnetConfig *datamodel.SubnetConfiguration) (bool, *api.JsonErr) {
	request.Status = datamodel.Completed

	if request.AllocationRequest.NodeConstraint != "" {
		err := a.Nodes.UpdateNodeConstrainedReservationSetByNodeID(request.AllocationRequest.NodeConstraint, request.NetworkID, request.ReservationID, false)
		if err != nil {
			err.Text = fmt.Sprintf("Failed to associate reservation request %s to node %s due to error %s", request.String(), request.AllocationRequest.NodeConstraint, err.String())
			return false, err
		}
	}

	if err := a.Subnets.UpdateSubnetReservationSet(request.ReservationID, request.AllocatedSubnet, false, subnetConfig); err != nil {
		err.Text = fmt.Sprintf("Failed to persist updating reservation request %s in subnet configuration due to error %s", request.String(), err.String())
		if err.Code == api.PropertyValueTooLargeStorageError {
			err := api.BadRequest(api.FailedToUpdateSubnetWithReservationInfo, err.Text)
			return a.markReservationAsFailed(request, err)
		}

		return false, err
	}

	if err := a.Reservations.PutIPReservationConfiguration(request); err != nil {
		err.Text = fmt.Sprintf("Failed to mark reservation request %s as succeded due to error %s", request.String(), err.String())
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

func (a *IPAllocator) markReservationAsFailed(req *datamodel.ReservationConfiguration, err *api.JsonErr) (bool, *api.JsonErr) {
	req.Status = datamodel.Failed
	req.FailureDetail = *err
	a.Logger.Errorf("Failed to process reservation request {%s} due to error {%s}", req.String(), err.String())
	if err := a.Reservations.PutIPReservationConfiguration(req); err != nil {
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

func (a *IPAllocator) RemoveAllAllocations(res datamodel.ReservationConfiguration) *api.JsonErr {
	var (
		rowKeys  []string
		allocIDs []string
	)

	for allocID, ipAddress := range res.SecondaryAllocations {
		if ipAddress != "" {
			rowKey := datamodel.EncodeIPAllocationKey(ipAddress)
			rowKeys = append(rowKeys, rowKey)
			continue
		}
		allocIDs = append(allocIDs, allocID)
	}

	// this is needed in cases where allocations were persisted correctly, but not back to the reservation
	// we make sure that any allocation ids associated to the reservation don't have lingering records
	if len(allocIDs) > 0 {
		allocRows, err := a.Allocations.QueryIPAllocationsByAllocationID(res.NetworkID, res.AllocationRequest.SubnetName, allocIDs)
		if err != nil {
			return err
		}
		for _, row := range allocRows {
			rowKeys = append(rowKeys, row.RowKey)
		}
	}

	return a.DeleteAllocations(res.NetworkID, res.AllocationRequest.SubnetName, rowKeys)
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

func (a *IPAllocator) DeleteAllocationsWithEtag(entities []datamodel.IPAllocationEntity) *api.JsonErr {
	if len(entities) == 0 {
		a.Logger.Debugf("DeleteAllocations: no allocations to delete")
		return nil
	}

	for _, entity := range entities {
		err := a.Allocations.DeleteIPAllocationEntityWithEtag(entity)
		if err != nil {
			a.Logger.Errorf("%s: %s", utils.TracePrefix(2), err)
			return err
		}
	}

	return nil
}

// clearAllocations will clear all properties for the given allocation ids
func (a *IPAllocator) clearAllocations(networkID, subnetName string, allocationIDs []string) *api.JsonErr {
	// todo: if the size of allocation ids is large enough, the method below could fail due to a large query filter generated.
	//  need to determine what the maximum size for allocation ids should be and create batches.
	entities, err := a.Allocations.QueryIPAllocationsByAllocationID(networkID, subnetName, allocationIDs)
	if err != nil {
		return err
	}

	if len(entities) == 0 {
		a.Logger.Debugf("vnet ID %s subnet %s: the following allocation ids were not found in allocation table: %+v", networkID, subnetName, allocationIDs)
		return nil
	}

	// will retain etags
	for i := range entities {
		entities[i].IPAllocation = ""
		entities[i].AllocationID = ""
		entities[i].ToBeDeletedOwner = ""
	}

	return a.UpdateAllocations(networkID, subnetName, entities)
}

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

func (a *IPAllocator) GetIPAllocationsWithToBeDeletedOwner(networkID, subnetName, ncid string) ([]datamodel.IPAllocationEntity, *api.JsonErr) {
	return a.Allocations.QueryIPAllocationsByToBeDeletedOwner(networkID, subnetName, ncid)
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

func (a *IPAllocator) MarkAllocationAsPendingCleanup(res datamodel.ReservationConfiguration, allocationIDs []string) *api.JsonErr {
	batch := make(map[string]map[string]interface{})

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

	// need to update the table with the NCID in the toBeDeletedProperty column
	for _, rowKey := range rowKeys {
		batch[rowKey] = map[string]interface{}{datamodel.ToBeDeletedOwnerProperty: res.AssociatedNCID}
	}

	// whether it's a conflict or a network error, we should retry
	return a.Allocations.UpdateIPAllocations(res.NetworkID, res.AllocationRequest.SubnetName, batch)
}

// PrepopulateSubnet will persist all rows for a given partition and subnet CIDR range.
// If the subnet is large enough, the operation may not be atomic since rows will need to be persisted in batches.
// Given the batch slicing strategy (batches for a given subnet should always contain the same elements, conflicts
// are handled), this operation is safe to retry.
func (a *IPAllocator) PrepopulateSubnet(vnetID, subnetName, subnetCIDR string) *api.JsonErr {
	_, subnet, err := net.ParseCIDR(subnetCIDR)
	if err != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError, "vnet ID %s subnet %s: could not parse subnet CIDR: %v", vnetID, subnetName, err)
	}

	allAllocs, err := generateIPAllocationEntitiesForSubnet(subnet)
	if err != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError, "vnet ID %s subnet %s: could not generate subnet rows: %v", vnetID, subnetName, err)
	}

	ch := a.rowBatchProducer(allAllocs)

	if err := a.rowBatchConsumer(ch, vnetID, subnetName); err != nil {
		return api.InternalServerError(api.StatusInternalServerError, err.Error())
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

func (a *IPAllocator) rowBatchConsumer(in chan []datamodel.IPAllocationEntity, vnetID, subnetName string) error {
	workerCount := 10
	var g sync.WaitGroup
	g.Add(workerCount)

	var batchError rowBatchError
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
						a.Logger.Debugf("batch starting with key %s already persisted", firstKey)
						return nil
					}

					// attempt to persist, but handle conflict since it is still possible.
					if err := a.Allocations.InsertIPAllocationEntities(vnetID, subnetName, entities); err != nil {
						if err.HTTPStatus == http.StatusConflict {
							a.Logger.Debugf("batch starting with key %s already exists", firstKey)
							return nil
						}
						return err
					}

					return nil
				}

				if err := cosmosThrottleRetrier.Do(context.TODO(), retryableFunc); err != nil {
					a.Logger.Errorf("could not persist batch starting with key %s: %v", firstKey, err)
					batchError.addKey(firstKey)
				}
			}
		}()
	}

	g.Wait()

	if len(batchError.keys) > 0 {
		return &batchError
	}

	return nil
}

type rowBatchError struct {
	keys []string
	m    sync.Mutex
}

func (r *rowBatchError) Error() string {
	return fmt.Sprintf("could not persist batches starting with keys: %+v", r.keys)
}

func (r *rowBatchError) addKey(key string) {
	r.m.Lock()
	r.keys = append(r.keys, key)
	r.m.Unlock()
}

func generateIPAllocationEntitiesForSubnet(subnet *net.IPNet) ([]datamodel.IPAllocationEntity, error) {
	subnetBlockSize := subnetMaxHostCount(subnet)

	loIPBound := int(utils.IP2Int(subnet.IP))
	hiIPBound := loIPBound + subnetBlockSize - 1

	reserved, err := allocationhelper.GetReservedAddressesInSubnet(subnet)
	if err != nil {
		return nil, err
	}

	var rows []datamodel.IPAllocationEntity
	for i := loIPBound; i <= hiIPBound; i++ {
		ip := utils.Int2IP(uint32(i)).String() + "/32"
		rowKey := datamodel.EncodeIPAllocationKey(ip)
		row := datamodel.IPAllocationEntity{RowKey: rowKey}

		if _, isReserved := reserved[ip]; isReserved {
			row.IPAllocation = reservedAddress
			row.AllocationID = reservedAddress
		}

		rows = append(rows, row)
	}

	return rows, nil
}
