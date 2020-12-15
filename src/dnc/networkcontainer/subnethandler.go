package networkcontainer

import (
	"context"
	"dnc/allocationhelper"
	"dnc/api"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/utils"
	"fmt"
	"net"
	"time"
)

// subnetHandler processes batches of reservation requests for a given vnet and subnet name
type subnetHandler struct {
	reservations IPReservationRepo
	subnets      SubnetRepo
	allocator    *IPAllocator
	logger       loggershim.LoggerInterface
	inputCh      chan reservationRequest
	subnetInfo   subnetInfo
	maxBatchSize int
	locker       *utils.SessionManager
}

type subnetKey struct {
	vnetID     string
	subnetName string
}

type subnetInfo struct {
	vnetID     string
	subnetName string
	baseSubnet *net.IPNet
}

type handlerDependencies struct {
	reservations IPReservationRepo
	subnets      SubnetRepo
	allocator    *IPAllocator
	logger       loggershim.LoggerInterface
	locker       *utils.SessionManager
}

func newSubnetHandler(info subnetInfo, deps handlerDependencies, maxBatchSize int) *subnetHandler {
	return &subnetHandler{
		reservations: deps.reservations,
		subnets:      deps.subnets,
		allocator:    deps.allocator,
		logger:       deps.logger,
		inputCh:      make(chan reservationRequest),
		subnetInfo:   info,
		maxBatchSize: maxBatchSize,
		locker:       deps.locker,
	}
}

func (s *subnetHandler) enqueue(req reservationRequest) {
	// note: enqueueing order is not deterministic. when multiple concurrent sends to the input channel are being
	// blocked, there is no guarantee that they will be consumed in order. in practice, this should be negligible
	// as the scheduler seems to be biased towards older goroutines. if behavior changes drastically in the future,
	// we can consider another data structure to collect work.
	s.inputCh <- req
}

// processAndBatch processes incoming messages, only batching while processing is ongoing
func (s *subnetHandler) processAndBatch(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-s.inputCh:
			processCtx, signalProcessDone := context.WithCancel(ctx)

			// any message will immediately trigger processing
			go func() {
				single := make(requestSet)
				single.put(req)
				s.process(single)
				signalProcessDone()
			}()

			// we start collecting the next batch asynchronously
			batchCh := s.nextBatch(processCtx)

			for {
				<-processCtx.Done()    // we wait until the current processing is done to start next batch
				nextBatch := <-batchCh // we always expect to receive a single set (empty or not) from this channel
				if nextBatch.len() == 0 {
					break // if there were no messages during previous execution, wait for next message
				}

				// otherwise, start collecting again and process current batch
				processCtx, signalProcessDone = context.WithCancel(ctx)
				batchCh = s.nextBatch(processCtx)

				s.process(nextBatch)
				signalProcessDone()
			}
		}
	}
}

func (s *subnetHandler) nextBatch(ctx context.Context) chan requestSet {
	out := make(chan requestSet)
	set := make(requestSet)
	go func() {
		defer func() { out <- set }()

		for {
			select {
			case <-ctx.Done():
				return
			case req := <-s.inputCh:
				set.put(req)
				if set.len() >= s.maxBatchSize {
					return
				}
			}
		}
	}()

	return out
}

func (s *subnetHandler) logf(format string, args ...interface{}) {
	s.logger.Debugf("[subnet handler] vnet %s subnet %s - %s", s.subnetInfo.vnetID, s.subnetInfo.subnetName, fmt.Sprintf(format, args...))
}

func (s *subnetHandler) logErrorf(format string, args ...interface{}) {
	s.logger.Errorf("[subnet handler] vnet %s subnet %s - %s", s.subnetInfo.vnetID, s.subnetInfo.subnetName, fmt.Sprintf(format, args...))
}

// todo: batch processing is only enabled for managed DNC.
//  as we onboard more scenarios onto mDNC, we also need to port certain functionality such as
//  reading allocations from nodes, which is specific to WebApps.
func (s *subnetHandler) process(batch requestSet) {
	start := time.Now()

	s.logf("processing reservation batch of %d. start time %v", batch.len(), start.String())

	reservations := s.cleanUpTBDs(batch)

	// todo: retry loop over read/reconcile/compute/store, which must consider partial successes on previous attempt
	allocResult, err := s.allocate(reservations)
	if err != nil {
		s.logErrorf("could not get allocations from subnet table: %v", err)
		batch.flushWithError(err)
		return
	}

	if len(allocResult.rowBatch) > 0 {
		s.logf("persisting %d allocations", len(allocResult.rowBatch))
		if err := s.allocator.saveAllocationBatch(s.subnetInfo.vnetID, s.subnetInfo.subnetName, allocResult.rowBatch); err != nil {
			s.logErrorf("could not save allocation batch: %v", err)
			batch.flushWithError(err)
			return
		}
	}

	s.logf("updating subnet with %d completed reservations", len(allocResult.completed))
	if err := s.updateSubnet(allocResult.completed); err != nil {
		s.logErrorf("could not update subnet: %v", err)
		batch.flushWithError(err)
		return
	}

	s.logf("updating reservations")
	reservations = append(allocResult.completed, allocResult.failed...)
	if err := s.reservations.UpdateReservations(reservations); err != nil {
		s.logErrorf("could not update reservations: %v", err)
		batch.flushWithError(err)
		return
	}

	batch.complete(reservations)
	s.logf("batch completed in %s", time.Since(start).String())
}

