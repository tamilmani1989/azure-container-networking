// Copyright 2017 Microsoft. All rights reserved.
// MIT License

package restserver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-container-networking/aitelemetry"
	"github.com/Azure/azure-container-networking/cns"
	"github.com/Azure/azure-container-networking/cns/common"
	"github.com/Azure/azure-container-networking/cns/dockerclient"
	"github.com/Azure/azure-container-networking/cns/hnsclient"
	"github.com/Azure/azure-container-networking/cns/imdsclient"
	"github.com/Azure/azure-container-networking/cns/ipamclient"
	"github.com/Azure/azure-container-networking/cns/logger"
	"github.com/Azure/azure-container-networking/cns/networkcontainers"
	"github.com/Azure/azure-container-networking/cns/nmagentclient"
	"github.com/Azure/azure-container-networking/cns/routes"
	acn "github.com/Azure/azure-container-networking/common"
	"github.com/Azure/azure-container-networking/platform"
	"github.com/Azure/azure-container-networking/store"
)

var (
	// Named Lock for accessing different states in httpRestServiceState
	namedLock = acn.InitNamedLock()
)

const (
	// Key against which CNS state is persisted.
	storeKey        = "ContainerNetworkService"
	swiftAPIVersion = "1"
	attach          = "Attach"
	detach          = "Detach"
	// Rest service state identifier for named lock
	stateJoinedNetworks = "JoinedNetworks"
)

// HTTPRestService represents http listener for CNS - Container Networking Service.
type HTTPRestService struct {
	*cns.Service
	dockerClient     *dockerclient.DockerClient
	imdsClient       *imdsclient.ImdsClient
	ipamClient       *ipamclient.IpamClient
	networkContainer *networkcontainers.NetworkContainers
	ContainerIPIDByOrchestratorContext map[string]string                 // OrchestratorContext is key and value is SecondaryIPID(uuid).
	ContainerIPConfigState             map[string]containerIPConfigState // seondaryipid(uuid) is key
	AllocatedIPCount                   map[string]struct{int} // key - ncid
	routingTable     *routes.RoutingTable
	store            store.KeyValueStore
	state            *httpRestServiceState
	lock             sync.Mutex
	dncPartitionKey  string
}

// containerstatus is used to save status of an existing container
type containerstatus struct {
	ID                            string
	VMVersion                     string
	HostVersion                   string
	CreateNetworkContainerRequest cns.CreateNetworkContainerRequest
}

// httpRestServiceState contains the state we would like to persist.
type httpRestServiceState struct {
	Location                           string
	NetworkType                        string
	OrchestratorType                   string
	NodeID                             string
	Initialized                        bool
	ContainerIDByOrchestratorContext   map[string]string          // OrchestratorContext is key and value is NetworkContainerID.
	ContainerStatus                    map[string]containerstatus        // NetworkContainerID is key.
	Networks                           map[string]*networkInfo
	TimeStamp                          time.Time
	joinedNetworks                     map[string]struct{}
}

type networkInfo struct {
	NetworkName string
	NicInfo     *imdsclient.InterfaceInfo
	Options     map[string]interface{}
}

// HTTPService describes the min API interface that every service should have.
type HTTPService interface {
	common.ServiceAPI
	SendNCSnapShotPeriodically(int, chan bool)
}

// NewHTTPRestService creates a new HTTP Service object.
func NewHTTPRestService(config *common.ServiceConfig) (HTTPService, error) {
	service, err := cns.NewService(config.Name, config.Version, config.Store)
	if err != nil {
		return nil, err
	}

	imdsClient := &imdsclient.ImdsClient{}
	routingTable := &routes.RoutingTable{}
	nc := &networkcontainers.NetworkContainers{}
	dc, err := dockerclient.NewDefaultDockerClient(imdsClient)

	if err != nil {
		return nil, err
	}

	ic, err := ipamclient.NewIpamClient("")
	if err != nil {
		return nil, err
	}

	serviceState := &httpRestServiceState{}
	serviceState.Networks = make(map[string]*networkInfo)
	serviceState.joinedNetworks = make(map[string]struct{})

	return &HTTPRestService{
		Service:          service,
		store:            service.Service.Store,
		dockerClient:     dc,
		imdsClient:       imdsClient,
		ipamClient:       ic,
		networkContainer: nc,
		routingTable:     routingTable,
		state:            serviceState,
	}, nil
}

// Start starts the CNS listener.
func (service *HTTPRestService) Start(config *common.ServiceConfig) error {

	err := service.Initialize(config)
	if err != nil {
		logger.Errorf("[Azure CNS]  Failed to initialize base service, err:%v.", err)
		return err
	}

	err = service.restoreState()
	if err != nil {
		logger.Errorf("[Azure CNS]  Failed to restore service state, err:%v.", err)
		return err
	}

	err = service.restoreNetworkState()
	if err != nil {
		logger.Errorf("[Azure CNS]  Failed to restore network state, err:%v.", err)
		return err
	}

	// Add handlers.
	listener := service.Listener
	// default handlers
	listener.AddHandler(cns.SetEnvironmentPath, service.setEnvironment)
	listener.AddHandler(cns.CreateNetworkPath, service.createNetwork)
	listener.AddHandler(cns.DeleteNetworkPath, service.deleteNetwork)
	listener.AddHandler(cns.ReserveIPAddressPath, service.reserveIPAddress)
	listener.AddHandler(cns.ReleaseIPAddressPath, service.releaseIPAddress)
	listener.AddHandler(cns.GetHostLocalIPPath, service.getHostLocalIP)
	listener.AddHandler(cns.GetIPAddressUtilizationPath, service.getIPAddressUtilization)
	listener.AddHandler(cns.GetUnhealthyIPAddressesPath, service.getUnhealthyIPAddresses)
	listener.AddHandler(cns.CreateOrUpdateNetworkContainer, service.createOrUpdateNetworkContainer)
	listener.AddHandler(cns.DeleteNetworkContainer, service.deleteNetworkContainer)
	listener.AddHandler(cns.GetNetworkContainerStatus, service.getNetworkContainerStatus)
	listener.AddHandler(cns.GetInterfaceForContainer, service.getInterfaceForContainer)
	listener.AddHandler(cns.SetOrchestratorType, service.setOrchestratorType)
	listener.AddHandler(cns.GetNetworkContainerByOrchestratorContext, service.getNetworkContainerByOrchestratorContext)
	listener.AddHandler(cns.AttachContainerToNetwork, service.attachNetworkContainerToNetwork)
	listener.AddHandler(cns.DetachContainerFromNetwork, service.detachNetworkContainerFromNetwork)
	listener.AddHandler(cns.CreateHnsNetworkPath, service.createHnsNetwork)
	listener.AddHandler(cns.DeleteHnsNetworkPath, service.deleteHnsNetwork)
	listener.AddHandler(cns.NumberOfCPUCoresPath, service.getNumberOfCPUCores)
	listener.AddHandler(cns.CreateHostNCApipaEndpointPath, service.createHostNCApipaEndpoint)
	listener.AddHandler(cns.DeleteHostNCApipaEndpointPath, service.deleteHostNCApipaEndpoint)
	listener.AddHandler(cns.PublishNetworkContainer, service.publishNetworkContainer)
	listener.AddHandler(cns.UnpublishNetworkContainer, service.unpublishNetworkContainer)
	listener.AddHandler(cns.AllocateIPConfig, service.allocateIPConfig)
	listener.AddHandler(cns.ReleaseIPConfig, service.releaseIPConfig)
	listener.AddHandler(cns.ReconcileState, service.releaseIPConfig)

	// handlers for v0.2
	listener.AddHandler(cns.V2Prefix+cns.SetEnvironmentPath, service.setEnvironment)
	listener.AddHandler(cns.V2Prefix+cns.CreateNetworkPath, service.createNetwork)
	listener.AddHandler(cns.V2Prefix+cns.DeleteNetworkPath, service.deleteNetwork)
	listener.AddHandler(cns.V2Prefix+cns.ReserveIPAddressPath, service.reserveIPAddress)
	listener.AddHandler(cns.V2Prefix+cns.ReleaseIPAddressPath, service.releaseIPAddress)
	listener.AddHandler(cns.V2Prefix+cns.GetHostLocalIPPath, service.getHostLocalIP)
	listener.AddHandler(cns.V2Prefix+cns.GetIPAddressUtilizationPath, service.getIPAddressUtilization)
	listener.AddHandler(cns.V2Prefix+cns.GetUnhealthyIPAddressesPath, service.getUnhealthyIPAddresses)
	listener.AddHandler(cns.V2Prefix+cns.CreateOrUpdateNetworkContainer, service.createOrUpdateNetworkContainer)
	listener.AddHandler(cns.V2Prefix+cns.DeleteNetworkContainer, service.deleteNetworkContainer)
	listener.AddHandler(cns.V2Prefix+cns.GetNetworkContainerStatus, service.getNetworkContainerStatus)
	listener.AddHandler(cns.V2Prefix+cns.GetInterfaceForContainer, service.getInterfaceForContainer)
	listener.AddHandler(cns.V2Prefix+cns.SetOrchestratorType, service.setOrchestratorType)
	listener.AddHandler(cns.V2Prefix+cns.GetNetworkContainerByOrchestratorContext, service.getNetworkContainerByOrchestratorContext)
	listener.AddHandler(cns.V2Prefix+cns.AttachContainerToNetwork, service.attachNetworkContainerToNetwork)
	listener.AddHandler(cns.V2Prefix+cns.DetachContainerFromNetwork, service.detachNetworkContainerFromNetwork)
	listener.AddHandler(cns.V2Prefix+cns.CreateHnsNetworkPath, service.createHnsNetwork)
	listener.AddHandler(cns.V2Prefix+cns.DeleteHnsNetworkPath, service.deleteHnsNetwork)
	listener.AddHandler(cns.V2Prefix+cns.NumberOfCPUCoresPath, service.getNumberOfCPUCores)
	listener.AddHandler(cns.V2Prefix+cns.CreateHostNCApipaEndpointPath, service.createHostNCApipaEndpoint)
	listener.AddHandler(cns.V2Prefix+cns.DeleteHostNCApipaEndpointPath, service.deleteHostNCApipaEndpoint)

	// Initialize HTTP client to be reused in CNS
	connectionTimeout, _ := service.GetOption(acn.OptHttpConnectionTimeout).(int)
	responseHeaderTimeout, _ := service.GetOption(acn.OptHttpResponseHeaderTimeout).(int)
	acn.InitHttpClient(connectionTimeout, responseHeaderTimeout)

	logger.SetContextDetails(service.state.OrchestratorType, service.state.NodeID)
	logger.Printf("[Azure CNS]  Listening.")

	return nil
}

