package datamodel

import (
	"bytes"
	"dnc/aiwrapper"
	"dnc/allocationhelper"
	"dnc/api"
	"dnc/configuration"
	"dnc/datastoreshim"
	"dnc/logger/loggershim"
	"dnc/nmagentcontracts"
	"dnc/utils"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-container-networking/aitelemetry"
	"github.com/Azure/azure-container-networking/cns"
	"github.com/Azure/azure-container-networking/cns/logger"
	"github.com/Azure/azure-sdk-for-go/storage"
)

var (
	dataStoreShim             *datastoreshim.DataStoreShim
	cnsSettings               configuration.CNSSettings
	requestBaseInterval              = 1000
	minVlan                   uint16 = 1
	maxVlan                   uint16 = 2048
	maxContainersPerNode             = 10
	httpClient                *http.Client
	defaultDNSServer          string
	enforceNICLimit           bool // zero value is false
	nicsAllowedPerVirtualCore uint16
	managed                   bool
)

// Init - Initialize repository.
func Init(
	cnsConfigSettings configuration.CNSSettings,
	dataStoreSettings configuration.DataStoreSettings,
	dncSettings configuration.DNCSettings,
	min, max uint16,
	httpClientArg *http.Client,
	wireServerIP string) error {
	requestBaseInterval = dncSettings.RequestRetryBaseInterval
	dataStoreShim = datastoreshim.NewDataStoreShim(dataStoreSettings, dncSettings, TableToColumnMap)
	if dataStoreSettings.TestPersistenceStoreInfo.Test != "" {
		RestoreState()
	}

	if min > minVlan {
		minVlan = min
	}

	if max > minVlan {
		maxVlan = max
	}

	if wireServerIP != "" {
		defaultDNSServer = "168.63.129.16"
	}

	cnsSettings = cnsConfigSettings
	httpClient = httpClientArg
	enforceNICLimit = dncSettings.EnforceNICLimit
	nicsAllowedPerVirtualCore = dncSettings.NICsAllowedPerVirtualCore
	managed = dncSettings.Managed

	return nil
}

func getNCLimitFromNumCores(numCPUCores int) int {
	// Total numbers of containers/nics based on number of cores
	// Convert to Float is needed, as Floor takes float datatype inputs
	totalNicsBasedOnCores := float64(nicsAllowedPerVirtualCore * uint16(numCPUCores))
	return int(math.Floor(totalNicsBasedOnCores) - 1) // -1 to account for VM management NIC
}