type batchAllocationResult struct {
	completed []datamodel.ReservationConfiguration
	failed    []datamodel.ReservationConfiguration
	rowBatch  map[string]map[string]interface{}
}

func (r *batchAllocationResult) reservationComplete(res datamodel.ReservationConfiguration) {
	res.Status = datamodel.Completed
	r.completed = append(r.completed, res)
}

func (r *batchAllocationResult) reservationFailed(res datamodel.ReservationConfiguration, err api.JsonErr) {
	res.Status = datamodel.Failed
	res.FailureDetail = err
	r.failed = append(r.failed, res)
}

func (s *subnetHandler) allocate(reservations []datamodel.ReservationConfiguration) (batchAllocationResult, error) {
	allocReadTime := time.Now()

	s.logf("getting allocation state")
	allocationState, err := s.allocator.getAllocationsFromSubnetTable(s.subnetInfo.vnetID, s.subnetInfo.subnetName)
	if err != nil {
		return batchAllocationResult{}, err
	}

	s.logf("fetched %d allocation rows in %s", len(allocationState), time.Since(allocReadTime).String())

	var result batchAllocationResult
	result.rowBatch = make(map[string]map[string]interface{})

	for i := 0; i < len(reservations); i++ {
		res := reservations[i]
		isComplete := reconcileReservation(&res, allocationState)

		if isComplete {
			// if complete, add to completed list and remove from reservations that still need to be processed
			result.reservationComplete(res)
			reservations = append(reservations[:i], reservations[i+1:]...)
			i--
			continue
		}

		// otherwise, add the reconciled state to process list
		reservations[i] = res
	}

	addressesInUse, _ := getAllAddressesInUse(s.subnetInfo.baseSubnet, allocationState)

	s.logf("%d addresses in use in %s", len(addressesInUse), s.subnetInfo.baseSubnet.String())

	var ipsInUse []net.IP
	for _, ipAddress := range addressesInUse {
		ipsInUse = append(ipsInUse, ipAddress)
	}

	s.logf("determining allocations")
	for _, reservation := range reservations {
		var primaryIP net.IP
		if reservation.AllocatedSubnet != "" {
			primIP, _, err := net.ParseCIDR(reservation.AllocatedSubnet)
			if err != nil {
				result.reservationFailed(reservation, *api.InternalServerError(api.StatusInternalServerError, err.Error()))
				continue
			}
			primaryIP = primIP
		} else {
			ips, err := findAllocatableIPs(s.subnetInfo.baseSubnet, ipsInUse, 1)
			if err != nil {
				result.reservationFailed(reservation, *api.InternalServerError(api.StatusInternalServerError, err.Error()))
				continue
			}
			primaryIP = ips[0]
		}

		if _, ok := addressesInUse[primaryIP.String()]; !ok {
			ipsInUse = append(ipsInUse, primaryIP)
			addressesInUse[primaryIP.String()] = primaryIP
		}

		reservationKey := datamodel.GetReservationKey(reservation.NetworkID, reservation.ReservationID)

		if reservation.AllocatedSubnet == "" {
			reservation.AllocatedSubnet = primaryIP.String() + "/32"

			rowKey := datamodel.EncodeIPAllocationKey(reservation.AllocatedSubnet)
			result.rowBatch[rowKey] = map[string]interface{}{ipAllocProperty: reservationKey, allocIDProperty: reservation.ReservationID}
		}

		unallocated := make(map[string]struct{})
		for allocationID, ipAddress := range reservation.SecondaryAllocations {
			if ipAddress == "" {
				unallocated[allocationID] = struct{}{}
			}
		}

		if len(unallocated) == 0 {
			result.reservationComplete(reservation)
			continue
		}

		availableIPs, err := findAllocatableIPs(s.subnetInfo.baseSubnet, ipsInUse, len(unallocated))
		if err != nil {
			result.reservationFailed(reservation, *api.InternalServerError(api.StatusInternalServerError, err.Error()))
			continue
		}

		i := 0
		for allocationID := range unallocated {
			encodedSubnet := availableIPs[i].String() + utils.PrefixDelimiter + utils.IPV4DefaultPrefix

			rowKey := datamodel.EncodeIPAllocationKey(encodedSubnet)
			result.rowBatch[rowKey] = map[string]interface{}{ipAllocProperty: reservationKey, allocIDProperty: allocationID}

			if _, ok := addressesInUse[availableIPs[i].String()]; !ok {
				ipsInUse = append(ipsInUse, availableIPs[i])
				addressesInUse[encodedSubnet] = availableIPs[i]
			}
			i++
			reservation.SecondaryAllocations[allocationID] = encodedSubnet
		}

		result.reservationComplete(reservation)
	}

	return result, nil
}