// Stop stops the CNS.
func (service *HTTPRestService) Stop() {
	service.Uninitialize()
	logger.Printf("[Azure CNS]  Service stopped.")
}

// GetPartitionKey - Get dnc/service partition key
func (service *HTTPRestService) GetPartitionKey() (dncPartitionKey string) {
	service.lock.Lock()
	dncPartitionKey = service.dncPartitionKey
	service.lock.Unlock()
	return
}

// Get the network info from the service network state
func (service *HTTPRestService) getNetworkInfo(networkName string) (*networkInfo, bool) {
	service.lock.Lock()
	defer service.lock.Unlock()
	networkInfo, found := service.state.Networks[networkName]

	return networkInfo, found
}

// Set the network info in the service network state
func (service *HTTPRestService) setNetworkInfo(networkName string, networkInfo *networkInfo) {
	service.lock.Lock()
	defer service.lock.Unlock()
	service.state.Networks[networkName] = networkInfo

	return
}

// Remove the network info from the service network state
func (service *HTTPRestService) removeNetworkInfo(networkName string) {
	service.lock.Lock()
	defer service.lock.Unlock()
	delete(service.state.Networks, networkName)

	return
}

// Handles requests to set the environment type.
func (service *HTTPRestService) setEnvironment(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure CNS] setEnvironment")

	var req cns.SetEnvironmentRequest
	err := service.Listener.Decode(w, r, &req)
	logger.Request(service.Name, &req, err)

	if err != nil {
		return
	}

	switch r.Method {
	case "POST":
		logger.Printf("[Azure CNS]  POST received for SetEnvironment.")
		service.state.Location = req.Location
		service.state.NetworkType = req.NetworkType
		service.state.Initialized = true
		service.saveState()
	default:
	}

	resp := &cns.Response{ReturnCode: 0}
	err = service.Listener.Encode(w, &resp)

	logger.Response(service.Name, resp, resp.ReturnCode, ReturnCodeToString(resp.ReturnCode), err)
}

// Handles CreateNetwork requests.
func (service *HTTPRestService) createNetwork(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure CNS] createNetwork")

	var err error
	returnCode := 0
	returnMessage := ""

	if service.state.Initialized {
		var req cns.CreateNetworkRequest
		err = service.Listener.Decode(w, r, &req)
		logger.Request(service.Name, &req, err)

		if err != nil {
			returnMessage = fmt.Sprintf("[Azure CNS] Error. Unable to decode input request.")
			returnCode = InvalidParameter
		} else {
			switch r.Method {
			case "POST":
				dc := service.dockerClient
				rt := service.routingTable
				err = dc.NetworkExists(req.NetworkName)

				// Network does not exist.
				if err != nil {
					switch service.state.NetworkType {
					case "Underlay":
						switch service.state.Location {
						case "Azure":
							logger.Printf("[Azure CNS] Creating network with name %v.", req.NetworkName)

							err = rt.GetRoutingTable()
							if err != nil {
								// We should not fail the call to create network for this.
								// This is because restoring routes is a fallback mechanism in case
								// network driver is not behaving as expected.
								// The responsibility to restore routes is with network driver.
								logger.Printf("[Azure CNS] Unable to get routing table from node, %+v.", err.Error())
							}

							nicInfo, err := service.imdsClient.GetPrimaryInterfaceInfoFromHost()
							if err != nil {
								returnMessage = fmt.Sprintf("[Azure CNS] Error. GetPrimaryInterfaceInfoFromHost failed %v.", err.Error())
								returnCode = UnexpectedError
								break
							}

							err = dc.CreateNetwork(req.NetworkName, nicInfo, req.Options)
							if err != nil {
								returnMessage = fmt.Sprintf("[Azure CNS] Error. CreateNetwork failed %v.", err.Error())
								returnCode = UnexpectedError
							}

							err = rt.RestoreRoutingTable()
							if err != nil {
								logger.Printf("[Azure CNS] Unable to restore routing table on node, %+v.", err.Error())
							}

							networkInfo := &networkInfo{
								NetworkName: req.NetworkName,
								NicInfo:     nicInfo,
								Options:     req.Options,
							}

							service.state.Networks[req.NetworkName] = networkInfo

						case "StandAlone":
							returnMessage = fmt.Sprintf("[Azure CNS] Error. Underlay network is not supported in StandAlone environment. %v.", err.Error())
							returnCode = UnsupportedEnvironment
						}
					case "Overlay":
						returnMessage = fmt.Sprintf("[Azure CNS] Error. Overlay support not yet available. %v.", err.Error())
						returnCode = UnsupportedEnvironment
					}
				} else {
					returnMessage = fmt.Sprintf("[Azure CNS] Received a request to create an already existing network %v", req.NetworkName)
					logger.Printf(returnMessage)
				}

			default:
				returnMessage = "[Azure CNS] Error. CreateNetwork did not receive a POST."
				returnCode = InvalidParameter
			}
		}

	} else {
		returnMessage = fmt.Sprintf("[Azure CNS] Error. CNS is not yet initialized with environment.")
		returnCode = UnsupportedEnvironment
	}

	resp := &cns.Response{
		ReturnCode: returnCode,
		Message:    returnMessage,
	}

	err = service.Listener.Encode(w, &resp)

	if returnCode == 0 {
		service.saveState()
	}

	logger.Response(service.Name, resp, resp.ReturnCode, ReturnCodeToString(resp.ReturnCode), err)
}

// Handles DeleteNetwork requests.
func (service *HTTPRestService) deleteNetwork(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure CNS] deleteNetwork")

	var req cns.DeleteNetworkRequest
	returnCode := 0
	returnMessage := ""
	err := service.Listener.Decode(w, r, &req)
	logger.Request(service.Name, &req, err)

	if err != nil {
		return
	}

	switch r.Method {
	case "POST":
		dc := service.dockerClient
		err := dc.NetworkExists(req.NetworkName)

		// Network does exist
		if err == nil {
			logger.Printf("[Azure CNS] Deleting network with name %v.", req.NetworkName)
			err := dc.DeleteNetwork(req.NetworkName)
			if err != nil {
				returnMessage = fmt.Sprintf("[Azure CNS] Error. DeleteNetwork failed %v.", err.Error())
				returnCode = UnexpectedError
			}
		} else {
			if err == fmt.Errorf("Network not found") {
				logger.Printf("[Azure CNS] Received a request to delete network that does not exist: %v.", req.NetworkName)
			} else {
				returnCode = UnexpectedError
				returnMessage = err.Error()
			}
		}

	default:
		returnMessage = "[Azure CNS] Error. DeleteNetwork did not receive a POST."
		returnCode = InvalidParameter
	}

	resp := &cns.Response{
		ReturnCode: returnCode,
		Message:    returnMessage,
	}

	err = service.Listener.Encode(w, &resp)

	if returnCode == 0 {
		service.removeNetworkInfo(req.NetworkName)
		service.saveState()
	}

	logger.Response(service.Name, resp, resp.ReturnCode, ReturnCodeToString(resp.ReturnCode), err)
}

// Handles CreateHnsNetwork requests.
func (service *HTTPRestService) createHnsNetwork(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure CNS] createHnsNetwork")

	var err error
	returnCode := 0
	returnMessage := ""

	var req cns.CreateHnsNetworkRequest
	err = service.Listener.Decode(w, r, &req)
	logger.Request(service.Name, &req, err)

	if err != nil {
		returnMessage = fmt.Sprintf("[Azure CNS] Error. Unable to decode input request.")
		returnCode = InvalidParameter
	} else {
		switch r.Method {
		case "POST":
			if err := hnsclient.CreateHnsNetwork(req); err == nil {
				// Save the newly created HnsNetwork name. CNS deleteHnsNetwork API
				// will only allow deleting these networks.
				networkInfo := &networkInfo{
					NetworkName: req.NetworkName,
				}
				service.setNetworkInfo(req.NetworkName, networkInfo)
				returnMessage = fmt.Sprintf("[Azure CNS] Successfully created HNS network: %s", req.NetworkName)
			} else {
				returnMessage = fmt.Sprintf("[Azure CNS] CreateHnsNetwork failed with error %v", err.Error())
				returnCode = UnexpectedError
			}
		default:
			returnMessage = "[Azure CNS] Error. CreateHnsNetwork did not receive a POST."
			returnCode = InvalidParameter
		}
	}

	resp := &cns.Response{
		ReturnCode: returnCode,
		Message:    returnMessage,
	}

	err = service.Listener.Encode(w, &resp)

	if returnCode == 0 {
		service.saveState()
	}

	logger.Response(service.Name, resp, resp.ReturnCode, ReturnCodeToString(resp.ReturnCode), err)
}

