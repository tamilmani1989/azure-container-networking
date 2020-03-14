package configuration

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

// Features supported by underlying data storage
const (
	SharedSubnetDelegation      = 0
	RegionalStorageAccountUsage = 1
)

// AzureStorageInfo - Details to connect to azure storage account.
type AzureStorageInfo struct {
	AccountName                     string
	Key                             string
	PartitionKey                    string
	EnableSharedSubnetDelegation    bool
	RegionalStorageConnectionString string
	Environment                     string
}

// AzureSQLInfo - Details to connect to azure sql database.
type AzureSQLInfo struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
}

// TestPersistenceStoreInfo - Details to initialize testpersistencestore
type TestPersistenceStoreInfo struct {
	Test                         string
	EnableSharedSubnetDelegation bool
}

// OrchestratorStorageInfo - Details to initialize orchestratorstorage
type OrchestratorStorageInfo struct {
	Dll string
}

// DataStoreSettings - Details related to underlying data store.
type DataStoreSettings struct {
	AzureStorageInfo         AzureStorageInfo
	AzureSQLInfo             AzureSQLInfo
	TestPersistenceStoreInfo TestPersistenceStoreInfo
	OrchestratorStorageInfo  OrchestratorStorageInfo
}

// ServerSettings - Details related to listener settings.
type ServerSettings struct {
	IPAddress string
	Port      string
}

// BlobStorageSettings - Details related to blob storage settings.
type BlobStorageSettings struct {
	AzureStorageInfo
	BlobName                   string
	ContainerName              string
	LeaseIntervalInSecs        int64
	LeaseRenewIntervalInSecs   int64
	AcquireRetryIntervalInSecs int64
}

// MasterElectionSettings - Details related to settings for master election.
type MasterElectionSettings struct {
	BlobStorageSettings      BlobStorageSettings
	TestDataElectionSettings TestDataElectionSettings
}

// CustomLoggerSettings :- Details related to settings for custom logging.
type CustomLoggerSettings struct {
	FnName string
	Dll    string
}

// TestDataElectionSettings - Details related to setting for test election.
type TestDataElectionSettings struct {
	Test string
}

// HostAgentConnectionStrings NMagent connection strings
type HostAgentConnectionStrings struct {
	JoinNetworkURLFmt             string
	GetNetworkConfigurationURLFmt string
	PutNetworkContainerURLFmt     string
	DeleteNetworkContainerURLFmt  string
	GetSystemTagDefinitionURLFmt  string
}

// HostAgentSettings NMAgent connection URL settings.
type HostAgentSettings struct {
	WireServerIP      string
	ConnectionStrings HostAgentConnectionStrings
}

// CNSConnectionStrings CNS connection URL strings
type CNSConnectionStrings struct {
	CreateOrUpdateNetworkContainerURLFmt string
	DeleteNetworkContainerURLFmt         string
	GetNetworkContainerStatusURLFmt      string
	SetOrchestratorTypeURLFmt            string
	GetNetworkcontainerstatusURLFmt      string
	GetNumOfCPUCoresURLFmt               string
	PublishNetworkContainerURLFmt        string
	UnpublishNetworkContainerURLFmt      string
}

// CNSSettings CNS settings
type CNSSettings struct {
	ConnectionStrings CNSConnectionStrings
}

// VlanSettings - VLAN inventory
type VlanSettings struct {
	Min uint16
	Max uint16
}

// DNCSettings DNC configuration settings
type DNCSettings struct {
	// Base Back off interval on failure to process a request. Default 1 sec
	RequestRetryBaseInterval int
	// Max time to back off on failure to process a request. Default 16 sec
	MaxRequestBackoffInterval int
	// Network refresh monitor interval. Default 5 mins
	NetworkRefreshInterval int
	// Count of networks to be refreshed in a batch. Default 45
	NetworkRefreshBatchSize int
	// Time interval to sleep between refresh of network batches. Default 1 min
	SleepIntervalBetweenNetworkBatches int
	// Time interval to sleep between refresh of network. Default 300 milliseconds
	SleepIntervalBetweenNetworkRefresh int
	// System tag refresh interval. Defualt 1 min.
	TagRefreshInterval int
	// Refresh interval for sending goal states to CNS for an NC. Default 1 min.
	NetworkContainerRefreshInterval int
	// Refresh interval for sending batch reports to host. Default 1 min.
	ReportToHostInterval int
	// Concurrent request to process in a given batch. Defualt 5.
	ConcurrentRequestsToProcess int
	// Flag to determine, if DNC should enforce NIC limit or not?
	EnforceNICLimit bool
	// Number of NICs that are allowed per virtual core
	NICsAllowedPerVirtualCore uint16
	// Flag to determine whether DNC should forward requests instead of relying on client to handle redirects
	ReverseProxyEnabled bool
	// Max time latency allowed for data store operations before recording the operation latency in the log.
	ThresholdForLoggingDataStoreOpLatencyMs int
}

