package configuration

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

// AzureStorageInfo - Details to connect to azure storage account.
type AzureStorageInfo struct {
	AccountName  string
	Key          string
	PartitionKey string
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
	Test string
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

type HostAgentConnectionStrings struct {
	JoinNetworkURLFmt             string
	GetNetworkConfigurationURLFmt string
	PutNetworkContainerURLFmt     string
	DeleteNetworkContainerURLFmt  string
	GetSystemTagDefinitionURLFmt  string
}

type HostAgentSettings struct {
	WireServerIP      string
	ConnectionStrings HostAgentConnectionStrings
}

type CNSConnectionStrings struct {
	CreateOrUpdateNetworkContainerURLFmt string
	DeleteNetworkContainerURLFmt         string
	GetNetworkContainerStatusURLFmt      string
	SetOrchestratorTypeURLFmt            string
	GetNetworkcontainerstatusURLFmt      string
}

type CNSSettings struct {
	ConnectionStrings CNSConnectionStrings
}

type VlanSettings struct {
	Min uint16
	Max uint16
}

type DNCSettings struct {
	RequestRetryBaseInterval        int
	NetworkRefreshInterval          int
	NetworkContainerRefreshInterval int
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
		log.Fatalf("Could not parse %s: %v\n", configFile, err)
	}

	return &config
}
