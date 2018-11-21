package datamodel

import (
	"dnc/allocationhelper"
	"dnc/api"
	"dnc/configuration"
	"dnc/datastoreshim"
	"dnc/logger/loggershim"
	"dnc/utils"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

var dataStoreShim *datastoreshim.DataStoreShim
var requestBaseInterval = 1000
var minVlan uint16 = 1
var maxVlan uint16 = 2048
var maxContainersPerNode = 10

// Init - Initialize repository.
func Init(dataStoreSettings configuration.DataStoreSettings, interval int, min, max uint16) error {
	requestBaseInterval = interval
	dataStoreShim = datastoreshim.NewDataStoreShim(dataStoreSettings, TableToColumnMap)
	if dataStoreSettings.TestPersistenceStoreInfo.Test != "" {
		RestoreState()
	}

	if min > minVlan {
		minVlan = min
	}

	if max > minVlan {
		maxVlan = max
	}

	return nil
}

// DumpState - Function to dump test state.
func DumpState() {
	for tableName := range TableToColumnMap {
		kvMap, _, _, _, err := dataStoreShim.QueryAll(tableName, nil)
		if err == nil {
			jsonStr, _ := json.Marshal(kvMap)
			fileName := tableName + ".db"
			fp, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(0664))
			if err == nil {
				fp.Write(jsonStr)
				fp.Close()
			}
		}
	}
}

// RestoreState - Function to restore test state.
func RestoreState() {
	for tableName := range TableToColumnMap {
		fileName := tableName + ".db"
		content, err := ioutil.ReadFile(fileName)
		if err == nil {
			var kvMap map[string]interface{}
			if err = json.Unmarshal(content, &kvMap); err == nil {
				for k, v := range kvMap {
					dataStoreShim.InsertOrUpdate(tableName, k, TableToColumnMap[tableName], v)
				}
			}
		}
	}
}

// PutSubnetConfiguration - Persist subnet configuration.
func PutSubnetConfiguration(subnetConfiguration *SubnetConfiguration) *api.JsonErr {
	subnetKey := GetSubnetKey(subnetConfiguration.NetworkID, subnetConfiguration.SubnetName)
	jsonSerializedBytes, err := json.Marshal(subnetConfiguration)
	if err != nil {
		loggershim.LS.Errorf(
			"%s: Failed to serialize subnet configuration. SubnetKey:%s , Error: %s\n",
			utils.TracePrefix(2),
			subnetKey,
			err.Error())

		return &api.JsonErr{
			Code:       api.JsonMarshallingFailed,
			HTTPStatus: http.StatusInternalServerError,
			Text:       fmt.Sprintf("Failed to serialize node configuration due to error %s", err.Error())}
	}

	loggershim.LS.Debugf("Updating %s with subnetKey %s at TableToColumnMap[%s] and %s\n", subnetConfigurationTableName, subnetKey, TableToColumnMap[subnetConfigurationTableName], string(jsonSerializedBytes[:]))
	err = dataStoreShim.InsertOrUpdate(subnetConfigurationTableName, subnetKey, TableToColumnMap[subnetConfigurationTableName], string(jsonSerializedBytes[:]))
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to persist updated configuration for subnet key %s. Error:%s",
			subnetKey,
			err.Error())
		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	return nil
}

// UpdateAccessTokenOnSubnet - Update access token in a subnet configuration.
func UpdateAccessTokenOnSubnet(networkID, subnetName, accessToken string) *api.JsonErr {
	subnetConfiguration, err := QuerySubnetConfiguration(networkID, subnetName)
	if err != nil {
		return err
	}

	subnetConfiguration.AccessToken = accessToken
	return PutSubnetConfiguration(subnetConfiguration)
}

// DeleteSubnetConfiguration - Delete an existing subnet configuration.
func DeleteSubnetConfiguration(networkID, ipSubnet string) *api.JsonErr {
	subnetKey := GetSubnetKey(networkID, ipSubnet)
	deleteErr := dataStoreShim.Delete(subnetConfigurationTableName, subnetKey)
	if deleteErr != nil {
		msg := fmt.Sprintf(
			"Failed to delete subnet configuration for subnetKey %s from data store. Error: %s",
			subnetKey,
			deleteErr.Error())
		loggershim.LS.Errorf("%s: %s\n", utils.TracePrefix(2), msg)
		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			HTTPStatus: http.StatusInternalServerError,
			Text:       msg}
	}

	return nil
}

// PutIPReservationConfiguration - Persist IP Reservation configuration
func PutIPReservationConfiguration(ipReservation *ReservationConfiguration) *api.JsonErr {
	reservationKey := GetReservationKey(ipReservation.NetworkID, ipReservation.ReservationID)
	jsonSerializedBytes, err := json.Marshal(ipReservation)
	if err != nil {
		loggershim.LS.Errorf(
			"%s: Failed to serialize reservation configuration. ipReservation:%s , error: %s\n",
			utils.TracePrefix(2),
			ipReservation.String(),
			err.Error())

		return &api.JsonErr{
			Code:       api.JsonMarshallingFailed,
			HTTPStatus: http.StatusInternalServerError,
			Text:       fmt.Sprintf("Failed to serialize node configuration due to error %s", err.Error())}
	}

	err = dataStoreShim.InsertOrUpdate(ipReservationsTableName,
		reservationKey,
		TableToColumnMap[ipReservationsTableName],
		string(jsonSerializedBytes[:]))
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to persist updated configuration for reservation %s due to error: %s\n",
			ipReservation.ReservationID,
			err.Error())
		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	return nil
}

// ValidateAndUpdateIPReservationDependencies - Validate and persist IPReservation request.
func ValidateAndUpdateIPReservationDependencies(
	reservationConfiguration *ReservationConfiguration) *api.JsonErr {
	var nodeConfiguration *NodeConfiguration
	var subnetConfiguration *SubnetConfiguration
	var networkContainerConfiguration *NCConfiguration
	var err *api.JsonErr

	subnetConfiguration, err = QuerySubnetConfiguration(
		reservationConfiguration.NetworkID,
		reservationConfiguration.AllocationRequest.SubnetName)
	if err != nil {
		return err
	}

	_, ipSubnet, parseErr := net.ParseCIDR(subnetConfiguration.IPSubnet)
	if parseErr != nil {
		msg := fmt.Sprintf(
			"Failed to parse IPSubnet %s for subnet %s network:%s due to error:%s",
			subnetConfiguration.IPSubnet,
			reservationConfiguration.AllocationRequest.SubnetName,
			reservationConfiguration.NetworkID,
			parseErr.Error())
		loggershim.LS.Errorf("%s: %s.\n", utils.TracePrefix(2), msg)
		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			HTTPStatus: http.StatusInternalServerError,
			Text:       msg}
	}

	if reservationConfiguration.AllocationRequest.IPConstraint != "" {
		ipSubnetConstraint, err := allocationhelper.GetIPSubnetForIPAllocationConstraint(
			reservationConfiguration.AllocationRequest.IPConstraint)
		if err != nil {
			return err
		}

		_, err = allocationhelper.IsSubnetContainedInBaseSubnet(ipSubnet, ipSubnetConstraint)
		if err != nil {
			loggershim.LS.Debugf("%s: %s.\n", utils.TracePrefix(2), err.String())
		}
	}

	if len(reservationConfiguration.AllocationRequest.NodeConstraint) != 0 {
		nodeConfiguration, err = QueryNodeConfiguration(
			reservationConfiguration.AllocationRequest.NodeConstraint)
		if err != nil {
			return err
		}

		if len(reservationConfiguration.AssociatedNCID) > 0 {
			networkContainerConfiguration, err = QueryNetworkContainer(
				reservationConfiguration.NetworkID,
				reservationConfiguration.AssociatedNCID)
			if err != nil {
				return err
			}
			if strings.Compare(
				reservationConfiguration.AllocationRequest.NodeConstraint,
				networkContainerConfiguration.GetAssociationInfo().NodeID) != 0 {
				msg := fmt.Sprintf(
					"Mismatch in node constraint associated with the reservation and network container. "+
						"NodeConstraint:%s , NetworkContainer associated node :%s",
					reservationConfiguration.AllocationRequest.NodeConstraint,
					networkContainerConfiguration.GetAssociationInfo().NodeID)
				return &api.JsonErr{
					Code:       api.NodeConstraintMismatch,
					Text:       msg,
					HTTPStatus: http.StatusBadRequest}
			}
		}

		err = UpdateNodeConstrainedReservationSet(
			reservationConfiguration.NetworkID,
			reservationConfiguration.ReservationID,
			false,
			nodeConfiguration)
		if err != nil {
			return err
		}
	}

	err = UpdateSubnetReservationSet(
		reservationConfiguration.NetworkID,
		reservationConfiguration.ReservationID,
		"",
		false,
		subnetConfiguration)

	return err
}

// UpdateSubnetReservationSet - Add/Remove reservation in subnet set
func UpdateSubnetReservationSet(
	networkID,
	reservationID string,
	allocatedSubnet string,
	remove bool,
	subnetConfiguration *SubnetConfiguration) *api.JsonErr {
	if subnetConfiguration.Reservations == nil {
		subnetConfiguration.Reservations = make(map[string]string)
	}

	if remove {
		delete(subnetConfiguration.Reservations, reservationID)
	} else {
		subnetConfiguration.Reservations[reservationID] = allocatedSubnet
	}

	err := PutSubnetConfiguration(subnetConfiguration)
	if err != nil {
		return err
	}

	return err
}

// UpdateNodeConstrainedReservationSetByNodeID - Add/Remove reservation to constrained reservation set in node configuration
func UpdateNodeConstrainedReservationSetByNodeID(
	nodeID,
	networkID,
	reservationID string,
	remove bool) *api.JsonErr {
	nodeConfiguration, err := QueryNodeConfiguration(nodeID)
	if err != nil {
		return err
	}

	return UpdateNodeConstrainedReservationSet(
		networkID,
		reservationID,
		remove,
		nodeConfiguration)
}

