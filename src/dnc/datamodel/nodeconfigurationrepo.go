package datamodel

import (
	"dnc/api"
	"dnc/datastore/azurestorage"
	"dnc/datastoreshim"
	"dnc/logger/loggershim"
	"dnc/utils"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Azure/azure-container-networking/cns"
)

type NodeConfigurationRepo struct {
	Datastore datastoreshim.DataStoreShim
}

const (
	NodeSetColumnName = "NodeSet"
)

// CreateOrUpdateNodeConfiguration - Creates Or Updates a node configuration. Register should be true if called from RegisterNode.
func (n *NodeConfigurationRepo) CreateOrUpdateNodeConfiguration(
	register bool,
	cores int,
	nodeID string,
	nodeInfo api.NodeInfoRequest,
	apiVersion string,
	nmAgentSupportedApis []string) *api.JsonErr {
	repositoryNodeConfiguration, err := n.TryQueryNodeConfiguration(nodeID)
	if err != nil {
		return err
	}

	if repositoryNodeConfiguration == nil {
		repositoryNodeConfiguration = new(NodeConfiguration)
		repositoryNodeConfiguration.APIVersion = apiVersion
		repositoryNodeConfiguration.NodeID = nodeID
		repositoryNodeConfiguration.DeployedNetworkContainerMap = make(map[string]map[string]string)
		repositoryNodeConfiguration.NetworkVlanIDMap = make(map[string]uint16)
		repositoryNodeConfiguration.ConstrainedReservations = make(map[string]string)
		repositoryNodeConfiguration.UpdateGreCapable = GreCapableNotSet
		repositoryNodeConfiguration.InterfaceGreKeyMap = make(map[string]map[string]uint16)
	} else if repositoryNodeConfiguration.NodeID == nodeID && len(repositoryNodeConfiguration.NetworkVlanIDMap) > 0 {
		if len(nodeInfo.IPAddresses) > 0 && repositoryNodeConfiguration.NodeInfo.IPAddresses[0] != nodeInfo.IPAddresses[0] {
			return &api.JsonErr{
				Code:       api.NodeProcessingFailedDueToInternalError,
				Text:       fmt.Sprintf("Failed to update node IP as there are allocations."),
				HTTPStatus: http.StatusBadRequest}
		}
	}

	if register {
		repositoryNodeConfiguration.Registered = true
		repositoryNodeConfiguration.NcLimit = getNCLimitFromNumCores(cores)
	} else {
		repositoryNodeConfiguration.NodeInfo = nodeInfo
	}

	repositoryNodeConfiguration.UpdateGreCapable = updateGreKeyCapability(repositoryNodeConfiguration, nmAgentSupportedApis)
	return n.PersistNodeConfiguration(repositoryNodeConfiguration)
}

func (n *NodeConfigurationRepo) PersistNodeConfiguration(nodeconfiguration *NodeConfiguration) *api.JsonErr {
	jsonSerializedBytes, err := json.Marshal(nodeconfiguration)
	if err != nil {
		return &api.JsonErr{
			Code:       api.JsonMarshallingFailed,
			HTTPStatus: http.StatusInternalServerError,
			Text:       fmt.Sprintf("%s: Failed to serialize node configuration %+v due to error %s", utils.TracePrefix(2), nodeconfiguration, err.Error())}
	}

	nodeKey := GetNodeKey(nodeconfiguration.NodeID)

	if err := n.Datastore.InsertOrUpdateProperties(
		nodeTableName,
		dncPartitionKey,
		nodeKey,
		map[string]interface{}{
			TableToColumnMap[nodeTableName]: string(jsonSerializedBytes),
			NodeSetColumnName:               nodeconfiguration.NodeInfo.NodeSet,
		},
	); err != nil {
		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			HTTPStatus: http.StatusInternalServerError,
			Text:       fmt.Sprintf("%s: Failed to persist node configuration %+v due to error %s", utils.TracePrefix(2), nodeconfiguration, err.Error())}
	}

	return nil
}

