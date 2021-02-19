package networkcontainer

import (
	"context"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"fmt"
	"net"
	"time"
)

// subnetHandler processes batches of reservation requests for a given vnet and subnet name
type subnetHandler struct {
	reservations  IPReservationRepo
	allocator     *IPAllocator
	logger        loggershim.LoggerInterface
	inputCh       chan reservationRequest
	subnetInfo    subnetInfo
	maxBatchSize  int
	scaleStrategy scalingStrategy
}

// scalingStrategy handles scale up and scale down of allocation requests.
type scalingStrategy interface {
	// scaleDown handles decreases in allocations (using corresponding tbdAllocationIDs) for reservations.
	// It should return the new state of reservations after tbdAllocationIDs have been processed.
	scaleDown(reservations []datamodel.ReservationConfiguration, tbdAllocationIDs map[string][]string) ([]datamodel.ReservationConfiguration, error)
	// ScaleUp handles allocations for reservations where allocatedSubnet or secondary allocations don't have a corresponding IP.
	// It should only be called with the output of ScaleDown.
	// It should return the new state of reservations after any allocations have been processed.
	scaleUp(reservations []datamodel.ReservationConfiguration) ([]datamodel.ReservationConfiguration, error)
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
	allocator    *IPAllocator
	logger       loggershim.LoggerInterface
}

func newSubnetHandler(info subnetInfo, deps handlerDependencies, maxBatchSize int, reuseAllocations bool) *subnetHandler {
	var scaleStrategy scalingStrategy
	if !reuseAllocations {
		scaleStrategy = &scanSubnetStrategy{
			allocator:  deps.allocator,
			subnetInfo: info,
			logger:     deps.logger,
		}
	} else {
		scaleStrategy = &prepopulatedStrategy{
			allocator:  deps.allocator,
			subnetInfo: info,
			logger:     deps.logger,
		}
	}

	return &subnetHandler{
		reservations:  deps.reservations,
		allocator:     deps.allocator,
		logger:        deps.logger,
		inputCh:       make(chan reservationRequest),
		subnetInfo:    info,
		maxBatchSize:  maxBatchSize,
		scaleStrategy: scaleStrategy,
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

	updatedReservations, err := s.scaleStrategy.scaleUp(reservations)
	if err != nil {
		s.logErrorf("could not update reservations: %v", err)
		batch.flushWithError(err)
		return
	}

	s.logf("updating reservations")
	if err := s.reservations.UpdateReservations(updatedReservations); err != nil {
		s.logErrorf("could not update reservations: %v", err)
		batch.flushWithError(err)
		return
	}

	batch.complete(updatedReservations)
	s.logf("batch completed in %s", time.Since(start).String())
}

func (s *subnetHandler) cleanUpTBDs(batch requestSet) []datamodel.ReservationConfiguration {
	s.logf("deallocating ips")

	reservations := batch.getAllReservations()
	reservationSet := toReservationSet(reservations)
	reservationToTBD := batch.getAllValidTBDAllocIDs() // or, to be cleared ips

	if len(reservationToTBD) == 0 {
		return reservations
	}

	var resWithTBD []datamodel.ReservationConfiguration
	for resID := range reservationToTBD {
		resWithTBD = append(resWithTBD, reservationSet[resID])
	}

	deallocatedReservations, err := s.scaleStrategy.scaleDown(resWithTBD, reservationToTBD)
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