// UpdateNodeConstrainedReservationSet - Add/Remove reservation to constrained reservation set in node configuration
func UpdateNodeConstrainedReservationSet(
	networkID,
	reservationID string,
	remove bool,
	nodeConfiguration *NodeConfiguration) *api.JsonErr {
	if nodeConfiguration.ConstrainedReservations == nil {
		nodeConfiguration.ConstrainedReservations = make(map[string]string)
	}

	if remove {
		delete(nodeConfiguration.ConstrainedReservations, reservationID)
	} else {
		nodeConfiguration.ConstrainedReservations[reservationID] = networkID
	}

	return persistNodeConfiguration(nodeConfiguration)
}

func GetNextVlanID(networkVlanIDMap map[string]uint16) (uint16, *api.JsonErr) {
	v := make(utils.Uint16s, 0, len(networkVlanIDMap))
	for _, value := range networkVlanIDMap {
		v = append(v, value)
	}

	sort.Sort(v)
	var next = minVlan
	for _, usedVlanID := range v {
		if next != usedVlanID {
			if next > maxVlan {
				return 0, &api.JsonErr{
					HTTPStatus: http.StatusInternalServerError,
					Text:       "Exceeded maximum number of VLAN IDs",
					Code:       api.OutOfVlanIDs}
			}

			break
		}

		next++
	}

	return next, nil
}

// UpdateNodeConfigurationNetworkContainerSet - Remove NC from node configuration
func UpdateNodeConfigurationNetworkContainerSet(
	nodeID, networkID, ncID, reservationID, containerType string, updateVlan, remove bool) *api.JsonErr {
	nodeConfig, err := TryQueryNodeConfiguration(nodeID)
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
		delete(vnetMap, ncID)
		if len(vnetMap) == 0 {
			delete(nodeConfig.DeployedNetworkContainerMap, networkID)

			if updateVlan {
				loggershim.LS.Debugf("Freeing VlanID:%d assigned to network %s on node %s\n",
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

			if updateVlan {
				nodeConfig.NetworkVlanIDMap[networkID], err = GetNextVlanID(nodeConfig.NetworkVlanIDMap)
				if err != nil {
					return err
				}

				loggershim.LS.Debugf("Assigning VlanID:%d to network %s on node %s\n",
					nodeConfig.NetworkVlanIDMap[networkID],
					networkID,
					nodeID)
			}
		}

		vnetMap[ncID] = reservationID
	}

	ncCount := 0
	for _, ncToVnetMap := range nodeConfig.DeployedNetworkContainerMap {
		ncCount += len(ncToVnetMap)
	}

	if ncCount > maxContainersPerNode {
		return &api.JsonErr{
			Code:       api.NodeContainerCapacityExhausted,
			Text:       fmt.Sprintf("Node [%s] has reached maximum capacity [%d]", nodeConfig.NodeID, maxContainersPerNode),
			HTTPStatus: http.StatusBadRequest}
	}

	return persistNodeConfiguration(nodeConfig)
}

// UpdateNCToReservationAssociation :-
func UpdateNCToReservationAssociation(
	ncConfiguration *NCConfiguration,
	disassociate bool) *api.JsonErr {
	reservationConfiguration, err := TryQueryIPReservation(
		ncConfiguration.NetworkID,
		ncConfiguration.ReservationID)
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to retrieve configuration for reservation %s in network: {%s} due to error:{%s}",
			ncConfiguration.ReservationID,
			ncConfiguration.NetworkID,
			err)
		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	if reservationConfiguration == nil {
		msg := fmt.Sprintf(
			"Reservation %s is not found", reservationConfiguration.ReservationID)
		return &api.JsonErr{
			Code:       api.ReservationNotFound,
			HTTPStatus: http.StatusNotFound,
			Text:       msg}
	}

	if disassociate {
		if strings.Compare(
			reservationConfiguration.AssociatedNCID,
			ncConfiguration.NetworkContainerID) != 0 {
			return nil
		}
		reservationConfiguration.AssociatedNCID = ""
	} else {
		if len(reservationConfiguration.AssociatedNCID) != 0 &&
			strings.Compare(reservationConfiguration.AssociatedNCID,
				ncConfiguration.NetworkContainerID) != 0 {
			msg := fmt.Sprintf(
				"Reservation %s is use with different network container %s. Request network container %s",
				reservationConfiguration.ReservationID,
				reservationConfiguration.AssociatedNCID,
				ncConfiguration.NetworkContainerID)
			return &api.JsonErr{
				Code:       api.ReservationInUse,
				HTTPStatus: http.StatusBadRequest,
				Text:       msg}
		}
		reservationConfiguration.AssociatedNCID = ncConfiguration.NetworkContainerID
	}

	return PutIPReservationConfiguration(reservationConfiguration)
}

// DeleteIPReservation - Delete an IP Reservation.
func DeleteIPReservation(networkID, reservationID string, force bool) *api.JsonErr {
	reservationConfiguration, err := TryQueryIPReservation(networkID, reservationID)
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to retrieve configuration for reservation %s in network: {%s} due to error:{%s}",
			reservationID,
			networkID,
			err)
		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	if reservationConfiguration == nil {
		return nil
	}

	reservationKey := GetReservationKey(networkID, reservationID)

	if err = DeleteIPAllocation(reservationConfiguration.NetworkID,
		reservationConfiguration.AllocationRequest.SubnetName,
		reservationKey,
		reservationConfiguration.AllocatedSubnet); err != nil {
		loggershim.LS.Errorf("Failed to delete IP allocation due to error %+v\n", err)
		return err
	}

	if len(reservationConfiguration.AssociatedNCID) != 0 && !force {
		msg := fmt.Sprintf(
			"Reservation %s is in use with container: [%s]",
			reservationID, reservationConfiguration.AssociatedNCID)

		loggershim.LS.Errorf("%s: %s\n", utils.TracePrefix(2), msg)

		return &api.JsonErr{
			Code:       api.ReservationInUse,
			HTTPStatus: http.StatusBadRequest,
			Text:       msg}

	}

	subnetConfiguration, err := TryQuerySubnetConfiguration(
		reservationConfiguration.NetworkID,
		reservationConfiguration.AllocationRequest.SubnetName)
	if err != nil {
		return nil
	}

	if subnetConfiguration != nil {
		if subnetConfiguration.Reservations != nil {
			delete(subnetConfiguration.Reservations, reservationID)
			err = PutSubnetConfiguration(subnetConfiguration)
			if err != nil {
				return err
			}
		}
	}

	if len(reservationConfiguration.AllocationRequest.NodeConstraint) != 0 {
		nodeConfiguration, err := TryQueryNodeConfiguration(reservationConfiguration.AllocationRequest.NodeConstraint)
		if err != nil {
			return err
		}
		if nodeConfiguration != nil {
			err := UpdateNodeConstrainedReservationSet(
				reservationConfiguration.NetworkID,
				reservationConfiguration.ReservationID,
				true,
				nodeConfiguration)
			if err != nil {
				return err
			}
		}
	}

	deleteErr := dataStoreShim.Delete(ipReservationsTableName, reservationKey)
	if deleteErr != nil {
		msg := fmt.Sprintf(
			"Failed to delete configuration for reservation key %s from data store. Error: %s",
			reservationKey,
			deleteErr.Error())
		loggershim.LS.Errorf("%s: %s\n", utils.TracePrefix(2), msg)
		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			HTTPStatus: http.StatusInternalServerError,
			Text:       msg}
	}

	return err
}

// DeleteIPAllocation - Delete an IP allocation entry from IPAllocations table.
func DeleteIPAllocation(networkID, subnetName, reservationKey, allocatedIP string) *api.JsonErr {
	if IsFeatureSupported(configuration.SharedSubnetDelegation) {
		loggershim.LS.Debugf("Deleting IP %s from IPAllocations table", allocatedIP)

		allocatedIPKey := EncodeIPAllocationKey(allocatedIP)
		savedReservationKey, err := TryQueryIPAllocation(networkID, subnetName, allocatedIPKey)
		if err != nil {
			loggershim.LS.Errorf("Error: Cannot access IPAllocations table")
			return err
		}

		if len(savedReservationKey) > 0 {
			loggershim.LS.Debugf("Found reservation %s for allocated IP %s", savedReservationKey, allocatedIP)

			// Delete the entry if the saved reservation matches the current reservation
			if reservationKey == savedReservationKey {
				partitionKey := GetSubnetKey(networkID, subnetName)
				err := dataStoreShim.DeleteInPartition(ipAllocationsTableName, partitionKey, allocatedIPKey)
				if err != nil {
					loggershim.LS.Errorf("Error: Cannot delete entry %s from IPAllocations table", allocatedIPKey)
					msg := fmt.Sprintf("Cannot delete entry %s from IPAllocations table", allocatedIPKey)

					return &api.JsonErr{
						Code:       api.StatusInternalServerError,
						HTTPStatus: http.StatusInternalServerError,
						Text:       msg}
				}
			} else {
				loggershim.LS.Errorf("Error: Reservation key mismatch in IPAllocations table")
				msg := fmt.Sprintf("Reservation key mismatch in IPAllocations table for IP %s", allocatedIP)

				return &api.JsonErr{
					Code:       api.InvalidIPSubnet,
					HTTPStatus: http.StatusInternalServerError,
					Text:       msg}
			}
		} else {
			loggershim.LS.Errorf("Error: Cannot find the ipAddress %s in IPAllocation table", allocatedIP)
		}
	}

	return nil
}

// AddIPAllocation - Add the IP allocation to the specified partition of IPAllocations table.
func AddIPAllocation(networkID, subnetName, allocatedIP, reservationKey string) *api.JsonErr {
	partitionKey := GetSubnetKey(networkID, subnetName)
	status, err := dataStoreShim.InsertInPartition(ipAllocationsTableName,
		partitionKey,
		allocatedIP,
		TableToColumnMap[ipAllocationsTableName],
		reservationKey)
	if err != nil {
		jsonErr := &api.JsonErr{}
		if status == http.StatusConflict {
			jsonErr.Code = api.ReservationInUse
			jsonErr.HTTPStatus = status
			jsonErr.Text = fmt.Sprintf("IP is unavailable for allocation")
		} else {
			jsonErr.Code = api.StatusInternalServerError
			jsonErr.HTTPStatus = http.StatusInternalServerError
			jsonErr.Text = fmt.Sprintf("Internal server error")
		}
		return jsonErr
	}

	return nil
}