// DeleteNode - Delete a node configuration.
func (n *NodeConfigurationRepo) DeleteNode(nodeID string) *api.JsonErr {
	repositoryNodeConfiguration, err := n.TryQueryNodeConfiguration(nodeID)
	if err != nil {
		return nil
	}

	if repositoryNodeConfiguration == nil {
		return nil
	}

	if repositoryNodeConfiguration.DeployedNetworkContainerMap != nil &&
		len(repositoryNodeConfiguration.DeployedNetworkContainerMap) != 0 {
		deployedNCString := repositoryNodeConfiguration.DumpDeployedNetworkContainers()
		msg := fmt.Sprintf("Node %s has network containers deployed. Deployed containers: [%s]", nodeID, deployedNCString)
		loggershim.LS.Errorf("%s: %s", utils.TracePrefix(2), msg)

		return &api.JsonErr{
			Code:       api.NodeInUse,
			HTTPStatus: http.StatusBadRequest,
			Text:       msg}
	}

	if repositoryNodeConfiguration.ConstrainedReservations != nil &&
		len(repositoryNodeConfiguration.ConstrainedReservations) != 0 {
		constrainedReservationsString := utils.PrintMapKeys(
			repositoryNodeConfiguration.ConstrainedReservations)
		msg := fmt.Sprintf(
			"Node %s has associated reservations. Reservations: [%s]",
			nodeID,
			constrainedReservationsString)
		loggershim.LS.Errorf("%s: %s", utils.TracePrefix(2), msg)
		return &api.JsonErr{
			Code:       api.NodeInUse,
			HTTPStatus: http.StatusBadRequest,
			Text:       msg}
	}

	nodeKey := GetNodeKey(nodeID)
	deleteErr := n.Datastore.Delete(nodeTableName, nodeKey)
	if deleteErr != nil {
		msg := fmt.Sprintf(
			"Failed to delete node configuration for node %s from data store. Error: %s",
			nodeID,
			deleteErr.Error())
		loggershim.LS.Errorf("%s: %s", utils.TracePrefix(2), msg)
		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			HTTPStatus: http.StatusInternalServerError,
			Text:       msg}
	}

	return nil
}

func (n *NodeConfigurationRepo) TryQueryNodeConfiguration(nodeID string) (*NodeConfiguration, *api.JsonErr) {
	nodeKey := GetNodeKey(nodeID)
	return n.TryQueryNodeConfigurationByKey(nodeKey)
}

