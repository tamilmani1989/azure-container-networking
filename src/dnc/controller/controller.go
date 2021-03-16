package controller

import (
	"bytes"
	"crypto/sha256"
	"dnc/aiwrapper"
	"dnc/api"
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

	"github.com/Azure/azure-container-networking/cns"
)

// todo: no need for these to be globals, should be part of struct
var (
	setOrchestratorTypeURLFmt string
	masterElectionShim        *masterelectionshim.MasterElectionShim
	config                    *configuration.Configuration
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
	return controller
}

// GetConfig :- retrieves the configuration passed to controller
func (c *Controller) GetConfig() *configuration.Configuration {
	return config
}

// PersistNodeInfo - Persist node configuration.
func (c *Controller) PersistNodeInfo(nodeID string, nodeInfo *api.NodeInfoRequest, apiVersion string) *api.JsonErr {
	// Can be used whenver we send logs/metric
	aiwrapper.OrchestratorType = nodeInfo.OrchestratorType

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
		loggershim.LS.Debugf("[CNS] create/update %s", nodeInfo.String())

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
	return datamodel.CreateOrUpdateNodeConfiguration(false, 0, nodeID, *nodeInfo, apiVersion, []string{})
}

// RegisterNode - Register node
func (c *Controller) RegisterNode(nodeID, apiVersion string, request *api.NodeRegistrationRequest) *api.JsonErr {
	sessionInfo := map[string]string{datamodel.NodeRepositoryName: nodeID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	return datamodel.CreateOrUpdateNodeConfiguration(true, request.NumCores, nodeID, api.NodeInfoRequest{}, apiVersion, request.NmAgentSupportedApis)
}

// DeleteNode - Delete node configuration
func (c *Controller) DeleteNode(nodeID string) *api.JsonErr {
	loggershim.LS.Debugf("[Request] DeleteNode: %s", nodeID)
	sessionInfo := map[string]string{datamodel.NodeRepositoryName: nodeID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	return datamodel.DeleteNode(nodeID)
}

// GetNodeInfo - Get node info
func (c *Controller) GetNodeInfo(nodeID string) (*api.NodeInfoResponse, *api.JsonErr) {
	// No read lock is required because Azure Storage uses snapshot isolation to allow
	// read operations to happen concurrently with writes in the same partition. It
	// guarantees reads to see consistent data while updates are happening
	return datamodel.GetNodeInfo(nodeID)
}

// GetNodeConfiguration - Get node configuration
func (c *Controller) GetNodeConfiguration(nodeID string) (*datamodel.NodeConfiguration, *api.JsonErr) {
	return datamodel.TryQueryNodeConfiguration(nodeID)
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

	return datamodel.GetReservationResponse(networkID, reservationID)
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
	sessionInfo := map[string]string{
		datamodel.ReservationRepositoryName: reservationID,
		datamodel.NetworkRepositoryName:     networkID,
	}
	// Check if the node session lock is required
	reservation, err := datamodel.TryQueryIPReservation(networkID, reservationID)
	if reservation.Configuration != nil && err == nil {
		if len(reservation.Configuration.AllocationRequest.NodeConstraint) != 0 {
			sessionInfo[datamodel.NodeRepositoryName] = reservation.Configuration.AllocationRequest.NodeConstraint
		}
	}

	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	return c.ipAllocator.DeleteIPReservation(networkID, reservation, false)
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
	networkID, networkContainerID string) (*api.RequestStatus, int, *api.JsonErr) {
	// No read lock is required because Azure Storage uses snapshot isolation to allow
	// read operations to happen concurrently with writes in the same partition. It
	// guarantees reads to see consistent data while updates are happening
	ncConfig, err := datamodel.GetNetworkContainerConfiguration(networkID, networkContainerID)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	if ncConfig == nil {
		msg := fmt.Sprintf(
			"NetworkContainer %s in network %s is not found", networkContainerID, networkID)
		return nil, http.StatusNotFound, &api.JsonErr{
			Code:       api.NetworkContainerNotFound,
			HTTPStatus: http.StatusNotFound,
			Text:       msg}
	}

	requestStatus := &api.RequestStatus{
		State:         ncConfig.State.String(),
		Status:        ncConfig.Status.String(),
		FailureDetail: ncConfig.FailureDetail}

	return requestStatus, generateStatus(ncConfig.Status, &ncConfig.FailureDetail), nil
}

// PersistNetworkJoinRequest - Persists network join request
func (c *Controller) PersistNetworkJoinRequest(
	networkID string,
	networkInfo api.NetworkInfoRequest,
	apiVersion string) *api.JsonErr {
	loggershim.LS.Debugf("[Request] Join Vnet %s", networkID)

	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	return joinNetwork(networkID, apiVersion)
}

func joinNetwork(networkID string, apiVersion string) *api.JsonErr {
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

// PersistSubnetDelegationRequest - Persist subnet delegation info.
func (c *Controller) PersistSubnetDelegationRequest(
	networkID,
	subnet string,
	subnetInfo api.SubnetRequest,
	isDelete bool) *api.JsonErr {
	loggershim.LS.Debugf("[Request] Put Subnet: %s for network: %s isDelete: %t", subnet, networkID, isDelete)

	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	err := datamodel.UpdateSubnet(networkID, subnet, subnetInfo, isDelete)
	if err != nil {
		loggershim.LS.Errorf("UpdateSubnet: subnet: %s failed. Error: %v", datamodel.GetSubnetKey(networkID, subnet), err)
	}

	return err
}

func (c *Controller) DeleteSubnetV2(networkID, subnetName string) *api.JsonErr {
	loggershim.LS.Debugf("[Request] DeleteSubnetV2: Network: %s, Subnet: %s", networkID, subnetName)
	if !config.PubSubProxySettings.EnablePubSubProxy {
		// SubnetV2 API is supported with PubSubProxy only when DNC is running in managed mode
		return api.BadRequestf(api.RequestNotsupported, "Request not supported")
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
	loggershim.LS.Debugf("Marking 'Joined as false' for subnet: %s in network: %s", subnetName, networkID)
	if err := datamodel.PutSubnetConfiguration(requestedSubnetConfig); err != nil {
		loggershim.LS.Errorf("Failed to persist subnet configuration for subnet: %s in network: %s. Error: %v", subnetName, networkID, err)
		return err
	}

	subnetConfigs, err := datamodel.TryQueryAllSubnetConfigsInNetworkConfig(networkConfiguration)
	if err != nil {
		loggershim.LS.Errorf("Failed to get all the subnet configs in network: %s. Error: %v", networkID, err)
		return err
	}

	// Delete the network if there are no other 'Joined' subnets left in this network
	for subnet, subnetConfig := range subnetConfigs {
		// If there is at least one 'Joined' subnet left in this network, there is no further processing needed
		if subnetConfig.Joined == true && subnet != subnetName {
			loggershim.LS.Debugf("Cannot delete the network: %s as subnet: %s is in 'Joined' state", networkID, subnet)
			return nil
		}
	}

	loggershim.LS.Debugf("Deleting network: %s", networkID)
	// Delete the network configuration and subnet configurations in the network
	if err = datamodel.DeleteNetworkConfiguration(networkID); err == nil {
		// Unsubscribe vnet notifications for PubSubProxy
		pubsubproxy.UnregisterVnet(networkID)
	}

	return err
}

// PersistSubnetDelegationRequest - Persist subnet delegation info for V2 API.
func (c *Controller) PersistSubnetDelegationRequestV2(networkID, subnet string, subnetRequest api.SubnetRequest, apiVersion string) *api.JsonErr {
	loggershim.LS.Debugf("[Request] PutSubnetV2: Network: %s, Subnet: %s", networkID, subnet)
	if !config.PubSubProxySettings.EnablePubSubProxy {
		// SubnetV2 API is supported with PubSubProxy only when DNC is running in managed mode
		return api.BadRequestf(api.RequestNotsupported, "Request not supported")
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
			Status:       datamodel.Pending,
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
		loggershim.LS.Debugf("Updating the subnet config for Network: %s, Subnet: %s with access token and marking it as Joined", networkID, subnet)
		subnetConfig.AccessToken = subnetRequest.AuthenticationToken
		subnetConfig.Joined = true

		// todo: how to handle updates to cidr size?
		if config.FeatureFlagSettings.UsePrepopulatedIPAllocations {
			loggershim.LS.Debugf("Prepopulating vnet id %s subnet %s %s", subnetConfig.NetworkID, subnetConfig.SubnetName, subnetConfig.IPSubnet)
			if err := c.ipAllocator.PrepopulateSubnet(subnetConfig.NetworkID, subnetConfig.SubnetName, subnetConfig.IPSubnet); err != nil {
				loggershim.LS.Errorf("Could not prepopulate vnet id %s subnet %s %s: %+v", subnetConfig.NetworkID, subnetConfig.SubnetName, subnetConfig.IPSubnet, err)
				return err
			}
			loggershim.LS.Debugf("Successfully prepopulated vnet id %s subnet %s %s", subnetConfig.NetworkID, subnetConfig.SubnetName, subnetConfig.IPSubnet)
		}
	}

	// Persist the subnet configuration
	if err := datamodel.PutSubnetConfiguration(subnetConfig); err != nil {
		loggershim.LS.Errorf("Failed to persist subnet config for Network: %s, Subnet: %s", networkID, subnet)
		return err
	}

	// Subscribe after persisting the Subnet Configuration to make sure once pubsubproxy receives vnet data via
	// notification, it will update the subnet configuration persisted above and updates the status to complete
	if !isNetworkJoined {
		loggershim.LS.Debugf("PutSubnetV2 Registering vnet with pubsubproxy: Network: %s, Subnet: %s", networkID, subnet)
		pubsubproxy.RegisterVnet(networkID)
	}

	loggershim.LS.Debugf("PutSubnetV2 successfully persisted subnet config for Network: %s, Subnet: %s", networkID, subnet)
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
	loggershim.LS.Debugf("[Request] DeleteNetwork: %s", networkID)
	sessionInfo := map[string]string{datamodel.NetworkRepositoryName: networkID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	err := datamodel.DeleteNetworkConfiguration(networkID)
	if err == nil && config.PubSubProxySettings.EnablePubSubProxy {
		// Unsubscribe vnet notifications for PubSubProxy
		pubsubproxy.UnregisterVnet(networkID)
	}

	return err
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

	repoNCConfig, err := datamodel.TryQueryNetworkContainer(networkID, ncID)
	if err != nil {
		return err
	}

	return datamodel.PromoteContainerState(repoNCConfig)
}

// SkipNodeCleanupForNetworkContainer - Attempt to promote container stuck in an undesirable state (as a result of unreachable node)
func (c *Controller) SkipNodeCleanupForNetworkContainer(networkID, ncID string) *api.JsonErr {
	sessionInfo := map[string]string{datamodel.NCRepositoryName: ncID}
	c.SessionManager.BeginWriteSession(sessionInfo)
	defer c.SessionManager.EndWriteSession(sessionInfo)

	repoNCConfig, err := datamodel.TryQueryNetworkContainer(networkID, ncID)
	if err != nil {
		return err
	}

	if repoNCConfig.FailureDetail.Code == api.DisAssociationFailed {
		return datamodel.PromoteContainerState(repoNCConfig)
	}

	return nil
}

// GetNetworkContainerConfiguration - Retrieves the complete network configuration
func (c *Controller) GetNetworkContainerConfiguration(networkID, ncID string) (*datamodel.NCConfiguration, *api.JsonErr) {
	// No read lock is required because Azure Storage uses snapshot isolation to allow
	// read operations to happen concurrently with writes in the same partition. It
	// guarantees reads to see consistent data while updates are happening
	ncConfig, err := datamodel.GetNetworkContainerConfiguration(networkID, ncID)
	if err != nil {
		return nil, err
	}

	return ncConfig, nil
}

// GetNetworkContainerCleanupStatus returns the cleanup status of the nc.
func (c *Controller) GetNetworkContainerCleanupStatus(networkID, ncID string) (api.NetworkContainerCleanupResponse, *api.JsonErr) {
	cleanupResp := api.NetworkContainerCleanupResponse{Completed: false}
	// First, check the network container table to see if the NC exists
	_, err := datamodel.GetNetworkContainerConfiguration(networkID, ncID)
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
	ncConfig, err := datamodel.GetNetworkContainerConfiguration(networkID, ncID)
	if err != nil {
		return nil, err
	}

	ncGoalState, isRetriable, err := datamodel.GetNCGoalState(ncConfig)
	if err != nil {
		if isRetriable {
			err.HTTPStatus = http.StatusProcessing
		}
		return nil, err
	}

	ncRequest, isRetriable, err := datamodel.GetNetworkContainerRequest(ncGoalState, ncConfig)
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
	loggershim.LS.Debugf("[Request] DeleteNetworkContainer: %s", ncID)
	sessionInfo := map[string]string{datamodel.NCRepositoryName: ncID}
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