// TryQueryNodeConfiguration - Query node configuration.
func TryQueryNodeConfiguration(nodeID string) (*NodeConfiguration, *api.JsonErr) {
	nodeKey := GetNodeKey(nodeID)
	return TryQueryNodeConfigurationByKey(nodeKey)
}

// TryQueryNodeConfigurationByKey - Query node configuration by node key.
func TryQueryNodeConfigurationByKey(nodeKey string) (*NodeConfiguration, *api.JsonErr) {
	var repositoryNodeConfiguration NodeConfiguration
	nodeID := GetNodeIDFromKey(nodeKey)
	property, status, err := dataStoreShim.Query(nodeTableName, nodeKey)
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

	if property == nil {
		return nil, nil
	}

	serializedNodeConfiguration := property.(string)
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

// QueryNodeConfiguration - Query node configuration
func QueryNodeConfiguration(nodeID string) (*NodeConfiguration, *api.JsonErr) {
	nodeConfiguration, err := TryQueryNodeConfiguration(nodeID)
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

// TryQueryNetworkConfiguration - Query network configuration.
func TryQueryNetworkConfiguration(networkID string) (*NetworkConfiguration, *api.JsonErr) {
	networkKey := GetNetworkKey(networkID)
	return TryQueryNetworkConfigurationByKey(networkKey)
}

// TryQueryNetworkConfigurationByKey - Query network configuration.
func TryQueryNetworkConfigurationByKey(networkKey string) (*NetworkConfiguration, *api.JsonErr) {
	value, status, err := dataStoreShim.Query(networkConfigurationTableName, networkKey)
	if err != nil && status != http.StatusNotFound {
		msg := fmt.Sprintf(
			"Failed to retrieve configuration for network key:[%s] due to error:[%s]",
			networkKey,
			err)
		return nil, &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	if value == nil {
		return nil, nil
	}

	serializedNetworkConfiguration := value.(string)
	var networkConfiguration NetworkConfiguration
	err = json.Unmarshal([]byte(serializedNetworkConfiguration), &networkConfiguration)
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to unmarshal configuration for network key:[%s] due to error:[%s]",
			networkKey,
			err)
		return nil, &api.JsonErr{
			Code:       api.JsonBodyUnMarshalFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	return &networkConfiguration, nil
}

// QueryNetworkConfiguration - Query subnet configuration.
func QueryNetworkConfiguration(networkID string) (*NetworkConfiguration, *api.JsonErr) {
	networkConfiguration, err := TryQueryNetworkConfiguration(networkID)
	if err != nil {
		return nil, err
	}

	if networkConfiguration == nil {
		msg := fmt.Sprintf(
			"Configuration for network:[%s] not found.",
			networkID)
		return nil, &api.JsonErr{
			Code:       api.NetworkNotFound,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
	}

	return networkConfiguration, nil
}

// TryQuerySubnetConfiguration - Query subnet configuration.
func TryQuerySubnetConfiguration(networkID, subnetName string) (*SubnetConfiguration, *api.JsonErr) {
	subnetKey := GetSubnetKey(networkID, subnetName)
	return TryQuerySubnetConfigurationByKey(subnetKey)
}

// TryQuerySubnetConfigurationByKey - Query subnet configuration.
func TryQuerySubnetConfigurationByKey(subnetKey string) (*SubnetConfiguration, *api.JsonErr) {
	value, status, err := dataStoreShim.Query(subnetConfigurationTableName, subnetKey)
	if err != nil && status != http.StatusNotFound {
		msg := fmt.Sprintf(
			"Failed to retrieve configuration for subnet key:[%s] due to error:[%s]",
			subnetKey,
			err)
		return nil, &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	if value == nil {
		return nil, nil
	}

	serializedSubnetConfiguration := value.(string)
	var subnetConfiguration SubnetConfiguration
	err = json.Unmarshal([]byte(serializedSubnetConfiguration), &subnetConfiguration)
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to unmarshal configuration for subnet key:[%s] due to error:[%s]",
			subnetKey,
			err)
		return nil, &api.JsonErr{
			Code:       api.JsonBodyUnMarshalFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	return &subnetConfiguration, nil
}

// QuerySubnetConfigurationByKey - Query subnet configuration.
func QuerySubnetConfigurationByKey(subnetKey string) (*SubnetConfiguration, *api.JsonErr) {
	subnetConfiguration, err := TryQuerySubnetConfigurationByKey(subnetKey)
	if err != nil {
		return nil, err
	}

	if subnetConfiguration == nil {
		msg := fmt.Sprintf(
			"Configuration for subnet key:[%s] not found.",
			subnetKey)
		return nil, &api.JsonErr{
			Code:       api.SubnetNotFound,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
	}

	return subnetConfiguration, nil
}

// QuerySubnetConfiguration - Query subnet configuration.
func QuerySubnetConfiguration(networkID, subnetName string) (*SubnetConfiguration, *api.JsonErr) {
	subnetConfiguration, err := TryQuerySubnetConfiguration(networkID, subnetName)
	if err != nil {
		return nil, err
	}

	if subnetConfiguration == nil {
		msg := fmt.Sprintf(
			"Configuration for subnet:[%s] in network [%s] not found.",
			subnetName,
			networkID)
		return nil, &api.JsonErr{
			Code:       api.SubnetNotFound,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
	}

	return subnetConfiguration, nil
}

// TryQueryIPReservationByKey - Query IP Reservation configuration.
func TryQueryIPReservationByKey(reservationKey string) (*ReservationConfiguration, *api.JsonErr) {
	value, status, err := dataStoreShim.Query(ipReservationsTableName, reservationKey)
	if err != nil && status != http.StatusNotFound {
		msg := fmt.Sprintf(
			"Failed to retrieve configuration for reservation key:[%s] due to error:[%s]",
			reservationKey,
			err)
		return nil, &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	if value == nil {
		return nil, nil
	}

	serializedReservationConfiguration := value.(string)
	var reservationConfiguration ReservationConfiguration
	err = json.Unmarshal([]byte(serializedReservationConfiguration), &reservationConfiguration)
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to unmarshal configuration for reservation key:[%s] due to error:[%s]",
			reservationKey,
			err)
		return nil, &api.JsonErr{
			Code:       api.JsonBodyUnMarshalFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	return &reservationConfiguration, nil
}

// TryQueryIPReservation - Query IP Reservation configuration.
func TryQueryIPReservation(networkID, reservationID string) (*ReservationConfiguration, *api.JsonErr) {
	reservationKey := GetReservationKey(networkID, reservationID)
	return TryQueryIPReservationByKey(reservationKey)
}

// QueryIPReservation - Query IP Reservation configuration.
func QueryIPReservation(networkID, reservationID string) (*ReservationConfiguration, *api.JsonErr) {
	reservationConfiguration, err := TryQueryIPReservation(networkID, reservationID)
	if err != nil {
		return nil, err
	}

	if reservationConfiguration == nil {
		msg := fmt.Sprintf(
			"Configuration for reservation:[%s] in network [%s] not found.",
			reservationID,
			networkID)
		return nil, &api.JsonErr{
			Code:       api.ReservationNotFound,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
	}

	return reservationConfiguration, nil
}

// TryQueryRequestContext - Query request context.
func TryQueryRequestContext(requestID string) (*RequestContext, *api.JsonErr) {
	var repoRequestContext RequestContext
	value, status, err := dataStoreShim.Query(pendingWorkItemsTableName, requestID)
	if err != nil && status != http.StatusNotFound {
		msg := fmt.Sprintf(
			"Failed to retrieve request context for key:[%s] due to error:[%s]",
			requestID,
			err.Error())
		return nil, &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}
	if value == nil {
		return nil, nil
	}

	serializedRequestContext := value.(string)
	err = json.Unmarshal([]byte(serializedRequestContext), &repoRequestContext)
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to unmarshal context for request with key:[%s] due to error:[%s]",
			requestID,
			err)
		return nil, &api.JsonErr{
			Code:       api.JsonBodyUnMarshalFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	return &repoRequestContext, nil
}

// TryQueryNetworkContainerByKey - Query a network container by key.
func TryQueryNetworkContainerByKey(ncKey string) (*NCConfiguration, *api.JsonErr) {
	var repoNCConfiguration NCConfiguration
	value, status, err := dataStoreShim.Query(networkContainerTableName, ncKey)
	if err != nil && status != http.StatusNotFound {
		msg := fmt.Sprintf(
			"Failed to retrieve configuration for network container key:[%s] due to error:[%s]",
			ncKey,
			err.Error())
		return nil, &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}
	
	if value == nil {
		return nil, nil
	}

	serializedNCConfiguration := value.(string)
	err = json.Unmarshal([]byte(serializedNCConfiguration), &repoNCConfiguration)
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to unmarshal configuration for network container with key:[%s] due to error:[%s]",
			ncKey,
			err)
		return nil, &api.JsonErr{
			Code:       api.JsonBodyUnMarshalFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	return &repoNCConfiguration, nil
}

// TryQueryNetworkContainer - Query a network container by key.
func TryQueryNetworkContainer(networkID, networkContainerID string) (*NCConfiguration, *api.JsonErr) {
	ncKey := GetNCKey(networkID, networkContainerID)
	return TryQueryNetworkContainerByKey(ncKey)
}

// QueryNetworkContainer - Query node configuration.
func QueryNetworkContainer(networkID, networkContainerID string) (*NCConfiguration, *api.JsonErr) {
	ncConfig, err := TryQueryNetworkContainer(networkID, networkContainerID)
	if err != nil {
		return nil, err
	}

	if ncConfig == nil {
		msg := fmt.Sprintf(
			"Configuration for network container with key:[%s] not found.",
			ncKey)
		return nil, &api.JsonErr{
			Code:       api.NetworkContainerNotFound,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
	}

	return ncConfig, nil
}

// TryQueryTagDefinition - Query a tag defintion.
func TryQueryTagDefinition(tagType, tagName string) (*TagDefinition, *api.JsonErr) {
	tagDefintionKey := GetTagDefintionKey(tagType, tagName)
	return TryQueryTagDefinitionByKey(tagDefintionKey)
}

// TryQueryTagDefinitionByKey - Query tag defintion by key.
func TryQueryTagDefinitionByKey(tagDefinitionKey string) (*TagDefinition, *api.JsonErr) {
	var repoTagDefinition TagDefinition
	value, status, err := dataStoreShim.Query(tagDefinitionTableName, tagDefinitionKey)
	if err != nil && status != http.StatusNotFound {
		msg := fmt.Sprintf(
			"Failed to retrieve defintion for tag key:[%s] due to error:[%s]",
			tagDefinitionKey,
			err.Error())
		return nil, &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}
	if value == nil {
		return nil, nil
	}

	serializedTagDefintion := value.(string)
	err = json.Unmarshal([]byte(serializedTagDefintion), &repoTagDefinition)
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to unmarshal defintion for tag with key:[%s] due to error:[%s]",
			tagDefinitionKey,
			err)
		return nil, &api.JsonErr{
			Code:       api.JsonBodyUnMarshalFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	return &repoTagDefinition, nil
}

// CreateOrUpdateTagDefintion - Creates Or Updates a tag defintion.
func CreateOrUpdateTagDefintion(
	tagType string,
	tagName string,
	values []string) *api.JsonErr {
	repositoryTagDefintion, err := TryQueryTagDefinition(tagType, tagName)
	if err != nil {
		return err
	}

	if repositoryTagDefintion == nil {
		loggershim.LS.Debugf("%s: Creating defintion for tag type %s Name:%s\n", utils.TracePrefix(2), tagType, tagName)
		repositoryTagDefintion = &TagDefinition{
			Name:            tagName,
			Type:            tagType,
			CreatedDateTime: time.Now()}
	}
	repositoryTagDefintion.Version++
	repositoryTagDefintion.Values = values
	repositoryTagDefintion.LastUpdatedDateTime = time.Now()

	return persistTagDefinition(repositoryTagDefintion)
}

func persistTagDefinition(tagDefinition *TagDefinition) *api.JsonErr {
	jsonSerializedBytes, err := json.Marshal(tagDefinition)
	if err != nil {
		loggershim.LS.Errorf(
			"%s: Failed to serialize tag defintition %s due to error: %s\n",
			utils.TracePrefix(2),
			tagDefinition.String(),
			err.Error())

		return &api.JsonErr{
			Code:       api.JsonMarshallingFailed,
			HTTPStatus: http.StatusInternalServerError,
			Text: fmt.Sprintf("Failed to serialize tag defintion for tag %s due to error %s",
				tagDefinition.String(),
				err.Error())}
	}

	tagDefintionKey := GetTagDefintionKey(tagDefinition.Type, tagDefinition.Name)
	err = dataStoreShim.InsertOrUpdate(
		tagDefinitionTableName,
		tagDefintionKey,
		TableToColumnMap[tagDefinitionTableName],
		string(jsonSerializedBytes[:]))

	if err != nil {
		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			HTTPStatus: http.StatusInternalServerError,
			Text: fmt.Sprintf("Failed to persist tag definition %s due to error %s",
				tagDefinition.String(),
				err.Error())}
	}

	return nil
}

// CreateOrUpdateNodeConfiguration - Creates Or Updates a node configuration.
func CreateOrUpdateNodeConfiguration(
	nodeID string,
	nodeInfo api.NodeInfoRequest,
	apiVersion string) *api.JsonErr {
	repositoryNodeConfiguration, err := TryQueryNodeConfiguration(nodeID)
	if err != nil {
		return err
	}

	if repositoryNodeConfiguration == nil {
		loggershim.LS.Debugf("%s: Creating node configuration for node %s\n", utils.TracePrefix(2), nodeID)
		repositoryNodeConfiguration = new(NodeConfiguration)
		repositoryNodeConfiguration.APIVersion = apiVersion
		repositoryNodeConfiguration.NodeID = nodeID
		repositoryNodeConfiguration.DeployedNetworkContainerMap = make(map[string]map[string]string)
		repositoryNodeConfiguration.NetworkVlanIDMap = make(map[string]uint16)
		repositoryNodeConfiguration.ConstrainedReservations = make(map[string]string)
	}

	repositoryNodeConfiguration.NodeInfo = nodeInfo
	return persistNodeConfiguration(repositoryNodeConfiguration)
}

func persistNodeConfiguration(nodeconfiguration *NodeConfiguration) *api.JsonErr {
	jsonSerializedBytes, err := json.Marshal(nodeconfiguration)
	if err != nil {
		loggershim.LS.Errorf(
			"%s: Failed to serialize node configuration %s due to error: %s\n",
			utils.TracePrefix(2),
			nodeconfiguration.String(),
			err.Error())

		return &api.JsonErr{
			Code:       api.JsonMarshallingFailed,
			HTTPStatus: http.StatusInternalServerError,
			Text:       fmt.Sprintf("%s: Failed to serialize node configuration %s due to error %s", utils.TracePrefix(2), nodeconfiguration.String(), err.Error())}
	}

	nodeKey := GetNodeKey(nodeconfiguration.NodeID)
	err = dataStoreShim.InsertOrUpdate(
		nodeTableName,
		nodeKey,
		TableToColumnMap[nodeTableName],
		string(jsonSerializedBytes[:]))

	if err != nil {
		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			HTTPStatus: http.StatusInternalServerError,
			Text:       fmt.Sprintf("%s: Failed to persist node configuration %s due to error %s", utils.TracePrefix(2), nodeconfiguration.String(), err.Error())}
	}

	return nil
}

// DeleteNode - Delete a node configuration.
func DeleteNode(nodeID string) *api.JsonErr {
	repositoryNodeConfiguration, err := TryQueryNodeConfiguration(nodeID)
	if err != nil {
		return nil
	}

	if repositoryNodeConfiguration == nil {
		return nil
	}

	if repositoryNodeConfiguration.DeployedNetworkContainerMap != nil &&
		len(repositoryNodeConfiguration.DeployedNetworkContainerMap) != 0 {
		deployedNCString := repositoryNodeConfiguration.DumpDeployedNetworkContainers()
		msg := fmt.Sprintf("Node has network containers deployed. Deployed containers: [%s]", deployedNCString)
		loggershim.LS.Errorf("%s: %s\n", utils.TracePrefix(2), msg)

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
			"Node has associated reservations. Reservations: [%s]",
			constrainedReservationsString)
		loggershim.LS.Errorf("%s: %s\n", utils.TracePrefix(2), msg)
		return &api.JsonErr{
			Code:       api.NodeInUse,
			HTTPStatus: http.StatusBadRequest,
			Text:       msg}
	}

	nodeKey := GetNodeKey(nodeID)
	deleteErr := dataStoreShim.Delete(nodeTableName, nodeKey)
	if deleteErr != nil {
		msg := fmt.Sprintf(
			"Failed to delete node configuration for node %s from data store. Error: %s",
			nodeID,
			deleteErr.Error())
		loggershim.LS.Errorf("%s: %s\n", utils.TracePrefix(2), msg)
		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			HTTPStatus: http.StatusInternalServerError,
			Text:       msg}
	}

	return nil
}

// GetNetworkSubnets = Query virtual network subnets.
func GetNetworkSubnets(networkID string) (map[string]string, error) {
	rowKey := GetNetworkKey(networkID)
	properties, statusCode, err := dataStoreShim.Query(networkConfigurationTableName, rowKey)
	if err != nil {
		if statusCode == http.StatusNotFound {
			vnetSubnets := make(map[string]string)
			return vnetSubnets, nil
		}
		return nil, err
	}
	serializedSubnetString := properties.(string)
	var vnetSubnets map[string]string
	err = json.Unmarshal([]byte(serializedSubnetString), &vnetSubnets)
	return vnetSubnets, err
}

// updateNetworkSubnets - Update network subnets.
func updateNetworkSubnets(networkID string, subnets map[string]string) *api.JsonErr {
	networkConfiguration, err := QueryNetworkConfiguration(networkID)
	if err != nil {
		return err
	}
	networkConfiguration.Subnets = subnets
	return PersistNetworkConfiguration(networkConfiguration)
}

// deleteVirtualNetworkSubnet = delete virtual network subnet.
func deleteVirtualNetworkSubnet(networkID string, subnet string) error {
	rowKey := GetNetworkKey(networkID)
	properties, statusCode, err := dataStoreShim.Query(networkConfigurationTableName, rowKey)
	if err != nil {
		if statusCode == http.StatusNotFound {
			return nil
		}
		return err
	}

	var vnetSubnets map[string]string
	err = json.Unmarshal(properties.([]byte), &vnetSubnets)
	if err != nil {
		return err
	}

	delete(vnetSubnets, subnet)
	if len(vnetSubnets) == 0 {
		dataStoreShim.Delete(networkConfigurationTableName, rowKey)
	} else {
		updateNetworkSubnets(networkID, vnetSubnets)
	}
	return nil
}

// AddRequestToPendingQueue - Add request to pending work item queue.
func AddRequestToPendingQueue(
	requestKey string,
	requestContext *RequestContext) *api.JsonErr {
	repoRequestContext, requestReadErr := TryQueryRequestContext(requestKey)
	if requestReadErr != nil {
		msg := fmt.Sprintf(
			"Failed to check if request context exist in repository. Key:%s, Error: %s",
			requestKey,
			requestReadErr.String())
		loggershim.LS.Errorf("%s: %s\n", utils.TracePrefix(2), msg)
		return requestReadErr
	}

	if repoRequestContext != nil {
		requestContext.CreatedDateTime = repoRequestContext.CreatedDateTime
	} else {
		requestContext.CreatedDateTime = time.Now().UTC()
		requestContext.RetryInterval = time.Duration(requestBaseInterval) * time.Millisecond
	}

	jsonSerializedBytes, err := json.Marshal(requestContext)
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to serialize request context. Key:%s RequestContext:%s , Error: %s",
			requestKey,
			requestContext.String(),
			err.Error())
		loggershim.LS.Errorf("%s: %s\n", utils.TracePrefix(2), msg)

		return &api.JsonErr{
			Code:       api.JsonMarshallingFailed,
			HTTPStatus: http.StatusInternalServerError,
			Text:       msg}
	}

	err = dataStoreShim.InsertOrUpdate(pendingWorkItemsTableName, requestKey, TableToColumnMap[pendingWorkItemsTableName], string(jsonSerializedBytes[:]))
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to add request %s associated with key %s to pending queue due to error:{%s}",
			requestContext.String(),
			requestKey,
			err.Error())
		loggershim.LS.Errorf("%s: %s\n", utils.TracePrefix(2), msg)
		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			HTTPStatus: http.StatusInternalServerError,
			Text:       msg}
	}

	return nil
}

// TryGetRequestFromPendingQueue - Try get request with given Id from pending queue.
func TryGetRequestFromPendingQueue(requestKey string) (*RequestContext, *api.JsonErr) {
	value, _, err := dataStoreShim.Query(
		pendingWorkItemsTableName,
		requestKey)
	if err != nil {
		loggershim.LS.Errorf("Failed to query pending work item queue due to error: %s\n", err.Error())
		return nil, &api.JsonErr{
			Code:       api.StatusInternalServerError,
			HTTPStatus: http.StatusInternalServerError,
			Text: fmt.Sprintf(
				"Failed to retrieve configuration for request key:%s. Error:%s",
				requestKey,
				err.Error())}
	}
	if value == nil {
		return nil, nil
	}
	serializedRequestContext := value.(string)
	var requestContext RequestContext
	err = json.Unmarshal([]byte(serializedRequestContext), &requestContext)
	if err != nil {
		loggershim.LS.Errorf("Failed to qunmarshal requestdue to error: %s\n", err.Error())
		return nil, &api.JsonErr{
			Code:       api.StatusInternalServerError,
			HTTPStatus: http.StatusInternalServerError,
			Text: fmt.Sprintf(
				"Failed to unmarshal configuration for request key:%s, Request:%s. Error:%s",
				requestKey,
				serializedRequestContext,
				err.Error())}
	}

	return &requestContext, nil
}

// GetAllTagDefinitions - Retrieve all tag defintitions
func GetAllTagDefinitions() (map[string]TagDefinition, error) {
	var hasMoreData bool
	var context interface{}
	var tagDefinitions map[string]interface{}
	var err error
	var status int
	tagDefinitionsMap := make(map[string]TagDefinition)
	for {
		tagDefinitions, hasMoreData, context, status, err = dataStoreShim.QueryAll(
			tagDefinitionTableName,
			context)
		if err != nil {
			if status == http.StatusNotFound {
				return make(map[string]TagDefinition), nil
			}

			return nil, err
		}

		for tagDefinitionsKey, value := range tagDefinitions {
			serializedTagDefinition := value.(string)
			var tagDefinition TagDefinition
			err = json.Unmarshal([]byte(serializedTagDefinition), &tagDefinition)
			if err != nil {
				loggershim.LS.Errorf("Unmarshalling tag defintition %s failed with error %s\n", tagDefinition.String(), err.Error())
				return nil, err
			}

			tagDefinitionsMap[tagDefinitionsKey] = tagDefinition
		}

		if !hasMoreData {
			break
		}
	}

	return tagDefinitionsMap, nil
}

// GetPendingRequests - Retrieve all request pending processing
func GetPendingRequests() (map[string]RequestContext, error) {
	var hasMoreData bool
	var context interface{}
	var requests map[string]interface{}
	var err error
	var status int
	requestsToProcess := make(map[string]RequestContext)
	for {
		requests, hasMoreData, context, status, err = dataStoreShim.QueryAll(
			pendingWorkItemsTableName,
			context)
		if err != nil {
			if status == http.StatusNotFound {
				return make(map[string]RequestContext), nil
			}

			return nil, err
		}

		for requestKey, value := range requests {
			serializedRequestContext := value.(string)
			var requestContext RequestContext
			err = json.Unmarshal([]byte(serializedRequestContext), &requestContext)
			requestsToProcess[requestKey] = requestContext
		}

		if !hasMoreData {
			break
		}
	}

	return requestsToProcess, nil
}

// RemovePendingRequest - Remove pending request from work item queue.
func RemovePendingRequest(requestID string) {
	dataStoreShim.Delete(pendingWorkItemsTableName, requestID)
}

// PersistNCConfiguration - Persist network container configuration.
func PersistNCConfiguration(ncConfiguration *NCConfiguration, addToPendingQueue bool) *api.JsonErr {
	var jsonSerializedBytes []byte
	var err error
	if ncConfiguration.FailureDetail != (api.JsonErr{}) {
		loggershim.LS.Errorf("[%s] Error: %s", utils.TracePrefix(3), ncConfiguration.FailureDetail.String())
	}

	jsonSerializedBytes, err = json.Marshal(ncConfiguration)
	if err != nil {
		msg := fmt.Sprintf("Failed to serialize node configuration for NC ID:%s , Error: %s",
			ncConfiguration.NetworkContainerID,
			err.Error())
		loggershim.LS.Errorf(
			"%s: %s\n",
			utils.TracePrefix(2),
			msg)

		return &api.JsonErr{
			Code:       api.JsonMarshallingFailed,
			HTTPStatus: http.StatusInternalServerError,
			Text:       msg}
	}

	ncKey := GetNCKey(ncConfiguration.NetworkID, ncConfiguration.NetworkContainerID)
	err = dataStoreShim.InsertOrUpdate(networkContainerTableName,
		ncKey,
		TableToColumnMap[networkContainerTableName],
		string(jsonSerializedBytes[:]))
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to persist updated configuration for network container key%s. Error:%s",
			ncKey,
			err.Error())
		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	if addToPendingQueue {
		requestContext := RequestContext{
			Type:      NetworkContainerRequest,
			NetworkID: ncConfiguration.NetworkID,
			NodeID:    ncConfiguration.GetAssociationInfo().NodeID}

		return AddRequestToPendingQueue(ncKey, &requestContext)
	}

	return nil
}