// TryQueryNodeConfigurationByKey - Query node configuration by node key.
func (n *NodeConfigurationRepo) TryQueryNodeConfigurationByKey(nodeKey string) (*NodeConfiguration, *api.JsonErr) {
	var repositoryNodeConfiguration NodeConfiguration
	nodeID := GetNodeIDFromKey(nodeKey)
	entity, status, err := n.Datastore.Query(nodeTableName, nodeKey)
	if err != nil && status != http.StatusNotFound {
		msg := fmt.Sprintf(
			"Failed to retrieve configuration for node:[%s] due to error:[%s]",
			nodeID,
			err)
		return nil, &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	if entity == nil {
		return nil, nil
	}

	serializedNodeConfiguration := entity.Properties[nodeConfigurationTag].(string)
	err = json.Unmarshal([]byte(serializedNodeConfiguration), &repositoryNodeConfiguration)
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to unmarshal configuration for node:[%s] due to error:[%s]",
			nodeID,
			err)
		return nil, &api.JsonErr{
			Code:       api.JsonBodyUnMarshalFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	return &repositoryNodeConfiguration, nil
}

func (n *NodeConfigurationRepo) TryQueryAllNodeConfigurations(nodeSet string) ([]NodeConfiguration, *api.JsonErr) {
	var filter string
	if nodeSet != "" {
		filter = azurestorage.BuildFilterOnProperty(dncPartitionKey, NodeSetColumnName, []string{nodeSet})
	} else {
		filter = azurestorage.BuildFilterOnPartition(dncPartitionKey)
	}

	entities, err := n.Datastore.QueryFilter(nodeTableName, filter)
	if err != nil {
		return nil, api.InternalServerError(api.StatusInternalServerError, err.Error())
	}

	nodeConfigs := make([]NodeConfiguration, len(entities))
	for i, entity := range entities {
		serializedNodeConfig := entity.Properties[nodeConfigurationTag].(string)
		var nodeConfiguration NodeConfiguration
		if err := json.Unmarshal([]byte(serializedNodeConfig), &nodeConfiguration); err != nil {
			return nil, api.InternalServerErrorf(api.JsonBodyUnMarshalFailed, "Could not deserialize node %s. Error: %v", entity.RowKey, err)
		}

		nodeConfigs[i] = nodeConfiguration
	}

	return nodeConfigs, nil
}

// GetNodeInfo - Get node configuration for a given node.
func (n *NodeConfigurationRepo) GetNodeInfo(nodeID string) (*api.NodeInfoResponse, *api.JsonErr) {
	repoNodeConfiguration, err := n.TryQueryNodeConfiguration(nodeID)
	if err != nil {
		return nil, err
	}

	if repoNodeConfiguration == nil {
		return nil, &api.JsonErr{
			Code:       api.NodeNotFound,
			Text:       fmt.Sprintf("Node %s not found", nodeID),
			HTTPStatus: http.StatusNotFound}
	}

	var networkContainerResponses []api.NetworkContainerResponse
	constrainedReservations := make(
		[]api.ReservationResponse,
		len(repoNodeConfiguration.ConstrainedReservations))
	index := 0

	for networkID, ncMap := range repoNodeConfiguration.DeployedNetworkContainerMap {
		for ncID := range ncMap {
			ncConfig, err := GetNetworkContainerConfiguration(networkID, ncID)
			if err != nil {
				return nil, err
			}

			networkContainerResponse, err := GetNetworkContainerResponse(ncConfig)
			if err != nil {
				return nil, err
			}

			networkContainerResponses = append(networkContainerResponses, *networkContainerResponse)
			index++
		}
	}

	index = 0
	for reservationID, networkID := range repoNodeConfiguration.ConstrainedReservations {
		reservation, err := QueryIPReservation(networkID, reservationID)
		if err != nil {
			return nil, err
		}

		if reservation.Configuration == nil {
			continue
		}

		constrainedReservations[index] = translateRepoToContractReservationResponse(reservation)
		index++
	}

	nodeConfiguration := api.NodeInfoResponse{
		NodeInfo:                repoNodeConfiguration.NodeInfo,
		NetworkContainers:       networkContainerResponses,
		ConstrainedReservations: constrainedReservations}

	return &nodeConfiguration, nil
}

// GetNodes - Get a slim node info for all nodes
func (n *NodeConfigurationRepo) GetNodes(nodeSet string) (*api.NodesResponse, *api.JsonErr) {
	nodeConfigurations, err := n.TryQueryAllNodeConfigurations(nodeSet)
	if err != nil {
		return nil, err
	}

	slimNodeInfos := make([]api.SlimNodeInfo, len(nodeConfigurations))

	// Build a slice of slim node infos to return based on query result
	for i, nodeConfiguration := range nodeConfigurations {
		slimNodeInfos[i] = api.SlimNodeInfo{Name: nodeConfiguration.NodeID}
	}

	nodes := api.NodesResponse{
		Nodes: slimNodeInfos,
	}

	return &nodes, nil
}

// UpdateNodeConstrainedReservationSet - Add/Remove reservation to constrained reservation set in node configuration
func (n *NodeConfigurationRepo) UpdateNodeConstrainedReservationSet(networkID, reservationID string, remove bool, nodeConfiguration *NodeConfiguration) *api.JsonErr {
	if nodeConfiguration.ConstrainedReservations == nil {
		nodeConfiguration.ConstrainedReservations = make(map[string]string)
	}

	if remove {
		delete(nodeConfiguration.ConstrainedReservations, reservationID)
	} else {
		nodeConfiguration.ConstrainedReservations[reservationID] = networkID
	}

	return n.PersistNodeConfiguration(nodeConfiguration)
}

// UpdateNodeConfigurationNetworkContainerSet - Remove NC from node configuration
func (n *NodeConfigurationRepo) UpdateNodeConfigurationNetworkContainerSet(nodeID, networkID, ncID, reservationID, containerType string, updateVlan, remove bool, ncLimit int) *api.JsonErr {
	nodeConfig, err := n.TryQueryNodeConfiguration(nodeID)
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to retrieve configuration for node %s due to error:%s",
			nodeID,
			err)
		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	if nodeConfig == nil {
		// There can be a race if Node is deleted before this NC was even processed
		// (DeployedNetworkContainerMap is Updated in NC prepare state).
		// Thus, skip if Node is not found.
		if remove {
			loggershim.LS.Debugf("Node :%s is null, skipping removing NC :%s, NetworkId :%s association",
				nodeID,
				ncID,
				networkID)
			return nil
		}

		msg := fmt.Sprintf(
			"Node:%s in the request is not found", nodeID)
		return &api.JsonErr{
			Code:       api.NodeNotFound,
			HTTPStatus: http.StatusNotFound,
			Text:       msg}
	}

	if nodeConfig.DeployedNetworkContainerMap == nil {
		if remove {
			return nil
		}

		nodeConfig.DeployedNetworkContainerMap = make(map[string]map[string]string)
	}

	vnetMap, ok := nodeConfig.DeployedNetworkContainerMap[networkID]
	// If remove and map doesn't exist, just return nil, otherwise operate on map.
	if remove && !ok {
		return nil
	}

	if remove {
		greErr := deleteGreKeyAllocation(*nodeConfig, networkID, ncID)
		if greErr != nil {
			return greErr
		}
		delete(vnetMap, ncID)
		if len(vnetMap) == 0 {
			delete(nodeConfig.DeployedNetworkContainerMap, networkID)

			// todo: this is hacky, but it prevents having to make another call in calling code to get the orchestrator type,
			// which is only stored at the node level. clean up
			if updateVlan && !(nodeConfig.NodeInfo.OrchestratorType == cns.KubernetesCRD) {
				loggershim.LS.Debugf("Freeing VlanID:%d assigned to network %s on node %s",
					nodeConfig.NetworkVlanIDMap[networkID],
					networkID,
					nodeID)
				delete(nodeConfig.NetworkVlanIDMap, networkID)
			}
		}
	} else { // Non remove operations.
		if !ok {
			vnetMap = make(map[string]string)
			nodeConfig.DeployedNetworkContainerMap[networkID] = vnetMap

			if updateVlan && !(nodeConfig.NodeInfo.OrchestratorType == cns.KubernetesCRD) {
				nodeConfig.NetworkVlanIDMap[networkID], err = GetNextVlanID(nodeConfig.NetworkVlanIDMap)
				if err != nil {
					return err
				}

				loggershim.LS.Debugf("Assigning VlanID:%d to network %s on node %s",
					nodeConfig.NetworkVlanIDMap[networkID],
					networkID,
					nodeID)
			}
		}

		vnetMap[ncID] = reservationID

		//Skip the NCLimit check if nodeType is BareMetal
		if nodeConfig.NodeInfo.NodeType == api.BareMetalNodeType {
			loggershim.LS.Debugf("Skipping NC Count Limit check for BareMetal Node %s.", nodeID)
			return n.PersistNodeConfiguration(nodeConfig)
		}

		ncCount := 0
		for _, ncToVnetMap := range nodeConfig.DeployedNetworkContainerMap {
			ncCount += len(ncToVnetMap)
		}

		if ncCount > ncLimit {
			return &api.JsonErr{
				Code:       api.NodeContainerCapacityExhausted,
				Text:       fmt.Sprintf("Node [%s] has reached maximum capacity [%d]", nodeConfig.NodeID, ncLimit),
				HTTPStatus: http.StatusBadRequest}
		}
	}

	return n.PersistNodeConfiguration(nodeConfig)
}

