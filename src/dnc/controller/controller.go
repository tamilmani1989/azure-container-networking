package controller

import (
	"bytes"
	"dnc/api"
	"dnc/configuration"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/masterelection/masterelectionshim"
	"dnc/replica"
	"dnc/utils"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Azure/azure-container-networking/cns"
)

var (
	setOrchestratorTypeURLFmt string
	masterElectionShim        *masterelectionshim.MasterElectionShim
	config                    *configuration.Configuration
)

// Controller - Controller for handling API calls
type Controller struct {
	Replica        *replica.Replica
	SessionManager *utils.SessionManager
}

// NewController - Create a NewController
func NewController(replica *replica.Replica, sessionsManager *utils.SessionManager, setOrchestratorTypeUrlFmt string,
	electionShim *masterelectionshim.MasterElectionShim, configuration *configuration.Configuration) *Controller {
	controller := new(Controller)
	controller.Replica = replica
	controller.SessionManager = sessionsManager
	setOrchestratorTypeURLFmt = setOrchestratorTypeUrlFmt
	masterElectionShim = electionShim
	config = configuration
	return controller
}

// PersistNodeInfo - Persist node configuration.
func (c *Controller) PersistNodeInfo(nodeID string, nodeInfo *api.NodeInfoRequest, apiVersion string) *api.JsonErr {
	sessionInfo := map[string]string{datamodel.NodeRepositoryName: nodeID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	// Send OrchestratorType to CNS
	info := &cns.SetOrchestratorTypeRequest{OrchestratorType: nodeInfo.OrchestratorType}
	var body bytes.Buffer
	var resp cns.Response

	json.NewEncoder(&body).Encode(info)
	setOrchestratorTypeUrl := fmt.Sprintf(
		setOrchestratorTypeURLFmt,
		nodeInfo.IPAddresses[0])
	w, respErr := http.Post(setOrchestratorTypeUrl, "application/json; charset=utf-8", &body)
	if respErr != nil {
		msg := fmt.Sprintf("Post to URL %v for node %s failed with error %s",
			setOrchestratorTypeUrl,
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
		msg := fmt.Sprintf("Create node %s failed with error code %d",
			nodeID,
			resp.ReturnCode)
		return &api.JsonErr{
			Code:       api.NodeProcessingFailedDueToInternalError,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
	}

	return datamodel.CreateOrUpdateNodeConfiguration(nodeID, *nodeInfo, apiVersion)
}

// DeleteNode - Delete node configuration
func (c *Controller) DeleteNode(nodeID string) *api.JsonErr {
	sessionInfo := map[string]string{datamodel.NodeRepositoryName: nodeID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	return datamodel.DeleteNode(nodeID)
}

// GetNodeInfo - Get node configuration.
func (c *Controller) GetNodeInfo(nodeID string) (*api.NodeInfoResponse, *api.JsonErr) {
	sessionInfo := map[string]string{datamodel.NodeRepositoryName: nodeID}
	c.SessionManager.BeginReadSession(sessionInfo)
	defer c.SessionManager.EndReadSession(sessionInfo)
	return datamodel.GetNodeInfo(nodeID)
}

// PersistIPReservationRequest - Persist reservation request
func (c *Controller) PersistIPReservationRequest(
	networkID string,
	reservationID string,
	allocationRequest api.AllocationRequest,
	apiVersion string) *api.JsonErr {
	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}

	if len(allocationRequest.NodeConstraint) > 0 {
		sessionInfo[datamodel.NodeRepositoryName] = allocationRequest.NodeConstraint
	}

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
	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	c.SessionManager.BeginReadSession(sessionInfo)
	defer c.SessionManager.EndReadSession(sessionInfo)

	return datamodel.GetReservationResponse(networkID, reservationID)
}

// GetReservationStatus - Get reservation status
func (c *Controller) GetReservationStatus(
	networkID,
	reservationID string) (*api.RequestStatus, *api.JsonErr) {
	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	c.SessionManager.BeginReadSession(sessionInfo)
	defer c.SessionManager.EndReadSession(sessionInfo)
	return datamodel.GetReservationStatus(networkID, reservationID)
}

// DeleteReservation - Delete reservation configuration.
func (c *Controller) DeleteReservation(
	networkID string,
	reservationID string) *api.JsonErr {
	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)
	return datamodel.DeleteIPReservation(networkID, reservationID, false)
}

func generateStatus(status datamodel.Status, errorInfo *api.JsonErr) int {
	switch status {
	case datamodel.Pending:
		fallthrough
	case datamodel.NotStarted:
		return http.StatusProcessing
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
	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	c.SessionManager.BeginReadSession(sessionInfo)
	defer c.SessionManager.EndReadSession(sessionInfo)

	vnetCfg, err := datamodel.QueryNetworkConfiguration(networkID)
	if err != nil {
		return nil, 0, err
	}

	generatedStatus := generateStatus(vnetCfg.Status, &vnetCfg.FailureDetail)
	var requestStatus *api.RequestStatus
	if generatedStatus != http.StatusProcessing {
		requestStatus = &api.RequestStatus{
			State:         vnetCfg.NetworkState.String(),
			Status:        vnetCfg.Status.String(),
			FailureDetail: vnetCfg.FailureDetail}
	}

	return requestStatus, generatedStatus, nil
}

// GetSubnetStatus - Get subnet status
func (c *Controller) GetSubnetStatus(networkID, subnetName string) (*api.RequestStatus, int, *api.JsonErr) {
	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	c.SessionManager.BeginReadSession(sessionInfo)
	defer c.SessionManager.EndReadSession(sessionInfo)

	_, err := datamodel.QuerySubnetConfiguration(
		networkID,
		subnetName)
	if err != nil {
		return nil, 0, err
	}

	requestStatus := api.RequestStatus{
		State:  datamodel.Join.String(),
		Status: datamodel.Completed.String()}

	return &requestStatus, http.StatusOK, nil
}

// GetSubnetConfiguration - Get subnet configuration
func (c *Controller) GetSubnetConfiguration(networkID, subnetName string) (*api.SubnetConfigurationResponse, *api.JsonErr) {
	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	c.SessionManager.BeginReadSession(sessionInfo)
	defer c.SessionManager.EndReadSession(sessionInfo)

	repoSubnetConfiguration, err := datamodel.QuerySubnetConfiguration(
		networkID,
		subnetName)
	if err != nil {
		return nil, err
	}

	ipAllocationsData := make(api.IPAllocationsData, len(repoSubnetConfiguration.Reservations))
	index := 0
	for reservationID := range repoSubnetConfiguration.Reservations {
		repoReservationConfiguration, err := datamodel.QueryIPReservation(networkID, reservationID)
		if err != nil {
			msg := fmt.Sprintf(
				"Failed to retrieve reservation configuration for reservation:[%s] in network [%s] due to error:{%s}",
				reservationID,
				networkID,
				err)
			return nil, &api.JsonErr{
				Code:       api.StatusInternalServerError,
				Text:       msg,
				HTTPStatus: http.StatusInternalServerError}
		}
		if repoReservationConfiguration == nil {
			msg := fmt.Sprintf(
				"Configuration for reservation {%s} associated with subnet:%s in network %s not found due to error:{%+v}",
				reservationID,
				subnetName,
				networkID,
				err)
			return nil, &api.JsonErr{
				Code:       api.StatusInternalServerError,
				Text:       msg,
				HTTPStatus: http.StatusInternalServerError}
		}
		ipAllocationsData[index] = api.IPAllocationData{
			IPAddress:                    repoReservationConfiguration.AllocatedSubnet,
			AssociatedNetworkContainerID: repoReservationConfiguration.AssociatedNCID,
			ReservationID:                repoReservationConfiguration.ReservationID}
		index++

	}

	dnsServers, err := datamodel.GetSubnetDNSServers(repoSubnetConfiguration)
	subnetConfigurationResponse := api.SubnetConfigurationResponse{
		NetworkID:         networkID,
		SubnetName:        subnetName,
		IPSubnet:          repoSubnetConfiguration.IPSubnet,
		GatewayIP:         repoSubnetConfiguration.Gateway,
		DNSServers:        dnsServers,
		IPAllocationsData: ipAllocationsData,
		AccessToken:       repoSubnetConfiguration.AccessToken}

	return &subnetConfigurationResponse, nil
}

// GetNetworkContainerStatus - Get network container status
func (c *Controller) GetNetworkContainerStatus(
	networkID, networkContainerID string) (*api.RequestStatus, int, *api.JsonErr) {
	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	c.SessionManager.BeginReadSession(sessionInfo)
	defer c.SessionManager.EndReadSession(sessionInfo)

	ncConfig, err := datamodel.GetNetworkContainerConfiguration(networkID, networkContainerID)
	if err != nil {
		return nil, 0, err
	}

	if ncConfig == nil {
		msg := fmt.Sprintf(
			"NetworkContainer %s in network %s is not found", networkContainerID, networkID)
		return nil, 0, &api.JsonErr{
			Code:       api.SubnetNotFound,
			HTTPStatus: http.StatusNotFound,
			Text:       msg}
	}
	generatedStatus := generateStatus(ncConfig.Status, &ncConfig.FailureDetail)
	var requestStatus *api.RequestStatus
	if generatedStatus != http.StatusProcessing {
		requestStatus = &api.RequestStatus{
			State:  ncConfig.State.String(),
			Status: ncConfig.Status.String()}
	}

	return requestStatus, generatedStatus, nil

}

// PersistNetworkJoinRequest - Persists network join request
func (c *Controller) PersistNetworkJoinRequest(
	networkID string,
	networkInfo api.NetworkInfoRequest,
	apiVersion string) *api.JsonErr {
	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	return datamodel.CreateNetwork(
		networkID,
		networkInfo,
		apiVersion)
}

// PersistSubnetDelegationRequest - Persist subnet delgation info.
func (c *Controller) PersistSubnetDelegationRequest(
	networkID,
	subnet string,
	subnetInfo api.SubnetRequest) *api.JsonErr {
	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	return datamodel.UpdateSubnet(networkID, subnet, subnetInfo)
}

// DeleteNetwork - Delete a network
func (c *Controller) DeleteNetwork(networkID string) *api.JsonErr {
	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	return datamodel.DeleteNetworkConfiguration(networkID)
}

// PersistNetworkContainerRequestV1 - Persist network container request using dynamic allocation
func (c *Controller) PersistNetworkContainerRequestV1(
	networkID, ncID string,
	ncRequestV1 *api.NetworkContainerRequestV1,
	apiVersion string) *api.JsonErr {
	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	return datamodel.CreateOrUpdateNCRequestV1(networkID, ncID, ncRequestV1, apiVersion)
}

// PersistNetworkContainerRequestV2 - Persist network container request using dynamic allocation
func (c *Controller) PersistNetworkContainerRequestV2(
	networkID, ncID string,
	ncRequestV2 *api.NetworkContainerRequestV2,
	apiVersion string) *api.JsonErr {
	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	loggershim.LS.Debugf("V2-Request: %s\n", ncRequestV2.String())
	return datamodel.CreateOrUpdateNCRequestV2(networkID, ncID, ncRequestV2, apiVersion)
}

// GetNetworkContainerConfiguration - Retrieves the network configuration
func (c *Controller) GetNetworkContainerConfiguration(networkID, ncID string) (*api.NetworkContainerResponse, *api.JsonErr) {
	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	ncConfig, err := datamodel.GetNetworkContainerConfiguration(networkID, ncID)
	if err != nil {
		return nil, err
	}

	return datamodel.GetNetworkContainerResponse(ncConfig)
}

// DeleteNetworkContainer - Deletes a network container.
func (c *Controller) DeleteNetworkContainer(networkID, ncID string) *api.JsonErr {
	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	return datamodel.ProcessNetworkContainerForRelease(networkID, ncID)
}

// GetPrimaryIPEndpoint - Retrieves the current primary replica endpoint
func (c *Controller) GetPrimaryIPEndpoint() (string, error) {
	return c.Replica.GetPrimaryIPEndpoint()
}

// GetReplicaIPEndpoint - Retrieves the endpoint of this replica.
func (c *Controller) GetReplicaIPEndpoint() (string, error) {
	return c.Replica.GetReplicaIPEndpoint()
}

// InitializeAzureStorage - Tries to establish a connection with azure storage with information provided in request.
func (c *Controller) InitializeAzureStorage(request api.AzureStorageRequest) *api.JsonErr {
	config.DataStoreSettings.AzureStorageInfo.AccountName = request.AccountName
	config.DataStoreSettings.AzureStorageInfo.Key = request.Key
	config.DataStoreSettings.AzureStorageInfo.PartitionKey = request.PartitionKey
	masterElectionShim.SetDataStoreSettings(config)
	masterElectionShim.Init()
	datamodel.Init(config.DataStoreSettings, config.DNCSettings.RequestRetryBaseInterval, config.VlanSettings.Min, config.VlanSettings.Max)
	c.UpdateReplicaStatus(true)

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
