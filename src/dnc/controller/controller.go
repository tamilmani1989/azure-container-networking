package controller

import (
	"dnc/api"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/replica"
	"dnc/utils"
	"fmt"
	"net/http"
)

// Controller - Controller for handling API calls
type Controller struct {
	Replica        *replica.Replica
	SessionManager *utils.SessionManager
}

const (
	nodeRepositoryName    = "NodeRepository"
	networkRepositoryName = "NetworkRepository"
)

// NewController - Create a NewController
func NewController(replica *replica.Replica) *Controller {
	controller := new(Controller)
	controller.Replica = replica
	controller.SessionManager = utils.NewSessionManager([]string{nodeRepositoryName, networkRepositoryName})
	return controller
}

// PersistNodeInfo - Persist node configuration.
func (c *Controller) PersistNodeInfo(nodeID string, nodeInfo *api.NodeInfoRequest, apiVersion string) *api.JsonErr {
	sessionInfo := map[string]string{nodeRepositoryName: nodeID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	return datamodel.CreateOrUpdateNodeConfiguration(nodeID, *nodeInfo, apiVersion)
}

// DeleteNode - Delete node configuration
func (c *Controller) DeleteNode(nodeID string) *api.JsonErr {
	sessionInfo := map[string]string{nodeRepositoryName: nodeID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	return datamodel.DeleteNode(nodeID)
}

// GetNodeInfo - Get node configuration.
func (c *Controller) GetNodeInfo(nodeID string) (*api.NodeInfoResponse, *api.JsonErr) {
	sessionInfo := map[string]string{nodeRepositoryName: nodeID}
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
	sessionInfo := map[string]string{networkRepositoryName: networkID}
	if len(allocationRequest.NodeConstraint) > 0 {
		sessionInfo[nodeRepositoryName] = allocationRequest.NodeConstraint
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
	sessionInfo := map[string]string{networkRepositoryName: networkID}
	c.SessionManager.BeginReadSession(sessionInfo)
	defer c.SessionManager.EndReadSession(sessionInfo)

	return datamodel.GetReservationResponse(networkID, reservationID)
}

// GetReservationStatus - Get reservation status
func (c *Controller) GetReservationStatus(
	networkID,
	reservationID string) (*api.RequestStatus, *api.JsonErr) {
	sessionInfo := map[string]string{networkRepositoryName: networkID}
	c.SessionManager.BeginReadSession(sessionInfo)
	defer c.SessionManager.EndReadSession(sessionInfo)
	return datamodel.GetReservationStatus(networkID, reservationID)
}

// DeleteReservation - Delete reservation configuration.
func (c *Controller) DeleteReservation(
	networkID string,
	reservationID string) *api.JsonErr {
	sessionInfo := map[string]string{networkRepositoryName: networkID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)
	return datamodel.DeleteIPReservation(networkID, reservationID, false)
}

func generateStatus(status datamodel.Status, errorInfo *api.JsonErr) int {
	switch status {
	case datamodel.Pending:
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
	subnetConfiguration, err := datamodel.QuerySubnetConfiguration(
		networkID,
		subnetName)
	if err != nil {
		return nil, err
	}

	ipAllocationsData := make(api.IPAllocationsData, len(subnetConfiguration.Reservations))
	index := 0
	for reservationID := range subnetConfiguration.Reservations {
		reservationConfiguration, err := datamodel.QueryIPReservation(networkID, reservationID)
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
		if reservationConfiguration == nil {
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
			IPAddress:                    reservationConfiguration.AllocatedSubnet,
			AssociatedNetworkContainerID: reservationConfiguration.AssociatedNCID,
			ReservationID:                reservationConfiguration.ReservationID}
		index++

	}
	subnetConfigurationResponse := api.SubnetConfigurationResponse{
		NetworkID:         networkID,
		SubnetName:        subnetName,
		IPSubnet:          subnetConfiguration.IPSubnet,
		GatewayIP:         subnetConfiguration.Gateway,
		DNSServers:        subnetConfiguration.DNSServers,
		IPAllocationsData: ipAllocationsData,
		AccessToken:       subnetConfiguration.AccessToken}

	return &subnetConfigurationResponse, nil
}

// GetNetworkContainerStatus - Get network container status
func (c *Controller) GetNetworkContainerStatus(
	networkID, networkContainerID string) (*api.RequestStatus, int, *api.JsonErr) {
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

func (c *Controller) PersistNetworkInfo(
	networkID string,
	networkInfo api.NetworkInfoRequest,
	apiVersion string) *api.JsonErr {
	return datamodel.CreateNetwork(
		networkID,
		networkInfo,
		apiVersion)
}

func (c *Controller) PersistSubnetInfo(
	networkID,
	subnet string,
	subnetInfo api.SubnetRequest) *api.JsonErr {
	return datamodel.UpdateSubnet(networkID, subnet, subnetInfo)
}
func (c *Controller) PersistNetworkContainerRequestV1(
	networkID, ncID string,
	ncRequestV1 *api.NetworkContainerRequestV1,
	apiVersion string) *api.JsonErr {
	return datamodel.CreateOrUpdateNCRequestV1(networkID, ncID, ncRequestV1, apiVersion)
}

func (c *Controller) DeleteNetwork(networkID string) *api.JsonErr {
	return datamodel.DeleteNetworkConfiguration(networkID)
}

func (c *Controller) PersistNetworkContainerRequestV2(
	networkID, ncID string,
	ncRequestV2 *api.NetworkContainerRequestV2,
	apiVersion string) *api.JsonErr {
	loggershim.LS.Log("V2-Request: %s\n", ncRequestV2.String())
	return datamodel.CreateOrUpdateNCRequestV2(networkID, ncID, ncRequestV2, apiVersion)
}

func (c *Controller) GetNetworkContainerConfiguration(networkID, ncID string) (*api.NetworkContainerResponse, *api.JsonErr) {
	ncConfig, err := datamodel.GetNetworkContainerConfiguration(networkID, ncID)
	if err != nil {
		return nil, err
	}

	return datamodel.GetNetworkContainerResponse(ncConfig)

}
func (c *Controller) DeleteNetworkContainer(networkID, ncID string) *api.JsonErr {
	return datamodel.ProcessNetworkContainerForRelease(networkID, ncID)
}

// func GetReservationsByNetworkID(networkID string) []string {
// 	filter := fmt.Sprintf("NetworkID eq '%s'", networkID)

// 	entitiesWithContext, _ := BlobStorage.GetEntities(IPReservationsTableName, filter)
// 	reservations := make([]string, len(entitiesWithContext))
// 	index := 0
// 	for _, entityWithContext := range entitiesWithContext {
// 		jsonSerializedBytes, err := json.Marshal(entityWithContext.Properties)
// 		if err != nil {
// 			loggershim.LS.Log("Failed to serialize entity properties",
// 				entityWithContext.Properties, err)
// 			continue
// 		}

// 		var reservationConfiguration ReservationConfiguration
// 		err = json.Unmarshal(jsonSerializedBytes, &reservationConfiguration)
// 		reservations[index] = reservationConfiguration.ReservationID
// 		index++
// 	}

// 	return reservations

// }

func (c *Controller) GetPrimaryIPEndpoint() (string, error) {
	return c.Replica.GetPrimaryIPEndpoint()
}

func (c *Controller) GetReplicaIPEndpoint() (string, error) {
	return c.Replica.GetReplicaIPEndpoint()
}