// DeleteNetworkContainer - Delete network container configuration.
func DeleteNetworkContainer(networkID, networkContainerID string) *api.JsonErr {
	ncKey := GetNCKey(networkID, networkContainerID)
	err := dataStoreShim.Delete(networkContainerTableName, ncKey)
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to delete network container %s in data store. Error: %s",
			networkContainerID,
			err.Error())
		loggershim.LS.Errorf("%s: %s\n", utils.TracePrefix(2), msg)
		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			HTTPStatus: http.StatusInternalServerError,
			Text:       msg}
	}

	return nil
}

// GetNetworkContainerConfiguration - Get network container configuration
func GetNetworkContainerConfiguration(networkID,
	networkContainerID string) (*NCConfiguration, *api.JsonErr) {
	networkContainerConfig, err := TryQueryNetworkContainer(
		networkID, networkContainerID)
	if err != nil {
		return nil, nil
	}
	if networkContainerConfig == nil {
		msg := fmt.Sprintf("Network container %s not found", networkContainerID)
		loggershim.LS.Errorf("%s: %s\n", utils.TracePrefix(2), msg)
		return nil, &api.JsonErr{
			Code:       api.NetworkContainerNotFound,
			HTTPStatus: http.StatusNotFound,
			Text:       msg}
	}

	return networkContainerConfig, nil
}

