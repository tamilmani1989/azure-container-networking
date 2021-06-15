package controller

import (
	"bytes"
	"crypto/sha256"
	"dnc/aiwrapper"
	"dnc/api"
	"dnc/client"
	"dnc/configuration"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/masterelection/masterelectionshim"
	"dnc/networkcontainer"
	"dnc/pubsubproxy"
	"dnc/replica"
	"dnc/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/Azure/azure-container-networking/cns"
)

type joinNetworkFunction func(networkID string, apiVersion string) *api.JsonErr
type deleteNetworkFunction func(networkID string) *api.JsonErr
type registerSubnetFunction func(networkID, subnetID string, isRegistered bool)
type deleteSubnetFunction func(networkID, subnetID string, networkConfig *datamodel.NetworkConfiguration) *api.JsonErr
type getNetworkStatusFunction func() datamodel.Status

// todo: no need for these to be globals, should be part of struct
var (
	setOrchestratorTypeURLFmt string
	masterElectionShim        *masterelectionshim.MasterElectionShim
	config                    *configuration.Configuration
	joinNetworkFunc           joinNetworkFunction
	deleteNetworkFunc         deleteNetworkFunction
	registerSubnetFunc        registerSubnetFunction
	deleteSubnetFunc          deleteSubnetFunction
	getNetworkStatusFunc      getNetworkStatusFunction
)

// Controller - Controller for handling API calls
type Controller struct {
	Replica        *replica.Replica
	SessionManager *utils.SessionManager
	httpClient     *http.Client
	cnsHttpClient  *http.Client
	ipAllocator    *networkcontainer.IPAllocator
}

// NewController - Create a NewController
func NewController(
	replica *replica.Replica,
	sessionsManager *utils.SessionManager,
	setOrchestratorTypeUrlFmt string,
	electionShim *masterelectionshim.MasterElectionShim,
	configuration *configuration.Configuration,
	httpClient *http.Client,
	cnsHttpClient *http.Client,
	ipAllocator *networkcontainer.IPAllocator) *Controller {
	controller := new(Controller)
	controller.Replica = replica
	controller.SessionManager = sessionsManager
	controller.httpClient = httpClient
	controller.cnsHttpClient = cnsHttpClient
	controller.ipAllocator = ipAllocator
	setOrchestratorTypeURLFmt = setOrchestratorTypeUrlFmt
	masterElectionShim = electionShim
	config = configuration

	// TODO: Create an interface for these
	if config.PubSubProxySettings.EnableRNC {
		joinNetworkFunc = joinNetworkViaRNC
		deleteNetworkFunc = deleteNetworkViaRNC
		registerSubnetFunc = registerSubnetViaRNC
		deleteSubnetFunc = deleteSubnetViaRNC
		getNetworkStatusFunc = getNetworkStatusViaRNC
	} else {
		joinNetworkFunc = joinNetworkViaRNM
		deleteNetworkFunc = deleteNetworkViaRNM
		registerSubnetFunc = registerSubnetViaRNM
		deleteSubnetFunc = deleteSubnetViaRNM
		getNetworkStatusFunc = getNetworkStatusViaRNM
	}

	return controller
}

// GetConfig :- retrieves the configuration passed to controller
func (c *Controller) GetConfig() *configuration.Configuration {
	return config
}

// PersistNodeInfo - Persist node configuration.
func (c *Controller) PersistNodeInfo(nodeID string, nodeInfo *api.NodeInfoRequest, apiVersion string) *api.JsonErr {
	// todo: can nodes for a DNC partition have different orchestrator types? if not, this should just be cached
	aiwrapper.SetOrchestratorType(nodeInfo.OrchestratorType)

	// Turn off the DNS-CNS for managed and baremetal node type
	if !config.DNCSettings.Managed && !api.IsBareMetalNode(nodeInfo.NodeType) {
		// Send OrchestratorType to CNS
		var (
			info = &cns.SetOrchestratorTypeRequest{
				OrchestratorType: nodeInfo.OrchestratorType,
				DncPartitionKey:  config.DataStoreSettings.AzureStorageInfo.PartitionKey,
				NodeID:           nodeID}
			body bytes.Buffer
			resp cns.Response
		)

		json.NewEncoder(&body).Encode(info)
		setOrchestratorTypeURL := fmt.Sprintf(
			setOrchestratorTypeURLFmt,
			nodeInfo.IPAddresses[0])
		loggershim.LS.Debugf("[CNS] create/update %+v", nodeInfo)

		w, respErr := c.cnsHttpClient.Post(setOrchestratorTypeURL, "application/json; charset=utf-8", &body)
		if respErr != nil {
			msg := fmt.Sprintf("Post to URL %v for node %s failed with error %s",
				setOrchestratorTypeURL,
				nodeID,
				respErr.Error())
			return &api.JsonErr{
				Code:       api.NodeProcessingFailedDueToInternalError,
				Text:       msg,
				HTTPStatus: http.StatusBadRequest}
		}

		defer w.Body.Close()

		respErr = utils.DecodeResponse(w, &resp)
		if respErr != nil {
			msg := fmt.Sprintf("Create node %s failed with error %s",
				nodeID,
				respErr.Error())
			return &api.JsonErr{
				Code:       api.NodeProcessingFailedDueToInternalError,
				Text:       msg,
				HTTPStatus: http.StatusBadRequest}
		}

		if resp.ReturnCode != 0 {
			msg := fmt.Sprintf("Create node %s failed with error code %d. %s",
				nodeID,
				resp.ReturnCode,
				resp.Message)
			return &api.JsonErr{
				Code:       api.NodeProcessingFailedDueToInternalError,
				Text:       msg,
				HTTPStatus: http.StatusBadRequest}
		}
	}

	sessionInfo := map[string]string{datamodel.NodeRepositoryName: nodeID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)
	// pass false, 0 and empty Apis slice for non-registration calls
	return datamodel.NodeConfigRepo.CreateOrUpdateNodeConfiguration(false, 0, nodeID, *nodeInfo, apiVersion, []string{})
}

