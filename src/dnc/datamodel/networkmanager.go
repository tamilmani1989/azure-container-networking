package datamodel

import (
	"bytes"
	"dnc/aiwrapper"
	"dnc/api"
	"dnc/configuration"
	"dnc/datastore/azurestorage"
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

const (
	networkManagementLBSupport string = "NetworkManagementLBSupport"
)

var (
	dataStoreShim             datastoreshim.DataStoreShim
	cnsSettings               configuration.CNSSettings
	hostAgentSettings         configuration.HostAgentSettings
	PendingCleanupRepo        *PendingResourceCleanupRepo
	requestBaseInterval              = 1000
	MinVlan                   uint16 = 1
	MaxVlan                   uint16 = 2048
	maxContainersPerNode             = 10
	httpClient                *http.Client
	cnsHttpClient             *http.Client
	defaultDNSServer          string
	enforceNICLimit           bool // zero value is false
	nicsAllowedPerVirtualCore uint16
	managed                   bool
	MinGre                    uint16 = 257
	MaxGre                    uint16 = 511
	//TODO: refactor package level variables
	disableResMapInSubnetConfig bool
	dncPartitionKey             string
	batchSize                   int
)

// Init - Initialize repository.
func Init(
	cnsConfigSettings configuration.CNSSettings,
	dataStoreSettings configuration.DataStoreSettings,
	dncSettings configuration.DNCSettings,
	hostAgentCfgSettings configuration.HostAgentSettings,
	min, max, greMin, greMax uint16,
	httpClientArg *http.Client,
	cnsHttpClientArg *http.Client,
	wireServerIP string) error {
	requestBaseInterval = dncSettings.RequestRetryBaseInterval
	ds, err := datastoreshim.NewDataStoreShim(dataStoreSettings, dncSettings.ThresholdForLoggingDataStoreOpLatencyMs, TableToColumnMap)
	if err != nil {
		return err
	}

	dataStoreShim = ds
	if dataStoreSettings.TestPersistenceStoreInfo.Test != "" {
		RestoreState()
	}

	PendingCleanupRepo = NewPendingResourceCleanupRepo(&ds, loggershim.LS)

	if min > MinVlan {
		MinVlan = min
	}

	if max > MinVlan {
		MaxVlan = max
	}

	if greMin > MinGre {
		MinGre = greMin
	}

	if greMax > MinGre {
		MaxGre = greMax
	}

	if wireServerIP != "" {
		defaultDNSServer = "168.63.129.16"
	}

	cnsSettings = cnsConfigSettings
	cnsHttpClient = cnsHttpClientArg
	httpClient = httpClientArg
	enforceNICLimit = dncSettings.EnforceNICLimit
	nicsAllowedPerVirtualCore = dncSettings.NICsAllowedPerVirtualCore
	managed = dncSettings.Managed
	hostAgentSettings = hostAgentCfgSettings
	disableResMapInSubnetConfig = dncSettings.DisableReservationMapInSubnetConfig
	dncPartitionKey = dataStoreSettings.AzureStorageInfo.PartitionKey
	batchSize = dataStoreSettings.MaxBatchSize

	return nil
}

func getNCLimitFromNumCores(numCPUCores int) int {
	// Total numbers of containers/nics based on number of cores
	// Convert to Float is needed, as Floor takes float datatype inputs
	totalNicsBasedOnCores := float64(nicsAllowedPerVirtualCore * uint16(numCPUCores))
	return int(math.Floor(totalNicsBasedOnCores) - 1) // -1 to account for VM management NIC
}

// todo: when is this used? it does not work for ip allocation state since that uses a different partition key
// DumpStorageState - Function to dump storage state.
func DumpStorageState(partitionKey string) error {
	for tableName := range TableToColumnMap {
		if err := dumpTable(tableName, partitionKey); err != nil {
			return err
		}
	}

	return nil
}

func dumpTable(tableName, partitionKey string) error {
	fileName := tableName + ".db"

	fp, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(0664))
	if err != nil {
		return err
	}
	defer fp.Close()

	var pageToken interface{}
	for {
		ee, hasMoreRows, nextPageToken, _, err := dataStoreShim.QueryAllRowsInPartition(tableName, partitionKey, pageToken)
		if err != nil {
			return err
		}

		kvMap := make(map[string]interface{})
		for _, e := range ee {
			kvMap[e.RowKey] = e.Properties
		}

		jsonStr, _ := json.Marshal(kvMap)
		if _, err := fp.Write(jsonStr); err != nil {
			return err
		}

		if !hasMoreRows {
			break
		}

		pageToken = nextPageToken
	}

	return nil
}