// HttpClientSettings - Http client settings to be used when making http calls to CNS and NMAgent from DNC
type HttpClientSettings struct {
	// ConnectionTimeout indicates the timeout used to establish the connection
	ConnectionTimeout int
	// ResponseHeaderTimeout indicates the timeout used to get the header
	ResponseHeaderTimeout int
}

// TelemetrySettings DNC telemetry settings
type TelemetrySettings struct {
	// Flag to disable the telemetry. Default is false
	DisableTelemetry bool
	// Configure how many bytes can be sent in one call to the data collector
	TelemetryBatchSizeBytes int
	// Configure the maximum delay before sending queued telemetry in milliseconds
	TelemetryBatchIntervalInSecs int
	// Enable thread for getting metadata from wireserver
	DisableMetadataRefreshThread bool
	// Refresh interval in milliseconds for metadata thread
	RefreshIntervalInSecs int
	// Disable debug logging for telemetry messages
	DebugMode bool
	// Disable logging to App Insights.
	DisableAiLogging bool
	// Retry wait time in seconds for AI telemetry.
	GetEnvRetryWaitTimeInSecs int
	// Retry count for AI telemetry.
	GetEnvRetryCount int
}

// Configuration - Details related to dnc configuration settings
type Configuration struct {
	OrchestartorType       string
	DataStoreSettings      DataStoreSettings
	ServerSettings         ServerSettings
	MasterElectionSettings MasterElectionSettings
	CustomLoggerSettings   CustomLoggerSettings
	HostAgentSettings      HostAgentSettings
	CNSSettings            CNSSettings
	VlanSettings           VlanSettings
	DNCSettings            DNCSettings
	HttpClientSettings     HttpClientSettings
	TelemetrySettings      TelemetrySettings
}

// Load the JSON config file
func Load(configFile string) *Configuration {
	var config Configuration
	var err error
	var absPath string
	var input = io.ReadCloser(os.Stdin)
	if absPath, err = filepath.Abs(configFile); err != nil {
		log.Fatalln(err)
	}

	if input, err = os.Open(absPath); err != nil {
		log.Fatalln(err)
	}

	// Read the config file
	jsonBytes, err := ioutil.ReadAll(input)
	input.Close()
	if err != nil {
		log.Fatalln(err)
	}

	// Parse the config
	if err := json.Unmarshal(jsonBytes, &config); err != nil {
		log.Fatalf("Could not parse %s: %v", configFile, err)
	}

	return &config
}

func SetDncConfigDefaults(config *Configuration) {
	setDNCSettingDefaults(&config.DNCSettings)
	setHttpSettingDefaults(&config.HttpClientSettings)
	setTelemetrySettingDefaults(&config.TelemetrySettings)
	setBlobStorageSettings(&config.MasterElectionSettings.BlobStorageSettings)
}