// Handles deleteHnsNetwork requests.
func (service *HTTPRestService) deleteHnsNetwork(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure CNS] deleteHnsNetwork")

	var err error
	var req cns.DeleteHnsNetworkRequest
	returnCode := 0
	returnMessage := ""

	err = service.Listener.Decode(w, r, &req)
	logger.Request(service.Name, &req, err)

	if err != nil {
		returnMessage = fmt.Sprintf("[Azure CNS] Error. Unable to decode input request.")
		returnCode = InvalidParameter
	} else {
		switch r.Method {
		case "POST":
			networkInfo, found := service.getNetworkInfo(req.NetworkName)
			if found && networkInfo.NetworkName == req.NetworkName {
				if err = hnsclient.DeleteHnsNetwork(req.NetworkName); err == nil {
					returnMessage = fmt.Sprintf("[Azure CNS] Successfully deleted HNS network: %s", req.NetworkName)
				} else {
					returnMessage = fmt.Sprintf("[Azure CNS] DeleteHnsNetwork failed with error %v", err.Error())
					returnCode = UnexpectedError
				}
			} else {
				returnMessage = fmt.Sprintf("[Azure CNS] Network %s not found", req.NetworkName)
				returnCode = InvalidParameter
			}
		default:
			returnMessage = "[Azure CNS] Error. DeleteHnsNetwork did not receive a POST."
			returnCode = InvalidParameter
		}
	}

	resp := &cns.Response{
		ReturnCode: returnCode,
		Message:    returnMessage,
	}

	err = service.Listener.Encode(w, &resp)

	if returnCode == 0 {
		service.removeNetworkInfo(req.NetworkName)
		service.saveState()
	}

	logger.Response(service.Name, resp, resp.ReturnCode, ReturnCodeToString(resp.ReturnCode), err)
}

// Handles ip reservation requests.
func (service *HTTPRestService) reserveIPAddress(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure CNS] reserveIPAddress")

	var req cns.ReserveIPAddressRequest
	returnMessage := ""
	returnCode := 0
	addr := ""
	address := ""
	err := service.Listener.Decode(w, r, &req)

	logger.Request(service.Name, &req, err)

	if err != nil {
		return
	}

	if req.ReservationID == "" {
		returnCode = ReservationNotFound
		returnMessage = fmt.Sprintf("[Azure CNS] Error. ReservationId is empty")
	}

	switch r.Method {
	case "POST":
		ic := service.ipamClient

		ifInfo, err := service.imdsClient.GetPrimaryInterfaceInfoFromMemory()
		if err != nil {
			returnMessage = fmt.Sprintf("[Azure CNS] Error. GetPrimaryIfaceInfo failed %v", err.Error())
			returnCode = UnexpectedError
			break
		}

		asID, err := ic.GetAddressSpace()
		if err != nil {
			returnMessage = fmt.Sprintf("[Azure CNS] Error. GetAddressSpace failed %v", err.Error())
			returnCode = UnexpectedError
			break
		}

		poolID, err := ic.GetPoolID(asID, ifInfo.Subnet)
		if err != nil {
			returnMessage = fmt.Sprintf("[Azure CNS] Error. GetPoolID failed %v", err.Error())
			returnCode = UnexpectedError
			break
		}

		addr, err = ic.ReserveIPAddress(poolID, req.ReservationID)
		if err != nil {
			returnMessage = fmt.Sprintf("[Azure CNS] ReserveIpAddress failed with %+v", err.Error())
			returnCode = AddressUnavailable
			break
		}

		addressIP, _, err := net.ParseCIDR(addr)
		if err != nil {
			returnMessage = fmt.Sprintf("[Azure CNS] ParseCIDR failed with %+v", err.Error())
			returnCode = UnexpectedError
			break
		}
		address = addressIP.String()

	default:
		returnMessage = "[Azure CNS] Error. ReserveIP did not receive a POST."
		returnCode = InvalidParameter

	}

	resp := cns.Response{
		ReturnCode: returnCode,
		Message:    returnMessage,
	}

	reserveResp := &cns.ReserveIPAddressResponse{Response: resp, IPAddress: address}
	err = service.Listener.Encode(w, &reserveResp)
	logger.Response(service.Name, reserveResp, resp.ReturnCode, ReturnCodeToString(resp.ReturnCode), err)
}

// Handles release ip reservation requests.
func (service *HTTPRestService) releaseIPAddress(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure CNS] releaseIPAddress")

	var req cns.ReleaseIPAddressRequest
	returnMessage := ""
	returnCode := 0

	err := service.Listener.Decode(w, r, &req)
	logger.Request(service.Name, &req, err)

	if err != nil {
		return
	}

	if req.ReservationID == "" {
		returnCode = ReservationNotFound
		returnMessage = fmt.Sprintf("[Azure CNS] Error. ReservationId is empty")
	}

	switch r.Method {
	case "POST":
		ic := service.ipamClient

		ifInfo, err := service.imdsClient.GetPrimaryInterfaceInfoFromMemory()
		if err != nil {
			returnMessage = fmt.Sprintf("[Azure CNS] Error. GetPrimaryIfaceInfo failed %v", err.Error())
			returnCode = UnexpectedError
			break
		}

		asID, err := ic.GetAddressSpace()
		if err != nil {
			returnMessage = fmt.Sprintf("[Azure CNS] Error. GetAddressSpace failed %v", err.Error())
			returnCode = UnexpectedError
			break
		}

		poolID, err := ic.GetPoolID(asID, ifInfo.Subnet)
		if err != nil {
			returnMessage = fmt.Sprintf("[Azure CNS] Error. GetPoolID failed %v", err.Error())
			returnCode = UnexpectedError
			break
		}

		err = ic.ReleaseIPAddress(poolID, req.ReservationID)
		if err != nil {
			returnMessage = fmt.Sprintf("[Azure CNS] ReleaseIpAddress failed with %+v", err.Error())
			returnCode = ReservationNotFound
		}

	default:
		returnMessage = "[Azure CNS] Error. ReleaseIP did not receive a POST."
		returnCode = InvalidParameter
	}

	resp := cns.Response{
		ReturnCode: returnCode,
		Message:    returnMessage,
	}

	err = service.Listener.Encode(w, &resp)
	logger.Response(service.Name, resp, resp.ReturnCode, ReturnCodeToString(resp.ReturnCode), err)
}

// Retrieves the host local ip address. Containers can talk to host using this IP address.
func (service *HTTPRestService) getHostLocalIP(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure CNS] getHostLocalIP")
	logger.Request(service.Name, "getHostLocalIP", nil)

	var found bool
	var errmsg string
	hostLocalIP := "0.0.0.0"

	if service.state.Initialized {
		switch r.Method {
		case "GET":
			switch service.state.NetworkType {
			case "Underlay":
				if service.imdsClient != nil {
					piface, err := service.imdsClient.GetPrimaryInterfaceInfoFromMemory()
					if err == nil {
						hostLocalIP = piface.PrimaryIP
						found = true
					} else {
						logger.Printf("[Azure-CNS] Received error from GetPrimaryInterfaceInfoFromMemory. err: %v", err.Error())
					}
				}

			case "Overlay":
				errmsg = "[Azure-CNS] Overlay is not yet supported."
			}

		default:
			errmsg = "[Azure-CNS] GetHostLocalIP API expects a GET."
		}
	}

	returnCode := 0
	if !found {
		returnCode = NotFound
		if errmsg == "" {
			errmsg = "[Azure-CNS] Unable to get host local ip. Check if environment is initialized.."
		}
	}

	resp := cns.Response{ReturnCode: returnCode, Message: errmsg}
	hostLocalIPResponse := &cns.HostLocalIPAddressResponse{
		Response:  resp,
		IPAddress: hostLocalIP,
	}

	err := service.Listener.Encode(w, &hostLocalIPResponse)

	logger.Response(service.Name, hostLocalIPResponse, resp.ReturnCode, ReturnCodeToString(resp.ReturnCode), err)
}

// Handles ip address utilization requests.
func (service *HTTPRestService) getIPAddressUtilization(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure CNS] getIPAddressUtilization")
	logger.Request(service.Name, "getIPAddressUtilization", nil)

	returnMessage := ""
	returnCode := 0
	capacity := 0
	available := 0
	var unhealthyAddrs []string

	switch r.Method {
	case "GET":
		ic := service.ipamClient

		ifInfo, err := service.imdsClient.GetPrimaryInterfaceInfoFromMemory()
		if err != nil {
			returnMessage = fmt.Sprintf("[Azure CNS] Error. GetPrimaryIfaceInfo failed %v", err.Error())
			returnCode = UnexpectedError
			break
		}

		asID, err := ic.GetAddressSpace()
		if err != nil {
			returnMessage = fmt.Sprintf("[Azure CNS] Error. GetAddressSpace failed %v", err.Error())
			returnCode = UnexpectedError
			break
		}

		poolID, err := ic.GetPoolID(asID, ifInfo.Subnet)
		if err != nil {
			returnMessage = fmt.Sprintf("[Azure CNS] Error. GetPoolID failed %v", err.Error())
			returnCode = UnexpectedError
			break
		}

		capacity, available, unhealthyAddrs, err = ic.GetIPAddressUtilization(poolID)
		if err != nil {
			returnMessage = fmt.Sprintf("[Azure CNS] Error. GetIPUtilization failed %v", err.Error())
			returnCode = UnexpectedError
			break
		}
		logger.Printf("[Azure CNS] Capacity %v Available %v UnhealthyAddrs %v", capacity, available, unhealthyAddrs)

	default:
		returnMessage = "[Azure CNS] Error. GetIPUtilization did not receive a GET."
		returnCode = InvalidParameter
	}

	resp := cns.Response{
		ReturnCode: returnCode,
		Message:    returnMessage,
	}

	utilResponse := &cns.IPAddressesUtilizationResponse{
		Response:  resp,
		Available: available,
		Reserved:  capacity - available,
		Unhealthy: len(unhealthyAddrs),
	}

	err := service.Listener.Encode(w, &utilResponse)
	logger.Response(service.Name, utilResponse, resp.ReturnCode, ReturnCodeToString(resp.ReturnCode), err)
}