// DumpState - Function to dump test state.
func DumpState() {
	for tableName := range TableToColumnMap {
		rows, _, _, _, err := dataStoreShim.QueryAllRows(tableName, nil)
		kvMap := make(map[string]map[string]interface{})
		for _, entity := range rows {
			kvMap[entity.RowKey] = entity.Properties
		}
		if err == nil {
			jsonStr, err := json.Marshal(kvMap)
			if err != nil {
				return
			}
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
			var kvMap map[string]map[string]interface{}
			err = json.Unmarshal(content, &kvMap)
			if err == nil {
				for k, v := range kvMap {
					dataStoreShim.InsertOrUpdateProperties(tableName, "", k, v)
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
	reservationRow := ReservationRow{
		RowKey:        GetReservationKey(ipReservation.NetworkID, ipReservation.ReservationID),
		Configuration: ipReservation,
		SubnetKey:     GetSubnetKey(ipReservation.NetworkID, ipReservation.AllocationRequest.SubnetName),
	}

	return PutIPReservation(reservationRow)
}

// PutIPReservation - Persist IP Reservation
func PutIPReservation(reservation ReservationRow) *api.JsonErr {
	jsonSerializedBytes, err := json.Marshal(reservation.Configuration)
	if err != nil {
		return &api.JsonErr{
			Code:       api.JsonMarshallingFailed,
			HTTPStatus: http.StatusInternalServerError,
			Text:       fmt.Sprintf("could not serialize reservation configuration: %s. Error: %v", reservation.Configuration.String(), err)}
	}

	row := map[string]interface{}{
		reservationConfigProperty: string(jsonSerializedBytes[:]),
		subnetKeyProperty:         reservation.SubnetKey}
	err = dataStoreShim.InsertOrUpdateProperties(ipReservationsTableName, dncPartitionKey, reservation.RowKey, row)
	if err != nil {
		loggershim.LS.Errorf("could not persist reservation: %s. Error: %v", reservation.RowKey, err)
		api.InternalServerError(api.StatusInternalServerError, err.Error())
	}

	return nil
}

// UpdateSubnetReservationSet - Add/Remove reservation in subnet set
func UpdateSubnetReservationSet(
	reservationID string,
	allocatedSubnet string,
	remove bool,
	subnetConfiguration *SubnetConfiguration) *api.JsonErr {
	// Do not add the reservation in Subnet Configuration table if the config flag is enabled
	if disableResMapInSubnetConfig {
		return nil
	}

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

	return PersistNodeConfiguration(nodeConfiguration)
}

// GetNextVlanID - Helper fucntion to get next gre key
func GetNextVlanID(vlanMap map[string]uint16) (uint16, *api.JsonErr) {
	return findNextAvailable(vlanMap, MinVlan, MaxVlan, api.OutOfVlanIDs, "Exceeded maximum number of VLAN IDs")
}

// GetNextGreKey - Helper fucntion to get next gre key
func GetNextGreKey(greMap map[string]uint16) (uint16, *api.JsonErr) {
	return findNextAvailable(greMap, MinGre, MaxGre, api.OutOfGreKeys, "Exceeded maximum number of GRE Keys")
}

// findNextAvailable - helper function to get next item in integer map.
func findNextAvailable(baseMap map[string]uint16,
	next, max uint16,
	errorCode api.ErrorCode,
	errorText string) (uint16, *api.JsonErr) {

	v := make(utils.Uint16s, 0, len(baseMap))
	for _, value := range baseMap {
		v = append(v, value)
	}

	sort.Sort(v)

	for _, usedVlanOrgre := range v {
		if next != usedVlanOrgre {

			break
		}

		next++
	}

	if next > max {
		return 0, &api.JsonErr{
			HTTPStatus: http.StatusInternalServerError,
			Text:       errorText,
			Code:       errorCode}
	}

	return next, nil
}

//deleteGreKeyAllocation will take care of only GREKey deletion
func deleteGreKeyAllocation(
	nodeConfig NodeConfiguration,
	networkID, ncID string) *api.JsonErr {

	if nodeConfig.InterfaceGreKeyMap == nil {
		loggershim.LS.Debugf("InterfaceGreKeyMap is empty on node %s, NC: %s",
			nodeConfig.NodeID, ncID)
		return nil
	}

	ncConfig, ncErr := TryQueryNetworkContainer(networkID, ncID)
	if ncErr != nil {
		return ncErr
	}

	interfaceID := ""
	if !ncConfig.ImplicitReservation {
		interfaceID = ncConfig.RequestV1.AssociationInfo.InterfaceID
	} else {
		interfaceID = ncConfig.RequestV2.AssociationInfo.InterfaceID
	}

	if interfaceID == "" {
		loggershim.LS.Debugf("No associated InterfaceID found for NC: %s on node %s ",
			ncID, nodeConfig.NodeID)
		return nil
	}

	if nodeConfig.InterfaceGreKeyMap[interfaceID] == nil {
		nodeConfig.InterfaceGreKeyMap[interfaceID] = make(map[string]uint16)
	}

	// GreKey allocation will be deleted from InterfaceGreMap in the node
	// since this will be called just before deleting NC container, GreKey need not be deleted from NCConfig
	// DNC will delete GreKey for implicit reservations too as DNC cannot guarantee where this reservation be used.
	delete(nodeConfig.InterfaceGreKeyMap[interfaceID], ncConfig.ReservationID)

	if len(nodeConfig.InterfaceGreKeyMap[interfaceID]) == 0 {
		delete(nodeConfig.InterfaceGreKeyMap, interfaceID)
	}

	return nil

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
		if nodeConfig.UpdateGreCapable == GreCapable {
			greErr := deleteGreKeyAllocation(*nodeConfig, networkID, ncID)
			if greErr != nil {
				return greErr
			}
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

	return PersistNodeConfiguration(nodeConfig)
}

// UpdateNCToReservationAssociation :-
func UpdateNCToReservationAssociation(
	ncConfiguration *NCConfiguration,
	disassociate bool) *api.JsonErr {
	reservation, err := TryQueryIPReservation(ncConfiguration.NetworkID, ncConfiguration.ReservationID)
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

	if reservation.Configuration == nil {
		msg := fmt.Sprintf(
			"Reservation %s is not found", reservation.Configuration.ReservationID)
		return &api.JsonErr{
			Code:       api.ReservationNotFound,
			HTTPStatus: http.StatusNotFound,
			Text:       msg}
	}

	if disassociate {
		if strings.Compare(
			reservation.Configuration.AssociatedNCID,
			ncConfiguration.NetworkContainerID) != 0 {
			return nil
		}
		reservation.Configuration.AssociatedNCID = ""
	} else {
		if len(reservation.Configuration.AssociatedNCID) != 0 &&
			strings.Compare(reservation.Configuration.AssociatedNCID,
				ncConfiguration.NetworkContainerID) != 0 {
			msg := fmt.Sprintf(
				"Reservation %s is use with different network container %s. Request network container %s",
				reservation.Configuration.ReservationID,
				reservation.Configuration.AssociatedNCID,
				ncConfiguration.NetworkContainerID)
			return &api.JsonErr{
				Code:       api.ReservationInUse,
				HTTPStatus: http.StatusBadRequest,
				Text:       msg}
		}
		reservation.Configuration.AssociatedNCID = ncConfiguration.NetworkContainerID
	}

	return PutIPReservationConfiguration(reservation.Configuration)
}

// DeleteIPReservation - Delete an IP Reservation.
func DeleteIPReservation(networkID, reservationID string, force bool) *api.JsonErr {
	reservation, err := TryQueryIPReservation(networkID, reservationID)
	if err != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError,
			"could not retrieve configuration for reservationKey: %s. Error: %s",
			GetReservationKey(networkID, reservationID), err)
	}

	if reservation.Configuration == nil {
		return nil
	}

	reservationKey := GetReservationKey(networkID, reservationID)

	if err = DeleteIPAllocation(reservation.Configuration.NetworkID,
		reservation.Configuration.AllocationRequest.SubnetName,
		reservationKey,
		reservation.Configuration.AllocatedSubnet); err != nil {
		loggershim.LS.Errorf("Failed to delete IP allocation due to error %+v", err)
		return err
	}

	if len(reservation.Configuration.AssociatedNCID) != 0 && !force {
		return api.BadRequestf(api.ReservationInUse,
			"Reservation for key: %s is in use with container: %s", reservationKey, reservation.Configuration.AssociatedNCID)
	}

	// Delete the reservation from subnetConfig if the config flag to disable reservationMap is not set
	if !disableResMapInSubnetConfig {
		subnetConfiguration, err := TryQuerySubnetConfiguration(
			reservation.Configuration.NetworkID,
			reservation.Configuration.AllocationRequest.SubnetName)
		if err != nil {
			return err
		}

		if subnetConfiguration != nil && subnetConfiguration.Reservations != nil {
			delete(subnetConfiguration.Reservations, reservationID)
			if err = PutSubnetConfiguration(subnetConfiguration); err != nil {
				return err
			}
		}
	}

	if len(reservation.Configuration.AllocationRequest.NodeConstraint) != 0 {
		nodeConfiguration, err := TryQueryNodeConfiguration(reservation.Configuration.AllocationRequest.NodeConstraint)
		if err != nil {
			return err
		}
		if nodeConfiguration != nil {
			err := UpdateNodeConstrainedReservationSet(
				reservation.Configuration.NetworkID,
				reservation.Configuration.ReservationID,
				true,
				nodeConfiguration)
			if err != nil {
				return err
			}
		}
	}

	deleteErr := dataStoreShim.Delete(ipReservationsTableName, reservationKey)
	if deleteErr != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError,
			"could not delete configuration for reservationKey: %s. Error: %s", reservationKey, deleteErr)
	}

	return err
}

// DeleteIPAllocation - Delete an IP allocation entry from IPAllocations table.
func DeleteIPAllocation(networkID, subnetName, reservationKey, allocatedIP string) *api.JsonErr {
	allocatedIPKey := EncodeIPAllocationKey(allocatedIP)
	savedReservationKey, err := TryQueryIPAllocation(networkID, subnetName, allocatedIPKey)
	if err != nil {
		return err
	}

	if savedReservationKey.IPAllocation == "" {
		loggershim.LS.Errorf("Error: Cannot find the ipAddress %s in IPAllocation table", allocatedIP)
		return nil
	}

	if reservationKey != savedReservationKey.IPAllocation {
		msg := fmt.Sprintf("Reservation key mismatch in IPAllocations table for IP %s", allocatedIP)
		return api.InternalServerError(api.InvalidIPSubnet, msg)
	}

	// Delete the entry if the saved reservation matches the current reservation
	partitionKey := GetSubnetKey(networkID, subnetName)
	if err := dataStoreShim.DeleteInPartition(ipAllocationsTableName, partitionKey, allocatedIPKey); err != nil {
		msg := fmt.Sprintf("Cannot delete entry %s from IPAllocations table due to error: %+v", allocatedIPKey, err)
		return api.InternalServerError(api.StatusInternalServerError, msg)
	}

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

//TODO: Add tests for UpdateIPAllocations
func UpdateIPAllocations(networkID, subnetName string, rows map[string]map[string]interface{}) *api.JsonErr {
	partitionKey := GetSubnetKey(networkID, subnetName)

	if err := dataStoreShim.MergeBatch(ipAllocationsTableName, partitionKey, rows); err != nil {
		storageErr, ok := err.(storage.AzureStorageServiceError)
		if !ok {
			return api.InternalServerError(api.StatusInternalServerError, err.Error())
		}

		switch storageErr.StatusCode {
		case http.StatusPreconditionFailed:
			return api.PreconditionFailed(api.UpdateIPAllocationFailed, storageErr.Message)
		default:
			return api.InternalServerError(api.StatusInternalServerError, storageErr.Message)
		}
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
	entity, status, err := dataStoreShim.Query(nodeTableName, nodeKey)
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
	var result []*storage.Entity
	var returnContext interface{}
	var status int
	var err error
	var networkGUIDs = make([]string, 0)

	for {
		result, hasMoreData, returnContext, status, err = dataStoreShim.QueryAllRows(networkConfigurationTableName, returnContext)
		if err != nil {
			if status == http.StatusNotFound {
				return make([]string, 0), nil
			}

			return nil, err
		}

		for _, entity := range result {
			serializedNetworkConfiguration := entity.Properties[networkConfigurationTag].(string)
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
	entity, status, err := dataStoreShim.Query(networkConfigurationTableName, networkKey)
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

	if entity == nil {
		return nil, nil
	}

	serializedNetworkConfiguration := entity.Properties[networkConfigurationTag].(string)
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
	entity, status, err := dataStoreShim.Query(subnetConfigurationTableName, subnetKey)
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

	if entity == nil {
		return nil, nil
	}

	serializedSubnetConfiguration := entity.Properties[subnetConfigurationTag].(string)
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

// TryQueryFeatureMigrationStatus - Query the feature migration status.
// Returns nil error if the key is not found in the storage.
func TryQueryFeatureMigrationStatus(featureName string) (bool, *api.JsonErr) {
	entity, status, err := dataStoreShim.Query(featureMigrationTableName, featureName)
	if err != nil && status != http.StatusNotFound {
		return false, api.InternalServerErrorf(api.StatusInternalServerError,
			"could not retrieve migration status for feature: %s", featureName)
	}

	if entity == nil {
		return false, nil
	}

	prop, ok := entity.Properties[featureMigrationTag]
	if !ok {
		loggershim.LS.Errorf("could not find FeatureMigrationComplete property in the result entity")
		return false, nil
	}

	return prop.(bool), nil
}

// TryQueryIPReservationByKey - Query IP Reservation configuration.
func TryQueryIPReservationByKey(reservationKey string) (ReservationRow, *api.JsonErr) {
	entity, status, err := dataStoreShim.Query(ipReservationsTableName, reservationKey)
	if err != nil && status != http.StatusNotFound {
		return ReservationRow{}, api.InternalServerErrorf(api.StatusInternalServerError,
			"could not retrieve configuration for reservation: %s Error: %v", reservationKey, err)
	}

	if entity == nil {
		return ReservationRow{}, nil
	}

	var reservationConfiguration ReservationConfiguration
	err = json.Unmarshal([]byte(entity.Properties[reservationConfigProperty].(string)), &reservationConfiguration)
	if err != nil {
		return ReservationRow{}, api.InternalServerErrorf(api.JsonBodyUnMarshalFailed,
			"could not deserialize reservation %s: %v", reservationKey, err)
	}

	reservationRow := ReservationRow{
		RowKey:        reservationKey,
		Configuration: &reservationConfiguration,
	}

	if prop, ok := entity.Properties[subnetKeyProperty]; ok {
		reservationRow.SubnetKey = prop.(string)
	}

	return reservationRow, nil
}

// TryQueryIPReservation - Try querying an IP Reservation. No error is returned if the reservation record is not found in the storage.
func TryQueryIPReservation(networkID, reservationID string) (ReservationRow, *api.JsonErr) {
	reservationKey := GetReservationKey(networkID, reservationID)
	return TryQueryIPReservationByKey(reservationKey)
}

// QueryIPReservation - Query IP Reservation. An error is returned if the reservation record is not found in the storage.
func QueryIPReservation(networkID, reservationID string) (ReservationRow, *api.JsonErr) {
	reservationRow, err := TryQueryIPReservation(networkID, reservationID)
	if err != nil {
		return ReservationRow{}, err
	}

	if reservationRow.Configuration == nil {
		return ReservationRow{}, api.NotFoundf(api.ReservationNotFound, "could not find reservation: %s in network: %s",
			reservationID, networkID)
	}

	return reservationRow, nil
}

func UpdateIPReservations(reservations []ReservationConfiguration) *api.JsonErr {
	rows := make(map[string]map[string]interface{})
	for _, res := range reservations {
		key := GetReservationKey(res.NetworkID, res.ReservationID)
		resJSON, err := json.Marshal(res)
		if err != nil {
			return api.InternalServerErrorf(api.JsonMarshallingFailed, "could not serialize reservation %s: %v", res.ReservationID, err)
		}
		rows[key] = map[string]interface{}{
			reservationConfigurationTag: string(resJSON),
			subnetKeyProperty:           GetSubnetKey(res.NetworkID, res.AllocationRequest.SubnetName)}
	}

	if err := dataStoreShim.UpsertBatch(ipReservationsTableName, rows); err != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError, "could not persist reservation batch: %v", err)
	}

	return nil
}

// TryQueryRequestContext - Query request context.
func TryQueryRequestContext(requestID string) (*RequestContext, *api.JsonErr) {
	entity, status, err := dataStoreShim.Query(pendingWorkItemsTableName, requestID)
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
	if entity == nil {
		return nil, nil
	}

	serializedRequestContext := entity.Properties[requestContextTag].(string)
	var repoRequestContext RequestContext
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
	entity, status, err := dataStoreShim.Query(networkContainerTableName, ncKey)
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

	if entity == nil {
		return nil, nil
	}

	serializedNCConfiguration := entity.Properties[networkContainerConfigurationTag].(string)
	var repoNCConfiguration NCConfiguration
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

// CreateOrUpdateNodeConfiguration - Creates Or Updates a node configuration. Register should be true if called from RegisterNode.
func CreateOrUpdateNodeConfiguration(
	register bool,
	cores int,
	nodeID string,
	nodeInfo api.NodeInfoRequest,
	apiVersion string,
	nmAgentSupportedApis []string) *api.JsonErr {
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
	return PersistNodeConfiguration(repositoryNodeConfiguration)
}

func PersistNodeConfiguration(nodeconfiguration *NodeConfiguration) *api.JsonErr {
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

// GetPendingRequests - Retrieve all request pending processing
func GetPendingRequests() (map[string]RequestContext, error) {
	var context interface{} = nil
	requestsToProcess := make(map[string]RequestContext)
	for {
		requests, hasMoreData, cx, status, err := dataStoreShim.QueryAllRows(pendingWorkItemsTableName, context)
		if err != nil {
			if status == http.StatusNotFound {
				return make(map[string]RequestContext), nil
			}

			return nil, err
		}

		for _, entity := range requests {
			serializedRequestContext := entity.Properties[requestContextTag].(string)
			var requestContext RequestContext
			err = json.Unmarshal([]byte(serializedRequestContext), &requestContext)
			requestsToProcess[entity.RowKey] = requestContext
		}

		if !hasMoreData {
			break
		}
		context = cx
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
		return DeallocateMAC, nil

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
	} else if networkContainerConfig.MarkedForDelete == true {
		// There is a valid race where NC MarkedForDelete is set however adding the request to PendingQ is missing
		// This is possible if call to DeleteNetworkContainer failed in interim. As a result caller will receive failure and
		// retry the operation. So ensure NC entries exists in both NetworkContainer and PendingQ tables.
		requestID := GetNCKey(networkContainerConfig.NetworkID, networkContainerConfig.NetworkContainerID)
		pendingQItem, err := TryQueryRequestContext(requestID)
		if err != nil {
			return err
		}

		if pendingQItem != nil {
			return nil
		}

		loggershim.LS.Debugf("NC %s is MarkedForDelete, requeue request to PendinqQueueTable", networkContainerID)
		requestContext := RequestContext{
			Type:      NetworkContainerRequest,
			NetworkID: networkContainerConfig.NetworkID,
			NodeID:    networkContainerConfig.GetAssociationInfo().NodeID}

		return AddRequestToPendingQueue(requestID, &requestContext)
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
	networkConfig, err := TryQueryNetworkConfiguration(networkID)
	if err != nil {
		return err
	}

	if networkConfig == nil {
		return api.NotFoundf(api.NetworkNotFound, "could not find network: %s", networkID)
	}

	if networkConfig.NetworkState == Join && networkConfig.Status != Completed {
		return api.BadRequestf(api.NetworkStatusInvalid,
			"Network join for %s is not complete. Current status: %s", networkID, networkConfig.Status.String())
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
		networkConfig.Subnets[subnet] = SubnetKeyPlaceholder
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
			return api.BadRequestf(api.SubnetNotFound,
				"could not find subnet: %s in network: %s. Failed to store subnet config", subnet, networkID)
		}

		requestContext := RequestContext{
			Type:      VirtualNetworkRequest,
			NetworkID: networkID}

		return AddRequestToPendingQueue(GetNetworkKey(networkID), &requestContext)
	}

	subnetConfiguration, err := TryQuerySubnetConfiguration(networkID, subnet)
	if err != nil {
		return err
	}

	if subnetConfiguration == nil {
		return api.BadRequestf(api.SubnetConfigurationNotFound,
			"could not find subnet configuration: %s in network: %s", subnet, networkID)
	}

	if subnetConfiguration.UpdateStatus == SubnetUpdatePending {
		// No error is returned as the request is treated as idempotent while
		// the same request issued earlier is in process
		return nil
	} else if subnetConfiguration.UpdateStatus == SubnetUpdateFailed {
		return api.NotFoundf(api.SubnetNotFound, "could not find subnet: %s in network: %s", subnet, networkID)
	}

	if !subnetConfiguration.IsDelegatedSubnet {
		return api.BadRequestf(api.SubnetIsNotDelegated, "subnet: %s in network: %s is not delegated", subnet, networkID)
	}

	if isDelete {
		reservations, err := GetReservationsInSubnet(networkID, subnet)
		if err != nil {
			return api.InternalServerErrorf(api.StatusInternalServerError,
				"could not retrieve reservations for subnet: %s. Error: %v", subnet, err)
		}

		if len(reservations) > 0 {
			return api.BadRequestf(api.InvalidSubnetRequest, "cannot remove auth token as subnet: %s has allocations", subnet)
		}
	}

	subnetConfiguration.Joined = !isDelete
	subnetConfiguration.AccessToken = subnetRequest.AuthenticationToken
	return PutSubnetConfiguration(subnetConfiguration)
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

	var subnetKeysToDelete []string
	for subnetName := range networkConfiguration.Subnets {
		subnetConfiguration, err := TryQuerySubnetConfiguration(networkID, subnetName)
		if err != nil {
			return err
		}

		if subnetConfiguration == nil {
			continue
		}

		reservations, err := GetReservationsInSubnet(networkID, subnetName)
		if err != nil {
			return err
		}

		if len(reservations) > 0 {
			reservationKeys := make([]string, 0, len(reservations))
			for _, reservation := range reservations {
				reservationKeys = append(reservationKeys, reservation.RowKey)
			}
			reservationsString := "[" + strings.Join(reservationKeys, ", ") + "]"
			return api.BadRequestf(api.NetworkInUse,
				"Subnet: %s in Network: %s has allocations: %s", subnetName, networkID, reservationsString)
		}

		subnetKeysToDelete = append(subnetKeysToDelete, subnetName)
	}

	for _, subnetName := range subnetKeysToDelete {
		if err := DeleteSubnetConfiguration(networkID, subnetName); err != nil {
			return err
		}
	}

	networkKey := GetNetworkKey(networkID)
	if deleteErr := dataStoreShim.Delete(networkConfigurationTableName, networkKey); deleteErr != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError,
			"could not delete configuration for network: %s. Error: %s", networkKey, deleteErr)
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
		err               *api.JsonErr
		ipConfiguration   *api.IPConfiguration
		nodeConfiguration *NodeConfiguration
		reservation       ReservationRow
	)

	if ncConfig == nil {
		return nil, &api.JsonErr{
			Code:       api.StatusInternalServerError,
			HTTPStatus: http.StatusInternalServerError,
			Text:       "GetNetworkContainerResponse on nil NCConfiguration"}
	}

	if reservation, err = QueryIPReservation(ncConfig.NetworkID, ncConfig.ReservationID); err != nil {
		loggershim.LS.Errorf("Failed to get IP reservation in GetNetworkContainerResponse for NC: %s,"+
			" NC state: %s. Error: %v", ncConfig.NetworkContainerID, ncConfig.State.String(), err)
	}

	if reservation.Configuration != nil {
		ipConfiguration, err = getIPConfiguration(reservation.Configuration)
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
		NCVersion:                  int(ncConfig.CNSVersion),
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
		reservation, err := QueryIPReservation(networkID, reservationID)
		if err != nil {
			return nil, err
		}

		if reservation.Configuration == nil {
			continue
		}

		constrainedReservations[index] = translateRepoToContractReservationResponse(reservation.Configuration)
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

	reservation, err := TryQueryIPReservation(networkID, reservationID)
	if reservation.Configuration != nil {
		return nil
	}

	reservation.Configuration = &ReservationConfiguration{
		NetworkID:         networkID,
		ReservationID:     reservationID,
		ReservationState:  Process,
		Status:            Pending,
		AllocationRequest: allocationRequest,
		APIVersion:        apiVersion}

	err = PutIPReservationConfiguration(reservation.Configuration)
	if err != nil {
		return err
	}

	requestContext := RequestContext{
		Type:      ReservationRequest,
		NetworkID: reservation.Configuration.NetworkID,
		NodeID:    reservation.Configuration.AllocationRequest.NodeConstraint}

	reservationKey := GetReservationKey(networkID, reservationID)
	return AddRequestToPendingQueue(reservationKey, &requestContext)
}

// GetReservationResponse - Retrieve reservation response.
func GetReservationResponse(
	networkID string,
	reservationID string) (*api.ReservationResponse, *api.JsonErr) {
	reservation, err := QueryIPReservation(networkID, reservationID)
	if err != nil {
		return nil, err
	}

	response := translateRepoToContractReservationResponse(reservation.Configuration)
	return &response, nil
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
		msg := fmt.Sprintf("Network container id %s currently being processed in state %v with status %v. Cannot update while processing",
			nc.NetworkContainerID, nc.State.String(), nc.Status.String())
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

// GetAllNetworkIDs - Retrieve all network IDs
func GetAllNetworkIDs() (map[string]bool, error) {
	var context interface{}
	networkIDs := make(map[string]bool)
	for {
		entities, hasMoreData, cx, status, err := dataStoreShim.QueryAllRows(networkConfigurationTableName, context)
		if err != nil {
			if status == http.StatusNotFound {
				return networkIDs, nil
			}

			return nil, err
		}

		for _, entity := range entities {
			networkIDs[entity.RowKey] = true
		}

		if !hasMoreData {
			break
		}
		context = cx
	}

	return networkIDs, nil
}

// GetAllNodeIDs - Retrieve all node IDs
func GetAllNodeIDs() (map[string]bool, error) {
	var context interface{}
	nodeIDs := make(map[string]bool)
	for {
		entities, hasMoreData, cx, status, err := dataStoreShim.QueryAllRows(nodeTableName, context)
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

const AllocationIDProperty = "AllocationID"

type AllocationRow struct {
	RowKey           string
	IPAllocation     string
	ToBeDeletedOwner string
	AllocationID     string
}

type ReservationRow struct {
	RowKey        string
	Configuration *ReservationConfiguration
	SubnetKey     string
}

// TryQueryAllIPAllocations - Query all IP allocations in the partition.
func TryQueryAllIPAllocations(networkID, subnetName string) ([]AllocationRow, *api.JsonErr) {
	partitionKey := GetSubnetKey(networkID, subnetName)

	var (
		rows      []AllocationRow
		pageToken interface{}
	)
	for {
		entities, hasNextPage, nextPageToken, _, err := dataStoreShim.QueryAllRowsInPartition(ipAllocationsTableName, partitionKey, pageToken)
		if err != nil {
			return nil, api.InternalServerError(api.StatusInternalServerError, err.Error())
		}

		for _, entity := range entities {
			row := AllocationRow{
				RowKey:       entity.RowKey, // todo: use metadata struct from different PR
				IPAllocation: entity.Properties[ipAllocationTag].(string),
			}

			/* TODO: Similar checks should be in place for all entity properties.
			   create a helper func that reads entity.Property and returns a value */
			if prop, ok := entity.Properties[AllocationIDProperty]; ok {
				row.AllocationID = prop.(string)
			}

			rows = append(rows, row)
		}

		if !hasNextPage {
			break
		}
		pageToken = nextPageToken
	}
	return rows, nil
}

func QueryIPAllocationsByAllocationID(networkID, subnetName string, allocationIDs []string) ([]AllocationRow, *api.JsonErr) {
	partitionKey := GetSubnetKey(networkID, subnetName)
	filter := azurestorage.BuildFilterOnProperty(partitionKey, AllocationIDProperty, allocationIDs)

	entities, err := dataStoreShim.QueryFilter(ipAllocationsTableName, filter)
	if err != nil {
		return nil, api.InternalServerError(api.StatusInternalServerError, err.Error())
	}
	var rows []AllocationRow
	for _, entity := range entities {
		row := AllocationRow{
			RowKey:       entity.RowKey, // todo: use metadata struct from different PR
			IPAllocation: entity.Properties[ipAllocationTag].(string),
		}

		if prop, ok := entity.Properties[AllocationIDProperty]; ok {
			row.AllocationID = prop.(string)
		}

		rows = append(rows, row)
	}

	return rows, nil
}

func getReservationsFromSubnetConfigTable(networkID, subnetName string) (map[string]ReservationRow, *api.JsonErr) {
	repoSubnetConfiguration, err := QuerySubnetConfiguration(networkID, subnetName)
	if err != nil {
		return nil, err
	}

	reservations := make(map[string]ReservationRow, len(repoSubnetConfiguration.Reservations))
	for reservationID := range repoSubnetConfiguration.Reservations {
		repoReservation, err := QueryIPReservation(networkID, reservationID)
		if err != nil {
			return nil, api.InternalServerErrorf(api.StatusInternalServerError,
				"could not retrieve reservation for reservationID %s: %v", reservationID, err)
		}

		reservations[repoReservation.RowKey] = repoReservation
	}

	return reservations, nil
}

func GetReservationsInSubnet(networkID, subnetName string) ([]ReservationRow, *api.JsonErr) {
	// Get the reservations from IPReservationsTable filtered on SubnetKey
	reservationsFromIPReservationsTable, err := TryQueryAllReservationsInSubnet(networkID, subnetName)
	if err != nil {
		return nil, err
	}

	// If config flag is enabled, return reservations from IPReservations table
	if disableResMapInSubnetConfig {
		return reservationsFromIPReservationsTable, nil
	}

	// If the config flag is disabled, get the reservations from subnet config table too
	// and return the union of reservations without duplicates
	reservationMapFromSubnetConfigTable, err := getReservationsFromSubnetConfigTable(networkID, subnetName)
	if err != nil {
		return nil, err
	}

	// Create a reservation slice with initial capacity equal to number of reservations found in IPReservationsTable
	reservationsInSubnet := make([]ReservationRow, 0, len(reservationsFromIPReservationsTable))

	// Take the union of the two sets of reservations without duplicates
	for _, reservation := range reservationsFromIPReservationsTable {
		// Include the reservation from IPReservationsTable only if it's not found in SubnetConfigTable
		if _, duplicate := reservationMapFromSubnetConfigTable[reservation.RowKey]; !duplicate {
			reservationsInSubnet = append(reservationsInSubnet, reservation)
		}
	}

	// Include all the reservations from SubnetConfigTable
	for _, reservation := range reservationMapFromSubnetConfigTable {
		reservationsInSubnet = append(reservationsInSubnet, reservation)
	}

	return reservationsInSubnet, nil
}

// TryQueryAllReservationsInSubnet returns all the reservations in the given subnet
func TryQueryAllReservationsInSubnet(networkID, subnetName string) ([]ReservationRow, *api.JsonErr) {
	subnetKey := []string{GetSubnetKey(networkID, subnetName)}
	filter := azurestorage.BuildFilterOnProperty(dncPartitionKey, subnetKeyProperty, subnetKey)
	return tryQueryReservations(filter)
}

// TryQueryAllReservations returns all the reservations in the dnc partition
func TryQueryAllReservations() ([]ReservationRow, *api.JsonErr) {
	filter := azurestorage.BuildFilterOnPartition(dncPartitionKey)
	return tryQueryReservations(filter)
}

func tryQueryReservations(filter string) ([]ReservationRow, *api.JsonErr) {
	entities, err := dataStoreShim.QueryFilter(ipReservationsTableName, filter)
	if err != nil {
		return nil, api.InternalServerError(api.StatusInternalServerError, err.Error())
	}

	rows := make([]ReservationRow, len(entities))
	for i, entity := range entities {
		var reservationConfiguration ReservationConfiguration
		if err = json.Unmarshal([]byte(entity.Properties[reservationConfigProperty].(string)), &reservationConfiguration); err != nil {
			return []ReservationRow{}, api.InternalServerErrorf(api.JsonBodyUnMarshalFailed,
				"could not deserialize reservation %s. Error: %v", entity.RowKey, err)
		}

		rows[i] = ReservationRow{
			RowKey:        entity.RowKey,
			Configuration: &reservationConfiguration,
		}

		if prop, ok := entity.Properties[subnetKeyProperty]; ok {
			rows[i].SubnetKey = prop.(string)
		}
	}

	return rows, nil
}

func TryQueryAllSubnetConfigurations() ([]SubnetConfiguration, *api.JsonErr) {
	filter := azurestorage.BuildFilterOnPartition(dncPartitionKey)
	entities, err := dataStoreShim.QueryFilter(subnetConfigurationTableName, filter)
	if err != nil {
		return nil, api.InternalServerErrorf(api.StatusInternalServerError, err.Error())
	}

	subnetConfigs := make([]SubnetConfiguration, len(entities))
	for i, entity := range entities {
		var config SubnetConfiguration
		if err := json.Unmarshal([]byte(entity.Properties[subnetConfigurationTag].(string)), &config); err != nil {
			return []SubnetConfiguration{}, api.InternalServerErrorf(api.JsonBodyUnMarshalFailed,
				"could not deserialize subnet config: %s. Error: %v", entity.RowKey, err)
		}

		subnetConfigs[i] = config
	}

	return subnetConfigs, nil
}

// TryQueryIPAllocation - Query for IP allocations
func TryQueryIPAllocation(networkID, subnetName, ipAddress string) (AllocationRow, *api.JsonErr) {
	partitionKey := GetSubnetKey(networkID, subnetName)
	return TryQueryIPAllocationByKey(partitionKey, ipAddress)
}

// TryQueryIPAllocationByKey - Query for IP allocations by the key in the specified partition
func TryQueryIPAllocationByKey(partitionKey, ipAddress string) (AllocationRow, *api.JsonErr) {
	entity, status, err := dataStoreShim.QueryInPartition(ipAllocationsTableName, partitionKey, ipAddress)
	if err != nil && status != http.StatusNotFound {
		msg := fmt.Sprintf(
			"Failed to retrieve IP allocation for key:[%s] in partition:[%s] due to error:[%s]",
			ipAddress,
			partitionKey,
			err.Error())
		return AllocationRow{}, &api.JsonErr{
			Code:       api.StatusInternalServerError,
			Text:       msg,
			HTTPStatus: http.StatusInternalServerError}
	}

	if entity == nil {
		return AllocationRow{}, nil
	}

	row := AllocationRow{
		IPAllocation: entity.Properties[ipAllocationTag].(string),
	}

	if prop, ok := entity.Properties[AllocationIDProperty]; ok {
		row.AllocationID = prop.(string)
	}

	return row, nil
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

// GetAllNetworkContainers - Retrieve all network containers
func GetAllNetworkContainers() (map[string]NCConfiguration, error) {
	var context interface{}
	ncConfigs := make(map[string]NCConfiguration)
	for {
		entities, hasMoreData, cx, status, err := dataStoreShim.QueryAllRows(networkContainerTableName, context)
		if err != nil {
			if status == http.StatusNotFound {
				return ncConfigs, nil
			}

			return nil, err
		}

		for _, entity := range entities {
			serializedNCConfig := entity.Properties[networkContainerConfigurationTag].(string)
			var ncConfig NCConfiguration
			err = json.Unmarshal([]byte(serializedNCConfig), &ncConfig)
			if err != nil {
				return nil, fmt.Errorf("Unmarshalling nc config %s failed with error %s", serializedNCConfig, err.Error())
			}

			ncConfigs[entity.RowKey] = ncConfig
		}

		if !hasMoreData {
			break
		}
		context = cx
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
	repoReservation, err := TryQueryIPReservation(repoNCConfig.NetworkID, repoNCConfig.ReservationID)
	if err != nil {
		return nil, true, err
	}

	if repoReservation.Configuration == nil {
		return nil, false, api.BadRequestf(api.ReservationNotFound,
			"reservation: %s not found. Associated NC: %s", repoNCConfig.ReservationID, repoNCConfig.NetworkContainerID)
	}

	// Query NodeConfiguration table with nodeid and handle errors.
	repoNodeConfig, err := TryQueryNodeConfiguration(nodeID)
	if err != nil {
		return nil, true, err
	}

	if repoNodeConfig == nil {
		return nil, false, api.BadRequestf(api.NodeNotFound,
			"configuration for node: %s not found. Associated NC: %s", nodeID, repoNCConfig.NetworkContainerID)
	}

	repoSubnetConfiguration, err := TryQuerySubnetConfiguration(
		repoNCConfig.NetworkID,
		repoReservation.Configuration.AllocationRequest.SubnetName)
	if err != nil {
		return nil, true, err
	}

	if repoSubnetConfiguration == nil {
		return nil, false, api.BadRequestf(api.SubnetNotFound, "configuration for subnet: %s not found. Network: %s",
			repoReservation.Configuration.AllocationRequest.SubnetName, repoNCConfig.NetworkID)
	}

	if repoSubnetConfiguration.Tags == nil {
		repoSubnetConfiguration.Tags = make([]nmagentcontracts.Tag, 0)
	}

	repoNetworkConfiguration, err := TryQueryNetworkConfiguration(repoNCConfig.NetworkID)
	if err != nil {
		return nil, true, err
	}

	if repoNetworkConfiguration == nil {
		return nil, false, api.BadRequestf(api.NetworkNotFound, "configuration for network: %s not found", repoNCConfig.NetworkID)
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
	for _, ipAddress := range repoReservation.Configuration.SecondaryAllocations {
		if ipAddress != "" {
			secondaryIPs = append(secondaryIPs, ipAddress)
		}
	}

	ncGoalState := &NCGoalState{
		NetworkContainerID:    repoNCConfig.NetworkContainerID,
		Version:               repoNCConfig.CNSVersion,
		NetworkID:             repoNCConfig.NetworkID,
		InfraNetworkID:        repoNodeConfig.NodeInfo.InfrastructureNetwork,
		AllocatedIPSubnet:     repoReservation.Configuration.AllocatedSubnet,
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
		LocalIP:               repoNCConfig.LocalIP,
		GreKey:                repoNCConfig.GreKey,
		OrchestratorType:      repoNodeConfig.NodeInfo.OrchestratorType}

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

	w, respErr := cnsHttpClient.Get(getNumOfCoresURL)
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
		context           interface{}
		requestsToProcess = make(map[string]NcPendingPublishRequestContext)
	)

	for {
		entities, hasMoreData, cx, status, err := dataStoreShim.QueryAllRows(ncPendingPublishContextTableName, context)
		if err != nil {
			if status == http.StatusNotFound {
				return make(map[string]NcPendingPublishRequestContext), nil
			}

			return nil, err
		}

		for _, entity := range entities {
			serializedRequestContext := entity.Properties[ncPendingPublishContextTag].(string)
			var requestContext NcPendingPublishRequestContext
			if err := json.Unmarshal([]byte(serializedRequestContext), &requestContext); err != nil {
				return nil, err
			}
			requestsToProcess[entity.RowKey] = requestContext
		}

		if !hasMoreData {
			break
		}

		context = cx
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

func QueryNcPendingPublishContext(key string) (*NcPendingPublishRequestContext, *api.JsonErr) {
	entity, status, err := dataStoreShim.Query(ncPendingPublishContextTableName, key)
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

	if entity == nil {
		return nil, nil
	}

	serializedContextData := entity.Properties[ncPendingPublishContextTag].(string)
	var pendingPublishContext NcPendingPublishRequestContext
	if err := json.Unmarshal([]byte(serializedContextData), &pendingPublishContext); err != nil {
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

//updateGreKeyCapability helps determine if a given Node is capable of getting GRE keys assigned
func updateGreKeyCapability(nodeConfiguration *NodeConfiguration, nmAgentSupportedApis []string) GreSetting {
	loggershim.LS.Debugf("updateGreKeyCapability: Node: %s",
		nodeConfiguration.NodeID)

	// We are checking if any GREKeys are already assigned here
	// returning if InterfaceGreKeyMap has more than one entry,
	if len(nodeConfiguration.InterfaceGreKeyMap) > 0 {
		for _, interfaceMap := range nodeConfiguration.InterfaceGreKeyMap {
			if len(interfaceMap) > 0 {
				loggershim.LS.Debugf("UpdateGreKey is enabled on Node: %s",
					nodeConfiguration.NodeID)
				return GreCapable
			}
		}
	}

	// if countOfNcs >0 then GreKey is not allocated for these NCs
	countOfNcs := 0
	for _, mapNCs := range nodeConfiguration.DeployedNetworkContainerMap {
		countOfNcs += len(mapNCs)
	}

	if countOfNcs > 0 {
		loggershim.LS.Debugf("UpdateGreKey: Node:%s has existing NCs without GRE allocations.",
			nodeConfiguration.NodeID)
		return ExistingNCsWithoutGRE
	}

	// In Managed DNC case, DNC cannot talk to CNS, CNS will solicit SupportedAPIs list while registering Node.
	if !managed {
		if len(nodeConfiguration.NodeInfo.IPAddresses) == 0 {
			loggershim.LS.Debugf("UpdateGreKey: Node:%s has no IpAddresses in NodeConfig ",
				nodeConfiguration.NodeID)
			return GreCapableNotSet
		}
		//Check if NMAgent is capable
		supportedApisList, errMsg := NmAgentSupportedApis(nodeConfiguration.NodeInfo.IPAddresses[0])
		if errMsg != nil {
			loggershim.LS.Debugf("UpdateGreKey: Node:%s error occured while getting supported Apis list from NMAgent %s",
				nodeConfiguration.NodeID, errMsg.Error())
			return GreCapableNotSet
		}
		nmAgentSupportedApis = supportedApisList
	}

	if len(nmAgentSupportedApis) > 0 {
		for _, apiName := range nmAgentSupportedApis {
			if apiName == networkManagementLBSupport {
				loggershim.LS.Debugf("UpdateGreKey: Node:%s is capable for GRE Allocations",
					nodeConfiguration.NodeID)
				return GreCapable
			}
		}

		return NmAgentIncapable
	}

	return GreCapableNotSet

}

// NmAgentSupportedApis queries CNS running on a node to get NMagent's Supported Apis list.
// If this request fails, then DNC will skip assiging GRE key.
// TODO check for Swift AKS case where DNC-CNS talk via k8s api server
func NmAgentSupportedApis(nodeIP string) ([]string, error) {
	loggershim.LS.Debugf("NMAgent supported Apis query at GetNmAgentSupportedApis: %s",
		cnsSettings.ConnectionStrings.NmAgentSupportedApisURLFmt)

	var (
		resp               cns.NmAgentSupportedApisResponse
		body               bytes.Buffer
		retSupportedList   []string
		defaultContentType string = "application/json; charset=UTF-8"
	)

	// Check configuration settings.
	if cnsSettings.ConnectionStrings.NmAgentSupportedApisURLFmt == "" {
		loggershim.LS.Debugf("No GetNmAgentSupportedApisFmt found. Returning default empty list")
		return retSupportedList, fmt.Errorf("No GetNmAgentSupportedApisFmt found")
	}

	//This is a parameter sent to CNS
	getNmAgentSupportedAPIURL := fmt.Sprintf(
		hostAgentSettings.ConnectionStrings.GetNmAgentSupportedApiURLFmt,
		hostAgentSettings.WireServerIP)

	supportedApisreq := cns.NmAgentSupportedApisRequest{
		GetNmAgentSupportedApisURL: getNmAgentSupportedAPIURL,
	}

	//CNS endpoint to get Node NMagent's SUported Apis
	getCnsNmAgentApisURL := fmt.Sprintf(
		cnsSettings.ConnectionStrings.NmAgentSupportedApisURLFmt,
		nodeIP)

	encodeErr := json.NewEncoder(&body).Encode(supportedApisreq)
	if encodeErr != nil {
		msg := fmt.Errorf(
			"Error encountered while encoding NMAgent Supported Apis request to node IP %s with error %+v ",
			nodeIP,
			encodeErr.Error())
		return retSupportedList, msg
	}

	w, err := httpClient.Post(getCnsNmAgentApisURL, defaultContentType, &body)
	if err != nil {
		msg := fmt.Errorf(
			"Error encountered while querying NMAgent Supported Apis for node IP %s with error %+v ",
			nodeIP,
			err.Error())
		return retSupportedList, msg
	}
	defer w.Body.Close()

	// This is expected, in case cns used doesn't support the API
	if w.StatusCode == http.StatusNotFound {
		msg := fmt.Errorf(
			"Error encountered while querying NMAgent Supported Apis for node IP %s with error %+v ",
			nodeIP,
			w.StatusCode)
		return retSupportedList, msg
	}

	if w.StatusCode != http.StatusOK {
		msg := fmt.Errorf(
			"Error encountered while querying NMAgent Supported Apis for node IP %s with error %+v ",
			nodeIP,
			w.StatusCode)
		return retSupportedList, msg
	}

	if respErr := utils.DecodeResponse(w, &resp); respErr != nil {
		msg := fmt.Errorf("Error encountered while decoding the NmAgentSupportedApisResponse for node IP %s with error %+v",
			nodeIP,
			respErr.Error())
		return retSupportedList, msg
	}

	if resp.Response.ReturnCode != 0 {
		msg := fmt.Errorf(
			"GetNmAgentSupportedApis for node IP %s failed with errorcode %v and msg %s",
			nodeIP,
			resp.Response.ReturnCode,
			resp.Response.Message)
		return retSupportedList, msg
	}

	return resp.SupportedApis, nil
}

// Process and complete the feature migration for feature flag - DisableReservationMapInSubnetConfig
func ProcessDisableReservationMapInSubnetConfig(feature string) *api.JsonErr {
	featureMigrationComplete, err := TryQueryFeatureMigrationStatus(feature)
	if err != nil {
		loggershim.LS.Errorf("could not get the feature migration status for %s. Error: %v", feature, err)
		return err
	}

	if featureMigrationComplete {
		loggershim.LS.Debugf("Feature migration for %s is already completed", feature)
		return nil
	}

	// Update the subnetkey column of reservation rows in IPReservations table
	if err = updateIPReservationsTableWithSubnet(); err != nil {
		loggershim.LS.Errorf("could not update all the reservation records with subnet keys. Error: %v", err)
		return err
	}

	loggershim.LS.Debugf("successfully updated all reservations with subnet keys")

	// Remove the reservation map from subnet configuration table
	if err = removeReservationMapFromSubnetConfig(); err != nil {
		loggershim.LS.Errorf("could not remove the reservation map from subnet config. Error: %v", err)
		return err
	}

	loggershim.LS.Debugf("successfully removed reservation map from all subnet configuration records")

	// Mark the feature migration complete in the FeatureMigration Table
	if err := dataStoreShim.InsertOrUpdate(featureMigrationTableName, feature,
		TableToColumnMap[featureMigrationTableName], true); err != nil {
		loggershim.LS.Errorf("could not persist feature migration status for feature: %s. Error: %v", feature, err)
		return api.InternalServerErrorf(api.StatusInternalServerError, err.Error())
	}

	loggershim.LS.Debugf("successfully completed feature migration for %s", feature)

	return nil
}

// Remove the reservation map from subnet configuration table
func removeReservationMapFromSubnetConfig() *api.JsonErr {
	// Get all the subnet configurations in the DNC partition
	subnetConfigs, err := TryQueryAllSubnetConfigurations()
	if err != nil {
		return err
	}

	updatedSubnetConfigs := make(map[string]map[string]interface{})
	for _, subnetConfig := range subnetConfigs {
		if subnetConfig.Reservations != nil {
			subnetKey := GetSubnetKey(subnetConfig.NetworkID, subnetConfig.SubnetName)
			subnetConfig.Reservations = nil
			subnetJSON, err := json.Marshal(subnetConfig)
			if err != nil {
				return api.InternalServerErrorf(api.JsonMarshallingFailed,
					"could not serialize config for subnet: %s. Error: %v", subnetKey, err)
			}

			updatedSubnetConfigs[subnetKey] = map[string]interface{}{
				TableToColumnMap[subnetConfigurationTableName]: string(subnetJSON)}

			if len(updatedSubnetConfigs) == batchSize {
				if err := dataStoreShim.UpsertBatch(subnetConfigurationTableName, updatedSubnetConfigs); err != nil {
					return api.InternalServerErrorf(api.StatusInternalServerError, "could not persist subnetconfig batch: %v", err)
				}

				updatedSubnetConfigs = make(map[string]map[string]interface{})
			}
		}
	}

	if len(updatedSubnetConfigs) > 0 {
		if err := dataStoreShim.UpsertBatch(subnetConfigurationTableName, updatedSubnetConfigs); err != nil {
			return api.InternalServerErrorf(api.StatusInternalServerError, "could not persist subnetconfig batch: %v", err)
		}
	}

	return nil
}

// Update the subnetkey column of reservation rows in IPReservations table
func updateIPReservationsTableWithSubnet() *api.JsonErr {
	// Get all the reservations in the DNC partition
	reservations, err := TryQueryAllReservations()
	if err != nil {
		loggershim.LS.Errorf("could not get all the reservations. Error: %v", err)
		return err
	}

	updatedReservations := make(map[string]map[string]interface{})
	for _, reservation := range reservations {
		// Update the reservation record only if the subnetKey is empty
		if reservation.SubnetKey == "" {
			resJSON, err := json.Marshal(reservation.Configuration)
			if err != nil {
				return api.InternalServerErrorf(api.JsonMarshallingFailed,
					"could not serialize reservation %s. Error: %v", reservation.RowKey, err)
			}

			subnetKey := GetSubnetKey(reservation.Configuration.NetworkID, reservation.Configuration.AllocationRequest.SubnetName)
			updatedReservations[reservation.RowKey] = map[string]interface{}{
				reservationConfigProperty: string(resJSON),
				subnetKeyProperty:         subnetKey}

			if len(updatedReservations) == batchSize {
				if err := dataStoreShim.UpsertBatch(ipReservationsTableName, updatedReservations); err != nil {
					return api.InternalServerErrorf(api.StatusInternalServerError, "could not persist reservation batch: %v", err)
				}

				updatedReservations = make(map[string]map[string]interface{})
			}
		}
	}

	if len(updatedReservations) > 0 {
		if err := dataStoreShim.UpsertBatch(ipReservationsTableName, updatedReservations); err != nil {
			return api.InternalServerErrorf(api.StatusInternalServerError, "could not persist reservation batch: %v", err)
		}
	}

	return nil
}