// DumpStorageState - Function to dump storage state.
func DumpStorageState(partitionKey string) (err error) {
	for tableName := range TableToColumnMap {
		var context interface{}
		var kvMap map[string]interface{}
		var hasMoreRows = true
		var innerErr error
		for hasMoreRows {
			kvMap, hasMoreRows, context, _, innerErr = dataStoreShim.QueryAllInPartition(tableName, partitionKey, context)
			if innerErr == nil {
				jsonStr, _ := json.Marshal(kvMap)
				fileName := tableName + ".db"
				fp, innerErr := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(0664))
				if innerErr == nil {
					fp.Write(jsonStr)
					fp.Close()
				}
			}

			if innerErr != nil {
				err = innerErr
			}
		}
	}

	return err
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
		return &api.JsonErr{
			Code:       api.JsonMarshallingFailed,
			HTTPStatus: http.StatusInternalServerError,
			Text:       fmt.Sprintf("%s: Failed to serialize subnet configuration. SubnetKey:%s , Error: %s", utils.TracePrefix(2), subnetKey, err.Error())}
	}

	err = dataStoreShim.InsertOrUpdate(subnetConfigurationTableName, subnetKey, TableToColumnMap[subnetConfigurationTableName], string(jsonSerializedBytes[:]))
	if err != nil {
		errorCode := api.StatusInternalServerError

		// Special case for the property value too large storage error
		if strings.Contains(err.Error(), "ErrorCode=PropertyValueTooLarge") {
			errorCode = api.PropertyValueTooLargeStorageError
		}

		msg := fmt.Sprintf(
			"Failed to persist updated configuration for subnet key %s. Error:%s",
			subnetKey,
			err.Error())

		return &api.JsonErr{
			Code:       errorCode,
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
			"Failed to delete subnetkey %s from data store. Error: %s",
			subnetKey,
			deleteErr.Error())
		loggershim.LS.Errorf("%s: %s", utils.TracePrefix(2), msg)
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
		return &api.JsonErr{
			Code:       api.JsonMarshallingFailed,
			HTTPStatus: http.StatusInternalServerError,
			Text:       fmt.Sprintf("Failed to serialize node configuration for reservation:%s due to error %s", ipReservation.String(), err.Error())}
	}

	err = dataStoreShim.InsertOrUpdate(ipReservationsTableName,
		reservationKey,
		TableToColumnMap[ipReservationsTableName],
		string(jsonSerializedBytes[:]))
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to persist updated configuration for reservation %s due to error: %s",
			ipReservation.ReservationID,
			err.Error())
		loggershim.LS.Errorf("%s: %s", utils.TracePrefix(2), msg)
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

	defer func() {
		if err != nil {
			loggershim.LS.Errorf("ValidateAndUpdateIPReservationDependencies failed. Error: %v", err)
		}
	}()

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
		err = &api.JsonErr{
			Code:       api.StatusInternalServerError,
			HTTPStatus: http.StatusInternalServerError,
			Text:       msg}
		return err
	}

	if reservationConfiguration.AllocationRequest.IPConstraint != "" {
		ipSubnetConstraint, err := allocationhelper.GetIPSubnetForIPAllocationConstraint(
			reservationConfiguration.AllocationRequest.IPConstraint)
		if err != nil {
			return err
		}

		_, err = allocationhelper.IsSubnetContainedInBaseSubnet(ipSubnet, ipSubnetConstraint)
		if err != nil {
			loggershim.LS.Errorf("%s: %s.", utils.TracePrefix(2), err.String())
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
				err = &api.JsonErr{
					Code:       api.NodeConstraintMismatch,
					Text:       msg,
					HTTPStatus: http.StatusBadRequest}
				return err
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

// GetNextVlanID - Generate new Vlan ID
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
	nodeID, networkID, ncID, reservationID, containerType string,
	updateVlan, remove bool,
	ncLimit int) *api.JsonErr {
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
		loggershim.LS.Errorf("Failed to delete IP allocation due to error %+v", err)
		return err
	}

	if len(reservationConfiguration.AssociatedNCID) != 0 && !force {
		msg := fmt.Sprintf(
			"Reservation %s is in use with container: [%s]",
			reservationID, reservationConfiguration.AssociatedNCID)

		loggershim.LS.Errorf("%s: %s", utils.TracePrefix(2), msg)

		return &api.JsonErr{
			Code:       api.ReservationInUse,
			HTTPStatus: http.StatusBadRequest,
			Text:       msg}
	}

	subnetConfiguration, err := TryQuerySubnetConfiguration(
		reservationConfiguration.NetworkID,
		reservationConfiguration.AllocationRequest.SubnetName)
	if err != nil {
		return err
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
		loggershim.LS.Errorf("%s: %s", utils.TracePrefix(2), msg)
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
		allocatedIPKey := EncodeIPAllocationKey(allocatedIP)
		savedReservationKey, err := TryQueryIPAllocation(networkID, subnetName, allocatedIPKey)
		if err != nil {
			return err
		}

		if len(savedReservationKey) > 0 {
			// Delete the entry if the saved reservation matches the current reservation
			if reservationKey == savedReservationKey {
				partitionKey := GetSubnetKey(networkID, subnetName)
				err := dataStoreShim.DeleteInPartition(ipAllocationsTableName, partitionKey, allocatedIPKey)
				if err != nil {
					return &api.JsonErr{
						Code:       api.StatusInternalServerError,
						HTTPStatus: http.StatusInternalServerError,
						Text:       fmt.Sprintf("Cannot delete entry %s from IPAllocations table due to error: %+v", allocatedIPKey, err)}
				}
			} else {
				return &api.JsonErr{
					Code:       api.InvalidIPSubnet,
					HTTPStatus: http.StatusInternalServerError,
					Text:       fmt.Sprintf("Reservation key mismatch in IPAllocations table for IP %s", allocatedIP)}
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
			jsonErr.Text = fmt.Sprintf("IP %+v is unavailable for allocation for key: %s", allocatedIP, reservationKey)
		} else {
			jsonErr.Code = api.StatusInternalServerError
			jsonErr.HTTPStatus = http.StatusInternalServerError
			jsonErr.Text = fmt.Sprintf("Internal server error")
		}
		return jsonErr
	}

	loggershim.LS.Debugf("Allocated IP: %s for reservation key: %s", allocatedIP, reservationKey)
	return nil
}

// AddIPAllocations adds a batch of allocations to the specified vnet and subnet name partition in IPAllocationsTable
func AddIPAllocations(networkID, subnetName string, rows map[string]map[string]interface{}) *api.JsonErr {
	partitionKey := GetSubnetKey(networkID, subnetName)

	if err := dataStoreShim.InsertBatch(ipAllocationsTableName, partitionKey, rows); err != nil {
		storageErr, ok := err.(storage.AzureStorageServiceError)
		if !ok {
			return api.InternalServerError(api.StatusInternalServerError, err.Error())
		}

		switch storageErr.StatusCode {
		case http.StatusConflict:
			return api.Conflict(api.AllocationFailed, storageErr.Message)
		default:
			return api.InternalServerError(api.StatusInternalServerError, storageErr.Message)
		}
	}

	return nil
}

func DeleteIPAllocations(networkID, subnetName string, keys []string) *api.JsonErr {
	partitionKey := GetSubnetKey(networkID, subnetName)

	if err := dataStoreShim.DeleteBatch(ipAllocationsTableName, partitionKey, keys); err != nil {
		return api.InternalServerError(api.StatusInternalServerError, err.Error())
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

// TryQueryAllNetworks queries for all networkconfigurations managed by this dnc and returns their IDs
func TryQueryAllNetworks() ([]string, error) {
	var hasMoreData bool
	var result map[string]interface{}
	var returnContext interface{}
	var status int
	var err error
	var networkGUIDs = make([]string, 0)

	for {
		result, hasMoreData, returnContext, status, err =
			dataStoreShim.QueryAll(networkConfigurationTableName, returnContext)
		if err != nil {
			if status == http.StatusNotFound {
				return make([]string, 0), nil
			}

			return nil, err
		}

		for _, value := range result {
			serializedNetworkConfiguration := value.(string)
			var networkConfiguration NetworkConfiguration
			err = json.Unmarshal([]byte(serializedNetworkConfiguration), &networkConfiguration)
			if err != nil {
				return nil, err
			}
			networkGUIDs = append(networkGUIDs, networkConfiguration.NetworkID)
		}

		if !hasMoreData {
			break
		}
	}

	return networkGUIDs, nil
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
			HTTPStatus: http.StatusNotFound}
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
			HTTPStatus: http.StatusNotFound}
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
			HTTPStatus: http.StatusNotFound}
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
			HTTPStatus: http.StatusNotFound}
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

// CreateOrUpdateNodeConfiguration - Creates Or Updates a node configuration. Register should be true if called from RegisterNode.
func CreateOrUpdateNodeConfiguration(
	register bool,
	cores int,
	nodeID string,
	nodeInfo api.NodeInfoRequest,
	apiVersion string) *api.JsonErr {
	repositoryNodeConfiguration, err := TryQueryNodeConfiguration(nodeID)
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

	return persistNodeConfiguration(repositoryNodeConfiguration)
}