// Handles retrieval of ip addresses that are available to be reserved from ipam driver.
func (service *HTTPRestService) getAvailableIPAddresses(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure CNS] getAvailableIPAddresses")
	logger.Request(service.Name, "getAvailableIPAddresses", nil)

	switch r.Method {
	case "GET":
	default:
	}

	resp := cns.Response{ReturnCode: 0}
	ipResp := &cns.GetIPAddressesResponse{Response: resp}
	err := service.Listener.Encode(w, &ipResp)

	logger.Response(service.Name, ipResp, resp.ReturnCode, ReturnCodeToString(resp.ReturnCode), err)
}

// Handles retrieval of reserved ip addresses from ipam driver.
func (service *HTTPRestService) getReservedIPAddresses(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure CNS] getReservedIPAddresses")
	logger.Request(service.Name, "getReservedIPAddresses", nil)

	switch r.Method {
	case "GET":
	default:
	}

	resp := cns.Response{ReturnCode: 0}
	ipResp := &cns.GetIPAddressesResponse{Response: resp}
	err := service.Listener.Encode(w, &ipResp)

	logger.Response(service.Name, ipResp, resp.ReturnCode, ReturnCodeToString(resp.ReturnCode), err)
}

// Handles retrieval of ghost ip addresses from ipam driver.
func (service *HTTPRestService) getUnhealthyIPAddresses(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure CNS] getUnhealthyIPAddresses")
	logger.Request(service.Name, "getUnhealthyIPAddresses", nil)

	returnMessage := ""
	returnCode := 0
	capacity := 0
	available := 0
	var unhealthyAddrs []string

	switch r.Method {
	case "GET":
		ic := service.ipamClient

		ifInfo, err := service.imdsClient.GetPrimaryInterfaceInfoFromMemory()
		if err != nil {
			returnMessage = fmt.Sprintf("[Azure CNS] Error. GetPrimaryIfaceInfo failed %v", err.Error())
			returnCode = UnexpectedError
			break
		}

		asID, err := ic.GetAddressSpace()
		if err != nil {
			returnMessage = fmt.Sprintf("[Azure CNS] Error. GetAddressSpace failed %v", err.Error())
			returnCode = UnexpectedError
			break
		}

		poolID, err := ic.GetPoolID(asID, ifInfo.Subnet)
		if err != nil {
			returnMessage = fmt.Sprintf("[Azure CNS] Error. GetPoolID failed %v", err.Error())
			returnCode = UnexpectedError
			break
		}

		capacity, available, unhealthyAddrs, err = ic.GetIPAddressUtilization(poolID)
		if err != nil {
			returnMessage = fmt.Sprintf("[Azure CNS] Error. GetIPUtilization failed %v", err.Error())
			returnCode = UnexpectedError
			break
		}
		logger.Printf("[Azure CNS] Capacity %v Available %v UnhealthyAddrs %v", capacity, available, unhealthyAddrs)

	default:
		returnMessage = "[Azure CNS] Error. GetUnhealthyIP did not receive a POST."
		returnCode = InvalidParameter
	}

	resp := cns.Response{
		ReturnCode: returnCode,
		Message:    returnMessage,
	}

	ipResp := &cns.GetIPAddressesResponse{
		Response:    resp,
		IPAddresses: unhealthyAddrs,
	}

	err := service.Listener.Encode(w, &ipResp)
	logger.Response(service.Name, ipResp, resp.ReturnCode, ReturnCodeToString(resp.ReturnCode), err)
}

// getAllIPAddresses retrieves all ip addresses from ipam driver.
func (service *HTTPRestService) getAllIPAddresses(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure CNS] getAllIPAddresses")
	logger.Request(service.Name, "getAllIPAddresses", nil)

	switch r.Method {
	case "GET":
	default:
	}

	resp := cns.Response{ReturnCode: 0}
	ipResp := &cns.GetIPAddressesResponse{Response: resp}
	err := service.Listener.Encode(w, &ipResp)

	logger.Response(service.Name, ipResp, resp.ReturnCode, ReturnCodeToString(resp.ReturnCode), err)
}

// Handles health report requests.
func (service *HTTPRestService) getHealthReport(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure CNS] getHealthReport")
	logger.Request(service.Name, "getHealthReport", nil)

	switch r.Method {
	case "GET":
	default:
	}

	resp := &cns.Response{ReturnCode: 0}
	err := service.Listener.Encode(w, &resp)

	logger.Response(service.Name, resp, resp.ReturnCode, ReturnCodeToString(resp.ReturnCode), err)
}

// saveState writes CNS state to persistent store.
func (service *HTTPRestService) saveState() error {
	logger.Printf("[Azure CNS] saveState")

	// Skip if a store is not provided.
	if service.store == nil {
		logger.Printf("[Azure CNS]  store not initialized.")
		return nil
	}

	// Update time stamp.
	service.state.TimeStamp = time.Now()
	err := service.store.Write(storeKey, &service.state)
	if err == nil {
		logger.Printf("[Azure CNS]  State saved successfully.\n")
	} else {
		logger.Errorf("[Azure CNS]  Failed to save state., err:%v\n", err)
	}

	return err
}

// restoreState restores CNS state from persistent store.
func (service *HTTPRestService) restoreState() error {
	logger.Printf("[Azure CNS] restoreState")

	// Skip if a store is not provided.
	if service.store == nil {
		logger.Printf("[Azure CNS]  store not initialized.")
		return nil
	}

	// Read any persisted state.
	err := service.store.Read(storeKey, &service.state)
	if err != nil {
		if err == store.ErrKeyNotFound {
			// Nothing to restore.
			logger.Printf("[Azure CNS]  No state to restore.\n")
			return nil
		}

		logger.Errorf("[Azure CNS]  Failed to restore state, err:%v\n", err)
		return err
	}

	logger.Printf("[Azure CNS]  Restored state, %+v\n", service.state)
	return nil
}

func (service *HTTPRestService) setOrchestratorType(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure CNS] setOrchestratorType")

	var (
		req           cns.SetOrchestratorTypeRequest
		returnMessage string
		returnCode    int
		nodeID        string
	)

	err := service.Listener.Decode(w, r, &req)
	if err != nil {
		return
	}

	service.lock.Lock()

	service.dncPartitionKey = req.DncPartitionKey
	nodeID = service.state.NodeID

	if nodeID == "" || nodeID == req.NodeID {
		switch req.OrchestratorType {
		case cns.ServiceFabric:
			fallthrough
		case cns.Kubernetes:
			fallthrough
		case cns.WebApps:
			fallthrough
		case cns.Batch:
			fallthrough
		case cns.DBforPostgreSQL:
			fallthrough
		case cns.AzureFirstParty:
			service.state.OrchestratorType = req.OrchestratorType
			service.state.NodeID = req.NodeID
			logger.SetContextDetails(req.OrchestratorType, req.NodeID)
			service.saveState()
		default:
			returnMessage = fmt.Sprintf("Invalid Orchestrator type %v", req.OrchestratorType)
			returnCode = UnsupportedOrchestratorType
		}
	} else {
		returnMessage = fmt.Sprintf("Invalid request since this node has already been registered as %s", nodeID)
		returnCode = InvalidRequest
	}

	service.lock.Unlock()

	resp := cns.Response{
		ReturnCode: returnCode,
		Message:    returnMessage,
	}

	err = service.Listener.Encode(w, &resp)
	logger.Response(service.Name, resp, resp.ReturnCode, ReturnCodeToString(resp.ReturnCode), err)
}

func (service *HTTPRestService) saveNetworkContainerGoalState(req cns.CreateNetworkContainerRequest) (int, string) {
	// we don't want to overwrite what other calls may have written
	service.lock.Lock()
	defer service.lock.Unlock()

	existing, ok := service.state.ContainerStatus[req.NetworkContainerid]
	var hostVersion string
	if ok {
		hostVersion = existing.HostVersion
	}

	if service.state.ContainerStatus == nil {
		service.state.ContainerStatus = make(map[string]containerstatus)
	}

	service.state.ContainerStatus[req.NetworkContainerid] =
		containerstatus{
			ID:                            req.NetworkContainerid,
			VMVersion:                     req.Version,
			CreateNetworkContainerRequest: req,
			HostVersion:                   hostVersion}

	switch req.NetworkContainerType {
	case cns.AzureContainerInstance:
		fallthrough
	case cns.Docker:
		fallthrough
	case cns.Basic:
		fallthrough
	case cns.JobObject:
		fallthrough
	case cns.COW:
		fallthrough
	case cns.WebApps:
		switch service.state.OrchestratorType {
		case cns.Kubernetes:
			fallthrough
		case cns.ServiceFabric:
			fallthrough
		case cns.Batch:
			fallthrough
		case cns.DBforPostgreSQL:
			fallthrough
		case cns.AzureFirstParty:
			fallthrough
		case cns.WebApps:
			var podInfo cns.KubernetesPodInfo
			err := json.Unmarshal(req.OrchestratorContext, &podInfo)
			if err != nil {
				errBuf := fmt.Sprintf("Unmarshalling %s failed with error %v", req.NetworkContainerType, err)
				return UnexpectedError, errBuf
			}

			logger.Printf("Pod info %v", podInfo)

			if service.state.ContainerIDByOrchestratorContext == nil {
				service.state.ContainerIDByOrchestratorContext = make(map[string]string)
			}

			service.state.ContainerIDByOrchestratorContext[podInfo.PodName+podInfo.PodNamespace] = req.NetworkContainerid
			break

		default:
			errMsg := fmt.Sprintf("Unsupported orchestrator type: %s", service.state.OrchestratorType)
			logger.Errorf(errMsg)
			return UnsupportedOrchestratorType, errMsg
		}
	default:
		errMsg := fmt.Sprintf("Unsupported network container type %s", req.NetworkContainerType)
		logger.Errorf(errMsg)
		return UnsupportedNetworkContainerType, errMsg
	}

	service.saveState()
	return 0, ""
}