// GetRevertState - Get revert state for a given network container state.
func GetRevertState(ncState NetworkContainerState) (NetworkContainerState, *api.JsonErr) {
	switch ncState {
	case Prepare:
		return Purge, nil

	case Reserve:
		return UnReserve, nil

	case AllocateLocalIP:
		return UnReserve, nil

	case Associate:
		return DisAssociate, nil

	case RefreshAssociation:
		return UnPublish, nil

	case Publish:
		return UnPublish, nil

	case Program:
		return UnPublish, nil

	default:
		return Invalid, &api.JsonErr{
			Code:       api.InvalidNetworkStateForDelete,
			Text:       fmt.Sprintf("Network container state:%s is invalid", ncState),
			HTTPStatus: http.StatusConflict}
	}
}

// ProcessNetworkContainerForRelease - Process network container configuration pending release.
func ProcessNetworkContainerForRelease(networkID, networkContainerID string) *api.JsonErr {
	networkContainerConfig, err := TryQueryNetworkContainer(
		networkID, networkContainerID)
	if err != nil {
		return err
	}

	if networkContainerConfig == nil {
		return nil
	}

	if !networkContainerConfig.MarkedForDelete {
		networkContainerConfig.MarkedForDelete = true
		networkContainerConfig.Status = Pending
		revertState, err := GetRevertState(networkContainerConfig.State)
		if err != nil {
			return err
		}
		loggershim.LS.Debugf("NC %s transitioning from state %s to %s",
			networkContainerConfig.NetworkContainerID,
			networkContainerConfig.State, revertState)
		networkContainerConfig.State = revertState
		return PersistNCConfiguration(networkContainerConfig, true)
	}

	return nil
}

// PersistNetworkConfiguration - Persist network configuration.
func PersistNetworkConfiguration(
	networkConfiguration *NetworkConfiguration) *api.JsonErr {
	networkKey := GetNetworkKey(networkConfiguration.NetworkID)
	jsonSerializedBytes, err := json.Marshal(networkConfiguration)
	if err != nil {
		msg := fmt.Sprintf("Failed to serialize network configuration for network %s. "+
			"NetworkConfiguration:%s , Error: %s",
			networkConfiguration.NetworkID,
			networkConfiguration.String(),
			err.Error())
		loggershim.LS.Errorf(
			"%s: %s\n",
			utils.TracePrefix(2),
			msg)

		return &api.JsonErr{
			Code:       api.JsonMarshallingFailed,
			HTTPStatus: http.StatusInternalServerError,
			Text:       msg}
	}
	err = dataStoreShim.InsertOrUpdate(
		networkConfigurationTableName,
		networkKey,
		TableToColumnMap[networkConfigurationTableName],
		string(jsonSerializedBytes[:]))
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to persist network configuration for network %s. Error:%s",
			networkConfiguration.NetworkID,
			err.Error())
		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	return nil

}