func persistNodeConfiguration(nodeconfiguration *NodeConfiguration) *api.JsonErr {
	jsonSerializedBytes, err := json.Marshal(nodeconfiguration)
	if err != nil {
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
	deleteErr := dataStoreShim.Delete(nodeTableName, nodeKey)
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
		loggershim.LS.Errorf("%s: %s", utils.TracePrefix(2), msg)
		return requestReadErr
	}

	if repoRequestContext != nil {
		requestContext.CreatedDateTime = repoRequestContext.CreatedDateTime
	} else {
		requestContext.CreatedDateTime = time.Now().UTC()
		requestContext.RetryInterval = time.Duration(requestBaseInterval) * time.Millisecond
		requestContext.StateTransitionDateTime = requestContext.CreatedDateTime
	}

	jsonSerializedBytes, err := json.Marshal(requestContext)
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to serialize request context. Key:%s RequestContext:%s , Error: %s",
			requestKey,
			requestContext.String(),
			err.Error())
		loggershim.LS.Errorf("%s: %s", utils.TracePrefix(2), msg)

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
		loggershim.LS.Errorf("%s: %s", utils.TracePrefix(2), msg)
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
		loggershim.LS.Errorf("Failed to query pending work item queue due to error: %s", err.Error())
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
		loggershim.LS.Errorf("Failed to qunmarshal requestdue to error: %s", err.Error())
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
				loggershim.LS.Errorf("Unmarshalling tag defintition %s failed with error %s", tagDefinition.String(), err.Error())
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
		loggershim.LS.Errorf("%s: %s", utils.TracePrefix(2), msg)

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
		loggershim.LS.Errorf("%s: %s", utils.TracePrefix(2), msg)

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
		loggershim.LS.Errorf("%s: %s", utils.TracePrefix(2), msg)
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
		return nil, err
	}

	if networkContainerConfig == nil {
		msg := fmt.Sprintf("Network container %s not found", networkContainerID)
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

	case AllocateMAC:
		return UnReserve, nil // TODO: introduce ned state for unreserve MAC address.

	case Associate:
		return DisAssociate, nil

	case RefreshAssociation:
		fallthrough
	case Publish:
		fallthrough
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
	loggershim.LS.Debugf("ProcessNetworkContainerForRelease: %s", networkContainerID)
	networkContainerConfig, err := TryQueryNetworkContainer(
		networkID, networkContainerID)
	if err != nil {
		return err
	}

	if networkContainerConfig == nil {
		return nil
	}

	if !networkContainerConfig.MarkedForDelete || networkContainerConfig.Status == Failed {
		networkContainerConfig.MarkedForDelete = true
		networkContainerConfig.Status = Pending
		networkContainerConfig.FailureDetail = api.JsonErr{}
		// only retrieve the revert state if NC is not in any of the delete states
		if !IsDeleteInProgress(networkContainerConfig.State) {
			revertState, err := GetRevertState(networkContainerConfig.State)
			if err != nil {
				return err
			}

			loggershim.LS.Debugf("NC %s transitioning from state %s to %s",
				networkContainerConfig.NetworkContainerID,
				networkContainerConfig.State, revertState)
			networkContainerConfig.State = revertState

			// This is intended to be a new request for deletion, any existing RequestContext for this NC in
			// the pending queue needs to be deleted.
			// It's possible to have a RequestContext in pending queue belonging to refreshAssociation when
			// deleteNC is triggered. This results in using the incorrect timestamps causing the unpublish
			// and in turn the complete deletion time calculation to be erroneous.
			requestID := GetNCKey(networkContainerConfig.NetworkID, networkContainerConfig.NetworkContainerID)
			RemovePendingRequest(requestID)
		}

		return PersistNCConfiguration(networkContainerConfig, AddToPendingQ)
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
		loggershim.LS.Errorf("%s: %s", utils.TracePrefix(2), msg)

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
	apiVersion string,
	addToPendingQueue bool) *api.JsonErr {
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

	if addToPendingQueue {
		return AddRequestToPendingQueue(networkKey, &requestContext)
	}

	return nil
}

// UpdateSubnet - Update subnet configuration
func UpdateSubnet(
	networkID,
	subnet string,
	subnetRequest api.SubnetRequest,
	isDelete bool) *api.JsonErr {
	var err *api.JsonErr
	var networkConfig *NetworkConfiguration
	defer func() {
		if err != nil {
			loggershim.LS.Errorf("UpdateSubnet failed. Error %v", err)
		}
	}()
	networkConfig, err = TryQueryNetworkConfiguration(networkID)
	if err != nil {
		return err
	}

	if networkConfig == nil {
		msg := fmt.Sprintf("Network with ID %s not found", networkID)
		err = &api.JsonErr{
			Code:       api.NetworkNotFound,
			HTTPStatus: http.StatusNotFound,
			Text:       msg}
		return err
	}

	if networkConfig.NetworkState == Join && networkConfig.Status != Completed {
		msg := fmt.Sprintf("Network with ID %s join is not completed. Current Status: %s",
			networkID,
			networkConfig.Status.String())
		err = &api.JsonErr{
			Code:       api.NetworkStatusInvalid,
			HTTPStatus: http.StatusBadRequest,
			Text:       msg}
		return err
	}

	var subnetFound = true
	if networkConfig.Subnets == nil || len(networkConfig.Subnets) == 0 {
		subnetFound = false
	} else if _, ok := networkConfig.Subnets[subnet]; !ok {
		subnetFound = false
	}

	if !subnetFound {
		loggershim.LS.Debugf("Subnet %s not found in Network with ID %s. Refreshing network", subnet, networkID)
		networkConfig.Status = Refreshing
		networkConfig.Subnets[subnet] = GetSubnetKey(networkID, subnet)
		if err = PersistNetworkConfiguration(networkConfig); err != nil {
			return err
		}

		subnetConfig := SubnetConfiguration{
			NetworkID:    networkID,
			SubnetName:   subnet,
			AccessToken:  subnetRequest.AuthenticationToken,
			Joined:       true,
			UpdateStatus: SubnetUpdatePending}
		if failureDetail := PutSubnetConfiguration(&subnetConfig); failureDetail != nil {
			msg := fmt.Sprintf("Subnet %s not found in Network with ID %s. Failure to insert subnet in config.",
				subnet,
				networkID)
			err = &api.JsonErr{
				Code:       api.SubnetNotFound,
				HTTPStatus: http.StatusBadRequest,
				Text:       msg}
			return err
		}

		requestContext := RequestContext{
			Type:      VirtualNetworkRequest,
			NetworkID: networkID}

		err = AddRequestToPendingQueue(GetNetworkKey(networkID), &requestContext)

		return err
	}

	subnetConfiguration, err := TryQuerySubnetConfiguration(networkID, subnet)
	if err != nil {
		return err
	}

	if subnetConfiguration == nil {
		msg := fmt.Sprintf("Subnet %s configuration not found. Network with ID %s",
			subnet,
			networkID)
		err = &api.JsonErr{
			Code:       api.SubnetConfigurationNotFound,
			HTTPStatus: http.StatusBadRequest,
			Text:       msg}
		return err
	}

	if subnetConfiguration.UpdateStatus == SubnetUpdatePending {
		// No error is returned as the request is treated as idempotent while
		// the same request issued earlier is in process
		return nil
	} else if subnetConfiguration.UpdateStatus == SubnetUpdateFailed {
		msg := fmt.Sprintf("Network %s does not contain the specified subnet %s",
			networkID,
			subnet)
		err = &api.JsonErr{
			Code:       api.SubnetNotFound,
			HTTPStatus: http.StatusNotFound,
			Text:       msg}
		return err
	}

	if !subnetConfiguration.IsDelegatedSubnet {
		msg := fmt.Sprintf("Subnet %s is not a delegated subnet. Network with ID %s",
			subnet,
			networkID)
		err = &api.JsonErr{
			Code:       api.SubnetIsNotDelegated,
			HTTPStatus: http.StatusBadRequest,
			Text:       msg}
		return err
	}

	if isDelete && len(subnetConfiguration.Reservations) > 0 {
		msg := fmt.Sprintf("Cannot remove authentication token while subnet %s has allocations", subnet)
		err = &api.JsonErr{
			Code:       api.InvalidSubnetRequest,
			HTTPStatus: http.StatusBadRequest,
			Text:       msg}
		return err
	}

	subnetConfiguration.Joined = !isDelete
	subnetConfiguration.AccessToken = subnetRequest.AuthenticationToken
	err = PutSubnetConfiguration(subnetConfiguration)

	return err
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
	associationInfo := ncConfiguration.GetAssociationInfo()
	err := UpdateNodeConfigurationNetworkContainerSet(
		associationInfo.NodeID,
		ncConfiguration.NetworkID,
		ncConfiguration.NetworkContainerID,
		ncConfiguration.ReservationID,
		associationInfo.ContainerType,
		updateVlan,
		true,
		0)

	return err
}

// GetNetworks = Retrieve vnets managed by this dnc
func GetNetworks() (*api.NetworksResponse, *api.JsonErr) {
	var networksResponse api.NetworksResponse
	networksResponse.NetworkInfos = make([]api.NetworkInfo, 0)
	networkIDs, err := TryQueryAllNetworks()
	if err != nil {
		msg := fmt.Sprintf("Failed to retrieve networks due to error:[%s]", err)
		return nil, &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError,
		}
	}

	for _, networkID := range networkIDs {
		networksResponse.NetworkInfos = append(networksResponse.NetworkInfos, api.NetworkInfo{GUID: networkID})
	}

	return &networksResponse, nil
}