func (service *HTTPRestService) createOrUpdateNetworkContainer(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure CNS] createOrUpdateNetworkContainer")

	var req cns.CreateNetworkContainerRequest
	returnMessage := ""
	returnCode := 0

	err := service.Listener.Decode(w, r, &req)
	logger.Request(service.Name, &req, err)
	if err != nil {
		return
	}

	if req.NetworkContainerid == "" {
		returnCode = NetworkContainerNotSpecified
		returnMessage = fmt.Sprintf("[Azure CNS] Error. NetworkContainerid is empty")
	}

	switch r.Method {
	case "POST":
		if req.NetworkContainerType == cns.WebApps {
			// try to get the saved nc state if it exists
			existing, ok := service.getNetworkContainerDetails(req.NetworkContainerid)

			// create/update nc only if it doesn't exist or it exists and the requested version is different from the saved version
			if !ok || (ok && existing.VMVersion != req.Version) {
				nc := service.networkContainer
				if err = nc.Create(req); err != nil {
					returnMessage = fmt.Sprintf("[Azure CNS] Error. CreateOrUpdateNetworkContainer failed %v", err.Error())
					returnCode = UnexpectedError
					break
				}
			}
		} else if req.NetworkContainerType == cns.AzureContainerInstance {
			// try to get the saved nc state if it exists
			existing, ok := service.getNetworkContainerDetails(req.NetworkContainerid)

			// create/update nc only if it doesn't exist or it exists and the requested version is different from the saved version
			if ok && existing.VMVersion != req.Version {
				nc := service.networkContainer
				netPluginConfig := service.getNetPluginDetails()
				if err = nc.Update(req, netPluginConfig); err != nil {
					returnMessage = fmt.Sprintf("[Azure CNS] Error. CreateOrUpdateNetworkContainer failed %v", err.Error())
					returnCode = UnexpectedError
					break
				}
			}
		}

		returnCode, returnMessage = service.saveNetworkContainerGoalState(req)

	default:
		returnMessage = "[Azure CNS] Error. CreateOrUpdateNetworkContainer did not receive a POST."
		returnCode = InvalidParameter
	}

	resp := cns.Response{
		ReturnCode: returnCode,
		Message:    returnMessage,
	}

	reserveResp := &cns.CreateNetworkContainerResponse{Response: resp}
	err = service.Listener.Encode(w, &reserveResp)

	// If the NC was created successfully, log NC snapshot.
	if returnCode == 0 {
		logNCSnapshot(req)
	}

	logger.Response(service.Name, reserveResp, resp.ReturnCode, ReturnCodeToString(resp.ReturnCode), err)
}

func (service *HTTPRestService) getNetworkContainerByID(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure CNS] getNetworkContainerByID")

	var req cns.GetNetworkContainerRequest
	returnMessage := ""
	returnCode := 0

	err := service.Listener.Decode(w, r, &req)
	logger.Request(service.Name, &req, err)
	if err != nil {
		return
	}

	resp := cns.Response{
		ReturnCode: returnCode,
		Message:    returnMessage,
	}

	reserveResp := &cns.GetNetworkContainerResponse{Response: resp}
	err = service.Listener.Encode(w, &reserveResp)
	logger.Response(service.Name, reserveResp, resp.ReturnCode, ReturnCodeToString(resp.ReturnCode), err)
}

func (service *HTTPRestService) getNetworkContainerResponse(req cns.GetNetworkContainerRequest) cns.GetNetworkContainerResponse {
	var containerID string
	var getNetworkContainerResponse cns.GetNetworkContainerResponse

	service.lock.Lock()
	defer service.lock.Unlock()

	switch service.state.OrchestratorType {
	case cns.Kubernetes:
		fallthrough
	case cns.ServiceFabric:
		fallthrough
	case cns.Batch:
		fallthrough
	case cns.DBforPostgreSQL:
		fallthrough
	case cns.AzureFirstParty:
		var podInfo cns.KubernetesPodInfo
		err := json.Unmarshal(req.OrchestratorContext, &podInfo)
		if err != nil {
			getNetworkContainerResponse.Response.ReturnCode = UnexpectedError
			getNetworkContainerResponse.Response.Message = fmt.Sprintf("Unmarshalling orchestrator context failed with error %v", err)
			return getNetworkContainerResponse
		}

		logger.Printf("pod info %+v", podInfo)
		containerID = service.state.ContainerIDByOrchestratorContext[podInfo.PodName+podInfo.PodNamespace]
		logger.Printf("containerid %v", containerID)
		break

	default:
		getNetworkContainerResponse.Response.ReturnCode = UnsupportedOrchestratorType
		getNetworkContainerResponse.Response.Message = fmt.Sprintf("Invalid orchestrator type %v", service.state.OrchestratorType)
		return getNetworkContainerResponse
	}

	containerStatus := service.state.ContainerStatus
	containerDetails, ok := containerStatus[containerID]
	if !ok {
		getNetworkContainerResponse.Response.ReturnCode = UnknownContainerID
		getNetworkContainerResponse.Response.Message = "NetworkContainer doesn't exist."
		return getNetworkContainerResponse
	}

	savedReq := containerDetails.CreateNetworkContainerRequest
	getNetworkContainerResponse = cns.GetNetworkContainerResponse{
		NetworkContainerID:         savedReq.NetworkContainerid,
		IPConfiguration:            savedReq.IPConfiguration,
		Routes:                     savedReq.Routes,
		CnetAddressSpace:           savedReq.CnetAddressSpace,
		MultiTenancyInfo:           savedReq.MultiTenancyInfo,
		PrimaryInterfaceIdentifier: savedReq.PrimaryInterfaceIdentifier,
		LocalIPConfiguration:       savedReq.LocalIPConfiguration,
		AllowHostToNCCommunication: savedReq.AllowHostToNCCommunication,
		AllowNCToHostCommunication: savedReq.AllowNCToHostCommunication,
	}

	return getNetworkContainerResponse
}

func (service *HTTPRestService) getNetworkContainerByOrchestratorContext(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure CNS] getNetworkContainerByOrchestratorContext")

	var req cns.GetNetworkContainerRequest

	err := service.Listener.Decode(w, r, &req)
	logger.Request(service.Name, &req, err)
	if err != nil {
		return
	}

	// getNetworkContainerByOrchestratorContext gets called for multitenancy and
	// setting the SDNRemoteArpMacAddress regKey is essential for the multitenancy
	// to work correctly in case of windows platform. Return if there is an error
	if err = platform.SetSdnRemoteArpMacAddress(); err != nil {
		logger.Printf("[Azure CNS] SetSdnRemoteArpMacAddress failed with error: %s", err.Error())
		return
	}

	getNetworkContainerResponse := service.getNetworkContainerResponse(req)
	returnCode := getNetworkContainerResponse.Response.ReturnCode
	err = service.Listener.Encode(w, &getNetworkContainerResponse)
	logger.Response(service.Name, getNetworkContainerResponse, returnCode, ReturnCodeToString(returnCode), err)
}

func (service *HTTPRestService) deleteNetworkContainer(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure CNS] deleteNetworkContainer")

	var req cns.DeleteNetworkContainerRequest
	returnMessage := ""
	returnCode := 0

	err := service.Listener.Decode(w, r, &req)
	logger.Request(service.Name, &req, err)
	if err != nil {
		return
	}

	if req.NetworkContainerid == "" {
		returnCode = NetworkContainerNotSpecified
		returnMessage = fmt.Sprintf("[Azure CNS] Error. NetworkContainerid is empty")
	}

	switch r.Method {
	case "POST":
		var containerStatus containerstatus
		var ok bool

		containerStatus, ok = service.getNetworkContainerDetails(req.NetworkContainerid)

		if !ok {
			logger.Printf("Not able to retrieve network container details for this container id %v", req.NetworkContainerid)
			break
		}

		if containerStatus.CreateNetworkContainerRequest.NetworkContainerType == cns.WebApps {
			nc := service.networkContainer
			if err := nc.Delete(req.NetworkContainerid); err != nil {
				returnMessage = fmt.Sprintf("[Azure CNS] Error. DeleteNetworkContainer failed %v", err.Error())
				returnCode = UnexpectedError
				break
			}
		}

		service.lock.Lock()
		defer service.lock.Unlock()

		if service.state.ContainerStatus != nil {
			delete(service.state.ContainerStatus, req.NetworkContainerid)
		}

		if service.state.ContainerIDByOrchestratorContext != nil {
			for orchestratorContext, networkContainerID := range service.state.ContainerIDByOrchestratorContext {
				if networkContainerID == req.NetworkContainerid {
					delete(service.state.ContainerIDByOrchestratorContext, orchestratorContext)
					break
				}
			}
		}

		service.saveState()
		break
	default:
		returnMessage = "[Azure CNS] Error. DeleteNetworkContainer did not receive a POST."
		returnCode = InvalidParameter
	}

	resp := cns.Response{
		ReturnCode: returnCode,
		Message:    returnMessage,
	}

	reserveResp := &cns.DeleteNetworkContainerResponse{Response: resp}
	err = service.Listener.Encode(w, &reserveResp)
	logger.Response(service.Name, reserveResp, resp.ReturnCode, ReturnCodeToString(resp.ReturnCode), err)
}

