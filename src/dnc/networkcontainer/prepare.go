package networkcontainer

import (
	"dnc/api"
	"dnc/cnsinterface"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/utils"
	"fmt"
	"net/http"

	"github.com/Azure/azure-container-networking/cns"
	"github.com/google/uuid"
)

// todo: convert to interfaces, which can be reused across state transitions
type PersistReservationFunc func(ipReservation *datamodel.ReservationConfiguration) *api.JsonErr

type UpdateNodeContainersFunc func(nodeID, networkID, ncID, reservationID, containerType string, updateVlan, remove bool, ncLimit int) *api.JsonErr

type PersistNCFunc func(ncConfiguration *datamodel.NCConfiguration, addToPendingQueue bool) *api.JsonErr

type Preparer struct {
	CNS                  cnsinterface.CnsInterface
	Locks                *utils.SessionManager
	PersistReservation   PersistReservationFunc
	UpdateNodeContainers UpdateNodeContainersFunc
	PersistNC            PersistNCFunc
	Logger               loggershim.LoggerInterface
}

// PrepareNetworkContainerV2 prepares a network container that requires implicit reservations, and attempts to move it to reserve state.
// Returns whether item should be removed from pending queue, and any error during preparation
func (p *Preparer) PrepareNetworkContainerV2(ncConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	if ncConfig.ReservationID == "" {
		reservationIDUUID, err := uuid.NewUUID()
		if err != nil {
			return false, &api.JsonErr{
				Code:       api.StatusInternalServerError,
				HTTPStatus: http.StatusInternalServerError,
				Text:       fmt.Sprintf("Failed to generate UUID due to error:%s", err.Error()),
			}
		}
		ncConfig.ReservationID = reservationIDUUID.String()
	}

	if err := p.registerNCinNode(*ncConfig); err != nil {
		// if NC limit has been reached the error is non-retryable
		if err.Code == api.NodeContainerCapacityExhausted {
			return p.markNCAsFailed(ncConfig, *err)
		}
		return false, err
	}

	res := datamodel.ReservationConfiguration{
		ReservationID:         ncConfig.ReservationID,
		NetworkID:             ncConfig.NetworkID,
		ReservationState:      datamodel.Allocate,
		Status:                datamodel.Pending,
		AllocationRequest:     ncConfig.RequestV2.AllocationRequest,
		AssociatedNCID:        ncConfig.NetworkContainerID,
		SecondaryReservations: make(map[string]string),
	}

	for i := 0; i < ncConfig.RequestV2.AllocationRequest.SecondaryIPCount; i++ {
		id, _ := uuid.NewUUID()
		res.SecondaryReservations[id.String()] = ""
	}

	if err := p.PersistReservation(&res); err != nil {
		return false, &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       err.Error(),
			HTTPStatus: http.StatusInternalServerError,
		}
	}

	p.Logger.Debugf("Successfully created reservation config: %+v for NC: %s", res, ncConfig.NetworkContainerID)
	err := p.moveNCToReserveState(ncConfig)
	return false, err
}

func (p *Preparer) registerNCinNode(ncConfig datamodel.NCConfiguration) *api.JsonErr {
	nodeID := ncConfig.GetAssociationInfo().NodeID

	ncLimit, err := p.CNS.GetNCLimitForNode(nodeID) // todo: this is just 1 in case of AKS Swift
	if err != nil {
		return err
	}

	nodeIDLock := map[string]string{datamodel.NodeRepositoryName: nodeID}
	p.Locks.BeginWriteSession(nodeIDLock)
	// Add NC to the container map on the node. This will fail if the node has reached the NC limit
	err = p.UpdateNodeContainers(
		nodeID,
		ncConfig.NetworkID,
		ncConfig.NetworkContainerID,
		ncConfig.ReservationID,
		ncConfig.RequestV2.AssociationInfo.ContainerType,
		shouldUpdateVlan(ncConfig.RequestV2.AssociationInfo.ContainerType),
		false,
		ncLimit,
	)
	p.Locks.EndWriteSession(nodeIDLock)

	if err != nil {
		err.Text = fmt.Sprintf("Failed to add NC %s to node %s. Error %v", ncConfig.NetworkContainerID, nodeID, err.Text)
		return err
	}
	return nil
}

func (p *Preparer) markNCAsFailed(ncConfig *datamodel.NCConfiguration, err api.JsonErr) (bool, *api.JsonErr) {
	ncConfig.Status = datamodel.Failed
	ncConfig.FailureDetail = err
	if err := p.PersistNC(ncConfig, datamodel.DontAddToPendingQ); err != nil {
		return false, err // don't remove from queue until failure can be persisted
	}
	return true, nil
}

func (p *Preparer) moveNCToReserveState(ncConfig *datamodel.NCConfiguration) *api.JsonErr {
	ncConfig.State = datamodel.Reserve
	ncConfig.Status = datamodel.Pending
	ncConfig.FailureDetail = api.JsonErr{}
	return p.PersistNC(ncConfig, datamodel.DontAddToPendingQ)
}

func shouldUpdateVlan(containerType string) bool {
	switch containerType {
	case cns.AzureContainerInstance:
	case cns.Docker:
	case cns.JobObject:
	case cns.COW:
	default:
		return false
	}
	return true
}

func (p *Preparer) PrepareNetworkContainer(repoNCConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	if !repoNCConfig.ImplicitReservation {
		return p.PrepareNetworkContainerV1(repoNCConfig)
	}

	return p.PrepareNetworkContainerV2(repoNCConfig)
}

// todo: moved with minor changes from requestmonitor.go. need to clean up further and add tests
func (p *Preparer) PrepareNetworkContainerV1(ncConfig *datamodel.NCConfiguration) (bool, *api.JsonErr) {
	associationInfo := ncConfig.GetAssociationInfo()

	ncLimit, err := p.CNS.GetNCLimitForNode(associationInfo.NodeID)
	if err != nil {
		return false, err
	}

	sessionInfoWrite := map[string]string{
		datamodel.ReservationRepositoryName: ncConfig.RequestV1.ReservationID,
		datamodel.NodeRepositoryName:        associationInfo.NodeID,
	}
	p.Locks.BeginWriteSession(sessionInfoWrite)
	defer p.Locks.EndWriteSession(sessionInfoWrite)

	err = datamodel.UpdateNodeConfigurationNetworkContainerSet(
		associationInfo.NodeID,
		ncConfig.NetworkID,
		ncConfig.NetworkContainerID,
		ncConfig.RequestV1.ReservationID,
		associationInfo.ContainerType,
		shouldUpdateVlan(associationInfo.ContainerType),
		false,
		ncLimit,
	)

	if err != nil {
		if err.HTTPStatus == http.StatusInternalServerError {
			return false, err
		}
		return p.markNCAsFailed(ncConfig, *err)
	}

	reservationConfiguration, err := datamodel.QueryIPReservation(ncConfig.NetworkID, ncConfig.ReservationID)
	if err != nil {
		if err.HTTPStatus == http.StatusInternalServerError {
			return false, err
		}
		return p.markNCAsFailed(ncConfig, *err)
	}

	reservationConfiguration.AssociatedNCID = ncConfig.NetworkContainerID
	if err := datamodel.PutIPReservationConfiguration(reservationConfiguration); err != nil {
		return false, err
	}

	err = p.moveNCToReserveState(ncConfig)
	return false, err
}