// GetNetworkConfiguration = Retrieve network configuration (e.g. delegated subnets in a network)
func GetNetworkConfiguration(networkID string) (*api.NetworkConfigurationResponse, *api.JsonErr) {
	var networkConfigurationResponse api.NetworkConfigurationResponse
	networkConfiguration, err := TryQueryNetworkConfiguration(networkID)
	if err != nil {
		return nil, err
	}

	if networkConfiguration == nil {
		return nil, &api.JsonErr{
			Code:       api.NetworkNotFound,
			HTTPStatus: http.StatusNotFound,
			Text:       fmt.Sprintf("Network %s not found", networkID)}
	}

	networkConfigurationResponse.DelegatedSubnets = make([]string, 0)
	for subnet := range networkConfiguration.Subnets {
		subnetConfiguration, err := TryQuerySubnetConfiguration(networkID, subnet)
		if err != nil {
			return nil, err
		}

		// Ignore the cases where there is a mismatch in Network and Subnet tables
		// Refresh network will handle/cleanup these cases
		if subnetConfiguration == nil {
			continue
		}

		if len(subnetConfiguration.AccessToken) > 0 {
			networkConfigurationResponse.DelegatedSubnets = append(networkConfigurationResponse.DelegatedSubnets, subnet)
		}
	}

	return &networkConfigurationResponse, nil
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

// GetSubnetDNSServers will query network config and return dns servers.
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
			HTTPStatus: http.StatusNotFound,
			Text:       msg}
	}

	return GetDNSServers(repoNetworkConfiguration, repoSubnetConfiguration)
}

// GetDNSServers - VNET config published by RNM to pubsub has empty DNS Servers for non-custom cases, and customer provided addresses for the custom cases
func GetDNSServers(repoNetworkConfiguration *NetworkConfiguration, repoSubnetConfiguration *SubnetConfiguration) ([]string, *api.JsonErr) {
	var dnsServers []string
	if repoNetworkConfiguration.DNSServers != nil &&
		len(repoNetworkConfiguration.DNSServers) != 0 {
		dnsServers = repoNetworkConfiguration.DNSServers
	} else {
		dnsServers = []string{defaultDNSServer}
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
		return nil, &api.JsonErr{
			Code:       api.SubnetNotFound,
			HTTPStatus: http.StatusBadRequest,
			Text:       msg}
	}

	dnsServers, err := GetSubnetDNSServers(repoSubnetConfiguration)
	if err != nil {
		return nil, err
	}

	secondaryIPs := make(map[string]api.IPInfo)
	for reservationID, ipAddress := range repoReservationConfiguration.SecondaryAllocations {
		secondaryIPs[reservationID] = api.IPInfo{IPAddress: ipAddress}
	}

	ipConfiguration := api.IPConfiguration{
		NetworkID:    repoReservationConfiguration.NetworkID,
		SubnetName:   repoReservationConfiguration.AllocationRequest.SubnetName,
		IPSubnet:     repoSubnetConfiguration.IPSubnet,
		IPAddress:    repoReservationConfiguration.AllocatedSubnet,
		GatewayIP:    repoSubnetConfiguration.Gateway,
		DNSServers:   dnsServers,
		SecondaryIPs: secondaryIPs,
	}

	return &ipConfiguration, nil
}