func (service *HTTPRestService) getNetworkContainerStatus(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure CNS] getNetworkContainerStatus")

	var req cns.GetNetworkContainerStatusRequest
	returnMessage := ""
	returnCode := 0

	err := service.Listener.Decode(w, r, &req)
	logger.Request(service.Name, &req, err)
	if err != nil {
		return
	}

	service.lock.Lock()
	defer service.lock.Unlock()
	var ok bool
	var containerDetails containerstatus

	containerInfo := service.state.ContainerStatus
	if containerInfo != nil {
		containerDetails, ok = containerInfo[req.NetworkContainerid]
	} else {
		ok = false
	}

	var hostVersion string
	var vmVersion string

	if ok {
		savedReq := containerDetails.CreateNetworkContainerRequest
		containerVersion, err := service.imdsClient.GetNetworkContainerInfoFromHost(
			req.NetworkContainerid,
			savedReq.PrimaryInterfaceIdentifier,
			savedReq.AuthorizationToken, swiftAPIVersion)

		if err != nil {
			returnCode = CallToHostFailed
			returnMessage = err.Error()
		} else {
			hostVersion = containerVersion.ProgrammedVersion
		}
	} else {
		returnMessage = "[Azure CNS] Never received call to create this container."
		returnCode = UnknownContainerID
	}

	resp := cns.Response{
		ReturnCode: returnCode,
		Message:    returnMessage,
	}

	networkContainerStatusReponse := cns.GetNetworkContainerStatusResponse{
		Response:           resp,
		NetworkContainerid: req.NetworkContainerid,
		AzureHostVersion:   hostVersion,
		Version:            vmVersion,
	}

	err = service.Listener.Encode(w, &networkContainerStatusReponse)
	logger.Response(service.Name, networkContainerStatusReponse, resp.ReturnCode, ReturnCodeToString(resp.ReturnCode), err)
}

func (service *HTTPRestService) getInterfaceForContainer(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure CNS] getInterfaceForContainer")

	var req cns.GetInterfaceForContainerRequest
	returnMessage := ""
	returnCode := 0

	err := service.Listener.Decode(w, r, &req)
	logger.Request(service.Name, &req, err)
	if err != nil {
		return
	}

	containerInfo := service.state.ContainerStatus
	containerDetails, ok := containerInfo[req.NetworkContainerID]
	var interfaceName string
	var ipaddress string
	var cnetSpace []cns.IPSubnet
	var dnsServers []string
	var version string

	if ok {
		savedReq := containerDetails.CreateNetworkContainerRequest
		interfaceName = savedReq.NetworkContainerid
		cnetSpace = savedReq.CnetAddressSpace
		ipaddress = savedReq.IPConfiguration.IPSubnet.IPAddress // it has to exist
		dnsServers = savedReq.IPConfiguration.DNSServers
		version = savedReq.Version
	} else {
		returnMessage = "[Azure CNS] Never received call to create this container."
		returnCode = UnknownContainerID
		interfaceName = ""
		ipaddress = ""
		version = ""
	}

	resp := cns.Response{
		ReturnCode: returnCode,
		Message:    returnMessage,
	}

	getInterfaceForContainerResponse := cns.GetInterfaceForContainerResponse{
		Response:                resp,
		NetworkInterface:        cns.NetworkInterface{Name: interfaceName, IPAddress: ipaddress},
		CnetAddressSpace:        cnetSpace,
		DNSServers:              dnsServers,
		NetworkContainerVersion: version,
	}

	err = service.Listener.Encode(w, &getInterfaceForContainerResponse)

	logger.Response(service.Name, getInterfaceForContainerResponse, resp.ReturnCode, ReturnCodeToString(resp.ReturnCode), err)
}

// restoreNetworkState restores Network state that existed before reboot.
func (service *HTTPRestService) restoreNetworkState() error {
	logger.Printf("[Azure CNS] Enter Restoring Network State")

	if service.store == nil {
		logger.Printf("[Azure CNS] Store is not initialized, nothing to restore for network state.")
		return nil
	}

	rebooted := false
	modTime, err := service.store.GetModificationTime()

	if err == nil {
		logger.Printf("[Azure CNS] Store timestamp is %v.", modTime)

		rebootTime, err := platform.GetLastRebootTime()
		if err == nil && rebootTime.After(modTime) {
			logger.Printf("[Azure CNS] reboot time %v mod time %v", rebootTime, modTime)
			rebooted = true
		}
	}

	if rebooted {
		for _, nwInfo := range service.state.Networks {
			enableSnat := true

			logger.Printf("[Azure CNS] Restore nwinfo %v", nwInfo)

			if nwInfo.Options != nil {
				if _, ok := nwInfo.Options[dockerclient.OptDisableSnat]; ok {
					enableSnat = false
				}
			}

			if enableSnat {
				err := platform.SetOutboundSNAT(nwInfo.NicInfo.Subnet)
				if err != nil {
					logger.Printf("[Azure CNS] Error setting up SNAT outbound rule %v", err)
					return err
				}
			}
		}
	}

	return nil
}

func (service *HTTPRestService) attachNetworkContainerToNetwork(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure CNS] attachNetworkContainerToNetwork")

	var req cns.ConfigureContainerNetworkingRequest
	err := service.Listener.Decode(w, r, &req)
	logger.Request(service.Name, &req, err)
	if err != nil {
		return
	}

	resp := service.attachOrDetachHelper(req, attach, r.Method)
	attachResp := &cns.AttachContainerToNetworkResponse{Response: resp}
	err = service.Listener.Encode(w, &attachResp)
	logger.Response(service.Name, attachResp, resp.ReturnCode, ReturnCodeToString(resp.ReturnCode), err)
}

func (service *HTTPRestService) detachNetworkContainerFromNetwork(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure CNS] detachNetworkContainerFromNetwork")

	var req cns.ConfigureContainerNetworkingRequest
	err := service.Listener.Decode(w, r, &req)
	logger.Request(service.Name, &req, err)
	if err != nil {
		return
	}

	resp := service.attachOrDetachHelper(req, detach, r.Method)
	detachResp := &cns.DetachContainerFromNetworkResponse{Response: resp}
	err = service.Listener.Encode(w, &detachResp)
	logger.Response(service.Name, detachResp, resp.ReturnCode, ReturnCodeToString(resp.ReturnCode), err)
}

func (service *HTTPRestService) attachOrDetachHelper(req cns.ConfigureContainerNetworkingRequest, operation, method string) cns.Response {
	if method != "POST" {
		return cns.Response{
			ReturnCode: InvalidParameter,
			Message:    "[Azure CNS] Error. " + operation + "ContainerToNetwork did not receive a POST."}
	}
	if req.Containerid == "" {
		return cns.Response{
			ReturnCode: DockerContainerNotSpecified,
			Message:    "[Azure CNS] Error. Containerid is empty"}
	}
	if req.NetworkContainerid == "" {
		return cns.Response{
			ReturnCode: NetworkContainerNotSpecified,
			Message:    "[Azure CNS] Error. NetworkContainerid is empty"}
	}

	existing, ok := service.getNetworkContainerDetails(cns.SwiftPrefix + req.NetworkContainerid)

	if !ok {
		return cns.Response{
			ReturnCode: NotFound,
			Message:    fmt.Sprintf("[Azure CNS] Error. Network Container %s does not exist.", req.NetworkContainerid)}
	}

	returnCode := 0
	returnMessage := ""
	switch service.state.OrchestratorType {
	case cns.Batch:
		var podInfo cns.KubernetesPodInfo
		err := json.Unmarshal(existing.CreateNetworkContainerRequest.OrchestratorContext, &podInfo)
		if err != nil {
			returnCode = UnexpectedError
			returnMessage = fmt.Sprintf("Unmarshalling orchestrator context failed with error %+v", err)
		} else {
			nc := service.networkContainer
			netPluginConfig := service.getNetPluginDetails()
			switch operation {
			case attach:
				err = nc.Attach(podInfo, req.Containerid, netPluginConfig)
			case detach:
				err = nc.Detach(podInfo, req.Containerid, netPluginConfig)
			}
			if err != nil {
				returnCode = UnexpectedError
				returnMessage = fmt.Sprintf("[Azure CNS] Error. "+operation+"ContainerToNetwork failed %+v", err.Error())
			}
		}

	default:
		returnMessage = fmt.Sprintf("[Azure CNS] Invalid orchestrator type %v", service.state.OrchestratorType)
		returnCode = UnsupportedOrchestratorType
	}

	return cns.Response{
		ReturnCode: returnCode,
		Message:    returnMessage}
}

func (service *HTTPRestService) getNetPluginDetails() *networkcontainers.NetPluginConfiguration {
	pluginBinPath, _ := service.GetOption(acn.OptNetPluginPath).(string)
	configPath, _ := service.GetOption(acn.OptNetPluginConfigFile).(string)
	return networkcontainers.NewNetPluginConfiguration(pluginBinPath, configPath)
}

// Retrieves the number of logic processors on a node. It will be primarily
// used to enforce per VM delegated NIC limit by DNC.
func (service *HTTPRestService) getNumberOfCPUCores(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure-CNS] getNumberOfCPUCores")
	logger.Request(service.Name, "getNumberOfCPUCores", nil)

	var (
		num        int
		returnCode int
		errMsg     string
	)

	switch r.Method {
	case "GET":
		num = runtime.NumCPU()
	default:
		errMsg = "[Azure-CNS] getNumberOfCPUCores API expects a GET."
		returnCode = UnsupportedVerb
	}

	resp := cns.Response{ReturnCode: returnCode, Message: errMsg}
	numOfCPUCoresResp := cns.NumOfCPUCoresResponse{
		Response:      resp,
		NumOfCPUCores: num,
	}

	err := service.Listener.Encode(w, &numOfCPUCoresResp)

	logger.Response(service.Name, numOfCPUCoresResp, resp.ReturnCode, ReturnCodeToString(resp.ReturnCode), err)
}

