package replica

import (
	"dnc/api"
	"dnc/configuration"
	"dnc/logger/loggershim"
	"dnc/utils"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

const httpScheme = "http://"

// GetRefreshMonitorStatus
type GetRefreshMonitorStatus func() api.RefreshMonitorStatus

// Replica = Struct to hold replica settings
type Replica struct {
	promoteToPrimaryChannel chan bool
	demotePrimaryChannel    chan bool
	serverSettings          configuration.ServerSettings
	httpClientSettings      configuration.HttpClientSettings
	primaryIPEndpoint       string
	isPrimary               struct{ flag int32 }
	isReady                 struct{ flag int32 }
	mux                     *sync.Mutex
	getRefreshMonitorStatus GetRefreshMonitorStatus
	reverseProxy            *httputil.ReverseProxy
}

// NewReplica - Create a NewReplica
func NewReplica(
	configServerSettings configuration.ServerSettings,
	httpClientSettings configuration.HttpClientSettings,
	getRefreshMonitorStatus GetRefreshMonitorStatus) *Replica {
	replica := new((Replica))
	replica.promoteToPrimaryChannel = make(chan bool)
	replica.demotePrimaryChannel = make(chan bool)
	replica.serverSettings = configServerSettings
	replica.httpClientSettings = httpClientSettings
	replica.mux = new(sync.Mutex)
	replica.getRefreshMonitorStatus = getRefreshMonitorStatus
	return replica
}

// UpdateReplicaStatus - Call back to update replica status.
func (replica *Replica) UpdateReplicaStatus(isReplicaPrimary bool) {
	var i int32
	if isReplicaPrimary {
		i = 1
	}

	atomic.StoreInt32(&(replica.isPrimary.flag), int32(i))
}

// SetPrimaryIPEndpoint - Callback to set primary IP Endpoint.
func (replica *Replica) SetPrimaryIPEndpoint(ipEndpoint string) {
	replica.mux.Lock()
	defer replica.mux.Unlock()
	replica.primaryIPEndpoint = ipEndpoint
	replica.setPrimaryReverseProxy(ipEndpoint)
}

// Set the reverse proxy to the primary to be used by this replica
func (replica *Replica) setPrimaryReverseProxy(ipEndpoint string) {
	target := httpScheme + ipEndpoint
	url, err := url.ParseRequestURI(target)
	if err != nil {
		loggershim.LS.Errorf("Failed to parse target URL when creating reverseproxy with error: %v", err)
		return
	}
	replica.reverseProxy = httputil.NewSingleHostReverseProxy(url)

	transport := &http.Transport{
		MaxIdleConnsPerHost: replica.httpClientSettings.MaxIdleConnsPerHost,
		IdleConnTimeout:     time.Duration(replica.httpClientSettings.IdleConnTimeout) * time.Second,
		DialContext: (&net.Dialer{
			Timeout: time.Duration(replica.httpClientSettings.ConnectionTimeout) * time.Second,
		}).DialContext,
	}
	replica.reverseProxy.Transport = transport

	// This is based on golang's reverseproxy, default error handler, only difference is
	// it logs using our custom logger, rather standard one.
	// https://golang.org/src/net/http/httputil/reverseproxy.go?3655:3716#L159
	replica.reverseProxy.ErrorHandler = func(rw http.ResponseWriter, r *http.Request, err error) {
		// Assumption is if this is called, error will not be nil
		// same as library implementation.
		loggershim.LS.Errorf("HTTP: Proxy error : %+v", err)
		rw.WriteHeader(http.StatusBadGateway)
		return
	}
}

// Get the reverse proxy to the primary used by this replica
func (replica *Replica) GetPrimaryReverseProxy() *httputil.ReverseProxy {
	replica.mux.Lock()
	defer replica.mux.Unlock()
	return replica.reverseProxy
}

// GetPrimaryIPEndpoint - Get primary IP Endpoint.
func (replica *Replica) GetPrimaryIPEndpoint() (string, error) {
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

// IsReplicaReady - Function to check if this replica is ready
func (replica *Replica) IsReplicaReady() bool {
	return atomic.LoadInt32(&(replica.isReady.flag)) != 0
}

// UpdateReadyStatus - Function to update replica ready flag
func (replica *Replica) UpdateReadyStatus(isReplicaReady bool) {
	var i int32
	if isReplicaReady {
		i = 1
	}

	atomic.StoreInt32(&(replica.isReady.flag), int32(i))
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
			loggershim.LS.Errorf("Failed to retrieve IP Address due to error: %s", err.Error())
			return "", err
		}
	}

	return ipAddress, nil
}

// GetRefreshMonitorStatus - Retrieve refresh monitor status.
func (replica *Replica) GetRefreshMonitorStatus() api.RefreshMonitorStatus {
	return replica.getRefreshMonitorStatus()
}