func (s *subnetHandler) cleanUpTBDs(batch requestSet) []datamodel.ReservationConfiguration {
	s.logf("deallocating ips")

	reservations := batch.getAllReservations()
	reservationSet := toReservationSet(reservations)
	reservationToTBD := batch.getAllValidTBDAllocIDs()

	if len(reservationToTBD) == 0 {
		return reservations
	}

	var resWithTBD []datamodel.ReservationConfiguration
	for resID := range reservationToTBD {
		resWithTBD = append(resWithTBD, reservationSet[resID])
	}

	deallocatedReservations, err := s.deallocateBatch(resWithTBD, reservationToTBD)
	if err != nil {
		s.logErrorf("could not deallocate batch: %v", err)
		for resID := range reservationToTBD {
			batch.removeWithError(resID, err)
			delete(reservationSet, resID)
		}
	} else {
		s.logf("updating deallocated reservations")
		if err := s.reservations.UpdateReservations(deallocatedReservations); err != nil {
			for _, newRes := range deallocatedReservations {
				batch.removeWithError(newRes.ReservationID, err)
				delete(reservationSet, newRes.ReservationID)
			}
		}

		for _, newRes := range deallocatedReservations {
			reservationSet[newRes.ReservationID] = newRes
		}
	}
	return toReservationSlice(reservationSet)
}

func (s *subnetHandler) updateSubnet(reservations []datamodel.ReservationConfiguration) *api.JsonErr {
	if len(reservations) == 0 {
		return nil
	}

	// todo: this is needed because both reserve and unreserve update the subnet record.
	//   once we index reservations with subnet, this can be removed.
	session := map[string]string{
		datamodel.NetworkRepositoryName: s.subnetInfo.vnetID,
	}
	s.locker.BeginWriteSession(session)
	defer s.locker.EndWriteSession(session)

	subnet, err := s.subnets.TryQuerySubnetConfiguration(s.subnetInfo.vnetID, s.subnetInfo.subnetName)
	if err != nil {
		return err
	}
	if subnet == nil {
		return api.NotFoundf(api.SubnetNotFound, "vnet %s subnet name %s not found", s.subnetInfo.vnetID, s.subnetInfo.subnetName)
	}

	if subnet.Reservations == nil {
		subnet.Reservations = make(map[string]string)
	}

	for _, res := range reservations {
		subnet.Reservations[res.ReservationID] = res.AllocatedSubnet
	}

	return s.subnets.PutSubnetConfiguration(subnet)
}

func (s *subnetHandler) deallocateBatch(reservations []datamodel.ReservationConfiguration, tbdIDs map[string][]string) ([]datamodel.ReservationConfiguration, error) {
	// todo: should the input reservations be copied?
	var rowKeys []string
	for i := 0; i < len(reservations); i++ {
		tbdIDs := tbdIDs[reservations[i].ReservationID]

		for _, allocID := range tbdIDs {
			if ipAddress, found := reservations[i].SecondaryAllocations[allocID]; found && ipAddress != "" {
				rowKeys = append(rowKeys, datamodel.EncodeIPAllocationKey(ipAddress))
			}

			delete(reservations[i].SecondaryAllocations, allocID)
		}
	}

	if err := s.allocator.deleteAllocations(s.subnetInfo.vnetID, s.subnetInfo.subnetName, rowKeys); err != nil {
		return nil, err
	}

	if err := s.reservations.UpdateReservations(reservations); err != nil {
		return nil, err
	}

	return reservations, nil
}

func getAllAddressesInUse(baseSubnet *net.IPNet, allocationState map[string]*net.IPNet) (map[string]net.IP, error) {
	addressesInUse := make(map[string]net.IP) // 10.0.0.4 key
	for _, subnet := range allocationState {
		addressesInUse[subnet.IP.String()] = subnet.IP
	}

	azureReserved, err := allocationhelper.GetReservedAddressesInSubnet(baseSubnet)
	if err != nil {
		return nil, err
	}

	for _, reservedIPSubnet := range azureReserved {
		addressesInUse[reservedIPSubnet.IP.String()] = reservedIPSubnet.IP
	}

	return addressesInUse, nil
}

func toReservationSet(rr []datamodel.ReservationConfiguration) map[string]datamodel.ReservationConfiguration {
	out := make(map[string]datamodel.ReservationConfiguration)
	for _, r := range rr {
		out[r.ReservationID] = r
	}
	return out
}

func toReservationSlice(set map[string]datamodel.ReservationConfiguration) []datamodel.ReservationConfiguration {
	var out []datamodel.ReservationConfiguration
	for _, v := range set {
		out = append(out, v)
	}
	return out
}