func (service *HTTPRestService) getNetworkContainerDetails(networkContainerID string) (containerstatus, bool) {
	service.lock.Lock()
	defer service.lock.Unlock()

	containerDetails, containerExists := service.state.ContainerStatus[networkContainerID]

	return containerDetails, containerExists
}

func (service *HTTPRestService) createHostNCApipaEndpoint(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure-CNS] createHostNCApipaEndpoint")

	var (
		err           error
		req           cns.CreateHostNCApipaEndpointRequest
		returnCode    int
		returnMessage string
		endpointID    string
	)

	err = service.Listener.Decode(w, r, &req)
	logger.Request(service.Name, &req, err)
	if err != nil {
		return
	}

	switch r.Method {
	case "POST":
		networkContainerDetails, found := service.getNetworkContainerDetails(req.NetworkContainerID)
		if found {
			if !networkContainerDetails.CreateNetworkContainerRequest.AllowNCToHostCommunication &&
				!networkContainerDetails.CreateNetworkContainerRequest.AllowHostToNCCommunication {
				returnMessage = fmt.Sprintf("HostNCApipaEndpoint creation is not supported unless " +
					"AllowNCToHostCommunication or AllowHostToNCCommunication is set to true")
				returnCode = InvalidRequest
			} else {
				if endpointID, err = hnsclient.CreateHostNCApipaEndpoint(
					req.NetworkContainerID,
					networkContainerDetails.CreateNetworkContainerRequest.LocalIPConfiguration,
					networkContainerDetails.CreateNetworkContainerRequest.AllowNCToHostCommunication,
					networkContainerDetails.CreateNetworkContainerRequest.AllowHostToNCCommunication,
					networkContainerDetails.CreateNetworkContainerRequest.EndpointPolicies); err != nil {
					returnMessage = fmt.Sprintf("CreateHostNCApipaEndpoint failed with error: %v", err)
					returnCode = UnexpectedError
				}
			}
		} else {
			returnMessage = fmt.Sprintf("CreateHostNCApipaEndpoint failed with error: Unable to find goal state for"+
				" the given Network Container: %s", req.NetworkContainerID)
			returnCode = UnknownContainerID
		}
	default:
		returnMessage = "createHostNCApipaEndpoint API expects a POST"
		returnCode = UnsupportedVerb
	}

	response := cns.CreateHostNCApipaEndpointResponse{
		Response: cns.Response{
			ReturnCode: returnCode,
			Message:    returnMessage,
		},
		EndpointID: endpointID,
	}

	err = service.Listener.Encode(w, &response)
	logger.Response(service.Name, response, response.Response.ReturnCode, ReturnCodeToString(response.Response.ReturnCode), err)
}

func (service *HTTPRestService) deleteHostNCApipaEndpoint(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure-CNS] deleteHostNCApipaEndpoint")

	var (
		err           error
		req           cns.DeleteHostNCApipaEndpointRequest
		returnCode    int
		returnMessage string
	)

	err = service.Listener.Decode(w, r, &req)
	logger.Request(service.Name, &req, err)
	if err != nil {
		return
	}

	switch r.Method {
	case "POST":
		if err = hnsclient.DeleteHostNCApipaEndpoint(req.NetworkContainerID); err != nil {
			returnMessage = fmt.Sprintf("Failed to delete endpoint for Network Container: %s "+
				"due to error: %v", req.NetworkContainerID, err)
			returnCode = UnexpectedError
		}
	default:
		returnMessage = "deleteHostNCApipaEndpoint API expects a DELETE"
		returnCode = UnsupportedVerb
	}

	response := cns.DeleteHostNCApipaEndpointResponse{
		Response: cns.Response{
			ReturnCode: returnCode,
			Message:    returnMessage,
		},
	}

	err = service.Listener.Encode(w, &response)
	logger.Response(service.Name, response, response.Response.ReturnCode, ReturnCodeToString(response.Response.ReturnCode), err)
}

// Check if the network is joined
func (service *HTTPRestService) isNetworkJoined(networkID string) bool {
	namedLock.LockAcquire(stateJoinedNetworks)
	defer namedLock.LockRelease(stateJoinedNetworks)

	_, exists := service.state.joinedNetworks[networkID]

	return exists
}

// Set the network as joined
func (service *HTTPRestService) setNetworkStateJoined(networkID string) {
	namedLock.LockAcquire(stateJoinedNetworks)
	defer namedLock.LockRelease(stateJoinedNetworks)

	service.state.joinedNetworks[networkID] = struct{}{}
}

// Join Network by calling nmagent
func (service *HTTPRestService) joinNetwork(
	networkID string,
	joinNetworkURL string) (*http.Response, error, error) {
	var err error
	joinResponse, joinErr := nmagentclient.JoinNetwork(
		networkID,
		joinNetworkURL)

	if joinErr == nil && joinResponse.StatusCode == http.StatusOK {
		// Network joined successfully
		service.setNetworkStateJoined(networkID)
		logger.Printf("[Azure-CNS] setNetworkStateJoined for network: %s", networkID)
	} else {
		err = fmt.Errorf("Failed to join network: %s", networkID)
	}

	return joinResponse, joinErr, err
}

// Publish Network Container by calling nmagent
func (service *HTTPRestService) publishNetworkContainer(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure-CNS] PublishNetworkContainer")

	var (
		err                 error
		req                 cns.PublishNetworkContainerRequest
		returnCode          int
		returnMessage       string
		publishResponse     *http.Response
		publishStatusCode   int
		publishResponseBody []byte
		publishError        error
		publishErrorStr     string
		isNetworkJoined     bool
	)

	err = service.Listener.Decode(w, r, &req)

	// reqCopy creates a copy of incoming request. It doesn't copy the authentication token info
	// to avoid logging it.
	reqCopy := cns.PublishNetworkContainerRequest{
		NetworkID:                 req.NetworkID,
		NetworkContainerID:        req.NetworkContainerID,
		JoinNetworkURL:            req.JoinNetworkURL,
		CreateNetworkContainerURL: strings.Split(req.CreateNetworkContainerURL, "authenticationToken")[0],
	}

	logger.Request(service.Name, &reqCopy, err)
	if err != nil {
		return
	}

	switch r.Method {
	case "POST":
		// Join the network
		publishResponse, publishError, err = service.joinNetwork(req.NetworkID, req.JoinNetworkURL)
		if err == nil {
			isNetworkJoined = true
		} else {
			returnMessage = err.Error()
			returnCode = NetworkJoinFailed
		}

		if isNetworkJoined {
			// Publish Network Container
			publishResponse, publishError = nmagentclient.PublishNetworkContainer(
				req.NetworkContainerID,
				req.CreateNetworkContainerURL,
				req.CreateNetworkContainerRequestBody)
			if publishError != nil || publishResponse.StatusCode != http.StatusOK {
				returnMessage = fmt.Sprintf("Failed to publish Network Container: %s", req.NetworkContainerID)
				returnCode = NetworkContainerPublishFailed
				logger.Errorf("[Azure-CNS] %s", returnMessage)
			}
		}
	default:
		returnMessage = "PublishNetworkContainer API expects a POST"
		returnCode = UnsupportedVerb
	}

	if publishError != nil {
		publishErrorStr = publishError.Error()
	}

	if publishResponse != nil {
		publishStatusCode = publishResponse.StatusCode

		var errParse error
		publishResponseBody, errParse = ioutil.ReadAll(publishResponse.Body)
		if errParse != nil {
			returnMessage = fmt.Sprintf("Failed to parse the publish body. Error: %v", errParse)
			returnCode = UnexpectedError
			logger.Errorf("[Azure-CNS] %s", returnMessage)
		}

		publishResponse.Body.Close()
	}

	response := cns.PublishNetworkContainerResponse{
		Response: cns.Response{
			ReturnCode: returnCode,
			Message:    returnMessage,
		},
		PublishErrorStr:     publishErrorStr,
		PublishStatusCode:   publishStatusCode,
		PublishResponseBody: publishResponseBody,
	}

	err = service.Listener.Encode(w, &response)
	logger.Response(service.Name, response, response.Response.ReturnCode, ReturnCodeToString(response.Response.ReturnCode), err)
}

