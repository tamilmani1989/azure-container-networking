package networkcontainer

import (
	"context"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/utils"
	"fmt"
)

// IPReservationDispatcher processes network container reservations in batch
// It is a demuxer/router for reservation requests. It manages a pool of subnet
// handlers, each for a distinct subnet.
type IPReservationDispatcher struct {
	logger              loggershim.LoggerInterface
	subnets             SubnetRepo
	handlerDeps         handlerDependencies
	handlerMaxBatchSize int
	inputCh             chan reservationRequest
	subnetHandlers      map[subnetKey]*subnetHandler
	locker              *utils.SessionManager
}

type reservationRequest struct {
	reservation      datamodel.ReservationConfiguration
	toBeDeletedIPIDs []string
	responseCh       chan response
}

type response struct {
	reservation datamodel.ReservationConfiguration
	err         error
}

type DispatcherDependencies struct {
	Logger       loggershim.LoggerInterface
	Reservations IPReservationRepo
	Allocator    *IPAllocator
	Subnets      SubnetRepo
}

type BatchConfig struct {
	MaxBatchSize int
}

func NewIPReservationDispatcher(deps DispatcherDependencies, locker *utils.SessionManager, maxBatchSize int) *IPReservationDispatcher {
	return &IPReservationDispatcher{
		logger:  deps.Logger,
		subnets: deps.Subnets,
		handlerDeps: handlerDependencies{
			reservations: deps.Reservations,
			subnets:      deps.Subnets,
			allocator:    deps.Allocator,
			logger:       deps.Logger,
			locker:       locker,
		},
		handlerMaxBatchSize: maxBatchSize,
		inputCh:             make(chan reservationRequest),
		subnetHandlers:      make(map[subnetKey]*subnetHandler),
	}
}

func (d *IPReservationDispatcher) logf(format string, args ...interface{}) {
	d.logger.Debugf("[ip reservation dispatcher] %s", fmt.Sprintf(format, args...))
}

func (d *IPReservationDispatcher) logErrorf(format string, args ...interface{}) {
	d.logger.Errorf("[ip reservation dispatcher] %s", fmt.Sprintf(format, args...))
}

// Reserve processes an network container for reservation/allocation.
func (d *IPReservationDispatcher) Reserve(res datamodel.ReservationConfiguration, toBeDeletedIPIDs []string) (datamodel.ReservationConfiguration, error) {
	responseCh := make(chan response)

	// todo: revisit passing a context. considered it initially, but tricky to determine what behavior
	//   should be if cancelled while nc is already being processed as part of a batch.
	d.inputCh <- reservationRequest{
		reservation:      res,
		toBeDeletedIPIDs: toBeDeletedIPIDs,
		responseCh:       responseCh,
	}

	response := <-responseCh

	return response.reservation, response.err
}

// Run starts the dispatch loop
func (d *IPReservationDispatcher) Run(ctx context.Context) {
	d.logf("starting dispatch loop")

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case req := <-d.inputCh:
				key := subnetKey{
					vnetID:     req.reservation.NetworkID,
					subnetName: req.reservation.AllocationRequest.SubnetName,
				}
				handler, found := d.subnetHandlers[key]
				if !found {
					d.logf("starting batch reservation on vnet ID %s, subnet name %s", key.vnetID, key.subnetName)
					h, err := d.getNewHandler(key)
					if err != nil {
						d.logErrorf("could not start subnet handler: %+v", err)
						req.responseCh <- response{err: err}
						continue
					}

					d.subnetHandlers[key] = h
					d.logf("starting subnet handler for vnet %s subnet %s", key.vnetID, key.subnetName)
					go h.processAndBatch(ctx)
					handler = h
				}

				go handler.enqueue(req)
			}
		}
	}()
}

func (d *IPReservationDispatcher) getNewHandler(key subnetKey) (*subnetHandler, error) {
	subnetConfig, err := d.subnets.TryQuerySubnetConfiguration(key.vnetID, key.subnetName)
	if err != nil {
		return nil, fmt.Errorf("could not get subnet configuration for vnet %s subnet name %s: %v", key.vnetID, key.subnetName, err)
	}

	if subnetConfig == nil {
		return nil, fmt.Errorf("did not find subnet configuration for vnet %s subnet name %s", key.vnetID, key.subnetName)
	}

	subnet, parseErr := utils.ParseIPNet(subnetConfig.IPSubnet)
	if parseErr != nil {
		return nil, fmt.Errorf("could not parse cidr %s: %v", subnetConfig.IPSubnet, parseErr)
	}

	subnetInfo := subnetInfo{
		vnetID:     key.vnetID,
		subnetName: key.subnetName,
		baseSubnet: subnet,
	}

	return newSubnetHandler(subnetInfo, d.handlerDeps, d.handlerMaxBatchSize), nil
}
