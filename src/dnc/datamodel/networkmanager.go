// todo: package datamodel should be only for repo layer. we should extract service layer functionality (which
//  depends and orchestrates repos, as well as other things like cns and nmagent clients) to a different package,
//  possibly networkcontainer.
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

	"github.com/google/uuid"

	"github.com/Azure/azure-container-networking/aitelemetry"
	"github.com/Azure/azure-container-networking/cns"
	"github.com/Azure/azure-container-networking/cns/logger"
	"github.com/Azure/azure-sdk-for-go/storage"
)

const (
	networkManagementLBSupport string = "NetworkManagementLBSupport"
)

// todo: convert these package level vars into a service struct
var (
	// todo: Init is called by some endpoints which updates these vars at runtime, needs to be thread safe
	dataStoreShim               datastoreshim.DataStoreShim
	cnsSettings                 configuration.CNSSettings
	hostAgentSettings           configuration.HostAgentSettings
	requestBaseInterval                = 1000
	MinVlan                     uint16 = 1
	MaxVlan                     uint16 = 2048
	maxContainersPerNode               = 10
	httpClient                  *http.Client
	cnsHttpClient               *http.Client
	defaultDNSServer            string
	enforceNICLimit             bool // zero value is false
	nicsAllowedPerVirtualCore   uint16
	managed                     bool
	MinGre                      uint16 = 257
	MaxGre                      uint16 = 511
	disableResMapInSubnetConfig bool
	dncPartitionKey             string
	batchSize                   int

	// todo: repos can be initialized by a helper method and compose a struct that can be shared.
	//  exporting variable temporarily, since this package is behaving as a singleton.

	PendingCleanupRepo   *PendingResourceCleanupRepo
	NcPendingPublishRepo *NcPendingPublishRequestRepo
	AllocRepo            *AllocationRepo
	IpReservationRepo    *ReservationRepo
	NcRepo               *NetworkContainerRepo
	NodeConfigRepo       *NodeConfigurationRepo
	IpReservationSetRepo *ReservationSetRepo
	NodeSetsRepo         *NodeSetRepo
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

	dncPartitionKey = dataStoreSettings.AzureStorageInfo.PartitionKey
	dataStoreShim = ds

	AllocRepo = &AllocationRepo{Datastore: ds}
	IpReservationRepo = NewReservationRepo(ds, dncPartitionKey, loggershim.LS)
	PendingCleanupRepo = NewPendingResourceCleanupRepo(ds, loggershim.LS)
	NcPendingPublishRepo = NewNcPendingPublishRequestRepo(ds)
	NodeConfigRepo = &NodeConfigurationRepo{Datastore: ds}
	IpReservationSetRepo = NewReservationSetRepo(ds, dncPartitionKey, loggershim.LS)
	NcRepo = NewNetworkContainerRepo(ds, dncPartitionKey, loggershim.LS)
	NodeConfigRepo = &NodeConfigurationRepo{Datastore: ds}
	NodeSetsRepo = &NodeSetRepo{Datastore: ds, PartitionKey: dncPartitionKey}

	if dataStoreSettings.TestPersistenceStoreInfo.Test != "" {
		RestoreState()
	}

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
	batchSize = dataStoreSettings.MaxBatchSize

	return nil
}