func setDNCSettingDefaults(dncSettings *DNCSettings) {
	if dncSettings.RequestRetryBaseInterval == 0 {
		// Set default to 1 sec.
		dncSettings.RequestRetryBaseInterval = 1000
	}

	if dncSettings.ConcurrentRequestsToProcess == 0 {
		// Set default to 5
		dncSettings.ConcurrentRequestsToProcess = 5
	}

	if dncSettings.MaxRequestBackoffInterval == 0 {
		// Set default to 16 secs
		dncSettings.MaxRequestBackoffInterval = 16000
	}

	if dncSettings.NetworkRefreshInterval == 0 {
		// Set default to 1 min.
		dncSettings.NetworkRefreshInterval = 60000
	}

	if dncSettings.TagRefreshInterval == 0 {
		// Set default to 1 min.
		dncSettings.TagRefreshInterval = 60000
	}

	if dncSettings.NetworkContainerRefreshInterval == 0 {
		// Set default to 1 min
		dncSettings.NetworkContainerRefreshInterval = 60000
	}

	if dncSettings.NetworkRefreshBatchSize == 0 {
		// Set default to 60
		dncSettings.NetworkRefreshBatchSize = 60
	}

	if dncSettings.SleepIntervalBetweenNetworkBatches == 0 {
		// Set default to 1 min
		dncSettings.SleepIntervalBetweenNetworkBatches = 60000
	}

	if dncSettings.ReportToHostInterval < 60000 {
		// Set default to 1 min
		dncSettings.ReportToHostInterval = 60000
	}

	if dncSettings.SleepIntervalBetweenNetworkRefresh == 0 {
		// Set default to 1 min
		dncSettings.SleepIntervalBetweenNetworkRefresh = 300
	}

	if dncSettings.NICsAllowedPerVirtualCore == 0 {
		// Set default to 3
		dncSettings.SleepIntervalBetweenNetworkRefresh = 3
	}

	if dncSettings.ThresholdForLoggingDataStoreOpLatencyMs == 0 {
		// Set default to 1 second
		dncSettings.ThresholdForLoggingDataStoreOpLatencyMs = 1000
	}
}

func setHttpSettingDefaults(httpClientSettings *HttpClientSettings) {
	if httpClientSettings.ConnectionTimeout == 0 {
		// set the default connection timeout to 5 seconds
		httpClientSettings.ConnectionTimeout = 5
	}

	if httpClientSettings.ResponseHeaderTimeout == 0 {
		// set the default response header timeout to 120 seconds
		httpClientSettings.ResponseHeaderTimeout = 120
	}
}

func setTelemetrySettingDefaults(telemetrySettings *TelemetrySettings) {
	if telemetrySettings.RefreshIntervalInSecs == 0 {
		// set the default refresh interval of metadata thread to 15 seconds
		telemetrySettings.RefreshIntervalInSecs = 15
	}

	if telemetrySettings.TelemetryBatchIntervalInSecs == 0 {
		// set the default AI telemetry batch interval to 7 seconds
		telemetrySettings.TelemetryBatchIntervalInSecs = 7
	}

	if telemetrySettings.TelemetryBatchSizeBytes == 0 {
		// set the default AI telemetry batch size to 16384 bytes
		telemetrySettings.TelemetryBatchSizeBytes = 16384
	}

	if telemetrySettings.GetEnvRetryWaitTimeInSecs == 0 {
		telemetrySettings.GetEnvRetryWaitTimeInSecs = 3
	}

	if telemetrySettings.GetEnvRetryCount == 0 {
		telemetrySettings.GetEnvRetryCount = 5
	}
}

func setBlobStorageSettings(blobStorageSettings *BlobStorageSettings) {
	// The max lease interval supported by blob is 60 seconds
	// https://docs.microsoft.com/en-us/rest/api/storageservices/lease-blob
	// Cap the lease interval to 60 seconds
	if blobStorageSettings.LeaseIntervalInSecs > 60 {
		blobStorageSettings.LeaseIntervalInSecs = 60
	}
}