// CreateNetwork - Creates network configuration
func CreateNetwork(
	networkID string,
	networkInfo api.NetworkInfoRequest,
	apiVersion string) *api.JsonErr {
	networkConfiguration, err := TryQueryNetworkConfiguration(networkID)
	if err != nil {
		return err
	}

	if networkConfiguration != nil {
		return nil
	}

	networkKey := GetNetworkKey(networkID)
	networkConfiguration = &NetworkConfiguration{
		NetworkID:    networkID,
		NetworkState: Join,
		Status:       Pending,
		APIVersion:   apiVersion}

	err = PersistNetworkConfiguration(networkConfiguration)
	if err != nil {
		return err
	}

	requestContext := RequestContext{
		Type:      VirtualNetworkRequest,
		NetworkID: networkID}

	return AddRequestToPendingQueue(networkKey, &requestContext)

}

// UpdateSubnet - Update subnet configuration
func UpdateSubnet(networkID, subnet string, subnetRequest api.SubnetRequest) *api.JsonErr {
	networkConfig, err := TryQueryNetworkConfiguration(networkID)
	if err != nil {
		return err
	}

	if networkConfig == nil {
		msg := fmt.Sprintf("Network with ID %s not found", networkID)
		return &api.JsonErr{
			Code:       api.NetworkNotFound,
			HTTPStatus: http.StatusNotFound,
			Text:       msg}
	}

	if networkConfig.NetworkState == Join && networkConfig.Status != Completed {
		msg := fmt.Sprintf("Network with ID %s join is not completed. Current Status: %s",
			networkID,
			networkConfig.Status.String())
		return &api.JsonErr{
			Code:       api.NetworkStatusInvalid,
			HTTPStatus: http.StatusBadRequest,
			Text:       msg}
	}

	if networkConfig.Subnets == nil || len(networkConfig.Subnets) == 0 {
		msg := fmt.Sprintf("No subnets found in Network with ID %s",
			networkID)
		return &api.JsonErr{
			Code:       api.SubnetNotFound,
			HTTPStatus: http.StatusBadRequest,
			Text:       msg}
	}

	_, ok := networkConfig.Subnets[subnet]
	if !ok {
		msg := fmt.Sprintf("Subnet %s not found in Network with ID %s",
			subnet,
			networkID)
		return &api.JsonErr{
			Code:       api.SubnetNotFound,
			HTTPStatus: http.StatusBadRequest,
			Text:       msg}
	}

	subnetConfiguration, err := TryQuerySubnetConfiguration(networkID, subnet)
	if err != nil {
		return err
	}

	if subnetConfiguration == nil {
		msg := fmt.Sprintf("Subnet %s configuration not found. Network with ID %s",
			subnet,
			networkID)
		return &api.JsonErr{
			Code:       api.SubnetConfigurationNotFound,
			HTTPStatus: http.StatusBadRequest,
			Text:       msg}
	}

	if !subnetConfiguration.IsDelegatedSubnet {
		msg := fmt.Sprintf("Subnet %s is not a delegated subnet. Network with ID %s",
			subnet,
			networkID)
		return &api.JsonErr{
			Code:       api.SubnetIsNotDelegated,
			HTTPStatus: http.StatusBadRequest,
			Text:       msg}

	}
	subnetConfiguration.Joined = true
	subnetConfiguration.AccessToken = subnetRequest.AuthenticationToken
	err = PutSubnetConfiguration(subnetConfiguration)
	if err != nil {
		return err
	}

	return nil
}

// GetNodeIDForNetworkContainer - Helper to retrieve node Id from NC configuration.
func GetNodeIDForNetworkContainer(networkID, ncID string) (string, *api.JsonErr) {
	networkContainerConfiguration, err := QueryNetworkContainer(networkID, ncID)
	if err != nil {
		return "", err
	}
	return networkContainerConfiguration.GetAssociationInfo().NodeID, nil
}

// PurgeNetworkContainerAssociations - Process relase of resevation associated with network container
func PurgeNetworkContainerAssociations(ncConfiguration *NCConfiguration, updateVlan bool) *api.JsonErr {
	nodeID := ncConfiguration.GetAssociationInfo().NodeID
	err := UpdateNodeConfigurationNetworkContainerSet(
		nodeID,
		ncConfiguration.NetworkID,
		ncConfiguration.NetworkContainerID,
		ncConfiguration.ReservationID,
		ncConfiguration.GetAssociationInfo().ContainerType,
		updateVlan,
		true)
	if err != nil {
		return err
	}

	if ncConfiguration.ImplicitReservation {
		err = DeleteIPReservation(ncConfiguration.NetworkID, ncConfiguration.ReservationID, true)
	} else {
		err = UpdateNCToReservationAssociation(ncConfiguration, true)
	}

	return err
}

// DeleteNetworkConfiguration :-
func DeleteNetworkConfiguration(networkID string) *api.JsonErr {
	networkConfiguration, err := TryQueryNetworkConfiguration(networkID)
	if err != nil {
		return err
	}

	if networkConfiguration == nil {
		return nil
	}

	subnetKeysToDelete := make([]string, len(networkConfiguration.Subnets))
	index := 0

	for subnetName := range networkConfiguration.Subnets {
		subnetConfiguration, err := TryQuerySubnetConfiguration(networkID, subnetName)
		if err != nil {
			return err
		}

		if subnetConfiguration == nil {
			continue
		}

		if len(subnetConfiguration.Reservations) > 0 {
			reservationsString := utils.PrintMapKeys(subnetConfiguration.Reservations)
			msg := fmt.Sprintf(
				"Subnet %s in network %s has allocations in use %s",
				subnetName,
				networkID,
				reservationsString)
			return &api.JsonErr{
				Code:       api.NetworkInUse,
				HTTPStatus: http.StatusBadRequest,
				Text:       msg}
		}

		subnetKeysToDelete[index] = subnetName
		index++
	}

	for _, subnetName := range subnetKeysToDelete {
		err := DeleteSubnetConfiguration(networkID, subnetName)
		if err != nil {
			return err
		}
	}

	networkKey := GetNetworkKey(networkID)
	deleteErr := dataStoreShim.Delete(networkConfigurationTableName, networkKey)
	if deleteErr != nil {
		msg := fmt.Sprintf(
			"Failed to delete network configuration network-key: %s network-id: %s due to error %s",
			networkKey,
			networkID,
			deleteErr.Error())
		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			HTTPStatus: http.StatusInternalServerError,
			Text:       msg}
	}

	return nil
}

func GetSubnetDNSServers(repoSubnetConfiguration *SubnetConfiguration) ([]string, *api.JsonErr) {
	repoNetworkConfiguration, err := TryQueryNetworkConfiguration(repoSubnetConfiguration.NetworkID)
	if err != nil {
		return nil, err
	}

	if repoNetworkConfiguration == nil {
		msg := fmt.Sprintf(
			"Configuration for network:%s not found.",
			repoSubnetConfiguration.NetworkID)

		return nil, &api.JsonErr{
			Code:       api.NetworkNotFound,
			HTTPStatus: http.StatusBadRequest,
			Text:       msg}
	}

	return GetDNSServers(repoNetworkConfiguration, repoSubnetConfiguration)
}

func GetDNSServers(repoNetworkConfiguration *NetworkConfiguration, repoSubnetConfiguration *SubnetConfiguration) ([]string, *api.JsonErr) {
	var dnsServers []string
	if repoNetworkConfiguration.DNSServers != nil &&
		len(repoNetworkConfiguration.DNSServers) != 0 {
		dnsServers = repoNetworkConfiguration.DNSServers
	} else {
		ipSubnet, err := utils.ParseIPNet(repoSubnetConfiguration.IPSubnet)
		if err != nil {
			msg := fmt.Sprintf("Failed to parse subnet %s for subnet %s for network %s due to error %s",
				repoSubnetConfiguration.IPSubnet,
				repoSubnetConfiguration.SubnetName,
				repoSubnetConfiguration.NetworkID,
				err.Error())
			loggershim.LS.Errorf("%s\n", msg)

			return nil, &api.JsonErr{
				Code:       api.InvalidIPSubnet,
				HTTPStatus: http.StatusInternalServerError,
				Text:       msg}
		}

		dnsServers = allocationhelper.GetDNSServerAddressesForSubnet(ipSubnet)
	}

	return dnsServers, nil
}

func getIPConfiguration(repoReservationConfiguration *ReservationConfiguration) (*api.IPConfiguration, *api.JsonErr) {
	repoSubnetConfiguration, err := TryQuerySubnetConfiguration(
		repoReservationConfiguration.NetworkID,
		repoReservationConfiguration.AllocationRequest.SubnetName)

	if err != nil {
		return nil, err
	}

	if repoSubnetConfiguration == nil {
		msg := fmt.Sprintf(
			"Configuration for subnet %s in network:%s not found.",
			repoReservationConfiguration.AllocationRequest.SubnetName,
			repoReservationConfiguration.NetworkID)
		loggershim.LS.Errorf("%s: %s\n", utils.TracePrefix(2), msg)
		return nil, &api.JsonErr{
			Code:       api.SubnetNotFound,
			HTTPStatus: http.StatusBadRequest,
			Text:       msg}
	}

	dnsServers, err := GetSubnetDNSServers(repoSubnetConfiguration)
	if err != nil {
		return nil, err
	}

	ipConfiguration := api.IPConfiguration{
		NetworkID:  repoReservationConfiguration.NetworkID,
		SubnetName: repoReservationConfiguration.AllocationRequest.SubnetName,
		IPSubnet:   repoSubnetConfiguration.IPSubnet,
		IPAddress:  repoReservationConfiguration.AllocatedSubnet,
		GatewayIP:  repoSubnetConfiguration.Gateway,
		DNSServers: dnsServers}

	return &ipConfiguration, nil
}