// GetNetworkContainerResponse - Get network container response for network config.
func GetNetworkContainerResponse(
	ncConfig *NCConfiguration) (*api.NetworkContainerResponse, *api.JsonErr) {
	var (
		err                      *api.JsonErr
		ipConfiguration          *api.IPConfiguration
		nodeConfiguration        *NodeConfiguration
		reservationConfiguration *ReservationConfiguration
	)

	if ncConfig == nil {
		return nil, &api.JsonErr{
			Code:       api.StatusInternalServerError,
			HTTPStatus: http.StatusInternalServerError,
			Text:       "GetNetworkContainerResponse on nil NCConfiguration"}
	}

	reservationConfiguration, err = QueryIPReservation(
		ncConfig.NetworkID,
		ncConfig.ReservationID)
	if err != nil {
		loggershim.LS.Errorf("Failed to get IP reservation in GetNetworkContainerResponse for NC: %s,"+
			" NC state: %s. Error: %v", ncConfig.NetworkContainerID, ncConfig.State.String(), err)
	}

	if reservationConfiguration != nil {
		ipConfiguration, err = getIPConfiguration(reservationConfiguration)
		if err != nil {
			loggershim.LS.Errorf("Failed to get IP configuration for reservation: %s "+
				"in GetNetworkContainerResponse for NC: %s. Error: %v",
				ncConfig.ReservationID, ncConfig.NetworkContainerID, err)
		}
	}

	nodeConfiguration, err = QueryNodeConfiguration(ncConfig.GetAssociationInfo().NodeID)
	if err != nil {
		loggershim.LS.Errorf("Failed to get node configuration for node: %s in GetNetworkContainerResponse."+
			"for NC: %s. Error: %v", ncConfig.GetAssociationInfo().NodeID, ncConfig.NetworkContainerID, err)
	}

	var vlanID uint16
	if nodeConfiguration == nil {
		vlanID = 0
	} else {
		vlanID = nodeConfiguration.NetworkVlanIDMap[ncConfig.NetworkID]
	}

	requestStatus := api.RequestStatus{
		State:         ncConfig.State.String(),
		Status:        ncConfig.Status.String(),
		FailureDetail: ncConfig.FailureDetail}
	associationInfo := ncConfig.GetAssociationInfo()
	response := api.NetworkContainerResponse{
		NcID:                       ncConfig.NetworkContainerID,
		AssociationInfo:            associationInfo,
		RequestStatus:              requestStatus,
		VlanID:                     vlanID,
		LocalIP:                    ncConfig.LocalIP,
		AllowHostToNCCommunication: ncConfig.GetAllowHostToNCCommunicationStatus(),
		AllowNCToHostCommunication: ncConfig.GetAllowNCToHostCommunicationStatus()}

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

// getOrchestratorContextList - Returns the list of orchestrator contexts for all the NCs for a given node.
func getOrchestratorContextListFromNodeConfig(nodeConfig *NodeConfiguration) (map[string][]byte, *api.JsonErr) {
	orchestratorContextList := make(map[string][]byte)

	for networkID, ncMap := range nodeConfig.DeployedNetworkContainerMap {
		for ncID := range ncMap {
			ncConfig, err := GetNetworkContainerConfiguration(networkID, ncID)
			if err == nil && ncConfig != nil {
				orchestratorContextList[ncConfig.NetworkContainerID] = ncConfig.GetAssociationInfo().OrchestratorContext
			}
		}
	}

	return orchestratorContextList, nil
}

func validateNodeTypePlacement(containerType, nodeType string) *api.JsonErr {
	if api.IsBareMetalNode(nodeType) != api.IsBareMetalContainer(containerType) {
		return &api.JsonErr{
			Code:       api.InvalidContainerPlacement,
			Text:       fmt.Sprintf("Invalid Node for the Container Type"),
			HTTPStatus: http.StatusBadRequest}
	}

	return nil
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

func sendNCCreationReqMetric(networkID, ncID, status string, associationInfo api.AssociationInfo) {
	metric := aitelemetry.Metric{
		Name: aiwrapper.NCCreationReqCountMetric,
		// Value 1.0 is to track number of NC Create Requests
		Value:            1.0,
		CustomDimensions: make(map[string]string),
	}

	metric.CustomDimensions[aiwrapper.NetworkIDStr] = networkID
	metric.CustomDimensions[aiwrapper.NetworkContainerIDStr] = ncID
	metric.CustomDimensions[aiwrapper.StatusStr] = status
	metric.CustomDimensions[aiwrapper.NodeIDStr] = associationInfo.NodeID
	metric.CustomDimensions[aiwrapper.ContainerTypeStr] = associationInfo.ContainerType

	go aiwrapper.SendMetric(metric)
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

	defer func() {
		var status string
		if err != nil {
			status = aiwrapper.Invalid
			loggershim.LS.Errorf("CreateOrUpdateNCRequestV1 failed. Error: %v", err)
		} else {
			status = aiwrapper.Valid
		}
		// Metric for getting number of NC Creation Requests
		sendNCCreationReqMetric(networkID, ncID, status, ncRequestV1.AssociationInfo)
	}()

	nodeID := ncRequestV1.AssociationInfo.NodeID
	repoNodeConfiguration, err := TryQueryNodeConfiguration(nodeID)
	if err != nil {
		return err
	}

	if repoNodeConfiguration == nil {
		return &api.JsonErr{
			Code:       api.NodeNotFound,
			Text:       fmt.Sprintf("Node %s not found", nodeID),
			HTTPStatus: http.StatusNotFound}
	}

	err = validateNodeTypePlacement(
		ncRequestV1.AssociationInfo.ContainerType,
		repoNodeConfiguration.NodeInfo.NodeType)
	if err != nil {
		return err
	}

	var orchestratorContextList map[string][]byte
	orchestratorContextList, err = getOrchestratorContextListFromNodeConfig(repoNodeConfiguration)
	if err != nil {
		return err
	}

	// Marshal the orchestrator context for the incoming request. This is needed to compare with the
	// existing orchestrator contexts because when saving the orchestrator context to NCConfig, it
	// gets marshaled.
	marshaledOrchestratorContext, _ := json.Marshal(ncRequestV1.AssociationInfo.OrchestratorContext)

	for ncid, orchestratorContext := range orchestratorContextList {
		if bytes.Compare(orchestratorContext, marshaledOrchestratorContext) == 0 {
			err = &api.JsonErr{
				Code:       api.StatusBadRequest,
				Text:       fmt.Sprintf("Illegal create nc operation due to duplicate orchestrator context with nc: %s", ncid),
				HTTPStatus: http.StatusBadRequest}
			return err
		}
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

	err = PersistNCConfiguration(&ncConfiguration, AddToPendingQ)
	return err
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
		return UpdateNCRequestV2(*ncConfig, *ncRequestV2)
	}

	defer func() {
		var status string
		if err != nil {
			status = aiwrapper.Invalid
			loggershim.LS.Errorf("CreateOrUpdateNCRequestV2 failed. Error: %v", err)
		} else {
			status = aiwrapper.Valid
		}
		// Metric for getting number of NC Creation Requests
		sendNCCreationReqMetric(networkID, ncID, status, ncRequestV2.AssociationInfo)
	}()

	nodeID := ncRequestV2.AssociationInfo.NodeID
	repoNodeConfiguration, err := TryQueryNodeConfiguration(nodeID)
	if err != nil {
		return err
	}

	if repoNodeConfiguration == nil {
		return &api.JsonErr{
			Code:       api.NodeNotFound,
			Text:       fmt.Sprintf("Node %s not found", nodeID),
			HTTPStatus: http.StatusNotFound}
	}

	err = validateNodeTypePlacement(
		ncRequestV2.AssociationInfo.ContainerType,
		repoNodeConfiguration.NodeInfo.NodeType)
	if err != nil {
		return err
	}

	var orchestratorContextList map[string][]byte
	orchestratorContextList, err = getOrchestratorContextListFromNodeConfig(repoNodeConfiguration)
	if err != nil {
		return err
	}

	// Marshal the orchestrator context for the incoming request. This is needed to compare with the
	// existing orchestrator contexts because when saving the orchestrator context to NCConfig, it
	// gets marshaled.
	marshaledOrchestratorContext, _ := json.Marshal(ncRequestV2.AssociationInfo.OrchestratorContext)

	for ncid, orchestratorContext := range orchestratorContextList {
		if bytes.Compare(orchestratorContext, marshaledOrchestratorContext) == 0 {
			err = &api.JsonErr{
				Code:       api.StatusBadRequest,
				Text:       fmt.Sprintf("Illegal create nc operation due to duplicate orchestrator context with nc: %s", ncid),
				HTTPStatus: http.StatusBadRequest}
			return err
		}
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
		err = &api.JsonErr{
			Code:       api.NetworkStateInvalid,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
		return err
	}

	if networkConfig.Status != Completed {
		msg := fmt.Sprintf(
			"Cannot create network container in network %v as the status of the join operation is not completed. Current State:%v",
			ncID,
			networkConfig.Status.String())
		err = &api.JsonErr{
			Code:       api.NetworkStateInvalid,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
		return err
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
		err = &api.JsonErr{
			Code:       api.SubnetIsNotJoined,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
		return err
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

	err = PersistNCConfiguration(&ncConfiguration, AddToPendingQ)
	return err
}

func UpdateNCRequestV2(nc NCConfiguration, request api.NetworkContainerRequestV2) *api.JsonErr {
	// It is possible that a previous NC request is already going through processing as an update is received
	// We must wait for the processing to be complete (or failed) before we queue up another item, otherwise, we risk a
	// race condition in which the updated records are persisted before current processing ends, and then
	// at the end of the initial processing, the pending queue item is delete, without having processed the update.
	if !(nc.Status == Completed || nc.Status == Failed) {
		msg := fmt.Sprintf("Network container id %s currently being processed in state %v. Cannot update while processing", nc.NetworkContainerID, nc.State.String())
		// 409 Conflict may not be the most accurate error for this condition, but should signal the client to retry the request later
		return api.Conflict(api.ProcessingPending, msg)
	}

	handling := getHandleLevel(nc.RequestV2, request)
	switch handling {
	case noOp:
		loggershim.LS.Debugf("update request for nc id %s received, nothing to update", nc.NetworkContainerID)
		return nil
	case persistWithoutQueueing:
		nc.RequestV2 = request
		return PersistNCConfiguration(&nc, DontAddToPendingQ)
	case process:
		nc.State = Prepare
		nc.Status = Pending
		nc.CNSVersion++
		nc.RequestV2 = request
		nc.MarkedForDelete = false // todo: handle this scenario when implementing deletes
		// both nc and pending item operations are forced upserts
		// nc lock should have been taken upstream, so this is safe.
		return PersistNCConfiguration(&nc, AddToPendingQ)
	default:
		return api.InternalServerError(api.StatusInternalServerError, fmt.Sprintf("unknown handling level %d", handling))
	}
}

type requestHandling int

const (
	noOp requestHandling = iota
	persistWithoutQueueing
	process
)

// We will only process updates when there are differences between existing nc state and nc request
// that should result in changes to IP allocations
// Persist if requests are different
func getHandleLevel(processed, requested api.NetworkContainerRequestV2) requestHandling {
	if requested.AllocationRequest.SecondaryIPCount != processed.AllocationRequest.SecondaryIPCount {
		return process
	}

	processedDeletions := make(map[string]struct{})
	for _, ipID := range processed.ToBeDeletedIPIds {
		processedDeletions[ipID] = struct{}{}
	}

	requestedDeletions := make(map[string]struct{})
	for _, ipID := range requested.ToBeDeletedIPIds {
		if _, ok := processedDeletions[ipID]; !ok {
			// a requested deletion has not been previously processed
			return process
		}
		requestedDeletions[ipID] = struct{}{}
	}

	// all requested ids have been processed, and thus are a subset of processed ids
	// if the requested deletion set is smaller, we should still persist it
	if len(requestedDeletions) < len(processedDeletions) {
		return persistWithoutQueueing
	}

	return noOp
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

	switch v := value.(type) {
	case string:
		return v, nil
	case map[string]interface{}:
		return v[ipAllocationTag].(string), nil
	default:
		return "", nil
	}
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
	b, ok := value.(bool)
	if !ok {
		return false, nil
	}
	return b, nil
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

// GetAllNetworkContainers - Retrieve all network containers
func GetAllNetworkContainers() (map[string]NCConfiguration, error) {
	var hasMoreData bool
	var context interface{}
	var networkConfigs map[string]interface{}
	var err error
	var status int
	ncConfigs := make(map[string]NCConfiguration)
	for {
		networkConfigs, hasMoreData, context, status, err = dataStoreShim.QueryAll(
			networkContainerTableName,
			context)
		if err != nil {
			if status == http.StatusNotFound {
				return ncConfigs, nil
			}

			return nil, err
		}

		for ncKey, value := range networkConfigs {
			serializedNCConfig := value.(string)
			var ncConfig NCConfiguration
			err = json.Unmarshal([]byte(serializedNCConfig), &ncConfig)
			if err != nil {
				return nil, fmt.Errorf("Unmarshalling nc config %s failed with error %s", serializedNCConfig, err.Error())
			}

			ncConfigs[ncKey] = ncConfig
		}

		if !hasMoreData {
			break
		}
	}

	return ncConfigs, nil
}

// GetNCGoalState - Get NC goal state from different tables.
// Return the following.
// 1. Constructed NC goal state
// 2. If there was an error, True if the error is retriable , False otherwise.
// 3. Error.
func GetNCGoalState(repoNCConfig *NCConfiguration) (*NCGoalState, bool, *api.JsonErr) {
	nodeID := repoNCConfig.GetAssociationInfo().NodeID
	repoReservationConfiguration, err := TryQueryIPReservation(repoNCConfig.NetworkID, repoNCConfig.ReservationID)
	if err != nil {
		return nil, true, err
	} else if repoReservationConfiguration == nil {
		msg := fmt.Sprintf(
			"IPReservation for Id %s associated with Network Container %s in Network %s not found.",
			repoNCConfig.ReservationID,
			repoNCConfig.NetworkContainerID,
			repoNCConfig.NetworkID)

		return nil, false, &api.JsonErr{
			Code:       api.ReservationNotFound,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
	}

	// Query NodeConfiguration table with nodeid and handle errors.
	repoNodeConfig, err := TryQueryNodeConfiguration(nodeID)
	if err != nil {
		return nil, true, err
	} else if repoNodeConfig == nil {
		msg := fmt.Sprintf(
			"Configuration for Node %v associated with the NetworkContainer %v not found.",
			nodeID,
			repoNCConfig.NetworkContainerID)

		return nil, false, &api.JsonErr{
			Code:       api.NodeNotFound,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
	}

	repoSubnetConfiguration, err := TryQuerySubnetConfiguration(
		repoNCConfig.NetworkID,
		repoReservationConfiguration.AllocationRequest.SubnetName)
	if err != nil {
		return nil, true, err
	} else if repoSubnetConfiguration == nil {
		msg := fmt.Sprintf(
			"Configuration for Subnet %v in Network %v not found.",
			repoReservationConfiguration.AllocationRequest.SubnetName,
			repoNCConfig.NetworkID)
		return nil, false, &api.JsonErr{
			Code:       api.SubnetNotFound,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
	}

	if repoSubnetConfiguration.Tags == nil {
		repoSubnetConfiguration.Tags = make([]nmagentcontracts.Tag, 0)
	}

	repoNetworkConfiguration, err := TryQueryNetworkConfiguration(repoNCConfig.NetworkID)
	if err != nil {
		return nil, true, err
	} else if repoNetworkConfiguration == nil {
		msg := fmt.Sprintf(
			"Configuration for Network %v not found.",
			repoNCConfig.NetworkID)
		return nil, false, &api.JsonErr{
			Code:       api.NetworkNotFound,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
	}

	dnsServers, err := GetDNSServers(repoNetworkConfiguration, repoSubnetConfiguration)
	var associatedInterfaceID string
	assocInfo := repoNCConfig.GetAssociationInfo()

	if len(assocInfo.InterfaceID) == 0 {
		associatedInterfaceID = repoNodeConfig.NodeInfo.IPAddresses[0]
	} else {
		associatedInterfaceID = assocInfo.InterfaceID
	}

	secondaryIPs := make([]string, 0)
	for _, ipAddress := range repoReservationConfiguration.SecondaryAllocations {
		if ipAddress != "" {
			secondaryIPs = append(secondaryIPs, ipAddress)
		}
	}

	ncGoalState := &NCGoalState{
		NetworkContainerID:    repoNCConfig.NetworkContainerID,
		Version:               repoNCConfig.CNSVersion,
		NetworkID:             repoNCConfig.NetworkID,
		InfraNetworkID:        repoNodeConfig.NodeInfo.InfrastructureNetwork,
		AllocatedIPSubnet:     repoReservationConfiguration.AllocatedSubnet,
		SecondaryIPs:          secondaryIPs,
		BaseSubnet:            repoSubnetConfiguration.IPSubnet,
		BaseSubnetName:        repoSubnetConfiguration.SubnetName,
		AccessToken:           repoSubnetConfiguration.AccessToken,
		GatewayIP:             repoSubnetConfiguration.Gateway,
		DNSServers:            dnsServers,
		AssociatedNodeID:      nodeID,
		AssociatedInterfaceID: associatedInterfaceID,
		NodeInterfaceID:       repoNodeConfig.NodeInfo.IPAddresses[0],
		CNetSubnets:           strings.Split(repoNetworkConfiguration.CNetSpace, ","),
		VNetSubnets:           strings.Split(repoNetworkConfiguration.VNetSpace, ","),
		TagsInfo:              make([]nmagentcontracts.Tag, 0),
		ContainerType:         assocInfo.ContainerType,
		OrchestratorContext:   assocInfo.OrchestratorContext,
		VnetVersion:           repoNetworkConfiguration.Version,
		LocalIP:               repoNCConfig.LocalIP}

	// todo: temporary. find better way to skip from publishing to pubsub proxy if orchestrated by k8s crd
	if repoNodeConfig.NodeInfo.OrchestratorType != cns.KubernetesCRD {
		ncGoalState.VlanID = int(repoNodeConfig.NetworkVlanIDMap[repoNCConfig.NetworkID])
	}

	return ncGoalState, false, nil
}

// The function queries CNS running on a node to get number of logical cores configuration data.
// If for some reason the request fails, we return default value of 10.
func GetNCLimitForNode(nodeIP string) (int, *api.JsonErr) {
	loggershim.LS.Debugf("getNCLimit called with settings values GetNumOfCPUCoresURLFmt: %s, EnforceNICLimit: %t",
		cnsSettings.ConnectionStrings.GetNumOfCPUCoresURLFmt,
		enforceNICLimit)

	// Check configuration settings.
	if cnsSettings.ConnectionStrings.GetNumOfCPUCoresURLFmt == "" || !enforceNICLimit {
		loggershim.LS.Debugf("Returning default NIC limit value")
		return utils.MaxContainersPerNode, nil
	}

	getNumOfCoresURL := fmt.Sprintf(
		cnsSettings.ConnectionStrings.GetNumOfCPUCoresURLFmt,
		nodeIP)

	w, respErr := httpClient.Get(getNumOfCoresURL)
	if respErr != nil {
		msg := fmt.Sprintf(
			"Error encountered while querying number of cores for node IP %s with error %+v ",
			nodeIP,
			respErr.Error())
		return 0, &api.JsonErr{
			Code:       api.NumberOfCPUCoresRetrievalFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	defer w.Body.Close()

	// This is expected, in case cns used doesn't support the API
	if w.StatusCode == http.StatusNotFound {
		loggershim.LS.Debugf("Error encountered while querying number of cores for node IP %s with error %+v ",
			nodeIP,
			w.StatusCode)
		return maxContainersPerNode, nil
	}

	if w.StatusCode != http.StatusOK {
		msg := fmt.Sprintf(
			"Error encountered while querying number of cores for node IP %s with error %+v ",
			nodeIP,
			w.StatusCode)
		return 0, &api.JsonErr{
			Code:       api.NumberOfCPUCoresRetrievalFailed,
			Text:       msg,
			HTTPStatus: w.StatusCode}
	}

	var resp cns.NumOfCPUCoresResponse
	if respErr = utils.DecodeResponse(w, &resp); respErr != nil {
		msg := fmt.Sprintf("Error encountered while decoding the NumOfCPUCoresResponse for node IP %s with error %+v",
			nodeIP,
			respErr.Error())
		return 0, &api.JsonErr{
			Code:       api.DecodeResponseFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	if resp.Response.ReturnCode != 0 {
		msg := fmt.Sprintf(
			"GetNumOfCPUCores for node IP %s failed with errorcode %v and msg %s",
			nodeIP,
			resp.Response.ReturnCode,
			resp.Response.Message)
		return 0, &api.JsonErr{
			Code:       api.NumberOfCPUCoresRetrievalFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	return getNCLimitFromNumCores(resp.NumOfCPUCores), nil
}

// GetNcPendingPublishRequestContexts - Retrieve all NC publish / unpublish pending request contexts
func GetNcPendingPublishRequestContexts() (map[string]NcPendingPublishRequestContext, error) {
	var (
		hasMoreData       = true
		context           interface{}
		requests          map[string]interface{}
		err               error
		status            int
		requestsToProcess = make(map[string]NcPendingPublishRequestContext)
	)

	for hasMoreData {
		requests, hasMoreData, context, status, err = dataStoreShim.QueryAll(
			ncPendingPublishContextTableName,
			context)
		if err != nil {
			if status == http.StatusNotFound {
				return make(map[string]NcPendingPublishRequestContext), nil
			}

			return nil, err
		}

		for requestKey, value := range requests {
			serializedRequestContext := value.(string)
			var requestContext NcPendingPublishRequestContext
			err = json.Unmarshal([]byte(serializedRequestContext), &requestContext)
			requestsToProcess[requestKey] = requestContext
		}
	}

	return requestsToProcess, nil
}

// RemoveNcPendingPublishContext - Remove pending NC publish / unpublish context from the table.
func RemoveNcPendingPublishContext(
	networkID, networkContainerID string,
	pubsubAzId string) {
	key := GetNcPendingPublishContextKey(networkID, networkContainerID, pubsubAzId)
	dataStoreShim.Delete(ncPendingPublishContextTableName, key)
}

// CreateOrUpdateNcPendingPublishRequest - Creates or updates NC publish / unpublish context from the table.
func CreateOrUpdateNcPendingPublishRequest(
	requestContext *NcPendingPublishRequestContext) *api.JsonErr {
	return persistNcPendingPublishContext(requestContext)
}

func persistNcPendingPublishContext(
	context *NcPendingPublishRequestContext) *api.JsonErr {
	var (
		jsonSerializedBytes []byte
		err                 error
	)

	jsonSerializedBytes, err = json.Marshal(context)
	if err != nil {
		msg := fmt.Sprintf("Failed to serialize pending NC publish context for NC: %s, Error: %s",
			context.NetworkContainerID,
			err.Error())
		loggershim.LS.Errorf("%s", msg)

		return &api.JsonErr{
			Code:       api.JsonMarshallingFailed,
			HTTPStatus: http.StatusInternalServerError,
			Text:       msg}
	}

	key := GetNcPendingPublishContextKey(context.NetworkID, context.NetworkContainerID, context.PubsubAzId)
	err = dataStoreShim.InsertOrUpdate(ncPendingPublishContextTableName,
		key,
		TableToColumnMap[ncPendingPublishContextTableName],
		string(jsonSerializedBytes[:]))
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to persist pending NC publish context for key: %s. Error: %s",
			key,
			err.Error())
		loggershim.LS.Errorf("%s", msg)

		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	return nil
}

func QueryNcPendingPublishContext(
	key string) (*NcPendingPublishRequestContext, *api.JsonErr) {
	var pendingPublishContext NcPendingPublishRequestContext
	value, status, err := dataStoreShim.Query(ncPendingPublishContextTableName, key)
	if err != nil && status != http.StatusNotFound {
		msg := fmt.Sprintf(
			"Failed to retrieve pending NC publish context for key:[%s] due to error:[%s]",
			key,
			err.Error())
		return nil, &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	if value == nil {
		return nil, nil
	}

	serializedContextData := value.(string)
	err = json.Unmarshal([]byte(serializedContextData), &pendingPublishContext)
	if err != nil {
		msg := fmt.Sprintf(
			"Failed to unmarshal pending NC publish context for key:[%s] due to error:[%s]",
			key,
			err)
		return nil, &api.JsonErr{
			Code:       api.JsonBodyUnMarshalFailed,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	return &pendingPublishContext, nil
}

// GetNetworkContainerRequest :- compile and retrieve cns.CreateNetworkContainerRequest to update CNS state
func GetNetworkContainerRequest(ncGoalState *NCGoalState, ncConfig *NCConfiguration) (*cns.CreateNetworkContainerRequest, bool, *api.JsonErr) {
	_, baseSubnetPrefix, err := utils.GetIPAddressStringAndPrefix(ncGoalState.BaseSubnet)
	if err != nil {
		msg := fmt.Sprintf("Failed to parse base subnet string %v due to error %v",
			ncGoalState.BaseSubnet,
			err.Error())
		failureDetail := api.JsonErr{
			Code:       api.InvalidIPSubnet,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
		ncConfig.Status = Failed
		ncConfig.FailureDetail = failureDetail
		return nil, true, PersistNCConfiguration(ncConfig, DontAddToPendingQ)
	}

	if ncConfig.ProgrammedTagsInfo == nil {
		ncConfig.ProgrammedTagsInfo = make(map[string]uint64)
	}

	var (
		allocatedIP      = strings.Split(ncGoalState.AllocatedIPSubnet, "/")[0]
		cnsCnetAddresses = make([]cns.IPSubnet, 0)
		ipSubnet         = cns.IPSubnet{
			IPAddress:    allocatedIP,
			PrefixLength: uint8(baseSubnetPrefix)}
		ipConfig = cns.IPConfiguration{
			DNSServers:       ncGoalState.DNSServers,
			GatewayIPAddress: ncGoalState.GatewayIP,
			IPSubnet:         ipSubnet}
	)

	localIPSubnet := cns.IPSubnet{
		IPAddress:    ncGoalState.LocalIP,
		PrefixLength: utils.LocalIPSubnetPrefixLength}
	localIPPrefix := fmt.Sprintf("%s/%d", localIPSubnet.IPAddress, localIPSubnet.PrefixLength)
	localGWIP, err := getLocalIPGateway(localIPPrefix)
	if err != nil {
		return nil, false, &api.JsonErr{
			HTTPStatus: http.StatusInternalServerError,
			Text:       fmt.Sprintf("getlocalgatewayIP failed: %v", err),
			Code:       api.StatusInternalServerError,
		}
	}

	localIPConfiguaration := cns.IPConfiguration{
		IPSubnet:         localIPSubnet,
		GatewayIPAddress: localGWIP,
	}

	authToken := ""
	prefix := cns.SwiftPrefix
	if managed {
		authToken = ncGoalState.AccessToken
		prefix = ""
	}

	return &cns.CreateNetworkContainerRequest{
		Version:                    strconv.FormatUint(ncConfig.CNSVersion, 10),
		NetworkContainerType:       ncGoalState.ContainerType,
		NetworkContainerid:         prefix + ncGoalState.NetworkContainerID,
		PrimaryInterfaceIdentifier: ncGoalState.AssociatedInterfaceID,
		IPConfiguration:            ipConfig,
		AuthorizationToken:         authToken,
		CnetAddressSpace:           cnsCnetAddresses,
		OrchestratorContext:        ncGoalState.OrchestratorContext,
		MultiTenancyInfo:           cns.MultiTenancyInfo{EncapType: cns.Vlan, ID: ncGoalState.VlanID},
		LocalIPConfiguration:       localIPConfiguaration,
		AllowHostToNCCommunication: ncConfig.GetAllowHostToNCCommunicationStatus(),
		AllowNCToHostCommunication: ncConfig.GetAllowNCToHostCommunicationStatus()}, false, nil
}

func getLocalIPGateway(ipsubnet string) (string, error) {
	_, ipNet, err := net.ParseCIDR(ipsubnet)
	if err != nil {
		logger.Errorf("parseCIDR for %v failed %v", ipsubnet, err)
		return "", err
	}

	ipstart := utils.IP2Int(ipNet.IP)
	ipstart = ipstart + 1
	return utils.Int2IP(ipstart).String(), nil
}