// RegisterNode - Register node
func (c *Controller) RegisterNode(nodeID, apiVersion string, request *api.NodeRegistrationRequest) *api.JsonErr {
	sessionInfo := map[string]string{datamodel.NodeRepositoryName: nodeID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	return datamodel.NodeConfigRepo.CreateOrUpdateNodeConfiguration(true, request.NumCores, nodeID, api.NodeInfoRequest{}, apiVersion, request.NmAgentSupportedApis)
}

// DeleteNode - Delete node configuration
func (c *Controller) DeleteNode(nodeID string) *api.JsonErr {
	loggershim.LS.Debugf("[Request] DeleteNode: %s", nodeID)
	sessionInfo := map[string]string{datamodel.NodeRepositoryName: nodeID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	return datamodel.NodeConfigRepo.DeleteNode(nodeID)
}

// GetNodeInfo - Get node info
func (c *Controller) GetNodeInfo(nodeID string) (*api.NodeInfoResponse, *api.JsonErr) {
	// No read lock is required because Azure Storage uses snapshot isolation to allow
	// read operations to happen concurrently with writes in the same partition. It
	// guarantees reads to see consistent data while updates are happening
	return datamodel.NodeConfigRepo.GetNodeInfo(nodeID)
}

// GetNodes - Get a list of slim node infos for all nodes
func (c *Controller) GetNodes(nodeSet string) (*api.NodesResponse, *api.JsonErr) {
	return datamodel.NodeConfigRepo.GetNodes(nodeSet)
}

// GetNodeConfiguration - Get node configuration
func (c *Controller) GetNodeConfiguration(nodeID string) (*datamodel.NodeConfiguration, *api.JsonErr) {
	return datamodel.NodeConfigRepo.TryQueryNodeConfiguration(nodeID)
}

// CreateNodeSet - Creates node set
func (c *Controller) CreateNodeSet(nodeSetName, apiVersion string) (datamodel.NodeSetEntity, *api.JsonErr) {
	nse := datamodel.NodeSetEntity{
		PartitionKey: c.GetConfig().DataStoreSettings.AzureStorageInfo.PartitionKey,
		RowKey:       datamodel.EncodeNodeSetRowKey(nodeSetName),
		NodeSetName:  nodeSetName,
		State:        datamodel.NodeSetCreate,
		Status:       datamodel.Completed,
		APIVersion:   apiVersion,
	}

	if err := datamodel.NodeSetsRepo.InsertNodeSet(nse); err != nil {
		return datamodel.NodeSetEntity{}, err
	}

	return nse, nil
}

// GetNodeSet - Gets a node set by its name
func (c *Controller) GetNodeSet(nodeSetName string) (*datamodel.NodeSetEntity, *api.JsonErr) {
	return datamodel.NodeSetsRepo.TryQueryNodeSet(nodeSetName)
}

// ForceDeleteNodeSet - Force delete a node set by its name
func (c *Controller) ForceDeleteNodeSet(nodeSetName string) *api.JsonErr {
	// todo: this is temporary, clean up of a node set should create a pending item to process nodes/ncs deletion
	//  and mark the node set as being in the process of deletion
	return datamodel.NodeSetsRepo.DeleteNodeSet(nodeSetName)
}

// DeleteNodeSet - Delete a node set by its name
func (c *Controller) DeleteNodeSet(nodeSetName string) *api.JsonErr {
	// todo: add validation to check for any associated nodes, in which case this call should fail.
	return datamodel.NodeSetsRepo.DeleteNodeSet(nodeSetName)
}

// createReservationSet creates reservation set and reservations associated with the set
func (c *Controller) createReservationSet(
	networkID string,
	reservationSetID string,
	request api.ReservationSetRequest,
	apiVersion string) *api.JsonErr {
	var err *api.JsonErr
	if err = datamodel.IpReservationSetRepo.CreateReservationSet(networkID, reservationSetID, request); err != nil {
		loggershim.LS.Errorf("failed to create reservation set: %s. Error: %v", reservationSetID, err)
		return err
	}

	loggershim.LS.Debugf("creating reservation set: %s with count %d", reservationSetID, request.NumReservations)
	// Create reservation records in the IPReservations table with each reservation tagged with the set ID.
	if err = datamodel.IpReservationRepo.CreateReservationsInSet(networkID, request.SubnetName,
		reservationSetID, request.NumReservations, apiVersion); err != nil {
		loggershim.LS.Errorf("failed to create reservations in set: %s. Error: %v", reservationSetID, err)
		return err
	}

	return nil
}

// updateReservationSet updates reservation set
func (c *Controller) updateReservationSet(
	networkID string,
	reservationSetID string,
	request api.ReservationSetRequest,
	reservationSet *datamodel.ReservationSet,
	apiVersion string) *api.JsonErr {
	// Fail the request if the set is already marked for deletion
	if reservationSet.MarkedForDelete {
		return api.BadRequestf(api.InvalidReservationRequest,
			"reservation set: %s is marked for deletion. Create/Update is not allowed.", reservationSetID)
	}

	// Return if there is no work to be done
	if reservationSet.Status == datamodel.Completed &&
		reservationSet.ReservationSetRequest.NumReservations == request.NumReservations {
		loggershim.LS.Debugf("Nothing to update for the reservation set: %s", reservationSetID)
		return nil
	}

	// Return error if the request is updating the num of reservations while the current request is pending
	if reservationSet.Status == datamodel.Pending && reservationSet.ReservationSetRequest.NumReservations != request.NumReservations {
		return api.Conflictf(api.ProcessingPending, "Reservation Set %s is currently being processed "+
			"in state %v with status %v. Cannot update while being processed",
			reservationSet.ReservationSetID, reservationSet.State.String(), reservationSet.Status.String())
	}

	// Fail the request if the network guid or subnet name is different than the the ones in the existing reservation set record
	if reservationSet.NetworkID != networkID {
		return api.BadRequestf(api.InvalidNetworkID, "network cannot be updated for existing reservation set")
	}

	if reservationSet.ReservationSetRequest.SubnetName != request.SubnetName {
		return api.BadRequestf(api.InvalidIPSubnet, "subnet cannot be updated for existing reservation set")
	}

	reservations, err := datamodel.IpReservationRepo.TryQueryAllReservationsInSet(networkID, reservationSetID)
	if err != nil {
		loggershim.LS.Errorf("failed to retrieve reservations in set: %s. Error: %v", reservationSetID, err)
		return err
	}

	// Delete all the failed reservations in this set
	for _, reservation := range reservations {
		if reservation.Configuration.Status == datamodel.Failed {
			if err := c.ipAllocator.DeleteIPReservation(reservation, false); err != nil {
				errMsg := fmt.Sprintf("could not delete failed reservation:%s in set: %s. Error: %v",
					reservation.Configuration.ReservationID, reservationSetID, err)
				loggershim.LS.Errorf(errMsg)
				return api.InternalServerErrorf(api.StatusInternalServerError, errMsg)
			}
		}
	}

	// Update the reservation set state and status
	if request.NumReservations > reservationSet.SuccessfulReservationCount {
		reservationSet.State = datamodel.ScaleUp
	} else {
		reservationSet.State = datamodel.ScaleDown
	}

	reservationSet.ReservationSetRequest = request
	reservationSet.Status = datamodel.Pending

	// Persist the updated reservation set
	if err = datamodel.IpReservationSetRepo.PersistReservationSet(reservationSet); err != nil {
		return err
	}

	if reservationSet.State == datamodel.ScaleUp {
		loggershim.LS.Debugf("attempting to scale up reservation set: %s to count %d", reservationSetID, request.NumReservations)
		if err = datamodel.IpReservationRepo.CreateReservationsInSet(networkID, request.SubnetName,
			reservationSetID, request.NumReservations, apiVersion); err != nil {
			loggershim.LS.Errorf("failed to create reservations in set: %s. Error: %v", reservationSetID, err)
			return err
		}

		return nil
	}

	loggershim.LS.Debugf("attempting to scale down reservation set: %s to count %d", reservationSetID, request.NumReservations)
	numReservationsToDelete := reservationSet.SuccessfulReservationCount - request.NumReservations
	numReservationsDeleted := 0
	// Delete the unassigned reservations to match the requested reservation count
	if numReservationsToDelete > 0 {
		for _, reservation := range reservations {
			if reservation.Configuration.AssociatedNCID == "" {
				if err := c.ipAllocator.DeleteIPReservation(reservation, false); err != nil {
					loggershim.LS.Errorf("failed to delete reservation: %s in set: %s. Error: %v",
						reservation.Configuration.ReservationID, reservationSetID, err)
					continue
				}
				numReservationsDeleted++
				if numReservationsDeleted == numReservationsToDelete {
					break
				}
			}
		}
	}

	// Update the reservation set
	reservationSet.SuccessfulReservationCount = reservationSet.SuccessfulReservationCount - numReservationsDeleted
	if numReservationsDeleted == numReservationsToDelete {
		reservationSet.Status = datamodel.Completed
	} else {
		reservationSet.Status = datamodel.Failed
	}

	if err = datamodel.IpReservationSetRepo.PersistReservationSet(reservationSet); err != nil {
		errMsg := fmt.Sprintf("could not persist reservationset: %s. Error: %v", reservationSetID, err)
		loggershim.LS.Errorf(errMsg)
		return api.InternalServerErrorf(api.StatusInternalServerError, errMsg)
	}

	if numReservationsDeleted != numReservationsToDelete {
		loggershim.LS.Errorf("failed to scale down reservation set: %s to count: %d. Existing reservation count: %d",
			reservationSetID, request.NumReservations, reservationSet.SuccessfulReservationCount)
		return api.BadRequestf(api.ReservationInUse, "could not delete the requested number of reservations. Requested: %d. Existing: %d",
			request.NumReservations, reservationSet.SuccessfulReservationCount)
	}

	loggershim.LS.Debugf("successfully completed reservation set request: %s with total reservation count %d",
		reservationSetID, request.NumReservations)
	return nil
}

// CreateOrUpdateReservationSetRequest - Create or update reservation set request
func (c *Controller) CreateOrUpdateReservationSetRequest(
	networkID string,
	reservationSetID string,
	request api.ReservationSetRequest,
	apiVersion string) (err *api.JsonErr) {
	loggershim.LS.Debugf("[Request] %+v ReservationSetID: %s", request, reservationSetID)

	defer func() {
		if err != nil {
			loggershim.LS.Errorf("CreateOrUpdateReservationSetRequest failed for reservation set: %s. error: %+v", reservationSetID, err)
		}
	}()

	var subnetConfig *datamodel.SubnetConfiguration
	if subnetConfig, err = datamodel.TryQuerySubnetConfiguration(networkID, request.SubnetName); err != nil {
		return
	}

	if subnetConfig == nil {
		err = api.BadRequestf(api.SubnetNotFound, "could not find subnet: %s in vnet: %s to create reservation set: %s",
			request.SubnetName, networkID, reservationSetID)
		return
	}

	// Return error if the subnet is not joined or if the subnet is marked for deletion
	if !subnetConfig.Joined {
		err = api.BadRequestf(api.SubnetIsNotJoined, "cannot create reservation set: %s as the subnet: %s "+
			"in vnet: %s is not registered with dnc", reservationSetID, request.SubnetName, networkID)
		return
	}

	if subnetConfig.MarkedForDelete {
		err = api.BadRequestf(api.SubnetMarkedForDeletion, "cannot create reservation set: %s as the subnet: %s " +
			"in vnet: %s is marked for deletion", reservationSetID, request.SubnetName, networkID)
		return
	}

	sessionInfo := map[string]string{datamodel.ReservationSetRepositoryName: reservationSetID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	// Check if the reservation set exists
	var reservationSet *datamodel.ReservationSet
	if reservationSet, err = datamodel.IpReservationSetRepo.TryQueryIPReservationSet(networkID, reservationSetID); err != nil {
		return
	}

	if reservationSet != nil {
		// Update existing reservation set
		if err = c.updateReservationSet(networkID, reservationSetID, request, reservationSet, apiVersion); err != nil {
			return err
		}
	} else {
		// Create new reservation set
		if err = c.createReservationSet(networkID, reservationSetID, request, apiVersion); err != nil {
			return err
		}
	}

	var reservationsToCreate []datamodel.ReservationRow
	reservationsToCreate, err = datamodel.IpReservationRepo.GetPendingReservationsInSet(networkID, reservationSetID)
	if err != nil {
		loggershim.LS.Errorf("failed to get pending reservations in set: %s. Error: %v", reservationSetID, err)
		return err
	}

	// Create pending work items for all the reservations as well as reservation set
	// todo: This should be in pendingworkitems repo. Move it there after separating that into a repo.
	pendingQueueRequests := make([]datamodel.PendingQueueRequest, len(reservationsToCreate)+1)
	createDateTime := time.Now().UTC()
	retryInterval := time.Second
	for i, reservationToCreate := range reservationsToCreate {
		pendingQueueRequests[i] = datamodel.PendingQueueRequest{
			Key: reservationToCreate.RowKey,
			Context: &datamodel.RequestContext{
				Type:                    datamodel.ReservationRequest,
				NetworkID:               networkID,
				CreatedDateTime:         createDateTime,
				RetryInterval:           retryInterval,
				StateTransitionDateTime: createDateTime,
			},
		}
	}

	pendingQueueRequestReservationSet := datamodel.PendingQueueRequest{
		Key: datamodel.GetReservationSetKey(networkID, reservationSetID),
		Context: &datamodel.RequestContext{
			Type:                    datamodel.ReservationSetRequest,
			NetworkID:               networkID,
			CreatedDateTime:         createDateTime,
			RetryInterval:           retryInterval,
			StateTransitionDateTime: createDateTime,
		},
	}

	pendingQueueRequests[len(reservationsToCreate)] = pendingQueueRequestReservationSet
	return datamodel.AddNewRequestsToPendingQueue(pendingQueueRequests)
}

// PersistIPReservationRequest - Persist reservation request
func (c *Controller) PersistIPReservationRequest(
	networkID string,
	reservationID string,
	allocationRequest api.AllocationRequest,
	apiVersion string) *api.JsonErr {
	loggershim.LS.Debugf("[Request] %s ReservationID: %s", allocationRequest.String(), reservationID)

	sessionInfo := map[string]string{datamodel.ReservationRepositoryName: reservationID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	return datamodel.PersistIPReservationRequest(
		networkID,
		reservationID,
		allocationRequest,
		apiVersion)
}

// GetReservationConfiguration - retrieve reservation configuration.
func (c *Controller) GetReservationConfiguration(
	networkID string,
	reservationID string) (*api.ReservationResponse, *api.JsonErr) {
	// No read lock is required because Azure Storage uses snapshot isolation to allow
	// read operations to happen concurrently with writes in the same partition. It
	// guarantees reads to see consistent data while updates are happening
	reservation, err := datamodel.QueryIPReservation(networkID, reservationID)
	if err != nil {
		return nil, err
	}
	return datamodel.GetReservationResponse(reservation)
}

// GetReservationSet - retrieves reservation set.
func (c *Controller) GetReservationSet(
	networkID string,
	reservationSetID string) (*api.ReservationSetResponse, *api.JsonErr) {
	reservationSet, err := datamodel.IpReservationSetRepo.TryQueryIPReservationSet(networkID, reservationSetID)
	if err != nil {
		return nil, err
	}

	if reservationSet == nil {
		return nil, api.NotFoundf(api.InvalidReservationSetID, "could not find reservation set: %s", reservationSetID)
	}

	reservations, err := datamodel.IpReservationRepo.TryQueryAllReservationsInSet(networkID, reservationSetID)
	if err != nil {
		return nil, err
	}

	var successfulReservationCount int
	reservationResponse := make([]api.ReservationResponse, len(reservations))
	for i, reservation := range reservations {
		response, err := datamodel.GetReservationResponse(reservation)
		if err != nil {
			return nil, err
		}

		reservationResponse[i] = *response

		if reservation.Configuration.Status == datamodel.Completed {
			successfulReservationCount++
		}
	}

	response := api.ReservationSetResponse{
		NetworkID:                  reservationSet.NetworkID,
		SubnetName:                 reservationSet.SubnetName,
		ReservationSetID:           reservationSet.ReservationSetID,
		RequestedReservationCount:  reservationSet.ReservationSetRequest.NumReservations,
		SuccessfulReservationCount: successfulReservationCount,
		TotalReservationCount:      len(reservations),
		Reservations:               reservationResponse,
		RequestStatus: api.RequestStatus{
			State:         reservationSet.State.String(),
			Status:        reservationSet.Status.String(),
			FailureDetail: reservationSet.FailureDetail,
		},
	}

	return &response, nil
}

// GetReservationSetStatus - Get reservation set status
func (c *Controller) GetReservationSetStatus(
	networkID string,
	reservationSetID string) (*api.RequestStatus, int, *api.JsonErr) {
	reservationSet, err := datamodel.IpReservationSetRepo.TryQueryIPReservationSet(networkID, reservationSetID)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	if reservationSet == nil {
		return nil, http.StatusNotFound, api.NotFoundf(api.InvalidReservationSetID, "could not find reservation set: %s", reservationSetID)
	}

	requestStatus := api.RequestStatus{
		State:         reservationSet.State.String(),
		Status:        reservationSet.Status.String(),
		FailureDetail: reservationSet.FailureDetail,
	}

	return &requestStatus, generateStatus(reservationSet.Status, &reservationSet.FailureDetail), err
}

// GetReservationStatus - Get reservation status
func (c *Controller) GetReservationStatus(
	networkID,
	reservationID string) (*api.RequestStatus, int, *api.JsonErr) {
	// No read lock is required because Azure Storage uses snapshot isolation to allow
	// read operations to happen concurrently with writes in the same partition. It
	// guarantees reads to see consistent data while updates are happening

	reservation, err := datamodel.QueryIPReservation(networkID, reservationID)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	requestStatus := &api.RequestStatus{
		State:         reservation.Configuration.ReservationState.String(),
		Status:        reservation.Configuration.Status.String(),
		FailureDetail: reservation.Configuration.FailureDetail}
	return requestStatus, generateStatus(reservation.Configuration.Status, &reservation.Configuration.FailureDetail), err
}

// DeleteReservation - Delete reservation configuration.
func (c *Controller) DeleteReservation(
	networkID string,
	reservationID string) *api.JsonErr {
	// Prevent explicit deletion of the reservation if it belongs to a reservation set
	reservation, err := datamodel.TryQueryIPReservation(networkID, reservationID)
	if err != nil {
		return err
	}

	if reservation.Configuration == nil {
		return nil
	}

	if reservation.SetKey != "" {
		return api.BadRequestf(api.InvalidReservationRequest, "reservation deletion is not allowed as it belongs to a set")
	}

	return c.deleteReservation(networkID, reservationID)
}

func (c *Controller) deleteReservation(
	networkID string,
	reservationID string) *api.JsonErr {
	sessionInfo := map[string]string{
		datamodel.NetworkRepositoryName:     networkID,
		datamodel.ReservationRepositoryName: reservationID,
	}

	reservation, err := datamodel.TryQueryIPReservation(networkID, reservationID)
	// Check if the node session lock is required
	if reservation.Configuration != nil && err == nil {
		if len(reservation.Configuration.AllocationRequest.NodeConstraint) != 0 {
			sessionInfo[datamodel.NodeRepositoryName] = reservation.Configuration.AllocationRequest.NodeConstraint
		}
	}

	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	return c.ipAllocator.DeleteIPReservation(reservation, false)
}

func generateStatus(status datamodel.Status, errorInfo *api.JsonErr) int {
	switch status {
	case datamodel.Pending:
		fallthrough
	case datamodel.NotStarted:
		return http.StatusAccepted
	case datamodel.Failed:
		return errorInfo.HTTPStatus
	case datamodel.Completed:
		return http.StatusOK
	case datamodel.Denied:
		return http.StatusForbidden
	}

	return http.StatusInternalServerError
}

// GetNetworkStatus - Get network status
func (c *Controller) GetNetworkStatus(networkID string) (*api.RequestStatus, int, *api.JsonErr) {
	// No read lock is required because Azure Storage uses snapshot isolation to allow
	// read operations to happen concurrently with writes in the same partition. It
	// guarantees reads to see consistent data while updates are happening
	vnetCfg, err := datamodel.QueryNetworkConfiguration(networkID)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	requestStatus := &api.RequestStatus{
		State:         vnetCfg.NetworkState.String(),
		Status:        vnetCfg.Status.String(),
		FailureDetail: vnetCfg.FailureDetail}

	return requestStatus, generateStatus(vnetCfg.Status, &vnetCfg.FailureDetail), nil
}

// GetSubnetStatus - Get subnet status
func (c *Controller) GetSubnetStatus(networkID, subnetName string) (*api.RequestStatus, int, *api.JsonErr) {
	var (
		requestError      *api.JsonErr
		requestReturnCode int
		requestStatus     api.RequestStatus
	)

	requestStatus.State = datamodel.Join.String()

	// No read lock is required because Azure Storage uses snapshot isolation to allow
	// read operations to happen concurrently with writes in the same partition. It
	// guarantees reads to see consistent data while updates are happening
	subnetConfig, err := datamodel.QuerySubnetConfiguration(networkID, subnetName)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	// If the subnet is not joined / registered, return StatusNotFound
	if !subnetConfig.Joined {
		return nil, http.StatusNotFound, api.NotFoundf(api.SubnetNotFound, "Subnet: %s in network: %s not found", subnetName, networkID)
	}

	switch subnetConfig.UpdateStatus {
	case datamodel.SubnetUpdateCompleted:
		requestStatus.Status = datamodel.Completed.String()
		requestReturnCode = http.StatusOK
		requestError = nil

	case datamodel.SubnetUpdateFailed:
		requestStatus.Status = datamodel.Failed.String()
		requestReturnCode = http.StatusNotFound
		requestError = api.NotFoundf(api.SubnetNotFound, "Network %s does not contain the specified subnet %s", networkID, subnetName)

	case datamodel.SubnetUpdatePending:
		requestStatus.Status = datamodel.Pending.String()
		requestReturnCode = http.StatusAccepted
		requestError = api.NotFoundf(api.JoinPending, "Network: %s subnet: %s join pending", networkID, subnetName)

	case datamodel.SubnetCleanupFailed:
		requestStatus.Status = datamodel.Failed.String()
		requestReturnCode = subnetConfig.FailureDetail.HTTPStatus
		requestError = api.InternalServerErrorf(subnetConfig.FailureDetail.Code, subnetConfig.FailureDetail.Text)

	case datamodel.SubnetCleanupPending:
		requestStatus.Status = datamodel.Pending.String()
		requestReturnCode = http.StatusAccepted
		requestError = api.NotFoundf(api.ProcessingPending, "Network: %s subnet: %s cleanup pending", networkID, subnetName)
	}

	return &requestStatus, requestReturnCode, requestError
}

// GetSubnetConfiguration - Get subnet configuration
func (c *Controller) GetSubnetConfiguration(networkID, subnetName string) (*api.SubnetConfigurationResponse, *api.JsonErr) {
	// No read lock is required because Azure Storage uses snapshot isolation to allow
	// read operations to happen concurrently with writes in the same partition. It
	// guarantees reads to see consistent data while updates are happening
	repoSubnetConfiguration, err := datamodel.QuerySubnetConfiguration(networkID, subnetName)
	if err != nil {
		return nil, err
	}

	// If the subnet is not joined, return StatusNotFound
	if !repoSubnetConfiguration.Joined {
		return nil, api.NotFoundf(api.SubnetNotFound, "Configuration for subnet: %s in network: %s not found", subnetName, networkID)
	}

	reservations, err := datamodel.GetReservationsInSubnet(networkID, subnetName)
	if err != nil {
		return nil, err
	}

	ipAllocationsData := make(api.IPAllocationsData, len(reservations))
	for i, reservation := range reservations {
		ipAllocationsData[i] = api.IPAllocationData{
			IPAddress:                    reservation.Configuration.AllocatedSubnet,
			AssociatedNetworkContainerID: reservation.Configuration.AssociatedNCID,
			ReservationID:                reservation.Configuration.ReservationID}
	}

	dnsServers, err := datamodel.GetSubnetDNSServers(repoSubnetConfiguration)
	subnetConfigurationResponse := api.SubnetConfigurationResponse{
		NetworkID:         networkID,
		SubnetName:        subnetName,
		IPSubnet:          repoSubnetConfiguration.IPSubnet,
		GatewayIP:         repoSubnetConfiguration.Gateway,
		DNSServers:        dnsServers,
		IPAllocationsData: ipAllocationsData,
		AccessToken:       fmt.Sprintf("%x", sha256.Sum256([]byte(repoSubnetConfiguration.AccessToken)))}

	return &subnetConfigurationResponse, nil
}

// GetNetworkContainerStatus - Get network container status
func (c *Controller) GetNetworkContainerStatus(
	networkID, networkContainerID string) (*api.NCRequestStatus, int, *api.JsonErr) {
	// No read lock is required because Azure Storage uses snapshot isolation to allow
	// read operations to happen concurrently with writes in the same partition. It
	// guarantees reads to see consistent data while updates are happening
	nc, err := datamodel.NcRepo.GetNetworkContainerData(networkID, networkContainerID)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	if nc.RowKey == "" {
		return nil, http.StatusNotFound, api.NotFoundf(api.NetworkContainerNotFound,
			"could not find NC: %s in network: %s", networkContainerID, networkID)
	}

	ncType := nc.Config.GetAssociationInfo().ContainerType
	//nicStatus set by default to Not Applicable in case it is a non-Baremetal scenario
	nicStatus := "N/A"
	//checking if NC type is Baremetal to call GetNICProgammingStatus and to set the nicStatus
	if ncType == api.BareMetalContainerType {
		aznmNICStatusData, aznmstatusError := pubsubproxy.GetNICProgrammingStatus(nc.Config.ReservationID)
		nicStatus = aznmNICStatusData
		//if the aznm NIC status returned is not nil, return the error string to api
		if aznmstatusError != nil {
			nicStatus = aznmstatusError.Error()
		}
	}

	var requestStatus = &api.NCRequestStatus{
		State:         nc.Config.State.String(),
		Status:        nc.Config.Status.String(),
		NICStatus:     nicStatus,
		FailureDetail: nc.Config.FailureDetail}

	return requestStatus, generateStatus(nc.Config.Status, &nc.Config.FailureDetail), nil
}

// PersistNetworkJoinRequest - Persists network join request
func (c *Controller) PersistNetworkJoinRequest(
	networkID string,
	networkInfo api.NetworkInfoRequest,
	apiVersion string) *api.JsonErr {
	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	return joinNetworkFunc(networkID, apiVersion)
}

func joinNetworkViaRNM(networkID string, apiVersion string) *api.JsonErr {
	loggershim.LS.Debugf("[Request] Join Vnet via RNM %s", networkID)
	if config.PubSubProxySettings.EnablePubSubProxy {
		// If PubSubProxy is enabled, do not add the vnet join request to the pending queue
		if err := datamodel.CreateNetwork(networkID, apiVersion, false); err != nil {
			return err
		}

		pubsubproxy.RegisterVnet(networkID)
		return nil
	}

	return datamodel.CreateNetwork(networkID, apiVersion, true)
}

func joinNetworkViaRNC(networkID string, apiVersion string) *api.JsonErr {
	loggershim.LS.Debugf("[Request] Join Vnet via RNC %s", networkID)
	if !config.PubSubProxySettings.EnablePubSubProxy {
		return api.BadRequestf(api.InvalidNetworkRequest, "Pubsubproxy should be enabled for network join via RNC")
	}

	// If DNC is subscribing to RNC published paths, there is no need to subscribe to vnet via pubsub proxy. RNC publishes
	// vnet and subnet data at different paths and DNC only needs subnet. Save the network for backward compatibility and
	// return. Pubsub proxy should be enabled for RNC path though as subnet subscription will need to go through pubsub proxy.
	// If PubSubProxy is enabled, do not add the vnet join request to the pending queue
	err := datamodel.CreateNetworkCompleted(networkID, apiVersion, false)
	if err == nil {
		loggershim.LS.Debugf("[Request] Finished Join Vnet via RNC %s", networkID)
	}

	return err
}

// PersistSubnetDelegationRequest - Persist subnet delegation info.
func (c *Controller) PersistSubnetDelegationRequest(
	networkID,
	subnet string,
	subnetInfo api.SubnetRequest,
	isDelete bool) *api.JsonErr {
	loggershim.LS.Debugf("[Request] Put Subnet: %s for network: %s isDelete: %t", subnet, networkID, isDelete)

	if err := ValidateSubnetRequest(subnet, ""); err != nil {
		return err
	}

	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	if err := c.updateSubnetV1(networkID, subnet, subnetInfo, isDelete); err != nil {
		loggershim.LS.Errorf("UpdateSubnet: subnet: %s failed. Error: %v", datamodel.GetSubnetKey(networkID, subnet), err)
		return err
	}

	return nil
}

func (c *Controller) updateSubnetV1(networkID, subnet string, subnetRequest api.SubnetRequest, isDelete bool) *api.JsonErr {
	networkConfig, err := datamodel.TryQueryNetworkConfiguration(networkID)
	if err != nil {
		return err
	}

	if networkConfig == nil {
		return api.NotFoundf(api.NetworkNotFound, "could not find network: %s", networkID)
	}

	if networkConfig.NetworkState == datamodel.Join && networkConfig.Status != datamodel.Completed {
		return api.BadRequestf(api.NetworkStatusInvalid, "Network join for %s is not complete. Current status: %s", networkID, networkConfig.Status.String())
	}

	var subnetFound = true
	if len(networkConfig.Subnets) == 0 {
		subnetFound = false
	}
	if _, ok := networkConfig.Subnets[subnet]; !ok {
		subnetFound = false
	}

	if !subnetFound {
		loggershim.LS.Debugf("Subnet %s not found in Network with ID %s. Refreshing network", subnet, networkID)
		networkConfig.Status = datamodel.Refreshing
		networkConfig.Subnets[subnet] = datamodel.SubnetKeyPlaceholder
		if err = datamodel.PersistNetworkConfiguration(networkConfig); err != nil {
			return err
		}

		subnetConfig := datamodel.SubnetConfiguration{
			NetworkID:    networkID,
			SubnetName:   subnet,
			AccessToken:  subnetRequest.AuthenticationToken,
			Joined:       true,
			UpdateStatus: datamodel.SubnetUpdatePending,
		}
		if err := datamodel.PutSubnetConfiguration(&subnetConfig); err != nil {
			return api.BadRequestf(api.SubnetNotFound, "Failed to store subnet config %s in network %s: %v", subnet, networkID, err)
		}

		requestContext := datamodel.RequestContext{
			Type:      datamodel.VirtualNetworkRequest,
			NetworkID: networkID,
		}

		return datamodel.AddRequestToPendingQueue(datamodel.GetNetworkKey(networkID), &requestContext)
	}

	subnetConfiguration, err := datamodel.TryQuerySubnetConfiguration(networkID, subnet)
	if err != nil {
		return err
	}

	if subnetConfiguration == nil {
		return api.BadRequestf(api.SubnetConfigurationNotFound, "could not find subnet configuration: %s in network: %s", subnet, networkID)
	}

	if subnetConfiguration.UpdateStatus == datamodel.SubnetUpdatePending {
		// No error is returned as the request is treated as idempotent while
		// the same request issued earlier is in process
		return nil
	}

	if subnetConfiguration.UpdateStatus == datamodel.SubnetUpdateFailed {
		return api.NotFoundf(api.SubnetNotFound, "could not find subnet: %s in network: %s", subnet, networkID)
	}

	if !subnetConfiguration.IsDelegatedSubnet {
		return api.BadRequestf(api.SubnetIsNotDelegated, "subnet: %s in network: %s is not delegated", subnet, networkID)
	}

	if isDelete {
		reservations, err := datamodel.GetReservationsInSubnet(networkID, subnet)
		if err != nil {
			return api.InternalServerErrorf(api.StatusInternalServerError,
				"could not retrieve reservations for subnet: %s. Error: %v", subnet, err)
		}

		if len(reservations) > 0 {
			return api.BadRequestf(api.InvalidSubnetRequest, "cannot remove auth token as subnet: %s has allocations", subnet)
		}
	}

	if config.FeatureFlagSettings.UsePrepopulatedIPAllocations && !isDelete {
		if err := c.ipAllocator.PrepopulateSubnet(networkID, subnet, subnetConfiguration.IPSubnet); err != nil {
			return err
		}
	}

	subnetConfiguration.Joined = !isDelete
	subnetConfiguration.AccessToken = subnetRequest.AuthenticationToken
	return datamodel.PutSubnetConfiguration(subnetConfiguration)
}

func (c *Controller) CleanupSubnet(networkID, subnetName string) *api.JsonErr {
	loggershim.LS.Debugf("[Request] CleanupSubnet: Network: %s, Subnet: %s", networkID, subnetName)

	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	return datamodel.CreateSubnetCleanupRequest(networkID, subnetName)
}

func (c *Controller) DeleteSubnetV2(networkID, subnetName string, apiversion string) *api.JsonErr {
	loggershim.LS.Debugf("[Request] DeleteSubnetV2: Network: %s, Subnet: %s", networkID, subnetName)
	if !config.PubSubProxySettings.EnablePubSubProxy {
		// SubnetV2 API is supported with PubSubProxy only when DNC is running in managed mode
		return api.BadRequestf(api.RequestNotsupported, "Request not supported")
	}

	if err := ValidateSubnetRequest(subnetName, apiversion); err != nil {
		return err
	}

	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	networkConfiguration, err := datamodel.TryQueryNetworkConfiguration(networkID)
	if err != nil {
		loggershim.LS.Errorf("Failed to retrieve network configuration for network: %s. Error: %v", networkID, err)
		return err
	}

	if networkConfiguration == nil {
		loggershim.LS.Debugf("No network configuration for network: %s", networkID)
		return nil
	}

	requestedSubnetConfig, err := datamodel.TryQuerySubnetConfiguration(networkID, subnetName)
	if err != nil {
		loggershim.LS.Errorf("Failed to retrieve subnet configuration for subnet: %s in network: %s. Error: %v", subnetName, networkID, err)
		return err
	}

	if requestedSubnetConfig == nil {
		loggershim.LS.Debugf("No subnet configuration for subnet: %s in network: %s", subnetName, networkID)
		return nil
	}

	// Delete subnet is not allowed while the subnet is already being cleaned up with CleanupAndDeleteSubnet API
	if requestedSubnetConfig.MarkedForDelete && requestedSubnetConfig.UpdateStatus == datamodel.SubnetCleanupPending {
		errMsg := fmt.Sprintf("Delete subnet not allowed while it's already being cleaned up. subnet: %s, network: %s", subnetName, networkID)
		loggershim.LS.Errorf(errMsg)
		return api.BadRequestf(api.StatusBadRequest, errMsg)
	}

	reservations, err := datamodel.TryQueryAllReservationsInSubnet(networkID, subnetName)
	if err != nil {
		loggershim.LS.Errorf("Failed to retrieve reservations for subnet: %s in network: %s. Error: %v", subnetName, networkID, err)
		return err
	}

	// Do not delete the subnet if there are allocations in the subnet
	if len(reservations) > 0 {
		reservationKeys := make([]string, 0, len(reservations))
		for _, reservation := range reservations {
			reservationKeys = append(reservationKeys, reservation.RowKey)
		}
		reservationsString := "[" + strings.Join(reservationKeys, ", ") + "]"
		return api.BadRequestf(api.SubnetInUse,
			"Subnet: %s in Network: %s has allocations: %s", subnetName, networkID, reservationsString)
	}

	// Mark the subnet as 'not joined' anymore and persist the updated subnet config
	requestedSubnetConfig.Joined = false
	requestedSubnetConfig.MarkedForDelete = true
	loggershim.LS.Debugf("Updating subnet: %s in network: %s for deletion", subnetName, networkID)
	if err := datamodel.PutSubnetConfiguration(requestedSubnetConfig); err != nil {
		loggershim.LS.Errorf("Failed to persist subnet configuration for subnet: %s in network: %s. Error: %v", subnetName, networkID, err)
		return err
	}

	if config.FeatureFlagSettings.UsePrepopulatedIPAllocations {
		if err := c.ipAllocator.DepopulateSubnet(networkID, subnetName); err != nil {
			loggershim.LS.Errorf("Failed to delete prepopulated allocations for subnet %s in network %s: %v", subnetName, networkID, err)
			return err
		}
	}

	return deleteSubnetFunc(networkID, subnetName, networkConfiguration)
}

// PersistSubnetDelegationRequest - Persist subnet delegation info for V2 API.
func (c *Controller) PersistSubnetDelegationRequestV2(networkID, subnet string, subnetRequest api.SubnetRequest, apiVersion string) *api.JsonErr {
	loggershim.LS.Debugf("[Request] PutSubnetV2: Network: %s, Subnet: %s", networkID, subnet)
	if !config.PubSubProxySettings.EnablePubSubProxy {
		// SubnetV2 API is supported with PubSubProxy only when DNC is running in managed mode
		return api.BadRequestf(api.RequestNotsupported, "Request not supported")
	}

	if err := ValidateSubnetRequest(subnet, apiVersion); err != nil {
		return err
	}

	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	networkConfig, err := datamodel.TryQueryNetworkConfiguration(networkID)
	if err != nil {
		loggershim.LS.Errorf("Failed to retrieve network config for Network: %s, Subnet: %s", networkID, subnet)
		return err
	}

	isNetworkJoined := true
	if networkConfig == nil {
		networkConfig = &datamodel.NetworkConfiguration{
			NetworkID:    networkID,
			NetworkState: datamodel.Join,
			Status:       getNetworkStatusFunc(),
			APIVersion:   apiVersion,
			Subnets:      make(map[string]string),
		}
		isNetworkJoined = false
	}

	// Add the subnet to the Network Configuration if not added already
	if _, ok := networkConfig.Subnets[subnet]; !ok {
		networkConfig.Subnets[subnet] = datamodel.SubnetKeyPlaceholder
		if err := datamodel.PersistNetworkConfiguration(networkConfig); err != nil {
			loggershim.LS.Errorf("Failed to persist network config for Network: %s, Subnet: %s", networkID, subnet)
			return err
		}
	}

	// Check if the Subnet Configuration for the subnet exists. If it doesn't exist, add a config with status SubnetUpdatePending
	// else just update the auth token and joined flag
	subnetConfig, err := datamodel.TryQuerySubnetConfiguration(networkID, subnet)
	if err != nil {
		loggershim.LS.Errorf("Failed to retrieve subnet config for Network: %s, Subnet: %s", networkID, subnet)
		return err
	}

	if subnetConfig == nil {
		loggershim.LS.Debugf("Creating new subnet config for Network: %s, Subnet: %s", networkID, subnet)
		subnetConfig = &datamodel.SubnetConfiguration{
			NetworkID:    networkID,
			SubnetName:   subnet,
			AccessToken:  subnetRequest.AuthenticationToken,
			Joined:       true,
			UpdateStatus: datamodel.SubnetUpdatePending,
		}
	} else {
		// Cannot join the subnet if it's marked for deletion
		if subnetConfig.MarkedForDelete {
			errMsg := fmt.Sprintf("Cannot register/join subnet: %s as it's marked for deletion",
				datamodel.GetSubnetKey(subnetConfig.NetworkID, subnetConfig.SubnetName))
			loggershim.LS.Errorf(errMsg)
			return api.BadRequest(api.StatusBadRequest, errMsg)
		}

		loggershim.LS.Debugf("Updating the subnet config for Network: %s, Subnet: %s with access token and marking it as Joined", networkID, subnet)
		subnetConfig.AccessToken = subnetRequest.AuthenticationToken
		subnetConfig.Joined = true

		// todo: how to handle updates to cidr size?
		if config.FeatureFlagSettings.UsePrepopulatedIPAllocations {
			if err := c.handleSubnetPrepopulation(subnetConfig); err != nil {
				loggershim.LS.Errorf("Could not prepopulate vnet id %s subnet %s %s: %v", subnetConfig.NetworkID, subnetConfig.SubnetName, subnetConfig.IPSubnet, err)
				return err
			}
		}
	}

	// Persist the subnet configuration
	if err := datamodel.PutSubnetConfiguration(subnetConfig); err != nil {
		loggershim.LS.Errorf("Failed to persist subnet config for Network: %s, Subnet: %s", networkID, subnet)
		return err
	}

	registerSubnetFunc(networkID, subnet, isNetworkJoined)
	loggershim.LS.Debugf("PutSubnetV2 successfully persisted subnet config for Network: %s, Subnet: %s", networkID, subnet)
	return nil
}

func (c *Controller) handleSubnetPrepopulation(subnet *datamodel.SubnetConfiguration) *api.JsonErr {
	if subnet.UpdateStatus != datamodel.SubnetUpdateCompleted {
		loggershim.LS.Debugf("Cannot prepopulate vnet id %s subnet %s as subnet is still in pending status. Will be handled on network update", subnet.NetworkID, subnet.SubnetName)
		return nil
	}

	vnetID := subnet.NetworkID
	subnetName := subnet.SubnetName
	subnetCIDR := subnet.IPSubnet

	loggershim.LS.Debugf("Prepopulating vnet id %s subnet %s %s", vnetID, subnetName, subnetCIDR)
	if err := c.ipAllocator.PrepopulateSubnet(vnetID, subnetName, subnetCIDR); err != nil {
		return err
	}

	loggershim.LS.Debugf("Successfully prepopulated vnet id %s subnet %s %s", vnetID, subnetName, subnetCIDR)
	return nil
}

// GetNetwork - Get network configuration (e.g. delegated subnets in network)
func (c *Controller) GetNetwork(networkID string) (*api.NetworkConfigurationResponse, *api.JsonErr) {
	// No read lock is required because Azure Storage uses snapshot isolation to allow
	// read operations to happen concurrently with writes in the same partition. It
	// guarantees reads to see consistent data while updates are happening

	return datamodel.GetNetworkConfiguration(networkID)
}

// GetNetworks - Get vnets managed by this DNC
func (c *Controller) GetNetworks() (*api.NetworksResponse, *api.JsonErr) {
	// No read lock is required because Azure Storage uses snapshot isolation to allow
	// read operations to happen concurrently with writes in the same partition. It
	// guarantees reads to see consistent data while updates are happening

	return datamodel.GetNetworks()
}

// DeleteNetwork - Delete a network
func (c *Controller) DeleteNetwork(networkID string) *api.JsonErr {
	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	return deleteNetworkFunc(networkID)
}

// DeleteNetwork - Delete a network
func deleteNetworkViaRNM(networkID string) *api.JsonErr {
	loggershim.LS.Debugf("[Request] DeleteNetworkViaRNM: %s", networkID)

	err := datamodel.DeleteNetworkConfiguration(networkID)
	if err == nil && config.PubSubProxySettings.EnablePubSubProxy {
		// Unsubscribe vnet notifications for PubSubProxy
		pubsubproxy.UnregisterVnet(networkID)
	}

	return err
}

// DeleteNetwork - Delete a network
func deleteNetworkViaRNC(networkID string) *api.JsonErr {
	loggershim.LS.Debugf("[Request] DeleteNetworkViaRNC: %s", networkID)
	// When RNC path is enabled, network is only deleted from Database. Since
	// we don't subscribe to vnet for RNC path, no need to unsubscribe it from pubsub
	return datamodel.DeleteNetworkConfiguration(networkID)
}

// PersistNetworkContainerRequestV1 - Persist network container request using dynamic allocation
func (c *Controller) PersistNetworkContainerRequestV1(
	networkID, ncID string,
	ncRequestV1 *api.NetworkContainerRequestV1,
	apiVersion string) *api.JsonErr {
	loggershim.LS.Debugf("[Request] %s NC:%s", ncRequestV1.String(), ncID)

	sessionInfoWrite := map[string]string{datamodel.NCRepositoryName: ncID}
	c.SessionManager.BeginWriteSession(sessionInfoWrite)
	defer c.SessionManager.EndWriteSession(sessionInfoWrite)

	sessionInfoRead := map[string]string{datamodel.NodeRepositoryName: ncRequestV1.AssociationInfo.NodeID}
	c.SessionManager.BeginReadSession(sessionInfoRead)
	defer c.SessionManager.EndReadSession(sessionInfoRead)

	return datamodel.CreateOrUpdateNCRequestV1(networkID, ncID, ncRequestV1, apiVersion)
}

// PersistNetworkContainerRequestV2 - Persist network container request using dynamic allocation
func (c *Controller) PersistNetworkContainerRequestV2(
	networkID, ncID string,
	ncRequestV2 *api.NetworkContainerRequestV2,
	apiVersion string) *api.JsonErr {
	loggershim.LS.Debugf("[Request] %s NC:%s", ncRequestV2.String(), ncID)

	sessionInfoWrite := map[string]string{datamodel.NCRepositoryName: ncID}
	c.SessionManager.BeginWriteSession(sessionInfoWrite)
	defer c.SessionManager.EndWriteSession(sessionInfoWrite)

	sessionInfoRead := map[string]string{
		datamodel.NodeRepositoryName:    ncRequestV2.AssociationInfo.NodeID,
		datamodel.NetworkRepositoryName: networkID,
	}
	c.SessionManager.BeginReadSession(sessionInfoRead)
	defer c.SessionManager.EndReadSession(sessionInfoRead)

	return datamodel.CreateOrUpdateNCRequestV2(networkID, ncID, ncRequestV2, apiVersion)
}

// PromoteContainerState - Attempt to promote container stuck in an undesirable state (as a result of unreachable node)
func (c *Controller) PromoteContainerState(networkID, ncID string) *api.JsonErr {
	sessionInfo := map[string]string{datamodel.NCRepositoryName: ncID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	nc, err := datamodel.NcRepo.TryQueryNetworkContainer(networkID, ncID)
	if err != nil {
		return err
	}

	if nc.RowKey == "" {
		return api.NotFoundf(api.NetworkContainerNotFound, "could not find NC: %s to promote the state",
			datamodel.GetNCKey(networkID, ncID))
	}

	return datamodel.PromoteContainerState(nc)
}

// SkipNodeCleanupForNetworkContainer - Attempt to promote container stuck in an undesirable state (as a result of unreachable node)
func (c *Controller) SkipNodeCleanupForNetworkContainer(networkID, ncID string) *api.JsonErr {
	sessionInfo := map[string]string{datamodel.NCRepositoryName: ncID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	nc, err := datamodel.NcRepo.TryQueryNetworkContainer(networkID, ncID)
	if err != nil {
		return err
	}

	if nc.RowKey == "" {
		return api.NotFoundf(api.NetworkContainerNotFound, "could not find the data for NC: %s", datamodel.GetNCKey(networkID, ncID))
	}

	if nc.Config.FailureDetail.Code == api.DisAssociationFailed {
		return datamodel.PromoteContainerState(nc)
	}

	return nil
}

// GetNetworkContainerConfiguration - Retrieves the complete network configuration
func (c *Controller) GetNetworkContainerConfiguration(networkID, ncID string) (*datamodel.NCConfiguration, *api.JsonErr) {
	// No read lock is required because Azure Storage uses snapshot isolation to allow
	// read operations to happen concurrently with writes in the same partition. It
	// guarantees reads to see consistent data while updates are happening
	nc, err := datamodel.NcRepo.GetNetworkContainerData(networkID, ncID)
	if err != nil {
		return nil, err
	}

	if nc.RowKey == "" {
		return nil, api.NotFoundf(api.NetworkContainerNotFound, "could not find NC: %s in network: %s", ncID, networkID)
	}

	if nc.Config.GetAssociationInfo().ContainerType == api.BareMetalContainerType {
		context := client.OrchestratorContext{}
		unmarshalErr := json.Unmarshal([]byte(nc.Config.GetAssociationInfo().OrchestratorContext), &context)
		if unmarshalErr != nil {
			loggershim.LS.Debugf("Unmarshalling OrchestratorContext Failed %+v", unmarshalErr)
			return nil, api.InternalServerError(api.StatusInternalServerError, unmarshalErr.Error())
		}
		//split string to take the latter part of PodName for LogicalContainerID
		logicalContainerID := strings.Split(context.PodName, "_")
		nc.Config.BMContainerInfo = fmt.Sprintf("{Physical ContainerID: %s, Logical NICID: %s, Logical ContainerID: %s}",
			nc.Config.GetInstanceID(), nc.Config.ReservationID, logicalContainerID[1])
	}

	return &nc.Config, nil
}

// GetNetworkContainerCleanupStatus returns the cleanup status of the nc.
func (c *Controller) GetNetworkContainerCleanupStatus(networkID, ncID string) (api.NetworkContainerCleanupResponse, *api.JsonErr) {
	cleanupResp := api.NetworkContainerCleanupResponse{Completed: false}
	// First, check the network container table to see if the NC exists
	_, err := datamodel.NcRepo.GetNetworkContainerData(networkID, ncID)
	if err == nil {
		// no error means the nc exists, so cleanup isn't done
		return cleanupResp, nil
	}

	if err.HTTPStatus != http.StatusNotFound {
		// expecting a 404, anything else should be considered an error and passed back to caller
		return cleanupResp, err
	}

	// NC doesn't exist. Now check pending cleanup jobs table to ensure there are no pending jobs for this NC
	pendingKey := datamodel.GetPendingResourceCleanupKey(datamodel.IPCleanup, networkID, ncID)
	entity, err := datamodel.PendingCleanupRepo.TryGetPendingCleanupRequest(pendingKey)
	if err != nil {
		// API returns error if it is not 404, bubble up the error to client
		return cleanupResp, err
	}

	if entity != nil {
		// Return as there still exists a pending Cleanup
		return cleanupResp, nil
	}

	// there are no pending jobs left for this NC, it is safe to return with Completed status
	cleanupResp.Completed = true
	return cleanupResp, nil
}

// GetNetworkContainerRequest - Retrieves cns.CreateNetworkContainerRequest
func (c *Controller) GetNetworkContainerRequest(networkID, ncID string) (*cns.CreateNetworkContainerRequest, *api.JsonErr) {
	nc, err := datamodel.NcRepo.GetNetworkContainerData(networkID, ncID)
	if err != nil {
		return nil, err
	}

	ncGoalState, isRetriable, err := datamodel.GetNCGoalState(&nc.Config)
	if err != nil {
		if isRetriable {
			err.HTTPStatus = http.StatusProcessing
		}
		return nil, err
	}

	ncRequest, isRetriable, err := datamodel.GetNetworkContainerRequest(ncGoalState, &nc)
	if err != nil {
		if isRetriable {
			err.HTTPStatus = http.StatusProcessing
		}
		return nil, err
	}

	return ncRequest, nil
}

// GetNetworkContainer - Retrieves the network container configuration
func (c *Controller) GetNetworkContainer(networkID, ncID string) (*api.NetworkContainerResponse, *api.JsonErr) {
	loggershim.LS.Debugf("[Request] GetNetworkContainer: %s", ncID)
	ncConfig, err := c.GetNetworkContainerConfiguration(networkID, ncID)
	if err != nil {
		return nil, err
	}

	return datamodel.GetNetworkContainerResponse(ncConfig)
}

// DeleteNetworkContainer - Deletes a network container.
func (c *Controller) DeleteNetworkContainer(networkID, ncID string) *api.JsonErr {
	loggershim.LS.Debugf("[Request] DeleteNetworkContainer: vnet id %s nc id %s", networkID, ncID)
	sessionInfo := map[string]string{datamodel.NCRepositoryName: ncID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	if err := datamodel.ProcessNetworkContainerForRelease(networkID, ncID); err != nil {
		loggershim.LS.Debugf("[Request] DeleteNetworkContainer: could not process vnet id %s nc id %s for release: %v", networkID, ncID, err)
		return err
	}

	return nil
}

// GetPrimaryIPEndpoint - Retrieves the current primary replica endpoint
func (c *Controller) GetPrimaryIPEndpoint() (string, error) {
	return c.Replica.GetPrimaryIPEndpoint()
}

// GetReplicaIPEndpoint - Retrieves the endpoint of this replica.
func (c *Controller) GetReplicaIPEndpoint() (string, error) {
	return c.Replica.GetReplicaIPEndpoint()
}

// GetReplicaIPEndpoint - Retrieves the tls endpoint of this replica.
func (c *Controller) GetReplicaTLSEndpoint() (string, error) {
	return c.Replica.GetReplicaTLSEndpoint()
}

// GetPrimaryReverseProxy - Get the reverse proxy to the primary used by this replica
func (c *Controller) GetPrimaryReverseProxy() *httputil.ReverseProxy {
	return c.Replica.GetPrimaryReverseProxy()
}

// InitializeAzureStorage - Tries to establish a connection with azure storage with information provided in request.
func (c *Controller) InitializeAzureStorage(request api.AzureStorageRequest) *api.JsonErr {
	loggershim.LS.Debugf("[Request] %s", request.String())
	config.DataStoreSettings.AzureStorageInfo.AccountName = request.AccountName
	config.DataStoreSettings.AzureStorageInfo.Key = request.Key
	config.DataStoreSettings.AzureStorageInfo.PartitionKey = request.PartitionKey
	masterElectionShim.SetDataStoreSettings(config)
	masterElectionShim.Init()

	err := datamodel.Init(
		config.CNSSettings,
		config.DataStoreSettings,
		config.DNCSettings,
		config.HostAgentSettings,
		config.VlanSettings.Min,
		config.VlanSettings.Max,
		config.GreSettings.Min,
		config.GreSettings.Min,
		c.httpClient,
		c.cnsHttpClient,
		config.HostAgentSettings.WireServerIP)
	if err != nil {
		return api.InternalServerError(api.StatusInternalServerError, err.Error())
	}

	c.UpdateReplicaStatus(true)

	// TODO: set the orchestrator for AI telemetry
	// setOrchestratorForTelemetry()

	return nil
}

// InitializeAzureStorageV2 - Tries to establish a connection with azure storage with information provided in request.
func (c *Controller) InitializeAzureStorageV2(request api.AzureStorageRequestV2) *api.JsonErr {
	loggershim.LS.Debugf("[Request] %s", request.String())
	config.DataStoreSettings.AzureStorageInfo.AccountName = request.AccountName
	config.DataStoreSettings.AzureStorageInfo.Key = request.Key
	config.DataStoreSettings.AzureStorageInfo.PartitionKey = request.PartitionKey
	config.DataStoreSettings.AzureStorageInfo.RegionalStorageConnectionString = request.RegionalStorageConnectionString
	masterElectionShim.SetDataStoreSettings(config)
	masterElectionShim.Init()

	err := datamodel.Init(
		config.CNSSettings,
		config.DataStoreSettings,
		config.DNCSettings,
		config.HostAgentSettings,
		config.VlanSettings.Min,
		config.VlanSettings.Max,
		config.GreSettings.Min,
		config.GreSettings.Min,
		c.httpClient,
		c.cnsHttpClient,
		config.HostAgentSettings.WireServerIP)
	if err != nil {
		return api.InternalServerError(api.StatusInternalServerError, err.Error())
	}

	c.UpdateReplicaStatus(true)

	// TODO: set the orchestrator for AI telemetry
	// setOrchestratorForTelemetry()

	return nil
}

// InitializeAzureStorageV3 - Tries to establish a connection with azure storage with information provided in request.
func (c *Controller) InitializeAzureStorageV3(request api.AzureStorageRequest, blobName string, containerName string) *api.JsonErr {
	loggershim.LS.Debugf("[Request] %s", request.String())
	config.DataStoreSettings.AzureStorageInfo.AccountName = request.AccountName
	config.DataStoreSettings.AzureStorageInfo.Key = request.Key
	config.DataStoreSettings.AzureStorageInfo.PartitionKey = request.PartitionKey
	config.MasterElectionSettings.BlobStorageSettings.BlobName = blobName
	config.MasterElectionSettings.BlobStorageSettings.ContainerName = containerName
	masterElectionShim.SetDataStoreSettings(config)
	masterElectionShim.Init()

	err := datamodel.Init(
		config.CNSSettings,
		config.DataStoreSettings,
		config.DNCSettings,
		config.HostAgentSettings,
		config.VlanSettings.Min,
		config.VlanSettings.Max,
		config.GreSettings.Min,
		config.GreSettings.Min,
		c.httpClient,
		c.cnsHttpClient,
		config.HostAgentSettings.WireServerIP)
	if err != nil {
		return api.InternalServerError(api.StatusInternalServerError, err.Error())
	}

	c.UpdateReplicaStatus(true)

	// TODO: set the orchestrator for AI telemetry
	// setOrchestratorForTelemetry()

	return nil
}

// IsReplicaReady - Function to check if this replica is ready
func (c *Controller) IsReplicaReady() bool {
	return c.Replica.IsReplicaReady()
}

// UpdateReplicaStatus - Function to update replica ready flag
func (c *Controller) UpdateReplicaStatus(isReplicaReady bool) {
	c.Replica.UpdateReadyStatus(isReplicaReady)
}

// RelinquishPrimaryStatus - Relinquish replica primary status
func (c *Controller) RelinquishPrimaryStatus() {
	loggershim.LS.Debugf("[Request] Relinquisihing primary replica status")
	masterElectionShim.RelinquishPrimaryStatus()
}

// GetRefreshMonitorStatus - Function to get replica monitor status
func (c *Controller) GetRefreshMonitorStatus() api.RefreshMonitorStatus {
	return c.Replica.GetRefreshMonitorStatus()
}

// RefreshNMAgentGoalState - Retrieves the current primary replica endpoint
func (c *Controller) RefreshNMAgentGoalState() {
	go c.RefreshNetworkContainersNMAgentGoalState()
}

// DumpStorageState - dumps DNC storage state
func (c *Controller) DumpStorageState() error {
	return datamodel.DumpStorageState(config.DataStoreSettings.AzureStorageInfo.PartitionKey)
}

// DeleteReservationSet - deletes reservation set
func (c *Controller) DeleteReservationSet(
	networkID string,
	reservationSetID string) *api.JsonErr {
	loggershim.LS.Debugf("[Request] Delete ReservationSetID: %s in vnet: %s", reservationSetID, networkID)
	sessionInfo := map[string]string{datamodel.ReservationSetRepositoryName: reservationSetID}

	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	reservations, err := datamodel.IpReservationRepo.TryQueryAllReservationsInSet(networkID, reservationSetID)
	if err != nil {
		loggershim.LS.Errorf("could not retrieve reservations in set: %s error: %v", reservationSetID, err)
		return err
	}

	// Check if any reservations are currently in use. If in use, return error
	var reservationsInUse []string
	for _, reservation := range reservations {
		if reservation.Configuration.AssociatedNCID != "" {
			reservationsInUse = append(reservationsInUse, reservation.Configuration.ReservationID)
		}
	}

	if len(reservationsInUse) > 0 {
		return api.BadRequestf(api.ReservationInUse,
			"deletion of the set is not allowed as reservations are in use: [%s]", strings.Join(reservationsInUse, ","))
	}

	// Mark the reservation set for deletion. This will prevent any further processing / new NC creations
	// in case the DNC fails in reservation deletion or restarts / crashes before completing this function and deleting the reservation set
	reservationSet, err := datamodel.IpReservationSetRepo.TryQueryIPReservationSet(networkID, reservationSetID)
	if err != nil {
		loggershim.LS.Errorf("could not retrieve reservation set: %s", reservationSetID)
		return err
	}

	if reservationSet == nil {
		errMsg := fmt.Sprintf("could not find reservation set: %s for deletion", reservationSetID)
		loggershim.LS.Errorf(errMsg)
		return api.BadRequestf(api.InvalidReservationSetID, errMsg)
	}

	reservationSet.MarkedForDelete = true
	if err = datamodel.IpReservationSetRepo.PersistReservationSet(reservationSet); err != nil {
		loggershim.LS.Errorf("could not persist reservation set: %+v error: %v", reservationSet, err)
		return err
	}

	// Delete all reservations
	for _, reservation := range reservations {
		reservationID := datamodel.GetReservationIDFromKey(reservation.RowKey)
		if err := c.deleteReservation(networkID, reservationID); err != nil {
			loggershim.LS.Errorf("could not delete reservation: %s in set: %s", reservationID, reservationSetID)
			return err
		}
	}

	if err := datamodel.IpReservationSetRepo.DeleteReservationSet(networkID, reservationSetID); err != nil {
		loggershim.LS.Errorf("could not delete reservation set: %s error: %v", reservationSetID, err)
		return err
	}

	loggershim.LS.Debugf("successfully deleted reservation set: %s", reservationSetID)

	return nil
}

func registerSubnetViaRNM(networkID, subnetNameorID string, isRegistered bool) {
	// Subscribe after persisting the Subnet Configuration to make sure once pubsubproxy receives vnet data via
	// notification, it will update the subnet configuration persisted above and updates the status to complete
	if !isRegistered {
		loggershim.LS.Debugf("PutSubnetV2 Registering vnet with pubsubproxy via RNM: Network: %s, Subnet: %s", networkID, subnetNameorID)
		pubsubproxy.RegisterVnet(networkID)
	}
}

func registerSubnetViaRNC(networkID, subnetNameorID string, isRegistered bool) {
	loggershim.LS.Debugf("PutSubnetV2 Registering subnet with pubsubproxy via RNC: Network: %s, Subnet: %s", networkID, subnetNameorID)
	pubsubproxy.RegisterSubnet(networkID, subnetNameorID)
}

func deleteSubnetViaRNM(networkID, subnetName string, networkConfiguration *datamodel.NetworkConfiguration) *api.JsonErr {
	loggershim.LS.Debugf("deleteSubnetViaRNM: Deleting subnet %s in network %s", subnetName, networkID)
	subnetConfigs, err := datamodel.TryQueryAllSubnetConfigsInNetworkConfig(networkConfiguration)
	if err != nil {
		loggershim.LS.Errorf("Failed to get all the subnet configs in network: %s. Error: %v", networkID, err)
		return err
	}

	// Delete the network if there are no other 'Joined' subnets left in this network
	for subnet, subnetConfig := range subnetConfigs {
		// If there is at least one 'Joined' subnet left in this network, there is no further processing needed
		if subnetConfig.Joined && subnet != subnetName {
			loggershim.LS.Debugf("Cannot delete the network: %s as subnet: %s is in 'Joined' state", networkID, subnet)
			return nil
		}
	}

	loggershim.LS.Debugf("deleteSubnetViaRNM: Deleting network: %s", networkID)
	// Delete the network configuration and subnet configurations in the network
	if err = datamodel.DeleteNetworkConfiguration(networkID); err == nil {
		// Unsubscribe vnet notifications for PubSubProxy
		pubsubproxy.UnregisterVnet(networkID)
	}

	return err
}

func deleteSubnetViaRNC(networkID, subnetId string, networkConfiguration *datamodel.NetworkConfiguration) *api.JsonErr {
	// Delete the subnet configurations
	loggershim.LS.Debugf("deleteSubnetViaRNC: Deleting subnet %s in network %s", subnetId, networkID)
	if err := datamodel.DeleteSubnetConfiguration(networkID, subnetId); err == nil {
		// Unsubscribe subnet notifications for PubSubProxy
		pubsubproxy.UnregisterSubnet(networkID, subnetId)
	}

	subnetConfigs, err := datamodel.TryQueryAllSubnetConfigsInNetworkConfig(networkConfiguration)
	if err != nil {
		loggershim.LS.Errorf("Failed to get all the subnet configs in network: %s. Error: %v", networkID, err)
		return err
	}

	// Delete the network if there are no other 'Joined' subnets left in this network
	joinedSubnetFound := false
	for subnet, subnetConfig := range subnetConfigs {
		// If there is at least one 'Joined' subnet left in this network, there is no further processing needed
		if subnetConfig.Joined && subnet != subnetId {
			joinedSubnetFound = true
			loggershim.LS.Debugf("deleteSubnetViaRNC: Not deleting network: %s as subnet: %s is in 'Joined' state", networkID, subnet)
		}
	}

	if !joinedSubnetFound {
		loggershim.LS.Debugf("deleteSubnetViaRNC: Deleting network: %s", networkID)
		return datamodel.DeleteNetworkConfiguration(networkID)
	}

	return nil
}

func ValidateSubnetRequest(subnetNameOrId string, apiVersion string) *api.JsonErr {
	if config.PubSubProxySettings.EnableRNC {

		if apiVersion != "2021-02-24" {
			return api.BadRequestf(api.RequestNotsupported, "Subnet request via RNC is only supported with API version 2021-02-24")
		}

		if !utils.IsValidUUID(subnetNameOrId) {
			return api.BadRequestf(api.InvalidSubnetName, "SubnetId %s must be guid if RNC path is enabled", subnetNameOrId)
		}
	}

	return nil
}

func getNetworkStatusViaRNM() datamodel.Status {
	return datamodel.Pending
}

func getNetworkStatusViaRNC() datamodel.Status {
	return datamodel.Completed
}