// Unpublish Network Container by calling nmagent
func (service *HTTPRestService) unpublishNetworkContainer(w http.ResponseWriter, r *http.Request) {
	logger.Printf("[Azure-CNS] UnpublishNetworkContainer")

	var (
		err                   error
		req                   cns.UnpublishNetworkContainerRequest
		returnCode            int
		returnMessage         string
		unpublishResponse     *http.Response
		unpublishStatusCode   int
		unpublishResponseBody []byte
		unpublishError        error
		unpublishErrorStr     string
		isNetworkJoined       bool
	)

	err = service.Listener.Decode(w, r, &req)

	// reqCopy creates a copy of incoming request. It doesn't copy the authentication token info
	// to avoid logging it.
	reqCopy := cns.UnpublishNetworkContainerRequest{
		NetworkID:                 req.NetworkID,
		NetworkContainerID:        req.NetworkContainerID,
		JoinNetworkURL:            req.JoinNetworkURL,
		DeleteNetworkContainerURL: strings.Split(req.DeleteNetworkContainerURL, "authenticationToken")[0],
	}

	logger.Request(service.Name, &reqCopy, err)
	if err != nil {
		return
	}

	switch r.Method {
	case "POST":
		// Join Network if not joined already
		isNetworkJoined = service.isNetworkJoined(req.NetworkID)
		if !isNetworkJoined {
			unpublishResponse, unpublishError, err = service.joinNetwork(req.NetworkID, req.JoinNetworkURL)
			if err == nil {
				isNetworkJoined = true
			} else {
				returnMessage = err.Error()
				returnCode = NetworkJoinFailed
			}
		}

		if isNetworkJoined {
			// Unpublish Network Container
			unpublishResponse, unpublishError = nmagentclient.UnpublishNetworkContainer(
				req.NetworkContainerID,
				req.DeleteNetworkContainerURL)
			if unpublishError != nil || unpublishResponse.StatusCode != http.StatusOK {
				returnMessage = fmt.Sprintf("Failed to unpublish Network Container: %s", req.NetworkContainerID)
				returnCode = NetworkContainerUnpublishFailed
				logger.Errorf("[Azure-CNS] %s", returnMessage)
			}

			if unpublishResponse != nil {
				var errParse error
				unpublishResponseBody, errParse = ioutil.ReadAll(unpublishResponse.Body)
				if errParse != nil {
					returnMessage = fmt.Sprintf("Failed to parse the unpublish body. Error: %v", errParse)
					returnCode = UnexpectedError
					logger.Errorf("[Azure-CNS] %s", returnMessage)
				}

				unpublishResponse.Body.Close()
			}
		}
	default:
		returnMessage = "UnpublishNetworkContainer API expects a POST"
		returnCode = UnsupportedVerb
	}

	if unpublishError != nil {
		unpublishErrorStr = unpublishError.Error()
	}

	if unpublishResponse != nil {
		unpublishStatusCode = unpublishResponse.StatusCode
	}

	response := cns.UnpublishNetworkContainerResponse{
		Response: cns.Response{
			ReturnCode: returnCode,
			Message:    returnMessage,
		},
		UnpublishErrorStr:     unpublishErrorStr,
		UnpublishStatusCode:   unpublishStatusCode,
		UnpublishResponseBody: unpublishResponseBody,
	}

	err = service.Listener.Encode(w, &response)
	logger.Response(service.Name, response, response.Response.ReturnCode, ReturnCodeToString(response.Response.ReturnCode), err)
}

func (service *HTTPRestService) allocateIPConfig(w http.ResponseWriter, r *http.Request) {
	var (
		err           error
		req           cns.GetNetworkContainerRequest
		ipConfig      cns.IPSubnet
		returnCode    int
		returnMessage string
	)

	err := service.Listener.Decode(w, r, &req)
	logger.Request(service.Name, &req, err)
	if err != nil {
		return
	}

	if ipConfig, err = getIPConfig(service, req); err != nil {
		returnCode = UnexpectedError
		returnMessage = fmt.Sprintf("AllocateIPConfig failed: %v", err)
	}

	resp := cns.Response{
		ReturnCode: returnCode,
		Message:    returnMessage,
	}

	service.lock.Lock()
	containerDetails := service.state.ContainerStatus[ipConfig.NCID]
	service.lock.Unlock()

	savedReq := containerDetails.CreateNetworkContainerRequest
	reserveResp := &cns.GetNetworkContainerResponse{
		Response:                   resp,
		NetworkContainerID:         savedReq.NetworkContainerid,
		IPConfiguration:            savedReq.IPConfiguration,
		Routes:                     savedReq.Routes,
		CnetAddressSpace:           savedReq.CnetAddressSpace,
		MultiTenancyInfo:           savedReq.MultiTenancyInfo,
		PrimaryInterfaceIdentifier: savedReq.PrimaryInterfaceIdentifier,
		LocalIPConfiguration:       savedReq.LocalIPConfiguration,
		AllowHostToNCCommunication: savedReq.AllowHostToNCCommunication,
		AllowNCToHostCommunication: savedReq.AllowNCToHostCommunication,
	}
	reserveResp.IPConfiguration.IPSubnet = ipConfig

	err = service.Listener.Encode(w, &reserveResp)
	logger.Response(service.Name, reserveResp, resp.ReturnCode, ReturnCodeToString(resp.ReturnCode), err)
}

func (service *HTTPRestService) releaseIPConfig(w http.ResponseWriter, r *http.Request) {
	var (
		err           error
		req           cns.GetNetworkContainerRequest
		ipConfig      cns.IPSubnet
		returnCode    int
		returnMessage string
	)

	err = service.Listener.Decode(w, r, &req)
	logger.Request(service.Name, &req, err)
	if err != nil {
		return
	}

	defer func() {
		resp := cns.Response{}

		if err ! = nil {
			returnCode = UnexpectedError
			returnMessage = err.Error()
		}

		err = service.Listener.Encode(w, &resp)
		logger.Response(service.Name, resp, resp.ReturnCode, ReturnCodeToString(resp.ReturnCode), err)
	}()

	service.lock.Lock()
	defer service.lock.Unlock()

	if service.state.OrchestratorContext != cns.Kubernetes {
		err = fmt.Errorf("AllocateIPconfig API supported only for kubernetes orchestrator")
		return ipState, err
	}

	// retrieve podinfo  from orchestrator context
	if err := json.UnMarshal(req.OrchestratorContext, &podInfo); err != nil {
		return ipState, err
	}


	ipID := service.state.ContainerIPIDByOrchestratorContext[podInfo.PodName+podInfo.PodNamespace]
	if ipID != nil {
		if ipState, isExist = service.state.containerIPConfigState[ipID]; isExist {

		}
		return ipState, fmt.Errorf("Pod->IPIP exists but IPID to IPConfig doesn't exist")
	} else {

	}

}

// If IPConfig is already allocated for pod, it returns that else it returns one of the available ipconfigs.
func getIPConfig(service *HTTPRestService, req cns.GetNetworkContainerRequest) (cns.ContainerIPConfigState, error) {

	service.lock.Lock()
	defer service.lock.Unlock()

	var (
		podInfo cns.KubernetesPodInfo
		ipState cns.ContainerIPConfigState
		isExist bool
	)

	if service.state.OrchestratorContext != cns.Kubernetes {
		return ipState, fmt.Errorf("AllocateIPconfig API supported only for kubernetes orchestrator")
	}

	// retrieve podinfo  from orchestrator context
	if err := json.UnMarshal(req.OrchestratorContext, &podInfo); err != nil {
		return ipState, err
	}

	// check if ipconfig already allocated for this pod and return that
	ipID := service.state.ContainerIPIDByOrchestratorContext[podInfo.PodName+podInfo.PodNamespace]
	if ipID != nil {
		if ipState, isExist = service.state.containerIPConfigState[ipID]; isExist {
			return ipState, nil
		}
		return ipState, fmt.Errorf("Pod->IPIP exists but IPID to IPConfig doesn't exist")
	}

	// return free ipconfig
	for _, ipState = range service.state.containerIPConfigState {
		if ipState.state == cns.Available {
			ipState.State = cns.Allocated
			return ipState
		}
	}
}

func logNCSnapshot(createNetworkContainerRequest cns.CreateNetworkContainerRequest) {
	var aiEvent = aitelemetry.Event{
		EventName:  logger.CnsNCSnapshotEventStr,
		Properties: make(map[string]string),
		ResourceID: createNetworkContainerRequest.NetworkContainerid,
	}

	aiEvent.Properties[logger.IpConfigurationStr] = fmt.Sprintf("%+v", createNetworkContainerRequest.IPConfiguration)
	aiEvent.Properties[logger.LocalIPConfigurationStr] = fmt.Sprintf("%+v", createNetworkContainerRequest.LocalIPConfiguration)
	aiEvent.Properties[logger.PrimaryInterfaceIdentifierStr] = createNetworkContainerRequest.PrimaryInterfaceIdentifier
	aiEvent.Properties[logger.MultiTenancyInfoStr] = fmt.Sprintf("%+v", createNetworkContainerRequest.MultiTenancyInfo)
	aiEvent.Properties[logger.CnetAddressSpaceStr] = fmt.Sprintf("%+v", createNetworkContainerRequest.CnetAddressSpace)
	aiEvent.Properties[logger.AllowNCToHostCommunicationStr] = fmt.Sprintf("%t", createNetworkContainerRequest.AllowNCToHostCommunication)
	aiEvent.Properties[logger.AllowHostToNCCommunicationStr] = fmt.Sprintf("%t", createNetworkContainerRequest.AllowHostToNCCommunication)
	aiEvent.Properties[logger.NetworkContainerTypeStr] = createNetworkContainerRequest.NetworkContainerType
	aiEvent.Properties[logger.OrchestratorContextStr] = fmt.Sprintf("%s", createNetworkContainerRequest.OrchestratorContext)

	logger.LogEvent(aiEvent)
}

// Sends network container snapshots to App Insights telemetry.
func (service *HTTPRestService) logNCSnapshots() {

	for _, ncStatus := range service.state.ContainerStatus {
		logNCSnapshot(ncStatus.CreateNetworkContainerRequest)
	}

	logger.Printf("[Azure CNS] Logging periodic NC snapshots. NC Count %d", len(service.state.ContainerStatus))
}

// Sets up periodic timer for sending network container snapshots
func (service *HTTPRestService) SendNCSnapShotPeriodically(ncSnapshotIntervalInMinutes int, stopSnapshot chan bool) {

	// Emit snapshot on startup and then emit it periodically.
	service.logNCSnapshots()

	snapshot := time.NewTicker(time.Minute * time.Duration(ncSnapshotIntervalInMinutes)).C
	for {
		select {
		case <-snapshot:
			service.logNCSnapshots()
		case <-stopSnapshot:
			return
		}
	}
}
