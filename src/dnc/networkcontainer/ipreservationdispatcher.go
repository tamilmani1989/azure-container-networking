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
	logger                     loggershim.LoggerInterface
	subnets                    SubnetRepo
	reservations               IPReservationRepo
	allocator                  *IPAllocator
	handlerMaxBatchSize        int
	usePrepopulatedAllocations bool
	pubsubProxyEnabled         bool
	inputCh                    chan reservationRequest
	subnetHandlers             map[subnetKey]*subnetHandler
}

type reservationRequest struct {
	reservation      datamodel.ReservationRow
	toBeDeletedIPIDs []string
	responseCh       chan response
}

type response struct {
	reservation datamodel.ReservationRow
	err         error
}

type DispatcherConfig struct {
	Logger                     loggershim.LoggerInterface
	Subnets                    SubnetRepo
	Reservations               IPReservationRepo
	Allocator                  *IPAllocator
	HandlerMaxBatchSize        int
	UsePrepopulatedAllocations bool
	PubSubProxyEnabled         bool
}

func NewIPReservationDispatcher(config DispatcherConfig) *IPReservationDispatcher {
	return &IPReservationDispatcher{
		logger:                     config.Logger,
		subnets:                    config.Subnets,
		reservations:               config.Reservations,
		allocator:                  config.Allocator,
		handlerMaxBatchSize:        config.HandlerMaxBatchSize,
		usePrepopulatedAllocations: config.UsePrepopulatedAllocations,
		pubsubProxyEnabled:         config.PubSubProxyEnabled,
		inputCh:                    make(chan reservationRequest),
		subnetHandlers:             make(map[subnetKey]*subnetHandler),
	}
}

func (d *IPReservationDispatcher) logf(format string, args ...interface{}) {
	d.logger.Debugf("[ip reservation dispatcher] %s", fmt.Sprintf(format, args...))
}

func (d *IPReservationDispatcher) logErrorf(format string, args ...interface{}) {
	d.logger.Errorf("[ip reservation dispatcher] %s", fmt.Sprintf(format, args...))
}

// Reserve processes an network container for reservation/allocation.
func (d *IPReservationDispatcher) Reserve(res datamodel.ReservationRow, toBeDeletedIPIDs []string) (datamodel.ReservationRow, error) {
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
					vnetID:     req.reservation.Configuration.NetworkID,
					subnetName: req.reservation.Configuration.AllocationRequest.SubnetName,
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

	handlerConfig := handlerConfig{
		subnetInfo:   subnetInfo,
		reservations: d.reservations,
		logger:       d.logger,
		maxBatchSize: d.handlerMaxBatchSize,
	}

	if d.usePrepopulatedAllocations {
		handlerConfig.scaleStrategy = &prepopulatedStrategy{
			allocator:          d.allocator,
			subnetInfo:         subnetInfo,
			logger:             d.logger,
			pubsubProxyEnabled: d.pubsubProxyEnabled,
		}
	} else {
		handlerConfig.scaleStrategy = &scanSubnetStrategy{
			allocator:          d.allocator,
			subnetInfo:         subnetInfo,
			logger:             d.logger,
			pubsubProxyEnabled: d.pubsubProxyEnabled,
		}
	}

	return newSubnetHandler(handlerConfig), nil
}
