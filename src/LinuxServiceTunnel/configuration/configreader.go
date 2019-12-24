package configuration

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Azure/azure-container-networking/log"
)

const (
	primaryIfName = "eth0"
	bridgeName    = "azure0"
	veth1Name     = "azstveth0"
	veth2Name     = "azstveth1"
)

type ServiceTunnelConfig struct {
	PortsToOpen          []int
	PrimaryInterfaceName string
	BridgeName           string
	Veth1Name            string
	Veth2Name            string
}

// Read servicetunnel config file and save it in an object
func ReadConfigFile(fileName string) (ServiceTunnelConfig, error) {
	var stConfig ServiceTunnelConfig

	// Get config path from env variable. If not, set config path to process directory
	configpath, found := os.LookupEnv("AZST_CONFIG_PATH")
	if !found {
		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			log.Errorf("[Configuration] Failed to find exe dir:%v", err)
			return stConfig, err
		}

		configpath = filepath.Join(dir, fileName)
	}

	log.Printf("config path: %v", configpath)

	// Read config file
	readBytes, err := ioutil.ReadFile(configpath)
	if err != nil {
		log.Errorf("Reading HandlerEnv failed with error %v", err)
		return stConfig, err
	}

	if err := json.Unmarshal(readBytes, &stConfig); err != nil {
		log.Errorf("Unmarshalling failed with err %v", err)
		return stConfig, err
	}

	return stConfig, nil
}

// Set default servicetunnel config values
func SetConfigDefaults(stConfig *ServiceTunnelConfig) {
	if stConfig.PrimaryInterfaceName == "" {
		stConfig.PrimaryInterfaceName = primaryIfName
	}

	if stConfig.BridgeName == "" {
		stConfig.BridgeName = bridgeName
	}

	if stConfig.Veth1Name == "" {
		stConfig.Veth1Name = veth1Name
	}

	if stConfig.Veth2Name == "" {
		stConfig.Veth2Name = veth2Name
	}
}