// QueryNodeConfiguration - Query node configuration
func (n *NodeConfigurationRepo) QueryNodeConfiguration(nodeID string) (*NodeConfiguration, *api.JsonErr) {
	nodeConfiguration, err := n.TryQueryNodeConfiguration(nodeID)
	if err != nil {
		return nil, err
	}

	if nodeConfiguration == nil {
		msg := fmt.Sprintf(
			"Configuration for node:[%s] not found.",
			nodeID)
		return nil, &api.JsonErr{
			Code:       api.NodeNotFound,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
	}

	return nodeConfiguration, nil
}

// UpdateNodeConstrainedReservationSetByNodeID - Add/Remove reservation to constrained reservation set in node configuration
func (n *NodeConfigurationRepo) UpdateNodeConstrainedReservationSetByNodeID(nodeID, networkID, reservationID string, remove bool) *api.JsonErr {
	nodeConfiguration, err := n.QueryNodeConfiguration(nodeID)
	if err != nil {
		return err
	}

	return n.UpdateNodeConstrainedReservationSet(
		networkID,
		reservationID,
		remove,
		nodeConfiguration)
}

func (n *NodeConfigurationRepo) UpdateNodeContainers(nodeID, networkID, ncID, reservationID, containerType string, updateVlan, remove bool, ncLimit int) *api.JsonErr {
	return n.UpdateNodeConfigurationNetworkContainerSet(nodeID, networkID, ncID, reservationID, containerType, updateVlan, remove, ncLimit)
}

// GetAllNodeIDs - Retrieve all node IDs
func (n *NodeConfigurationRepo) GetAllNodeIDs() (map[string]bool, error) {
	var context interface{}
	nodeIDs := make(map[string]bool)
	for {
		entities, hasMoreData, cx, status, err := n.Datastore.QueryAllRows(nodeTableName, context)
		if err != nil {
			if status == http.StatusNotFound {
				return nodeIDs, nil
			}

			return nil, err
		}

		for _, entity := range entities {
			nodeIDs[entity.RowKey] = true
		}

		if !hasMoreData {
			break
		}

		context = cx
	}

	return nodeIDs, nil
}

// GetAnyNode - Retrieve any node
func (n *NodeConfigurationRepo) GetAnyNode() (NodeConfiguration, *api.JsonErr) {
	// todo: could have more efficient fetch by using $top=1
	entities, _, _, _, err := n.Datastore.QueryAllRows(nodeTableName, nil)
	if err != nil {
		return NodeConfiguration{}, api.InternalServerErrorf(api.StatusInternalServerError, "could not query all nodes: %v", err)
	}

	if len(entities) < 1 {
		return NodeConfiguration{}, api.NotFound(api.NodeNotFound, "no nodes found")
	}

	nodeJSON, ok := entities[0].Properties[nodeConfigurationTag].(string)
	if !ok {
		return NodeConfiguration{}, api.InternalServerErrorf(api.StatusInternalServerError, "node key %s does not contain string property %s", entities[0].RowKey, nodeConfigurationTag)
	}

	var node NodeConfiguration
	if err := json.Unmarshal([]byte(nodeJSON), &node); err != nil {
		return NodeConfiguration{}, api.InternalServerErrorf(api.StatusInternalServerError, "could not unmarshal node key %s: %v", entities[0].RowKey, err)
	}

	return node, nil
}
