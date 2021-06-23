package networkcontainer

import (
	"dnc/api"
	"dnc/cnsinterface"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/utils"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// todo: convert to interfaces, which can be reused across state transitions
type UpdateNodeContainersFunc func(nodeID, networkID, ncID, reservationID, containerType string, updateVlan, remove bool, ncLimit int) *api.JsonErr

type PersistNCFunc func(nc *datamodel.NetworkContainerEntity) *api.JsonErr

type Preparer struct {
	CNS                  cnsinterface.CnsInterface
	Locks                *utils.SessionManager
	Reservations         IPReservationRepo
	UpdateNodeContainers UpdateNodeContainersFunc
	PersistNC            PersistNCFunc
	Logger               loggershim.LoggerInterface
}

// PrepareNetworkContainerV2 prepares a network container that requires implicit reservations, and attempts to move it to reserve state.
// Returns whether item should be removed from pending queue, and any error during preparation
func (p *Preparer) PrepareNetworkContainerV2(repoNCData *datamodel.NetworkContainerEntity) (bool, *api.JsonErr) {
	var res datamodel.ReservationConfiguration
	if repoNCData.Config.ReservationID == "" {
		res = newReservation(repoNCData.Config)
		repoNCData.Config.ReservationID = res.ReservationID
	} else {
		existingRes, err := p.Reservations.TryQueryIPReservation(repoNCData.Config.NetworkID, repoNCData.Config.ReservationID)
		switch {
		case err != nil:
			return false, err
		case existingRes.Configuration == nil: // reservation id previously assigned, but no reservation created
			res = newReservation(repoNCData.Config)
			p.Logger.Debugf("creating reservation for nc id %s, reservation id %s", repoNCData.Config.NetworkContainerID, repoNCData.Config.ReservationID)
		default: // reservation previously created, this is preparing an update
			p.Logger.Debugf("processing update for nc id %s, reservation id %s", repoNCData.Config.NetworkContainerID, existingRes.Configuration.ReservationID)
			res, err = getUpdatedReservation(repoNCData.Config.RequestV2, *existingRes.Configuration)
			if err != nil {
				return p.markNCAsFailed(repoNCData, *err)
			}
		}
	}

	if err := p.registerNCinNode(repoNCData); err != nil {
		// if NC limit has been reached the error is non-retryable
		if err.Code == api.NodeContainerCapacityExhausted {
			return p.markNCAsFailed(repoNCData, *err)
		}
		return false, err
	}

	reservation := datamodel.ReservationRow{
		RowKey:        datamodel.GetReservationKey(repoNCData.Config.NetworkID, repoNCData.Config.ReservationID),
		Configuration: &res,
		SubnetKey:     datamodel.GetSubnetKey(repoNCData.Config.NetworkID, repoNCData.Config.RequestV2.AllocationRequest.SubnetName),
		SetKey:        "",
	}

	if err := p.Reservations.PersistIPReservation(reservation); err != nil {
		return false, api.InternalServerError(api.StatusInternalServerError, err.Error())
	}

	p.Logger.Debugf("successfully persisted reservation id %s, reservation config: %+v for nc %s", reservation.RowKey, res, repoNCData.Config.NetworkContainerID)

	err := p.moveNCToReserveState(repoNCData)
	return false, err
}

func newReservation(nc datamodel.NCConfiguration) datamodel.ReservationConfiguration {
	resID := nc.ReservationID
	if resID == "" {
		resID = uuid.New().String()
	}
	res := datamodel.ReservationConfiguration{
		ReservationID:        resID,
		NetworkID:            nc.NetworkID,
		ReservationState:     datamodel.Allocate,
		Status:               datamodel.Pending,
		AllocationRequest:    nc.RequestV2.AllocationRequest,
		AssociatedNCID:       nc.NetworkContainerID,
		SecondaryAllocations: make(map[string]string),
	}
	for i := 0; i < nc.RequestV2.AllocationRequest.SecondaryIPCount; i++ {
		res.SecondaryAllocations[uuid.New().String()] = ""
	}
	return res
}

// getUpdatedReservation compares the current allocation request and any ips that need to be deleted to an existing reservation.
// If any IP allocations need to be made, it will return a reservation with new allocation ids that need to be processed,
// and the reservation will be in a Pending state.
// For any deletions that need to be processed, the ids will remain as part of the reservation, and should only be removed
// once the associated records in the allocations table have been properly deleted.
func getUpdatedReservation(req api.NetworkContainerRequestV2, res datamodel.ReservationConfiguration) (datamodel.ReservationConfiguration, *api.JsonErr) {
	// requested to be deleted ip ids, which may or may not be part of the reservation anymore
	requestedDeletions := make(map[string]struct{})
	for _, allocID := range req.ToBeDeletedIPIds {
		requestedDeletions[allocID] = struct{}{}
	}

	// we capture a count of allocations to be decreased for any tbd ips that exist in the current secondary allocations
	// we use this to check for the state of allocations after accounting for removal of ips, and see if the request can
	// be processed further.
	allocCount := len(res.SecondaryAllocations)
	for allocID := range res.SecondaryAllocations {
		if _, found := requestedDeletions[allocID]; found {
			allocCount--
		}
	}

	// request count - (allocations - valid tbd ids)
	ipCountDiff := req.AllocationRequest.SecondaryIPCount - allocCount

	// we couldn't remove enough to meet the requested count, and we can't arbitrarily choose ids to remove.
	// fail processing, client must retry with a different request
	if ipCountDiff < 0 {
		diff := len(res.SecondaryAllocations) - req.AllocationRequest.SecondaryIPCount
		msg := fmt.Sprintf("not enough ips to be deleted to meet allocation request, required at least %d ids to be within existing reservation", diff)
		return datamodel.ReservationConfiguration{}, api.UnprocessableEntity(api.InvalidNetworkContainerRequest, msg)
	}

	// whether we process further allocations or just deletes, we always want the reservation to have the latest allocation request
	res.AllocationRequest = req.AllocationRequest

	// after accounting for ips that will be deleted, reservation meets the requested count, so no further allocation is needed
	if ipCountDiff == 0 {
		res.ReservationState = datamodel.Allocate
		res.Status = datamodel.Completed
		return res, nil
	}

	// otherwise, we still need to add keys to meet the allocation request and mark reservation as pending
	for i := 0; i < ipCountDiff; i++ {
		res.SecondaryAllocations[uuid.New().String()] = ""
	}

	res.ReservationState = datamodel.Allocate
	res.Status = datamodel.Pending
	return res, nil
}

func (p *Preparer) registerNCinNode(nc *datamodel.NetworkContainerEntity) *api.JsonErr {
	nodeID := nc.Config.GetAssociationInfo().NodeID

	ncLimit, err := p.CNS.GetNCLimitForNode(nodeID)
	if err != nil {
		return err
	}

	nodeIDLock := map[string]string{datamodel.NodeRepositoryName: nodeID}
	p.Locks.BeginWriteSession(nodeIDLock)
	// Add NC to the container map on the node. This will fail if the node has reached the NC limit
	err = p.UpdateNodeContainers(
		nodeID,
		nc.Config.NetworkID,
		nc.Config.NetworkContainerID,
		nc.Config.ReservationID,
		nc.Config.RequestV2.AssociationInfo.ContainerType,
		ShouldUpdateVlan(nc.Config.RequestV2.AssociationInfo.ContainerType),
		false,
		ncLimit,
	)
	p.Locks.EndWriteSession(nodeIDLock)

	if err != nil {
		err.Text = fmt.Sprintf("Failed to add NC %s to node %s. Error %v", nc.Config.NetworkContainerID, nodeID, err.Text)
		return err
	}
	return nil
}

func (p *Preparer) markNCAsFailed(repoNCData *datamodel.NetworkContainerEntity, err api.JsonErr) (bool, *api.JsonErr) {
	repoNCData.Config.Status = datamodel.Failed
	repoNCData.Config.FailureDetail = err
	if err := p.PersistNC(repoNCData); err != nil {
		return false, err // don't remove from queue until failure can be persisted
	}
	return true, nil
}

func (p *Preparer) moveNCToReserveState(repoNCData *datamodel.NetworkContainerEntity) *api.JsonErr {
	repoNCData.Config.State = datamodel.Reserve
	repoNCData.Config.Status = datamodel.Pending
	repoNCData.Config.FailureDetail = api.JsonErr{}
	repoNCData.Config.LogStateCompletion(datamodel.Prepare)
	return p.PersistNC(repoNCData)
}

func (p *Preparer) PrepareNetworkContainer(repoNCData *datamodel.NetworkContainerEntity) (bool, *api.JsonErr) {
	if !repoNCData.Config.ImplicitReservation {
		return p.PrepareNetworkContainerV1(repoNCData)
	}

	return p.PrepareNetworkContainerV2(repoNCData)
}

// PrepareNetworkContainerV1 prepare, validate the NC request v1.
// todo: moved with minor changes from requestmonitor.go. need to clean up further and add tests
func (p *Preparer) PrepareNetworkContainerV1(repoNCData *datamodel.NetworkContainerEntity) (bool, *api.JsonErr) {
	associationInfo := repoNCData.Config.GetAssociationInfo()

	// Process reservation if the reservation ID is specified in the NC config
	// In case of reservation set, no reservation ID will be specified and the allocation of the unassigned reservation
	// will be processed during reserve phase
	if repoNCData.Config.ReservationID != "" {
		reservation, err := datamodel.QueryIPReservation(repoNCData.Config.NetworkID, repoNCData.Config.ReservationID)
		if err != nil {
			if err.HTTPStatus == http.StatusInternalServerError {
				return false, err
			}
			return p.markNCAsFailed(repoNCData, *err)
		}

		// Compare with frontendIP in a case insensitive manner or any purpose tag defined, DNC should prevent create NC on this IP
		requestType := reservation.Configuration.AllocationRequest.RequestType
		if strings.EqualFold(requestType, api.FrontendIP) || requestType != "" {
			err = api.BadRequestf(api.ReservationTypeNotMatch,
				"Reservation has been reserved as %s, can't create container with it", requestType)
			return p.markNCAsFailed(repoNCData, *err)
		}
	}

	ncLimit, err := p.CNS.GetNCLimitForNode(associationInfo.NodeID)
	if err != nil {
		return false, err
	}

	sessionInfoWrite := map[string]string{
		datamodel.ReservationRepositoryName: repoNCData.Config.RequestV1.ReservationID,
		datamodel.NodeRepositoryName:        associationInfo.NodeID,
	}
	p.Locks.BeginWriteSession(sessionInfoWrite)
	defer p.Locks.EndWriteSession(sessionInfoWrite)

	err = datamodel.NodeConfigRepo.UpdateNodeConfigurationNetworkContainerSet(
		associationInfo.NodeID,
		repoNCData.Config.NetworkID,
		repoNCData.Config.NetworkContainerID,
		repoNCData.Config.RequestV1.ReservationID,
		associationInfo.ContainerType,
		ShouldUpdateVlan(associationInfo.ContainerType),
		false,
		ncLimit,
	)

	if err != nil {
		if err.HTTPStatus == http.StatusInternalServerError {
			return false, err
		}
		return p.markNCAsFailed(repoNCData, *err)
	}

	err = p.moveNCToReserveState(repoNCData)
	return false, err
}
