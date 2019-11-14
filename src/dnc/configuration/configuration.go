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
}

// OverrideConfigFromEnv - Overrides some static configuration with environment variables
func OverrideConfigFromEnv(config *Configuration) {

	if blobName := os.Getenv("DNCBlobName"); len(blobName) != 0 {
		config.MasterElectionSettings.BlobStorageSettings.BlobName = blobName
	}

	if containerName := os.Getenv("DNCContainerName"); len(containerName) != 0 {
		config.MasterElectionSettings.BlobStorageSettings.ContainerName = containerName
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

	if dataStorePartitionKey := os.Getenv("DataStorePartitionKey"); len(dataStorePartitionKey) != 0 {
		config.DataStoreSettings.AzureStorageInfo.PartitionKey = dataStorePartitionKey
	}

	if masterName := os.Getenv("MasterElectionStorageName"); len(masterName) != 0 {
		config.MasterElectionSettings.BlobStorageSettings.AccountName = masterName
	}

	if masterKey := os.Getenv("MasterElectionStorageKey"); len(masterKey) != 0 {
		config.MasterElectionSettings.BlobStorageSettings.Key = masterKey
	}

	if masterElectionPartitionKey := os.Getenv("MasterElectionPartitionKey"); len(masterElectionPartitionKey) != 0 {
		config.MasterElectionSettings.BlobStorageSettings.PartitionKey = masterElectionPartitionKey
	}
}
