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

// DataStoreSettings - Details related to underlying data store.
type DataStoreSettings struct {
	AzureStorageInfo         AzureStorageInfo
	AzureSQLInfo             AzureSQLInfo
	TestPersistenceStoreInfo TestPersistenceStoreInfo
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

// TestDataElectionSettings - Details related to setting for test election.
type TestDataElectionSettings struct {
	Test string
}

// Configuration - Details related to dnc configuration settings
type Configuration struct {
	OrchestartorType       string
	DataStoreSettings      DataStoreSettings
	ServerSettings         ServerSettings
	MasterElectionSettings MasterElectionSettings
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
		log.Fatalln("Could not parse %q: %v", configFile, err)
	}

	return &config
}
