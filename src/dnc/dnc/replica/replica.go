package replica

import (
	"fmt"
	"sync"
	"log"
	"dnc/utils"
	"dnc/configuration"
	"sync/atomic"
)

// Replica = Struct to hold replica settings
type Replica  struct {
	promoteToPrimaryChannel chan bool
	demotePrimaryChannel chan bool
	serverSettings configuration.ServerSettings
	primaryIPEndpoint string
	isPrimary struct{ flag int32 }	
	mux *sync.Mutex
}

// NewReplica - Create a NewReplica
func NewReplica(configServerSettings configuration.ServerSettings) (*Replica) {
	replica := new((Replica) )
	replica.promoteToPrimaryChannel = make(chan bool)
	replica.demotePrimaryChannel = make(chan bool)
	replica.serverSettings = configServerSettings
	replica.mux = new(sync.Mutex)
	return replica
}

// UpdateReplicaStatus - Call back to update replica status.
func (replica *Replica ) UpdateReplicaStatus(isReplicaPrimary bool) {
	var i int32
	if isReplicaPrimary {
		i = 1
	}
	atomic.StoreInt32(&(replica.isPrimary.flag), int32(i))
}
// SetPrimaryIPEndpoint - Callback to set primary IP Endpoint.
func (replica *Replica ) SetPrimaryIPEndpoint(ipEndpoint string ) {
	replica.mux.Lock()
	replica.primaryIPEndpoint = ipEndpoint
	replica.mux.Unlock()
}

// GetPrimaryIPEndpoint - Get primary IP Endpoint.
func (replica *Replica ) GetPrimaryIPEndpoint() (string, error) {
	replica.mux.Lock()
	defer replica.mux.Unlock()
	if replica.primaryIPEndpoint == "" {
		return "", fmt.Errorf("Failed to retrieve primary IP endpoint")
	}
	
	return replica.primaryIPEndpoint, nil
}

// IsReplicaPrimary - Function to check if this replica is priamry
func (replica *Replica) IsReplicaPrimary() bool {
	if atomic.LoadInt32(&(replica.isPrimary.flag)) != 0 {
		return true
	}
	return false
}

// GetReplicaIPEndpoint - Method to check if the given replica is primary.
func (replica *Replica) GetReplicaIPEndpoint() (string, error) {
	replicaIP, err := replica.getReplicaIPAddress()
	if err != nil {
		return "", err
	}

	return replicaIP + ":" + replica.serverSettings.Port, nil

}

func (replica *Replica) getReplicaIPAddress() (string, error) {
	ipAddress := replica.serverSettings.IPAddress
	var err error
	if ipAddress == "" {
		ipAddress, err = utils.GetLocalIP()
		if err != nil {
			log.Println("Failed to retrieve IP Address")
			return "", err
		}
	}

	return ipAddress, nil
}