// OverrideConfigFromEnv - Overrides some static configuration with environment variables
func OverrideConfigFromEnv(config *Configuration) {

	if blobName := os.Getenv("DNCBlobName"); len(blobName) != 0 {
		config.MasterElectionSettings.BlobStorageSettings.BlobName = blobName
	}

	if containerName := os.Getenv("DNCContainerName"); len(containerName) != 0 {
		config.MasterElectionSettings.BlobStorageSettings.ContainerName = containerName
	}

	if ipAddress := os.Getenv("DNCIpAddress"); len(ipAddress) != 0 {
		config.ServerSettings.IPAddress = ipAddress
	}

	if port := os.Getenv("DNCPort"); len(port) != 0 {
		config.ServerSettings.Port = port
	}

	if accountName := os.Getenv("DataStoreStorageName"); len(accountName) != 0 {
		config.DataStoreSettings.AzureStorageInfo.AccountName = accountName
	}

	if accountKey := os.Getenv("DataStoreStorageKey"); len(accountKey) != 0 {
		config.DataStoreSettings.AzureStorageInfo.Key = accountKey
	}

	if accountEnv := os.Getenv("DataStoreStorageEnvironment"); len(accountEnv) != 0 {
		config.DataStoreSettings.AzureStorageInfo.Environment = accountEnv
	}

	if dataStorePartitionKey := os.Getenv("DataStorePartitionKey"); len(dataStorePartitionKey) != 0 {
		config.DataStoreSettings.AzureStorageInfo.PartitionKey = dataStorePartitionKey
	}

	if masterName := os.Getenv("MasterElectionStorageName"); len(masterName) != 0 {
		config.MasterElectionSettings.BlobStorageSettings.AccountName = masterName
	}

	if masterKey := os.Getenv("MasterElectionStorageKey"); len(masterKey) != 0 {
		config.MasterElectionSettings.BlobStorageSettings.Key = masterKey
	}

	if masterEnv := os.Getenv("MasterElectionStorageEnvironment"); len(masterEnv) != 0 {
		config.MasterElectionSettings.BlobStorageSettings.Environment = masterEnv
	}

	if masterElectionPartitionKey := os.Getenv("MasterElectionPartitionKey"); len(masterElectionPartitionKey) != 0 {
		config.MasterElectionSettings.BlobStorageSettings.PartitionKey = masterElectionPartitionKey
	}

	if leaseIntervalInSecs := os.Getenv("LeaseIntervalInSecs"); len(leaseIntervalInSecs) != 0 {
		config.MasterElectionSettings.BlobStorageSettings.LeaseIntervalInSecs = ParseInt64(leaseIntervalInSecs)
	}

	if leaseRenewIntervalInSecs := os.Getenv("LeaseRenewIntervalInSecs"); len(leaseRenewIntervalInSecs) != 0 {
		config.MasterElectionSettings.BlobStorageSettings.LeaseRenewIntervalInSecs = ParseInt64(leaseRenewIntervalInSecs)
	}

	if acquireRetryIntervalInSecs := os.Getenv("AcquireRetryIntervalInSecs"); len(acquireRetryIntervalInSecs) != 0 {
		config.MasterElectionSettings.BlobStorageSettings.AcquireRetryIntervalInSecs = ParseInt64(acquireRetryIntervalInSecs)
	}

	if requestRetryBaseInterval := os.Getenv("RequestRetryBaseInterval"); len(requestRetryBaseInterval) != 0 {
		config.DNCSettings.RequestRetryBaseInterval = ParseInt(requestRetryBaseInterval)
	}

	if maxRequestBackoffInterval := os.Getenv("MaxRequestBackoffInterval"); len(maxRequestBackoffInterval) != 0 {
		config.DNCSettings.MaxRequestBackoffInterval = ParseInt(maxRequestBackoffInterval)
	}

	if networkRefreshInterval := os.Getenv("NetworkRefreshInterval"); len(networkRefreshInterval) != 0 {
		config.DNCSettings.NetworkRefreshInterval = ParseInt(networkRefreshInterval)
	}

	if networkRefreshBatchSize := os.Getenv("NetworkRefreshBatchSize"); len(networkRefreshBatchSize) != 0 {
		config.DNCSettings.NetworkRefreshBatchSize = ParseInt(networkRefreshBatchSize)
	}

	if sleepIntervalBetweenNetworkBatches := os.Getenv("SleepIntervalBetweenNetworkBatches"); len(sleepIntervalBetweenNetworkBatches) != 0 {
		config.DNCSettings.SleepIntervalBetweenNetworkBatches = ParseInt(sleepIntervalBetweenNetworkBatches)
	}

	if sleepIntervalBetweenNetworkRefresh := os.Getenv("SleepIntervalBetweenNetworkRefresh"); len(sleepIntervalBetweenNetworkRefresh) != 0 {
		config.DNCSettings.SleepIntervalBetweenNetworkRefresh = ParseInt(sleepIntervalBetweenNetworkRefresh)
	}

	if networkContainerRefreshInterval := os.Getenv("NetworkContainerRefreshInterval"); len(networkContainerRefreshInterval) != 0 {
		config.DNCSettings.NetworkContainerRefreshInterval = ParseInt(networkContainerRefreshInterval)
	}

	if tagRefreshInterval := os.Getenv("TagRefreshInterval"); len(tagRefreshInterval) != 0 {
		config.DNCSettings.TagRefreshInterval = ParseInt(tagRefreshInterval)
	}

	if reportToHostInterval := os.Getenv("ReportToHostInterval"); len(reportToHostInterval) != 0 {
		config.DNCSettings.ReportToHostInterval = ParseInt(reportToHostInterval)
	}

	if concurrentRequestsToProcess := os.Getenv("ConcurrentRequestsToProcess"); len(concurrentRequestsToProcess) != 0 {
		config.DNCSettings.ConcurrentRequestsToProcess = ParseInt(concurrentRequestsToProcess)
	}

	if enforceNICLimit := os.Getenv("EnforceNICLimit"); len(enforceNICLimit) != 0 {
		config.DNCSettings.EnforceNICLimit = ParseBool(enforceNICLimit)
	}

	if nicsAllowedPerVirtualCore := os.Getenv("NICsAllowedPerVirtualCore"); len(nicsAllowedPerVirtualCore) != 0 {
		config.DNCSettings.NICsAllowedPerVirtualCore = ParseUInt16(nicsAllowedPerVirtualCore)
	}

	if reverseProxyEnabled := os.Getenv("ReverseProxyEnabled"); len(reverseProxyEnabled) != 0 {
		config.DNCSettings.ReverseProxyEnabled = ParseBool(reverseProxyEnabled)
	}

	if connectionTimeout := os.Getenv("ConnectionTimeout"); len(connectionTimeout) != 0 {
		config.HttpClientSettings.ConnectionTimeout = ParseInt(connectionTimeout)
	}

	if responseHeaderTimeout := os.Getenv("ResponseHeaderTimeout"); len(responseHeaderTimeout) != 0 {
		config.HttpClientSettings.ResponseHeaderTimeout = ParseInt(responseHeaderTimeout)
	}

	if telemetryBatchSizeBytes := os.Getenv("TelemetryBatchSizeBytes"); len(telemetryBatchSizeBytes) != 0 {
		config.TelemetrySettings.TelemetryBatchSizeBytes = ParseInt(telemetryBatchSizeBytes)
	}

	if telemetryBatchIntervalInSecs := os.Getenv("TelemetryBatchIntervalInSecs"); len(telemetryBatchIntervalInSecs) != 0 {
		config.TelemetrySettings.TelemetryBatchIntervalInSecs = ParseInt(telemetryBatchIntervalInSecs)
	}

	if refreshIntervalInSecs := os.Getenv("RefreshIntervalInSecs"); len(refreshIntervalInSecs) != 0 {
		config.TelemetrySettings.RefreshIntervalInSecs = ParseInt(refreshIntervalInSecs)
	}

	if disableMetadataRefreshThread := os.Getenv("DisableMetadataRefreshThread"); len(disableMetadataRefreshThread) != 0 {
		config.TelemetrySettings.DisableMetadataRefreshThread = ParseBool(disableMetadataRefreshThread)
	}

	if disableTelemetry := os.Getenv("DisableTelemetry"); len(disableTelemetry) != 0 {
		config.TelemetrySettings.DisableTelemetry = ParseBool(disableTelemetry)
	}

	if debugMode := os.Getenv("DebugMode"); len(debugMode) != 0 {
		config.TelemetrySettings.DebugMode = ParseBool(debugMode)
	}

	if disableAiLogging := os.Getenv("DisableAiLogging"); len(disableAiLogging) != 0 {
		config.TelemetrySettings.DisableAiLogging = ParseBool(disableAiLogging)
	}

	if getEnvRetryWaitTimeInSecs := os.Getenv("GetEnvRetryWaitTimeInSecs"); len(getEnvRetryWaitTimeInSecs) != 0 {
		config.TelemetrySettings.GetEnvRetryWaitTimeInSecs = ParseInt(getEnvRetryWaitTimeInSecs)
	}

	if getEnvRetryCount := os.Getenv("GetEnvRetryCount"); len(getEnvRetryCount) != 0 {
		config.TelemetrySettings.GetEnvRetryCount = ParseInt(getEnvRetryCount)
	}

	if thresholdForLoggingDataStoreOpLatencyMs :=
		os.Getenv("ThresholdForLoggingDataStoreOpLatencyMs"); len(thresholdForLoggingDataStoreOpLatencyMs) != 0 {
		config.DNCSettings.ThresholdForLoggingDataStoreOpLatencyMs = ParseInt(thresholdForLoggingDataStoreOpLatencyMs)
	}
}