// GetNetworkContainerResponse - Get network container response for network config.
func GetNetworkContainerResponse(
	ncConfig *NCConfiguration) (*api.NetworkContainerResponse, *api.JsonErr) {
	var ipConfiguration *api.IPConfiguration
	var nodeConfiguration *NodeConfiguration

	if ncConfig.State != Prepare &&
		!(ncConfig.State == Reserve && ncConfig.Status != Completed) {
		reservationConfiguration, err := QueryIPReservation(
			ncConfig.NetworkID,
			ncConfig.ReservationID)
		if err != nil {
			return nil, err
		}

		ipConfiguration, err = getIPConfiguration(reservationConfiguration)
		if err != nil {
			return nil, err
		}

		nodeConfiguration, err = QueryNodeConfiguration(ncConfig.GetAssociationInfo().NodeID)
		if err != nil {
			return nil, err
		}
	}

	requestStatus := api.RequestStatus{
		State:         ncConfig.State.String(),
		Status:        ncConfig.Status.String(),
		FailureDetail: ncConfig.FailureDetail}
	associationInfo := ncConfig.GetAssociationInfo()
	response := api.NetworkContainerResponse{
		NcID:            ncConfig.NetworkContainerID,
		AssociationInfo: associationInfo,
		RequestStatus:   requestStatus,
		VlanID:          nodeConfiguration.NetworkVlanIDMap[ncConfig.NetworkID]}

	if ipConfiguration != nil {
		response.IPConfiguration = *ipConfiguration
	}

	return &response, nil
}

func translateRepoToContractReservationResponse(
	reservationConfiguration *ReservationConfiguration) api.ReservationResponse {
	requestStatus := api.RequestStatus{
		State:         reservationConfiguration.ReservationState.String(),
		Status:        reservationConfiguration.Status.String(),
		FailureDetail: reservationConfiguration.FailureDetail}

	return api.ReservationResponse{
		NetworkID:         reservationConfiguration.NetworkID,
		ReservationID:     reservationConfiguration.ReservationID,
		AllocationRequest: reservationConfiguration.AllocationRequest,
		AllocatedIPSubnet: reservationConfiguration.AllocatedSubnet,
		RequestStatus:     requestStatus}
}

// GetNodeInfo - Get node configuration for a given node.
func GetNodeInfo(nodeID string) (*api.NodeInfoResponse, *api.JsonErr) {
	repoNodeConfiguration, err := TryQueryNodeConfiguration(nodeID)
	if err != nil {
		return nil, err
	}

	if repoNodeConfiguration == nil {
		return nil, &api.JsonErr{
			Code:       api.NodeNotFound,
			Text:       fmt.Sprintf("Node %s not found", nodeID),
			HTTPStatus: http.StatusNotFound}
	}

	if repoNodeConfiguration.DeployedNetworkContainerMap == nil {
		repoNodeConfiguration.DeployedNetworkContainerMap = make(map[string]map[string]string)
	}

	if repoNodeConfiguration.ConstrainedReservations == nil {
		repoNodeConfiguration.ConstrainedReservations = make(map[string]string)
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
		reservationConfiguration, err := QueryIPReservation(networkID, reservationID)
		if err != nil {
			return nil, err
		}

		if reservationConfiguration == nil {
			continue
		}

		constrainedReservations[index] = translateRepoToContractReservationResponse(reservationConfiguration)
		index++
	}

	nodeConfiguration := api.NodeInfoResponse{
		NodeInfo:                repoNodeConfiguration.NodeInfo,
		NetworkContainers:       networkContainerResponses,
		ConstrainedReservations: constrainedReservations}

	return &nodeConfiguration, nil
}

// PersistIPReservationRequest :-
func PersistIPReservationRequest(networkID string,
	reservationID string,
	allocationRequest api.AllocationRequest,
	apiVersion string) *api.JsonErr {

	reservationConfiguration, err := TryQueryIPReservation(networkID, reservationID)
	if reservationConfiguration != nil {
		return nil
	}

	reservationConfiguration = &ReservationConfiguration{
		NetworkID:         networkID,
		ReservationID:     reservationID,
		ReservationState:  Process,
		Status:            Pending,
		AllocationRequest: allocationRequest,
		APIVersion:        apiVersion}

	err = PutIPReservationConfiguration(reservationConfiguration)
	if err != nil {
		return err
	}

	requestContext := RequestContext{
		Type:      ReservationRequest,
		NetworkID: reservationConfiguration.NetworkID,
		NodeID:    reservationConfiguration.AllocationRequest.NodeConstraint}

	reservationKey := GetReservationKey(networkID, reservationID)
	return AddRequestToPendingQueue(reservationKey, &requestContext)
}

// GetReservationResponse - Retrieve reservation response.
func GetReservationResponse(
	networkID string,
	reservationID string) (*api.ReservationResponse, *api.JsonErr) {
	reservationConfiguration, err := QueryIPReservation(networkID, reservationID)
	if err != nil {
		return nil, err
	}

	response := translateRepoToContractReservationResponse(reservationConfiguration)
	return &response, nil
}

// GetReservationStatus - Get reservation status
func GetReservationStatus(networkID, reservationID string) (*api.RequestStatus, *api.JsonErr) {
	repoReservationConfiguration, err := QueryIPReservation(networkID, reservationID)
	if err != nil {
		return nil, err
	}

	if repoReservationConfiguration == nil {
		msg := fmt.Sprintf(
			"Reservation %s in Network %s is not found", reservationID, networkID)
		return nil, &api.JsonErr{
			Code:       api.ReservationNotFound,
			HTTPStatus: http.StatusNotFound,
			Text:       msg}
	}

	var requestStatus *api.RequestStatus
	requestStatus = &api.RequestStatus{
		State:         repoReservationConfiguration.ReservationState.String(),
		Status:        repoReservationConfiguration.Status.String(),
		FailureDetail: repoReservationConfiguration.FailureDetail}

	return requestStatus, nil
}