// todo: this is only needed temporarily. having the datastore shim injected is a larger refactor.
func GetDatastoreSingleton() datastoreshim.DataStoreShim {
	return dataStoreShim
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
		subnetKeyProperty:         reservation.SubnetKey,
		setKeyProperty:            reservation.SetKey}
	err = dataStoreShim.InsertOrUpdateProperties(ipReservationsTableName, dncPartitionKey, reservation.RowKey, row)
	if err != nil {
		loggershim.LS.Errorf("could not persist reservation: %s. Error: %v", reservation.RowKey, err)
		return api.InternalServerError(api.StatusInternalServerError, err.Error())
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

// GetNextVlanID - Helper fucntion to get next gre key
func GetNextVlanID(vlanMap map[string]uint16) (uint16, *api.JsonErr) {
	return findNextAvailable(vlanMap, MinVlan, MaxVlan, api.OutOfVlanIDs, "Exceeded maximum number of VLAN IDs")
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

// GetNextGreKey - Helper fucntion to get next gre key
func GetNextGreKey(greMap map[string]uint16) (uint16, *api.JsonErr) {
	return findNextAvailableGrekey(greMap, MinGre, MaxGre, api.OutOfGreKeys, "Exceeded maximum number of GRE Keys")
}

// findNextAvailableGrekey - helper function to get next item in integer map.
func findNextAvailableGrekey(baseMap map[string]uint16,
	min, next uint16,
	errorCode api.ErrorCode,
	errorText string) (uint16, *api.JsonErr) {

	v := make(utils.Uint16s, 0, len(baseMap))
	for _, value := range baseMap {
		v = append(v, value)
	}

	sort.Sort(v)

	len := len(v)
	for i := len - 1; i >= 0; i-- {
		if next != v[i] {
			loggershim.LS.Debugf("Finished searching gre key, checked %d existing keys", len-i)
			break
		}
		next--
	}
	loggershim.LS.Debugf("Finished getting a gre key, base map length is %d, gre key value is %d, min is %v, max is %v", len, next, min, MaxGre)

	if next < min {
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

	nc, ncErr := NcRepo.TryQueryNetworkContainer(networkID, ncID)
	if ncErr != nil {
		return ncErr
	}

	if nc.RowKey == "" {
		return api.NotFoundf(api.NetworkContainerNotFound, "could not find data for NC: %s", GetNCKey(networkID, ncID))
	}

	interfaceID := ""
	if !nc.Config.ImplicitReservation {
		interfaceID = nc.Config.RequestV1.AssociationInfo.InterfaceID
	} else {
		interfaceID = nc.Config.RequestV2.AssociationInfo.InterfaceID
	}

	if interfaceID == "" {
		loggershim.LS.Debugf("No associated InterfaceID found for NC: %s on node %s ", ncID, nodeConfig.NodeID)
		return nil
	}

	if nodeConfig.InterfaceGreKeyMap[interfaceID] == nil {
		nodeConfig.InterfaceGreKeyMap[interfaceID] = make(map[string]uint16)
	}

	// GreKey allocation will be deleted from InterfaceGreMap in the node
	// since this will be called just before deleting NC container, GreKey need not be deleted from NCConfig
	// DNC will delete GreKey for implicit reservations too as DNC cannot guarantee where this reservation be used.
	delete(nodeConfig.InterfaceGreKeyMap[interfaceID], nc.Config.ReservationID)

	if len(nodeConfig.InterfaceGreKeyMap[interfaceID]) == 0 {
		delete(nodeConfig.InterfaceGreKeyMap, interfaceID)
	}

	return nil
}

// UpdateNCToReservationAssociation :-
func UpdateNCToReservationAssociation(
	nc *NetworkContainerEntity,
	disassociate bool) *api.JsonErr {
	reservation, err := TryQueryIPReservation(nc.Config.NetworkID, nc.Config.ReservationID)
	if err != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError,
			"could not retrieve configuration for reservation: %s in vnet: %s. error: %s",
			nc.Config.ReservationID, nc.Config.NetworkID, err)
	}

	// If the reservation is found, there is no need to throw an error as there is nothing to update
	if reservation.Configuration == nil {
		return nil
	}

	if disassociate {
		if strings.Compare(
			reservation.Configuration.AssociatedNCID,
			nc.Config.NetworkContainerID) != 0 {
			return nil
		}
		reservation.Configuration.AssociatedNCID = ""
	} else {
		if len(reservation.Configuration.AssociatedNCID) != 0 &&
			strings.Compare(reservation.Configuration.AssociatedNCID, nc.Config.NetworkContainerID) != 0 {
			return api.BadRequestf(api.ReservationInUse,
				"Reservation: %s is in use with different network container: %s. Request network container: %s",
				reservation.Configuration.ReservationID, reservation.Configuration.AssociatedNCID, nc.Config.NetworkContainerID)
		}
		reservation.Configuration.AssociatedNCID = nc.Config.NetworkContainerID
	}

	return IpReservationRepo.PersistIPReservation(reservation)
}

// TryQueryAllNetworks queries for all networkconfigurations managed by this dnc and returns their IDs
// TODO: Change all the repo methods to return *api.JsonErr instead of error
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
		return nil, api.NotFoundf(api.SubnetNotFound, "Configuration for subnet: %s in network: %s not found", subnetName, networkID)
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

	if prop, ok := entity.Properties[setKeyProperty]; ok {
		reservationRow.SetKey = prop.(string)
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
		msg := fmt.Sprintf("Failed to serialize request context. Key:%s RequestContext:%+v, Error: %s",
			requestKey, requestContext, err.Error())
		loggershim.LS.Errorf("%s: %s", utils.TracePrefix(2), msg)

		return &api.JsonErr{
			Code:       api.JsonMarshallingFailed,
			HTTPStatus: http.StatusInternalServerError,
			Text:       msg}
	}

	err = dataStoreShim.InsertOrUpdate(pendingWorkItemsTableName, requestKey, TableToColumnMap[pendingWorkItemsTableName], string(jsonSerializedBytes[:]))
	if err != nil {
		msg := fmt.Sprintf("Failed to add request %+v associated with key %s to pending queue due to error:{%s}",
			requestContext, requestKey, err.Error())
		loggershim.LS.Errorf("%s: %s", utils.TracePrefix(2), msg)
		return &api.JsonErr{
			Code:       api.StatusInternalServerError,
			HTTPStatus: http.StatusInternalServerError,
			Text:       msg}
	}

	return nil
}

type PendingQueueRequest struct {
	Key     string
	Context *RequestContext
}

func (p *PendingQueueRequest) ToStorageEntity(dncPartitionKey string) (storage.Entity, *api.JsonErr) {
	e := storage.Entity{
		Table:        &storage.Table{Name: pendingWorkItemsTableName},
		PartitionKey: dncPartitionKey,
		RowKey:       p.Key,
		//todo: Add ETag
		//OdataEtag:     p.ETag,
	}

	jsonSerializedBytes, err := json.Marshal(p.Context)
	if err != nil {
		return storage.Entity{}, api.InternalServerErrorf(api.JsonMarshallingFailed,
			"could not serialize request context. key: %s, context: %s. error: %v", p.Key, p.Context, err)
	}

	e.Properties = map[string]interface{}{
		requestContextTag: string(jsonSerializedBytes[:]),
	}

	return e, nil
}

// AddNewRequestsToPendingQueue - Add batch of new requests to pending work item queue.
func AddNewRequestsToPendingQueue(
	requests []PendingQueueRequest) *api.JsonErr {
	storageEntities := make([]storage.Entity, len(requests))
	for i, request := range requests {
		se, err := request.ToStorageEntity(dncPartitionKey)
		if err != nil {
			return err
		}
		storageEntities[i] = se
	}

	if err := dataStoreShim.UpsertEntityBatch(pendingWorkItemsTableName, storageEntities); err != nil {
		storageErr, ok := err.(storage.AzureStorageServiceError)
		if ok {
			return &api.JsonErr{
				Code:       api.AzureStorageError,
				Text:       storageErr.Message,
				HTTPStatus: storageErr.StatusCode}
		}

		return api.InternalServerError(api.StatusInternalServerError, err.Error())
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
func RemovePendingRequest(requestID string) *api.JsonErr {
	if err := dataStoreShim.Delete(pendingWorkItemsTableName, requestID); err != nil {
		switch e := err.(type) {
		case storage.AzureStorageServiceError:
			if e.StatusCode == http.StatusNotFound {
				return nil
			}
			return &api.JsonErr{Code: api.AzureStorageError, Text: e.Message, HTTPStatus: e.StatusCode}
		default:
			return api.InternalServerError(api.StatusInternalServerError, e.Error())
		}
	}

	return nil
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
		return UnReserve, nil

	case Associate:
		return DisAssociate, nil

	case RefreshAssociation:
		fallthrough
	case Publish:
		fallthrough
	case PrepareSecondaryIpCleanup:
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

func storageEntityFromRequestContext(key string, req RequestContext) (storage.Entity, error) {
	bs, err := json.Marshal(req)
	if err != nil {
		return storage.Entity{}, err
	}

	e := storage.Entity{
		Table:        &storage.Table{Name: pendingWorkItemsTableName},
		PartitionKey: dncPartitionKey,
		RowKey:       key,
		Properties: map[string]interface{}{
			requestContextTag: string(bs),
		},
	}

	return e, nil
}

func UpsertPendingQueueItem(key string, req RequestContext) *api.JsonErr {
	now := time.Now().UTC()

	req.CreatedDateTime = now
	req.RetryInterval = time.Duration(requestBaseInterval) * time.Millisecond
	req.StateTransitionDateTime = now

	e, err := storageEntityFromRequestContext(key, req)
	if err != nil {
		return api.InternalServerError(api.StatusInternalServerError, err.Error())
	}

	if err := dataStoreShim.InsertOrUpdateEntity(pendingWorkItemsTableName, e); err != nil {
		return api.InternalServerError(api.StatusInternalServerError, err.Error())
	}

	return nil
}

// ProcessNetworkContainerForRelease - Process network container configuration pending release.
func ProcessNetworkContainerForRelease(networkID, networkContainerID string) *api.JsonErr {
	loggershim.LS.Debugf("ProcessNetworkContainerForRelease: vnet id %s nc id %s", networkID, networkContainerID)
	nc, err := NcRepo.TryQueryNetworkContainer(networkID, networkContainerID)
	if err != nil {
		return err
	}

	if nc.RowKey == "" {
		loggershim.LS.Debugf("ProcessNetworkContainerForRelease: vnet id %s nc id %s not found, nothing to process", networkID, networkContainerID)
		return nil
	}

	if !nc.Config.MarkedForDelete || nc.Config.Status == Failed {
		nc.Config.MarkedForDelete = true
		nc.Config.Status = Pending
		nc.Config.FailureDetail = api.JsonErr{}
		// only retrieve the revert state if NC is not in any of the delete states
		if !IsDeleteInProgress(nc.Config.State) {
			revertState, err := GetRevertState(nc.Config.State)
			if err != nil {
				return err
			}

			loggershim.LS.Debugf("NC %s transitioning from state %s to %s",
				nc.Config.NetworkContainerID, nc.Config.State, revertState)
			nc.Config.State = revertState

			// This is intended to be a new request for deletion, any existing RequestContext for this NC in
			// the pending queue needs to be deleted.
			// It's possible to have a RequestContext in pending queue belonging to refreshAssociation when
			// deleteNC is triggered. This results in using the incorrect timestamps causing the unpublish
			// and in turn the complete deletion time calculation to be erroneous.
			requestID := GetNCKey(nc.Config.NetworkID, nc.Config.NetworkContainerID)
			if err := RemovePendingRequest(requestID); err != nil {
				return err
			}
		}

		loggershim.LS.Debugf("ProcessNetworkContainerForRelease: vnet %s nc id %s", networkID, networkContainerID)
		if err = NcRepo.PersistNetworkContainerData(&nc); err != nil {
			// todo: make sure the error is being logged in the caller.
			return err
		}

		// Add request to the pending queue
		requestContext := RequestContext{
			Type:      NetworkContainerRequest,
			NetworkID: nc.Config.NetworkID,
			NodeID:    nc.Config.GetAssociationInfo().NodeID,
		}

		ncKey := GetNCKey(nc.Config.NetworkID, nc.Config.NetworkContainerID)
		return AddRequestToPendingQueue(ncKey, &requestContext)
	}

	if nc.Config.MarkedForDelete {
		// There is a valid race where NC MarkedForDelete is set however adding the request to PendingQ is missing
		// This is possible if call to DeleteNetworkContainer failed in interim. As a result caller will receive failure and
		// retry the operation. So ensure NC entries exists in both NetworkContainer and PendingQ tables.
		requestID := GetNCKey(nc.Config.NetworkID, nc.Config.NetworkContainerID)
		pendingQItem, err := TryQueryRequestContext(requestID)
		if err != nil {
			return err
		}

		if pendingQItem != nil {
			return nil
		}

		loggershim.LS.Debugf("NC %s is MarkedForDelete, re-queue request to PendinqQueueTable", networkContainerID)
		requestContext := RequestContext{
			Type:      NetworkContainerRequest,
			NetworkID: nc.Config.NetworkID,
			NodeID:    nc.Config.GetAssociationInfo().NodeID}

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
func CreateNetwork(networkID, apiVersion string, addToPendingQueue bool) *api.JsonErr {
	return createNetworkWithStateAndStatus(networkID, apiVersion, addToPendingQueue, Join, Pending)
}

// CreateNetwork - Creates network configuration
func CreateNetworkCompleted(networkID, apiVersion string, addToPendingQueue bool) *api.JsonErr {
	return createNetworkWithStateAndStatus(networkID, apiVersion, addToPendingQueue, Join, Completed)
}

// CreateNetwork - Creates network configuration
func createNetworkWithStateAndStatus(
	networkID, apiVersion string, addToPendingQueue bool, state NetworkState, status Status) *api.JsonErr {
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
		NetworkState: state,
		Status:       status,
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

// PurgeNetworkContainerAssociations - Process release of reservation associated with network container
func PurgeNetworkContainerAssociations(nc *NetworkContainerEntity, updateVlan bool) *api.JsonErr {
	associationInfo := nc.Config.GetAssociationInfo()
	err := NodeConfigRepo.UpdateNodeConfigurationNetworkContainerSet(
		associationInfo.NodeID,
		nc.Config.NetworkID,
		nc.Config.NetworkContainerID,
		nc.Config.ReservationID,
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

	nodeConfiguration, err = NodeConfigRepo.QueryNodeConfiguration(ncConfig.GetAssociationInfo().NodeID)
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

	stateLog := make([]api.StateTransitionStats, 0, len(ncConfig.StateTransitionLog))
	for k, v := range ncConfig.StateTransitionLog {
		stateLog = append(stateLog, api.StateTransitionStats{State: k, CompletedOn: v.CompletedOn})
	}

	sort.Slice(stateLog, func(i, j int) bool {
		return stateLog[j].CompletedOn.After(stateLog[i].CompletedOn)
	})

	response := api.NetworkContainerResponse{
		NcID:                       ncConfig.NetworkContainerID,
		AssociationInfo:            associationInfo,
		RequestStatus:              requestStatus,
		VlanID:                     vlanID,
		LocalIP:                    ncConfig.LocalIP,
		MacAddress:                 ncConfig.MACAddress,
		NCVersion:                  int(ncConfig.CNSVersion),
		AllowHostToNCCommunication: ncConfig.GetAllowHostToNCCommunicationStatus(),
		AllowNCToHostCommunication: ncConfig.GetAllowNCToHostCommunicationStatus(),
		CreatedDateTime:            ncConfig.CreatedDateTime,
		StateTransitionLog:         stateLog,
	}

	if ipConfiguration != nil {
		response.IPConfiguration = *ipConfiguration
	}

	return &response, nil
}

func translateRepoToContractReservationResponse(
	reservation ReservationRow) api.ReservationResponse {
	config := reservation.Configuration
	requestStatus := api.RequestStatus{
		State:         config.ReservationState.String(),
		Status:        config.Status.String(),
		FailureDetail: config.FailureDetail}

	reservationSetID, _ := GetSetIDFromReservationSetKey(reservation.SetKey)
	response := api.ReservationResponse{
		NetworkID:         config.NetworkID,
		ReservationID:     config.ReservationID,
		AllocationRequest: config.AllocationRequest,
		AllocatedIPSubnet: config.AllocatedSubnet,
		ReservationSetID:  reservationSetID,
		RequestStatus:     requestStatus}

	if reservation.SetKey != "" {
		response.ReservationSetID = GetReservationIDFromKey(reservation.SetKey)
	}

	return response
}

// getOrchestratorContextList - Returns the list of orchestrator contexts for all the NCs for a given node.
func getOrchestratorContextListFromNodeConfig(nodeConfig NodeConfiguration) (map[string][]byte, *api.JsonErr) {
	orchestratorContextList := make(map[string][]byte)

	for networkID, ncMap := range nodeConfig.DeployedNetworkContainerMap {
		for ncID := range ncMap {
			nc, err := NcRepo.GetNetworkContainerData(networkID, ncID)
			if err == nil && nc.RowKey != "" {
				orchestratorContextList[nc.Config.NetworkContainerID] = nc.Config.GetAssociationInfo().OrchestratorContext
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

// PersistIPReservationRequest :-
func PersistIPReservationRequest(networkID string,
	reservationID string,
	allocationRequest api.AllocationRequest,
	apiVersion string) *api.JsonErr {
	reservation, err := TryQueryIPReservation(networkID, reservationID)
	if err != nil {
		return err
	}

	if reservation.Configuration != nil {
		return nil
	}

	reservation = ReservationRow{
		RowKey:    GetReservationKey(networkID, reservationID),
		SubnetKey: GetSubnetKey(networkID, allocationRequest.SubnetName),
		Configuration: &ReservationConfiguration{
			NetworkID:         networkID,
			ReservationID:     reservationID,
			ReservationState:  Process,
			Status:            Pending,
			AllocationRequest: allocationRequest,
			APIVersion:        apiVersion},
		SetKey: "",
	}

	err = IpReservationRepo.PersistIPReservation(reservation)
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
	reservation ReservationRow) (*api.ReservationResponse, *api.JsonErr) {
	response := translateRepoToContractReservationResponse(reservation)
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

func EnsureNCPendingWorkItem(vnetID, ncID, nodeID string) *api.JsonErr {
	ncKey := GetNCKey(vnetID, ncID)
	pendingItem, err := TryQueryRequestContext(ncKey)
	if err != nil {
		return err
	}

	if pendingItem != nil {
		return nil
	}

	loggershim.LS.Debugf("request for nc id %s received, but no pending item found", ncID)
	requestContext := RequestContext{
		NetworkID: vnetID,
		NodeID:    nodeID,
		Type:      NetworkContainerRequest,
	}

	return UpsertPendingQueueItem(ncKey, requestContext)
}

// CreateOrUpdateNCRequestV1 :-
func CreateOrUpdateNCRequestV1(networkID, ncID string, ncRequestV1 *api.NetworkContainerRequestV1, apiVersion string) *api.JsonErr {
	if err := createOrUpdateNCRequestV1(networkID, ncID, ncRequestV1, apiVersion); err != nil {
		loggershim.LS.Errorf("CreateOrUpdateNCRequestV1 failed. Error: %v", err)
		sendNCCreationReqMetric(networkID, ncID, aiwrapper.Invalid, ncRequestV1.AssociationInfo)
		return err
	}

	sendNCCreationReqMetric(networkID, ncID, aiwrapper.Valid, ncRequestV1.AssociationInfo)
	return nil

}
func createOrUpdateNCRequestV1(
	networkID, ncID string,
	ncRequestV1 *api.NetworkContainerRequestV1,
	apiVersion string) *api.JsonErr {
	nc, err := NcRepo.TryQueryNetworkContainer(networkID, ncID)
	if err != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError,
			"could not retrieve configuration for network container: %s. error: %v", ncID, err)
	}

	// there is no update supported for V1 request, but we should still ensure the corresponding pending queue item
	// exists, as its creation is not atomic with NC creation.
	if nc.RowKey != "" {
		if nc.Config.State == Prepare && nc.Config.Status == Pending {
			return EnsureNCPendingWorkItem(networkID, ncID, nc.Config.GetAssociationInfo().NodeID)
		}
		return nil
	}

	nodeID := ncRequestV1.AssociationInfo.NodeID

	node, err := NodeConfigRepo.TryQueryNodeEntity(GetNodeKey(nodeID))
	if err != nil {
		return err
	}

	if node.RowKey == "" {
		return api.NotFoundf(api.NodeNotFound, "Node: %s not found", nodeID)
	}

	if err := validateNodeTypePlacement(ncRequestV1.AssociationInfo.ContainerType, node.Config.NodeInfo.NodeType); err != nil {
		return err
	}

	orchestratorContextList, err := getOrchestratorContextListFromNodeConfig(node.Config)
	if err != nil {
		return err
	}

	// Marshal the orchestrator context for the incoming request. This is needed to compare with the
	// existing orchestrator contexts because when saving the orchestrator context to NCConfig, it
	// gets marshaled.
	marshaledOrchestratorContext, _ := json.Marshal(ncRequestV1.AssociationInfo.OrchestratorContext)

	for ncid, orchestratorContext := range orchestratorContextList {
		if bytes.Compare(orchestratorContext, marshaledOrchestratorContext) == 0 {
			return api.BadRequestf(api.StatusBadRequest,
				"Illegal create nc operation due to duplicate orchestrator context with nc: %s", ncid)
		}
	}

	nc = NetworkContainerEntity{
		RowKey: GetNCKey(networkID, ncID),
		Config: NCConfiguration{
			NetworkID:           networkID,
			NetworkContainerID:  ncID,
			ReservationID:       ncRequestV1.ReservationID,
			RequestV1:           *ncRequestV1,
			ImplicitReservation: false,
			State:               Prepare,
			Status:              Pending,
			APIVersion:          apiVersion,
			CreatedDateTime:     time.Now().UTC(),
			CNSVersion:          0,
		},
		NodeSetKey: node.NodeSetKey,
	}

	if err := NcRepo.PersistNetworkContainerData(&nc); err != nil {
		return err
	}

	// Add request to the pending queue
	requestContext := RequestContext{
		Type:      NetworkContainerRequest,
		NetworkID: nc.Config.NetworkID,
		NodeID:    nc.Config.GetAssociationInfo().NodeID}

	return AddRequestToPendingQueue(GetNCKey(networkID, ncID), &requestContext)
}

func CreateSubnetCleanupRequest(networkID, subnetName string) *api.JsonErr {
	networkConfiguration, err := TryQueryNetworkConfiguration(networkID)
	if err != nil {
		loggershim.LS.Errorf("Failed to retrieve network configuration for network: %s. Error: %v", networkID, err)
		return err
	}

	if networkConfiguration == nil {
		loggershim.LS.Debugf("No network configuration for network: %s", networkID)
		return nil
	}

	requestedSubnetConfig, err := TryQuerySubnetConfiguration(networkID, subnetName)
	if err != nil {
		loggershim.LS.Errorf("Failed to retrieve subnet configuration for subnet: %s in network: %s. Error: %v", subnetName, networkID, err)
		return err
	}

	if requestedSubnetConfig == nil {
		loggershim.LS.Debugf("No subnet configuration for subnet: %s in network: %s", subnetName, networkID)
		return nil
	}

	// If subnet cleanup is already in process, ensure that the pending work item exists. Add pending item if it doesn't exist.
	if requestedSubnetConfig.MarkedForDelete && requestedSubnetConfig.UpdateStatus == SubnetCleanupPending {
		return ensureSubnetCleanupPendingItem(networkID, subnetName)
	}

	// Update the subnet configuration for pending cleanup
	// Do not mark the Joined to false because that will cause the GET status on this subnet to return 404. Subnet cleanup
	// is async operation and GET status needs to return either cleanup pending / cleanup failed. No NCs will be allowed to be created
	// in this subnet as it's marked for deletion.
	requestedSubnetConfig.UpdateStatus = SubnetCleanupPending
	requestedSubnetConfig.MarkedForDelete = true
	requestedSubnetConfig.FailureDetail = api.JsonErr{}
	loggershim.LS.Debugf("Marking subnet: %s in network: %s for deletion and pending cleanup", subnetName, networkID)
	if err := PutSubnetConfiguration(requestedSubnetConfig); err != nil {
		loggershim.LS.Errorf("Failed to persist subnet configuration for subnet: %s in network: %s. Error: %v", subnetName, networkID, err)
		return err
	}

	// Add subnet cleanup request to pending queue
	requestContext := RequestContext{
		Type:       SubnetCleanupRequest,
		NetworkID:  networkID,
		SubnetName: subnetName,
	}

	subnetKey := GetSubnetKey(networkID, subnetName)
	return AddRequestToPendingQueue(subnetKey, &requestContext)
}

// Ensure that the pending work item exists for the subnet which is in pending cleanup state
func ensureSubnetCleanupPendingItem(networkID, subnetName string) *api.JsonErr {
	key := GetSubnetKey(networkID, subnetName)
	pendingItem, err := TryQueryRequestContext(key)
	if err != nil {
		return err
	}

	if pendingItem != nil {
		loggershim.LS.Debugf("Subnet cleanup is already in process for subnet: %s in network: %s", subnetName, networkID)
		return nil
	}

	loggershim.LS.Debugf("subnet %s has pending cleanup status, but no pending item found", key)

	// Add subnet cleanup request to pending queue
	requestContext := RequestContext{
		Type:       SubnetCleanupRequest,
		NetworkID:  networkID,
		SubnetName: subnetName,
	}

	subnetKey := GetSubnetKey(networkID, subnetName)
	return UpsertPendingQueueItem(subnetKey, requestContext)
}

// CreateOrUpdateNCRequestV2 - Persists NetworkContainerRequestV2
func CreateOrUpdateNCRequestV2(networkID, ncID string, ncRequestV2 *api.NetworkContainerRequestV2, apiVersion string) *api.JsonErr {
	if err := createOrUpdateNCRequestV2(networkID, ncID, ncRequestV2, apiVersion); err != nil {
		loggershim.LS.Errorf("CreateOrUpdateNCRequestV2 failed. Error: %v", err)
		sendNCCreationReqMetric(networkID, ncID, aiwrapper.Invalid, ncRequestV2.AssociationInfo)
		return err
	}

	sendNCCreationReqMetric(networkID, ncID, aiwrapper.Valid, ncRequestV2.AssociationInfo)
	return nil
}

func createOrUpdateNCRequestV2(
	networkID, ncID string,
	ncRequestV2 *api.NetworkContainerRequestV2,
	apiVersion string) *api.JsonErr {
	nc, err := NcRepo.TryQueryNetworkContainer(networkID, ncID)
	if err != nil {
		return api.InternalServerErrorf(api.StatusInternalServerError,
			"could not retrieve configuration for network container: %s. error: %v", ncID, err)
	}

	// Update if the NC record is already present
	if nc.RowKey != "" {
		return UpdateNCRequestV2(&nc, *ncRequestV2)
	}

	nodeID := ncRequestV2.AssociationInfo.NodeID
	node, err := NodeConfigRepo.TryQueryNodeEntity(GetNodeKey(nodeID))
	if err != nil {
		return err
	}

	if node.RowKey == "" {
		return api.NotFoundf(api.NodeNotFound, "Node: %s not found", nodeID)
	}

	// todo: this check is prone to race conditions with node set cleanup, but will require further refactoring to take care
	nodeSet := node.Config.NodeInfo.NodeSet
	if nodeSet != "" {
		ns, err := NodeSetsRepo.TryQueryNodeSet(nodeSet)
		if err != nil {
			return api.InternalServerErrorf(api.StatusInternalServerError, "node %s: could not verify node set %s: %v", nodeID, nodeSet, err)
		}

		if ns.RowKey == "" {
			return api.BadRequestf(api.EntityNotFound, "node %s: node set %s not found", nodeID, nodeSet)
		}

		if ns.MarkedForDelete {
			return api.BadRequestf(api.EntityMarkedForDeletion, "node %s: node set %s marked for deletion", nodeID, nodeSet)
		}
	}

	if err := validateNodeTypePlacement(ncRequestV2.AssociationInfo.ContainerType, node.Config.NodeInfo.NodeType); err != nil {
		return err
	}

	orchestratorContextList, err := getOrchestratorContextListFromNodeConfig(node.Config)
	if err != nil {
		return err
	}

	// Marshal the orchestrator context for the incoming request. This is needed to compare with the
	// existing orchestrator contexts because when saving the orchestrator context to NCConfig, it
	// gets marshaled.
	marshaledOrchestratorContext, _ := json.Marshal(ncRequestV2.AssociationInfo.OrchestratorContext)

	for ncid, orchestratorContext := range orchestratorContextList {
		if bytes.Compare(orchestratorContext, marshaledOrchestratorContext) == 0 {
			return api.BadRequestf(api.StatusBadRequest,
				"Illegal create nc operation due to duplicate orchestrator context with nc: %s", ncid)
		}
	}

	networkConfig, err := QueryNetworkConfiguration(networkID)
	if err != nil {
		return err
	}

	if networkConfig.NetworkState != Join {
		return api.BadRequestf(api.NetworkStateInvalid, "Cannot create network container: %s in network: %s "+
			"as network is not in join state. Current network state: %s", ncID, networkID, networkConfig.NetworkState.String())
	}

	if networkConfig.Status != Completed {
		return api.BadRequestf(api.NetworkStateInvalid, "Cannot create network container: %s in network: %s "+
			"as the status of the join operation is not completed. Current network state: %s", ncID, networkID, networkConfig.NetworkState.String())
	}

	subnetConfig, err := QuerySubnetConfiguration(networkID, ncRequestV2.AllocationRequest.SubnetName)
	if err != nil {
		return err
	}

	if !subnetConfig.Joined {
		return api.BadRequestf(api.SubnetIsNotJoined, "Cannot create network container: %s in network: %s, subnet: %s"+
			" as subnet has not completed the join operation", ncID, networkID, ncRequestV2.AllocationRequest.SubnetName)
	}

	if subnetConfig.MarkedForDelete {
		return api.BadRequestf(api.SubnetMarkedForDeletion, "Cannot create network container: %s in network: %s, subnet: %s"+
			" as subnet is marked for deletion", ncID, networkID, ncRequestV2.AllocationRequest.SubnetName)
	}

	nc = NetworkContainerEntity{
		RowKey: GetNCKey(networkID, ncID),
		Config: NCConfiguration{
			NetworkID:           networkID,
			NetworkContainerID:  ncID,
			ReservationID:       uuid.New().String(),
			RequestV2:           *ncRequestV2,
			ImplicitReservation: true,
			State:               Prepare,
			Status:              Pending,
			APIVersion:          apiVersion,
			CreatedDateTime:     time.Now().UTC(),
			CNSVersion:          0,
		},
		SubnetKey:  GetSubnetKey(networkID, ncRequestV2.AllocationRequest.SubnetName),
		NodeSetKey: node.NodeSetKey,
	}

	if err := NcRepo.PersistNetworkContainerData(&nc); err != nil {
		return err
	}

	// Add request to the pending queue
	requestContext := RequestContext{
		Type:      NetworkContainerRequest,
		NetworkID: nc.Config.NetworkID,
		NodeID:    nc.Config.GetAssociationInfo().NodeID}

	return AddRequestToPendingQueue(GetNCKey(networkID, ncID), &requestContext)
}

func UpdateNCRequestV2(nc *NetworkContainerEntity, request api.NetworkContainerRequestV2) *api.JsonErr {
	// It is possible that a previous NC request is already going through processing as an update is received
	// We must wait for the processing to be complete (or failed) before we queue up another item, otherwise, we risk a
	// race condition in which the updated records are persisted before current processing ends, and then
	// at the end of the initial processing, the pending queue item is delete, without having processed the update.
	if nc.Config.State == Prepare && nc.Config.Status == Pending {
		// Since the initial NC creation is not atomic, the NC record could have been stored but not a pending work item.
		// This could happen for a number of reasons but most likely due to throttling. We check for existence of an
		// associated work item. If one doesn't exist, we attempt to create it and return any errors. Otherwise, we return conflict.
		if err := EnsureNCPendingWorkItem(nc.Config.NetworkID, nc.Config.NetworkContainerID, nc.Config.GetAssociationInfo().NodeID); err != nil {
			return err
		}

		msg := fmt.Sprintf("Network container id %s currently being processed in state %v with status %v. Cannot update while processing",
			nc.Config.NetworkContainerID, nc.Config.State.String(), nc.Config.Status.String())
		// 409 Conflict may not be the most accurate error for this condition, but should signal the client to retry the request later
		return api.Conflict(api.ProcessingPending, msg)
	}

	handling := getHandleLevel(nc.Config.RequestV2, request, nc.Config.Status)
	switch handling {
	case noOp:
		loggershim.LS.Debugf("update request for nc id %s received, nothing to update", nc.Config.NetworkContainerID)
		return nil
	case persistWithoutQueueing:
		nc.Config.RequestV2 = request
		return NcRepo.PersistNetworkContainerData(nc)
	case process:
		nc.Config.State = Prepare
		nc.Config.Status = Pending
		nc.Config.CNSVersion++
		nc.Config.RequestV2 = request
		nc.Config.MarkedForDelete = false // todo: handle this scenario when implementing deletes
		// both nc and pending item operations are forced upserts
		// nc lock should have been taken upstream, so this is safe.
		if err := NcRepo.PersistNetworkContainerData(nc); err != nil {
			return err
		}

		// Add request to pending queue table
		requestContext := RequestContext{
			Type:      NetworkContainerRequest,
			NetworkID: nc.Config.NetworkID,
			NodeID:    nc.Config.GetAssociationInfo().NodeID,
		}
		return AddRequestToPendingQueue(GetNCKey(nc.Config.NetworkID, nc.Config.NetworkContainerID), &requestContext)
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
// that should result in changes to IP allocations, or after a failure.
// Persist if requests are different
func getHandleLevel(processed, requested api.NetworkContainerRequestV2, ncStatus Status) requestHandling {
	if requested.AllocationRequest.SecondaryIPCount != processed.AllocationRequest.SecondaryIPCount || ncStatus == Failed {
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

type ReservationRow struct {
	RowKey        string
	Configuration *ReservationConfiguration
	SubnetKey     string
	SetKey        string
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

		//todo: move this code to reservationrepo and use the utility func to convert between storage entity and reservation
		if prop, ok := entity.Properties[subnetKeyProperty]; ok {
			rows[i].SubnetKey = prop.(string)
		}

		if prop, ok := entity.Properties[setKeyProperty]; ok {
			rows[i].SetKey = prop.(string)
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

// TryQueryAllSubnetConfigsInNetworkConfig - Retrieve all subnet configs in the network
func TryQueryAllSubnetConfigsInNetworkConfig(networkConfig *NetworkConfiguration) (map[string]*SubnetConfiguration, *api.JsonErr) {
	if networkConfig == nil {
		return nil, api.BadRequestf(api.NetworkNotFound, "Null network configuration provided")
	}

	subnetConfigs := make(map[string]*SubnetConfiguration, len(networkConfig.Subnets))
	for subnetName := range networkConfig.Subnets {
		subnetConfig, err := TryQuerySubnetConfiguration(networkConfig.NetworkID, subnetName)
		if err != nil {
			return nil, err
		}

		if subnetConfig != nil {
			subnetConfigs[subnetName] = subnetConfig
		}
	}

	return subnetConfigs, nil
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
	repoNodeConfig, err := NodeConfigRepo.TryQueryNodeConfiguration(nodeID)
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
		OrchestratorType:      repoNodeConfig.NodeInfo.OrchestratorType,
		AZIdOrName:            repoNodeConfig.NodeInfo.AZID}

	// todo: temporary. find better way to skip from publishing to pubsub proxy if orchestrated by k8s crd
	if repoNodeConfig.NodeInfo.OrchestratorType != cns.KubernetesCRD {
		ncGoalState.VlanID = int(repoNodeConfig.NetworkVlanIDMap[repoNCConfig.NetworkID])
	}

	return ncGoalState, false, nil
}

// The function queries CNS running on a node to get number of logical cores configuration data.
// If for some reason the request fails, we return default value of 10.
func GetNCLimitForNode(nodeIP string) (int, *api.JsonErr) {
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

// GetNetworkContainerRequest :- compile and retrieve cns.CreateNetworkContainerRequest to update CNS state
func GetNetworkContainerRequest(ncGoalState *NCGoalState, nc *NetworkContainerEntity) (*cns.CreateNetworkContainerRequest, bool, *api.JsonErr) {
	_, baseSubnetPrefix, err := utils.GetIPAddressStringAndPrefix(ncGoalState.BaseSubnet)
	if err != nil {
		msg := fmt.Sprintf("Failed to parse base subnet string %v due to error %v",
			ncGoalState.BaseSubnet,
			err.Error())
		failureDetail := api.JsonErr{
			Code:       api.InvalidIPSubnet,
			Text:       msg,
			HTTPStatus: http.StatusBadRequest}
		nc.Config.Status = Failed
		nc.Config.FailureDetail = failureDetail
		//todo: persist the failure in the caller rather than doing it here.
		return nil, true, NcRepo.PersistNetworkContainerData(nc)
	}

	if nc.Config.ProgrammedTagsInfo == nil {
		nc.Config.ProgrammedTagsInfo = make(map[string]uint64)
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
		Version:                    strconv.FormatUint(nc.Config.CNSVersion, 10),
		NetworkContainerType:       ncGoalState.ContainerType,
		NetworkContainerid:         prefix + ncGoalState.NetworkContainerID,
		PrimaryInterfaceIdentifier: ncGoalState.AssociatedInterfaceID,
		IPConfiguration:            ipConfig,
		AuthorizationToken:         authToken,
		CnetAddressSpace:           cnsCnetAddresses,
		OrchestratorContext:        ncGoalState.OrchestratorContext,
		MultiTenancyInfo:           cns.MultiTenancyInfo{EncapType: cns.Vlan, ID: ncGoalState.VlanID},
		LocalIPConfiguration:       localIPConfiguaration,
		AllowHostToNCCommunication: nc.Config.GetAllowHostToNCCommunicationStatus(),
		AllowNCToHostCommunication: nc.Config.GetAllowNCToHostCommunicationStatus(),
		EndpointPolicies:           nc.Config.GetEndpointPolicies()}, false, nil
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
	reservations, err := IpReservationRepo.TryQueryAllReservations()
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
				subnetKeyProperty:         subnetKey,
				setKeyProperty:            reservation.SetKey}

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