// CreateOrUpdateNCRequestV1 :-
func CreateOrUpdateNCRequestV1(
	networkID, ncID string,
	ncRequestV1 *api.NetworkContainerRequestV1,
	apiVersion string) *api.JsonErr {
	ncConfig, err := TryQueryNetworkContainer(networkID, ncID)
	if err != nil {
		msg := fmt.Sprintf(
			"Failed when trying to retrieve configuration for network container:%s due to error:%s",
			ncID,
			err)
		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	if ncConfig != nil {
		return nil
	}

	nodeConfig, err := TryQueryNodeConfiguration(ncRequestV1.AssociationInfo.NodeID)
	if err != nil {
		return err
	}

	if nodeConfig == nil {
		msg := fmt.Sprintf(
			"Failed when trying to retrieve configuration for node:%s due to error:%s",
			ncRequestV1.AssociationInfo.NodeID,
			err)
		return &api.JsonErr{
			Code:       api.NodeNotFound,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	ncConfiguration := NCConfiguration{
		NetworkID:           networkID,
		NetworkContainerID:  ncID,
		ReservationID:       ncRequestV1.ReservationID,
		RequestV1:           *ncRequestV1,
		ImplicitReservation: false,
		State:               Prepare,
		Status:              Pending,
		APIVersion:          apiVersion,
		CNSVersion:          0}

	return PersistNCConfiguration(&ncConfiguration, true)
}

// CreateOrUpdateNCRequestV2 - Persists NetworkContainerRequestV2
func CreateOrUpdateNCRequestV2(
	networkID, ncID string,
	ncRequestV2 *api.NetworkContainerRequestV2,
	apiVersion string) *api.JsonErr {
	ncConfig, err := TryQueryNetworkContainer(networkID, ncID)
	if err != nil {
		msg := fmt.Sprintf(
			"Failed when trying to retrieve configuration for network container:%s due to error:%s",
			ncID,
			err)
		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	if ncConfig != nil {
		return nil
	}

	nodeConfig, err := TryQueryNodeConfiguration(ncRequestV2.AssociationInfo.NodeID)
	if err != nil {
		return err
	}

	if nodeConfig == nil {
		msg := fmt.Sprintf(
			"Failed when trying to retrieve configuration for node:%s due to error:%s",
			ncRequestV2.AssociationInfo.NodeID,
			err)
		return &api.JsonErr{
			Code:       api.NodeNotFound,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	networkConfig, err := QueryNetworkConfiguration(networkID)
	if err != nil {
		return err
	}

	if networkConfig.NetworkState != Join {
		msg := fmt.Sprintf(
			"Cannot create network container in network %v as it is not in join state. Current State:%v",
			ncID,
			networkConfig.NetworkState.String())
		return &api.JsonErr{
			Code:       api.NetworkStateInvalid,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
	}

	if networkConfig.Status != Completed {
		msg := fmt.Sprintf(
			"Cannot create network container in network %v as the status of the join operation is not completed. Current State:%v",
			ncID,
			networkConfig.Status.String())
		return &api.JsonErr{
			Code:       api.NetworkStateInvalid,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
	}

	subnetConfig, err := QuerySubnetConfiguration(networkID, ncRequestV2.AllocationRequest.SubnetName)
	if err != nil {
		return err
	}

	if !subnetConfig.Joined {
		msg := fmt.Sprintf(
			"Cannot create network container in network %v subnet %v as it has not completed the join operation.",
			ncID,
			ncRequestV2.AllocationRequest.SubnetName)
		return &api.JsonErr{
			Code:       api.SubnetIsNotJoined,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
	}

	ncConfiguration := NCConfiguration{
		NetworkID:           networkID,
		NetworkContainerID:  ncID,
		RequestV2:           *ncRequestV2,
		ImplicitReservation: true,
		State:               Prepare,
		Status:              Pending,
		APIVersion:          apiVersion,
		CNSVersion:          0}

	return PersistNCConfiguration(&ncConfiguration, true)
}

type NetworkContainerInfo struct {
	NetworkID          string
	SubnetName         string
	ReservationID      string
	AssociationInfo    api.AssociationInfo
	NetworkContainerID string
	ProgrammedTagsInfo map[string]uint64
}

// GetNetworkContainersInSubnet - Get network containers in a subnet
func GetNetworkContainersInSubnet(
	repoSubnetConfiguration *SubnetConfiguration,
	includePendingDeletes bool) ([]*NetworkContainerInfo, *api.JsonErr) {
	networkContainersInfo := make([]*NetworkContainerInfo, len(repoSubnetConfiguration.Reservations))
	containersCount := 0
	for reservationID := range repoSubnetConfiguration.Reservations {
		repoReservation, err := TryQueryIPReservation(repoSubnetConfiguration.NetworkID, reservationID)
		if err != nil {
			return nil, err
		}

		if repoReservation == nil {
			continue
		}

		if len(repoReservation.AssociatedNCID) == 0 {
			continue
		}

		repoNetworkContainerConfiguration, err := TryQueryNetworkContainer(
			repoSubnetConfiguration.NetworkID,
			repoReservation.AssociatedNCID)
		if err != nil {
			return nil, err
		}

		if repoNetworkContainerConfiguration == nil {
			continue
		}

		if !includePendingDeletes && repoNetworkContainerConfiguration.MarkedForDelete {
			continue
		}

		associationInfo := repoNetworkContainerConfiguration.GetAssociationInfo()
		networkContainersInfo[containersCount] = &NetworkContainerInfo{
			NetworkID:          repoSubnetConfiguration.NetworkID,
			SubnetName:         repoSubnetConfiguration.SubnetName,
			ReservationID:      repoReservation.ReservationID,
			NetworkContainerID: repoNetworkContainerConfiguration.NetworkContainerID,
			AssociationInfo:    associationInfo,
			ProgrammedTagsInfo: repoNetworkContainerConfiguration.ProgrammedTagsInfo}

		containersCount++
	}

	return networkContainersInfo[:containersCount], nil
}

// GetAllNetworkIDs - Retrieve all network IDs
func GetAllNetworkIDs() (map[string]bool, error) {
	var hasMoreData bool
	var context interface{}
	var networksMap map[string]interface{}
	var err error
	var status int
	networkIDs := make(map[string]bool)
	for {
		networksMap, hasMoreData, context, status, err = dataStoreShim.QueryAll(
			networkConfigurationTableName,
			context)
		if err != nil {
			if status == http.StatusNotFound {
				return networkIDs, nil
			}

			return nil, err
		}

		for networkKey := range networksMap {
			networkIDs[networkKey] = true
		}

		if !hasMoreData {
			break
		}
	}

	return networkIDs, nil
}

// GetAllNodeIDs - Retrieve all node IDs
func GetAllNodeIDs() (map[string]bool, error) {
	var hasMoreData bool
	var context interface{}
	var nodesMap map[string]interface{}
	var err error
	var status int
	nodeIDs := make(map[string]bool)
	for {
		nodesMap, hasMoreData, context, status, err = dataStoreShim.QueryAll(
			nodeTableName,
			context)
		if err != nil {
			if status == http.StatusNotFound {
				return nodeIDs, nil
			}

			return nil, err
		}

		for nodeID := range nodesMap {
			nodeIDs[nodeID] = true
		}

		if !hasMoreData {
			break
		}
	}

	return nodeIDs, nil
}

// TryQueryAllIPAllocations - Query all IP allocations in the partition.
func TryQueryAllIPAllocations(networkID, subnetName string, context interface{}) (
	map[string]interface{}, bool, interface{}, *api.JsonErr) {
	partitionKey := GetSubnetKey(networkID, subnetName)
	data, hasMoreRows, retContext, statusCode, err :=
		dataStoreShim.QueryAllInPartition(ipAllocationsTableName, partitionKey, context)

	if err != nil && statusCode != http.StatusNotFound {
		msg := fmt.Sprintf(
			"Failed to retrieve IP allocations for partition key:[%s] due to error:[%s]",
			partitionKey,
			err.Error())
		return nil, false, nil, &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	if data == nil {
		return nil, false, nil, nil
	}

	return data, hasMoreRows, retContext, nil
}

// TryQueryIPAllocation - Query for IP allocations
func TryQueryIPAllocation(networkID, subnetName, ipAddress string) (string, *api.JsonErr) {
	partitionKey := GetSubnetKey(networkID, subnetName)
	return TryQueryIPAllocationByKey(partitionKey, ipAddress)
}

// TryQueryIPAllocationByKey - Query for IP allocations by the key in the specified partition
func TryQueryIPAllocationByKey(partitionKey, ipAddress string) (string, *api.JsonErr) {
	value, status, err := dataStoreShim.QueryInPartition(ipAllocationsTableName, partitionKey, ipAddress)
	if err != nil && status != http.StatusNotFound {
		msg := fmt.Sprintf(
			"Failed to retrieve IP allocation for key:[%s] in partition:[%s] due to error:[%s]",
			ipAddress,
			partitionKey,
			err.Error())
		return "", &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	if value == nil {
		return "", nil
	}

	return value.(string), nil
}

// EncodeIPAllocationKey - Encodes the primary key for IP allocations table
func EncodeIPAllocationKey(allocatedSubnet string) string {
	charReplacer := strings.NewReplacer("/", "_")
	return charReplacer.Replace(allocatedSubnet)
}

// DecodeIPAllocationKey - Decodes the primary key for IP allocations table
func DecodeIPAllocationKey(allocatedSubnet string) string {
	charReplacer := strings.NewReplacer("_", "/")
	return charReplacer.Replace(allocatedSubnet)
}

// QueryAllIPReservations - Query all IP Reservation configurations.
func QueryAllIPReservations() ([]ReservationConfiguration, *api.JsonErr) {
	var (
		err                       *api.JsonErr
		reservationConfigurations []ReservationConfiguration
		hasMoreRows               = true
		context                   interface{}
		data                      map[string]interface{}
	)

	for hasMoreRows {
		data, hasMoreRows, context, err = TryQueryAllIPReservations(context)
		if err != nil {
			return nil, err
		}

		for reservationKey, value := range data {
			serializedReservationConfiguration := value.(string)
			var reservationConfiguration ReservationConfiguration
			err := json.Unmarshal([]byte(serializedReservationConfiguration), &reservationConfiguration)
			if err != nil {
				msg := fmt.Sprintf(
					"Failed to unmarshal configuration for reservation key:[%s] due to error:[%s]",
					reservationKey,
					err)
				return nil, &api.JsonErr{
					Code:       api.JsonBodyUnMarshalFailed,
					Text:       msg,
					HTTPStatus: http.StatusInternalServerError}
			}

			reservationConfigurations = append(reservationConfigurations, reservationConfiguration)
		}
	}

	return reservationConfigurations, nil
}

// TryQueryAllIPReservations - Query all IP Reservation configurations in the partition.
func TryQueryAllIPReservations(context interface{}) (
	map[string]interface{}, bool, interface{}, *api.JsonErr) {
	data, hasMoreRows, retContext, statusCode, err := dataStoreShim.QueryAll(ipReservationsTableName, context)

	if err != nil && statusCode != http.StatusNotFound {
		msg := fmt.Sprintf(
			"Failed to retrieve IP reservations due to error:[%s]",
			err.Error())
		return nil, false, nil, &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	if data == nil {
		return nil, false, nil, nil
	}

	return data, hasMoreRows, retContext, nil
}

// ProcessSharedSubnetDelegationMigration - Migrate IP allocations from IPReservations to IPAllocations table
func ProcessSharedSubnetDelegationMigration() error {
	if IsFeatureSupported(configuration.SharedSubnetDelegation) {
		// check if the migration for shared subnet delegation is pending
		migrationComplete, err := TryQuerySharedSubnetDelegationMigration()
		if err != nil {
			loggershim.LS.Errorf("sharedSubnetDelegationMigration table query failed due to error %v", err)
			return fmt.Errorf("Error querying sharedSubnetDelegationMigration table")
		}

		if !migrationComplete {
			loggershim.LS.Debugf("Triggering migration for shared subnet delegation")
			reservationConfigurations, err := QueryAllIPReservations()
			if err != nil {
				loggershim.LS.Errorf("Failed to get all IP reservations due to error %v", err)
				return fmt.Errorf("Failed to get all IP reservations with error %s", err.Text)
			}

			// Migrate IP allocations from IPReservations to IPAllocations table
			for _, reservationConfiguration := range reservationConfigurations {
				reservationKey := GetReservationKey(reservationConfiguration.NetworkID, reservationConfiguration.ReservationID)

				if len(reservationConfiguration.AllocatedSubnet) != 0 {
					if err = AddIPAllocation(reservationConfiguration.NetworkID,
						reservationConfiguration.AllocationRequest.SubnetName,
						EncodeIPAllocationKey(reservationConfiguration.AllocatedSubnet),
						reservationKey); err != nil {
						loggershim.LS.Errorf("addIPAllocation for %v failed due to %s",
							reservationConfiguration.AllocatedSubnet, err.Text)
						if err.Code == api.StatusInternalServerError {
							return fmt.Errorf("Failed to add IP in IPAllocations table with err %s", err.Text)
						}
					}
				}
			}

			if err = UpdateSharedSubnetDelegationMigration(true); err != nil {
				loggershim.LS.Errorf("Failed to update SharedSubnetDelegationMigration table due to error %s", err)
			} else {
				loggershim.LS.Debugf("Migration for shared subnet delegation completed successfully")
			}
		} else {
			loggershim.LS.Debugf("Migration for shared subnet delegation already completed")
		}
	}

	return nil
}

// TryQuerySharedSubnetDelegationMigration - Query migration table in a specified partition.
func TryQuerySharedSubnetDelegationMigration() (bool, *api.JsonErr) {
	value, status, err := dataStoreShim.Query(sharedSubnetDelegationMigrationTableName, "migration-status")
	if err != nil && status != http.StatusNotFound {
		msg := fmt.Sprintf(
			"Failed to retrieve migration status due to error:[%s]", err)
		return false, &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	if value == nil {
		return false, nil
	}

	return value.(bool), nil
}

// UpdateSharedSubnetDelegationMigration - Updates the migration table indicating the completion of the migration operation.
func UpdateSharedSubnetDelegationMigration(completionStatus bool) *api.JsonErr {
	err := dataStoreShim.InsertOrUpdate(sharedSubnetDelegationMigrationTableName,
		"migration-status",
		TableToColumnMap[sharedSubnetDelegationMigrationTableName],
		completionStatus)
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to update SharedSubnetDelegationMigration table. Error:%s",
			err.Error())
		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	return nil
}

// IsFeatureSupported - Function to validate if the given feature is supported
func IsFeatureSupported(featureID int) bool {
	return dataStoreShim.IsFeatureSupported(featureID)
}
