// Copyright 2017 Microsoft. All rights reserved.
// MIT License

package restserver

import (
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"net"

	"github.com/Azure/azure-container-networking/cns"
	"github.com/Azure/azure-container-networking/cns/common"
	"github.com/Azure/azure-container-networking/cns/dockerclient"
	//"github.com/Azure/azure-container-networking/cns/hnsclient"
	"github.com/Azure/azure-container-networking/cns/imdsclient"
	"github.com/Azure/azure-container-networking/cns/ipamclient"
	"github.com/Azure/azure-container-networking/cns/routes"
	"github.com/Azure/azure-container-networking/log"
	"github.com/Azure/azure-container-networking/store"
)

const (
	// Key against which CNS state is persisted.
	storeKey   = "ContainerNetworkService"
	bridgeMode = "bridge"
)

// httpRestService represents http listener for CNS - Container Networking Service.
type httpRestService struct {
	*cns.Service
	dockerClient *dockerclient.DockerClient
	imdsClient   *imdsclient.ImdsClient
	ipamClient   *ipamclient.IpamClient
	routingTable *routes.RoutingTable
	//hnsClient          *hnsclient.HnsClient
	store store.KeyValueStore
	state httpRestServiceState
}

type networkInfo struct {
	Subnet        string
	InternetResID string
}

// httpRestServiceState contains the state we would like to persist.
type httpRestServiceState struct {
	Location       string
	NetworkType    string
	Initialized    bool
	TimeStamp      time.Time
	InternetPoolID string
	NetworkInfoMap map[string]*networkInfo
}

// HTTPService describes the min API interface that every service should have.
type HTTPService interface {
	common.ServiceAPI
}

func executeShellCommand(command string) error {
	log.Debugf("[Azure-CNS] %s", command)
	cmd := exec.Command("sh", "-c", command)
	err := cmd.Start()
	if err != nil {
		return err
	}
	return cmd.Wait()
}

func (service *httpRestService) getInternetBridgePool(pool string) error {
	log.Printf("Create Internet Bridge Pool %v", pool)

	asID, err := service.ipamClient.GetAddressSpace()
	if err != nil {
		log.Printf("[Azure CNS] Error while getting address space %v", err.Error())
		return err
	}

	service.state.InternetPoolID, err = service.ipamClient.GetPoolID(asID, pool, "internet")
	if err != nil {
		log.Printf("[Azure CNS] Error while allocating Pool%v", err.Error())
		return err
	}

	return nil
}

// NewHTTPRestService creates a new HTTP Service object.
func NewHTTPRestService(config *common.ServiceConfig) (HTTPService, error) {
	service, err := cns.NewService(config.Name, config.Version, config.Store)
	if err != nil {
		return nil, err
	}

	imdsClient := &imdsclient.ImdsClient{}
	routingTable := &routes.RoutingTable{}
	//hnsclient := &hnsclient.HnsClient{}
	dc, err := dockerclient.NewDefaultDockerClient(imdsClient)
	if err != nil {
		return nil, err
	}

	ic, err := ipamclient.NewIpamClient("")
	if err != nil {
		return nil, err
	}

	httpRestService := &httpRestService{
		Service:      service,
		store:        service.Service.Store,
		dockerClient: dc,
		imdsClient:   imdsClient,
		ipamClient:   ic,
		routingTable: routingTable,
		//hnsClient:    hnsclient,
	}
	httpRestService.state.NetworkInfoMap = make(map[string]*networkInfo)

	return httpRestService, nil
}

// Start starts the CNS listener.
func (service *httpRestService) Start(config *common.ServiceConfig) error {

	err := service.Initialize(config)
	if err != nil {
		log.Printf("[Azure CNS]  Failed to initialize base service, err:%v.", err)
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

	// handlers for v0.1
	listener.AddHandler(cns.V1Prefix+cns.SetEnvironmentPath, service.setEnvironment)
	listener.AddHandler(cns.V1Prefix+cns.CreateNetworkPath, service.createNetwork)
	listener.AddHandler(cns.V1Prefix+cns.DeleteNetworkPath, service.deleteNetwork)
	listener.AddHandler(cns.V1Prefix+cns.ReserveIPAddressPath, service.reserveIPAddress)
	listener.AddHandler(cns.V1Prefix+cns.ReleaseIPAddressPath, service.releaseIPAddress)
	listener.AddHandler(cns.V1Prefix+cns.GetHostLocalIPPath, service.getHostLocalIP)
	listener.AddHandler(cns.V1Prefix+cns.GetIPAddressUtilizationPath, service.getIPAddressUtilization)
	listener.AddHandler(cns.V1Prefix+cns.GetUnhealthyIPAddressesPath, service.getUnhealthyIPAddresses)

	service.restoreState()

	log.Printf("nw to subnet map %v", service.state.NetworkInfoMap)
	log.Printf("[Azure CNS]  Listening.")

	return nil
}

// Stop stops the CNS.
func (service *httpRestService) Stop() {
	service.Uninitialize()
	log.Printf("[Azure CNS]  Service stopped.")
}

// Handles requests to set the environment type.
func (service *httpRestService) setEnvironment(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Azure CNS] setEnvironment")

	var req cns.SetEnvironmentRequest
	err := service.Listener.Decode(w, r, &req)
	log.Request(service.Name, &req, err)

	if err != nil {
		return
	}

	switch r.Method {
	case "POST":
		log.Printf("[Azure CNS]  POST received for SetEnvironment.")
		service.state.Location = req.Location
		service.state.NetworkType = req.NetworkType
		service.state.Initialized = true
		service.saveState()
	default:
	}

	resp := &cns.Response{ReturnCode: 0}
	err = service.Listener.Encode(w, &resp)

	log.Response(service.Name, resp, err)
}

// Handles CreateNetwork requests.
func (service *httpRestService) createNetwork(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Azure CNS] createNetwork")

	var err error
	returnCode := 0
	returnMessage := ""

	if service.state.Initialized {
		var req cns.CreateNetworkRequest
		err = service.Listener.Decode(w, r, &req)
		log.Request(service.Name, &req, err)

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
							log.Printf("[Azure CNS] Going to create network with name %v.", req.NetworkName)

							err = rt.GetRoutingTable()
							if err != nil {
								// We should not fail the call to create network for this.
								// This is because restoring routes is a fallback mechanism in case
								// network driver is not behaving as expected.
								// The responsibility to restore routes is with network driver.
								log.Printf("[Azure CNS] Unable to get routing table from node, %+v.", err.Error())
							}

							primaryNic, err := service.imdsClient.GetPrimaryInterfaceInfoFromMemory()
							if err != nil {
								returnMessage = fmt.Sprintf("[Azure CNS] Error. GetPrimaryInterfaceFromHost failed %v.", err.Error())
								log.Printf(returnMessage)
								returnCode = UnexpectedError
							} else {
								// docker limits itself to only one subnet per network
								// so even if we have multiple NICs, each in different subnet, we have to pick one.
								enableSnat := true
								var options map[string]interface{}

								if req.Options != nil {
									if _, ok := options[OptDisableSnat]; ok {
										enableSnat = false
									}
								}

								if enableSnat {
									options = make(map[string]interface{})
									options[NetworkMode] = bridgeMode
								}

								err = dc.CreateNetwork(req.NetworkName, primaryNic.Subnet, defaultNetworkPluginName, options, nil)
								if err != nil {
									returnMessage = fmt.Sprintf("[Azure CNS] Error. CreateNetwork failed %v.", err.Error())
									returnCode = UnexpectedError
								}

								if enableSnat {
									cmd := fmt.Sprintf("iptables -t nat -A POSTROUTING -m iprange ! --dst-range 168.63.129.16 -m addrtype ! --dst-type local ! -d %v -j MASQUERADE",
										primaryNic.Subnet)
									err = executeShellCommand(cmd)
									if err != nil {
										returnMessage = "SNAT Iptable rule was not set"
										log.Printf(returnMessage)
										returnCode = UnexpectedError
									}
								}
							}
							err = rt.RestoreRoutingTable()
							if err != nil {
								log.Printf("[Azure CNS] Unable to restore routing table on node, %+v.", err.Error())
							}

						case "StandAlone":
							returnMessage = fmt.Sprintf("[Azure CNS] Error. Underlay network is not supported in StandAlone environment. %v.", err.Error())
							returnCode = UnsupportedEnvironment
						}

					case "Overlay":
						log.Printf("[Azure CNS] Going to create an overlay network with name %v.", req.NetworkName)

						var subnetStr string
						vnetSpace := fmt.Sprintf("%v/%v", req.OverlayConfiguration.OverlaySubnet.IPAddress,
							req.OverlayConfiguration.OverlaySubnet.PrefixLength)

						log.Printf("[Azure-CNS] Created overlay subnet as %v", subnetStr)
						options := make(map[string]interface{})
						ipamOptions := make(map[string]interface{})
						var nodeIDList string

						for _, nodeConfig := range req.OverlayConfiguration.NodeConfig {

							log.Printf("[Azure CNS] Node config %v\n", nodeConfig)

							if nodeConfig.NodeIP == req.OverlayConfiguration.LocalNodeIP {
								subnetStr = fmt.Sprintf("%v/%v", nodeConfig.NodeSubnet.IPAddress,
									nodeConfig.NodeSubnet.PrefixLength)
								continue
							}

							nodeIDList = fmt.Sprintf("%s,%s", nodeConfig.NodeID, nodeIDList)

							nodeIP := net.ParseIP(nodeConfig.NodeIP)
							options[nodeConfig.NodeID] = fmt.Sprintf("%s,%s/%v", nodeIP, nodeConfig.NodeSubnet.IPAddress, nodeConfig.NodeSubnet.PrefixLength)
						}

						log.Printf("[Azure CNS] node id list %v\n", nodeIDList)

						if nodeIDList != "" {
							if nodeIDList[len(nodeIDList)-1] == ',' {
								nodeIDList = nodeIDList[:len(nodeIDList)-1]
							}
						}

						options["internetBridgeSubnet"] = "169.254.0.0/16"

						err = service.getInternetBridgePool("169.254.0.0/16")
						if err != nil {
							returnMessage = fmt.Sprintf("[Azure CNS] Error while creating internet bridge Pool %v", err.Error())
							log.Printf(returnMessage)
							returnCode = UnexpectedError
							break
						}

						reservationID, err := exec.Command("uuidgen").Output()
						if err != nil {
							returnMessage = fmt.Sprintf("[Azure CNS] Error while generating Guid %v", err.Error())
							log.Printf(returnMessage)
							returnCode = UnexpectedError
							break
						}

						log.Printf("[Azure CNS] reservation ID %v for internet bridge", reservationID)
						reservedIP, err := service.ipamClient.ReserveIPAddress(service.state.InternetPoolID, string(reservationID))
						if err != nil {
							returnMessage = fmt.Sprintf("[Azure CNS] Error while reserving IP for internet bridge %v", err.Error())
							log.Printf(returnMessage)
							returnCode = UnexpectedError
							break
						}

						log.Printf("[Azure CNS] reserved address %v for internet bridge", reservedIP)

						options["nodeList"] = nodeIDList
						options["vxlanID"] = req.OverlayConfiguration.VxLanID
						options["networkName"] = req.NetworkName
						options["vnetSpace"] = vnetSpace
						options["internetBridgeIP"] = reservedIP
						options["localNodeIP"] = req.OverlayConfiguration.LocalNodeIP
						options["localNodeInterface"] = req.OverlayConfiguration.LocalNodeInterface
						options["localNodeGateway"] = req.OverlayConfiguration.LocalNodeGateway
						ipamOptions["network.name"] = req.NetworkName

						err = dc.CreateNetwork(req.NetworkName, subnetStr, overlayPluginName, options, ipamOptions)
						if err != nil {
							returnMessage = fmt.Sprintf("[Azure CNS] Error. CreateNetwork %v of type overlay failed %v.", req.NetworkName, err.Error())
							returnCode = UnexpectedError
						}

						nwInfo := &networkInfo{
							Subnet:        subnetStr,
							InternetResID: string(reservationID),
						}

						service.state.NetworkInfoMap[req.NetworkName] = nwInfo

						log.Printf("Network to Subnet map %v\n", service.state.NetworkInfoMap)
						service.saveState()

					}
				} else {
					returnMessage = fmt.Sprintf("[Azure CNS] Received a request to create an already existing network %v", req.NetworkName)
					log.Printf(returnMessage)
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

	log.Response(service.Name, resp, err)
}

// Handles DeleteNetwork requests.
func (service *httpRestService) deleteNetwork(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Azure CNS] deleteNetwork")

	var req cns.DeleteNetworkRequest
	returnCode := 0
	returnMessage := ""
	err := service.Listener.Decode(w, r, &req)
	log.Request(service.Name, &req, err)

	if err != nil {
		return
	}

	switch r.Method {
	case "POST":
		dc := service.dockerClient
		err := dc.NetworkExists(req.NetworkName)

		// Network does exist
		if err == nil {
			log.Printf("[Azure CNS] Goign to delete network with name %v.", req.NetworkName)
			err := dc.DeleteNetwork(req.NetworkName)
			if err != nil {
				returnMessage = fmt.Sprintf("[Azure CNS] Error. DeleteNetwork failed %v.", err.Error())
				returnCode = UnexpectedError
			}
		} else {
			if err == fmt.Errorf("Network not found") {
				log.Printf("[Azure CNS] Received a request to delete network that does not exist: %v.", req.NetworkName)
			} else {
				returnCode = UnexpectedError
				returnMessage = err.Error()
			}
		}

	default:
		returnMessage = "[Azure CNS] Error. DeleteNetwork did not receive a POST."
		returnCode = InvalidParameter
	}

	nwInfo := service.state.NetworkInfoMap[req.NetworkName]
	if nwInfo != nil {
		service.ipamClient.ReleaseIPAddress(service.state.InternetPoolID, nwInfo.InternetResID)
	}

	delete(service.state.NetworkInfoMap, req.NetworkName)
	service.saveState()

	resp := &cns.Response{
		ReturnCode: returnCode,
		Message:    returnMessage,
	}

	err = service.Listener.Encode(w, &resp)

	log.Response(service.Name, resp, err)
}

// Handles ip reservation requests.
func (service *httpRestService) reserveIPAddress(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Azure CNS] reserveIPAddress")

	// reserve ip should take in the network name as a parameter
	// let's make it mandatory parameter as we have no one to be backward compatible wit
	var req cns.ReserveIPAddressRequest
	returnMessage := ""
	returnCode := 0
	addr := ""
	address := ""

	err := service.Listener.Decode(w, r, &req)
	subnetStr := ""
	log.Request(service.Name, &req, err)

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

		// check if subnet for the network was provided during network create call
		// if yes, then use it instead of asking the IMDS
		// Logic remains same in release.
		if service.state.NetworkType == "Underlay" {
			ifInfo, err := service.imdsClient.GetPrimaryInterfaceInfoFromMemory()
			if err != nil {
				returnMessage = fmt.Sprintf("[Azure CNS] Error. GetPrimaryIfaceInfo failed %v", err.Error())
				returnCode = UnexpectedError
				break
			}
			subnetStr = ifInfo.Subnet
		} else if service.state.NetworkType == "Overlay" {
			subnetStr = service.state.NetworkInfoMap[req.NetworkName].Subnet
			if subnetStr == "" {
				returnMessage = fmt.Sprintf("[Azure CNS] Error. Subnet doesn't exist for this network %v", req.NetworkName)
				returnCode = UnexpectedError
				break
			}
		}

		asID, err := ic.GetAddressSpace()
		if err != nil {
			returnMessage = fmt.Sprintf("[Azure CNS] Error. GetAddressSpace failed %v", err.Error())
			returnCode = UnexpectedError
			break
		}

		poolID, err := ic.GetPoolID(asID, subnetStr, req.NetworkName)
		if err != nil {
			returnMessage = fmt.Sprintf("[Azure CNS] Error. GetPoolID failed %v", err.Error())
			returnCode = UnexpectedError
			break
		}

		addr, err = ic.ReserveIPAddress(poolID, req.ReservationID)
		if err != nil {
			returnMessage = fmt.Sprintf("ReserveIpAddress failed with %+v", err.Error())
			returnCode = AddressUnavailable
			break
		}

		addressIP, _, err := net.ParseCIDR(addr)
		if err != nil {
			returnMessage = fmt.Sprintf("ParseCIDR failed with %+v", err.Error())
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

	log.Response(service.Name, reserveResp, err)
}

// Handles release ip reservation requests.
func (service *httpRestService) releaseIPAddress(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Azure CNS] releaseIPAddress")

	var req cns.ReleaseIPAddressRequest
	returnMessage := ""
	returnCode := 0
	subnetStr := ""

	err := service.Listener.Decode(w, r, &req)
	log.Request(service.Name, &req, err)

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

		if service.state.NetworkType == "Underlay" {
			ifInfo, err := service.imdsClient.GetPrimaryInterfaceInfoFromMemory()
			if err != nil {
				returnMessage = fmt.Sprintf("[Azure CNS] Error. GetPrimaryIfaceInfo failed %v", err.Error())
				returnCode = UnexpectedError
				break
			}
			subnetStr = ifInfo.Subnet
		} else if service.state.NetworkType == "Overlay" {
			subnetStr = service.state.NetworkInfoMap[req.NetworkName].Subnet
			if subnetStr == "" {
				returnMessage = fmt.Sprintf("[Azure CNS] Error. Subnet doesn't exist for this network %v", req.NetworkName)
				returnCode = UnexpectedError
				break
			}
		}

		asID, err := ic.GetAddressSpace()
		if err != nil {
			returnMessage = fmt.Sprintf("[Azure CNS] Error. GetAddressSpace failed %v", err.Error())
			returnCode = UnexpectedError
			break
		}

		poolID, err := ic.GetPoolID(asID, subnetStr, req.NetworkName)
		if err != nil {
			returnMessage = fmt.Sprintf("[Azure CNS] Error. GetPoolID failed %v", err.Error())
			returnCode = UnexpectedError
			break
		}

		err = ic.ReleaseIPAddress(poolID, req.ReservationID)
		if err != nil {
			returnMessage = fmt.Sprintf("ReleaseIpAddress failed with %+v", err.Error())
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

	log.Response(service.Name, resp, err)
}

// Retrieves the host local ip address. Containers can talk to host using this IP address.
func (service *httpRestService) getHostLocalIP(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Azure CNS] getHostLocalIP")
	log.Request(service.Name, "getHostLocalIP", nil)

	var found bool
	var errmsg string
	hostLocalIP := "0.0.0.0"

	if service.state.Initialized {
		switch r.Method {
		case "GET":
			found, hostLocalIP, errmsg = service.getOSSpecificHostLocalIP(service.state.NetworkType)
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

	log.Response(service.Name, hostLocalIPResponse, err)
}

// Handles ip address utilization requests.
func (service *httpRestService) getIPAddressUtilization(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Azure CNS] getIPAddressUtilization")
	log.Request(service.Name, "getIPAddressUtilization", nil)

	var req cns.IPAddressesUtilizationRequest
	returnMessage := ""
	returnCode := 0
	capacity := 0
	available := 0
	subnetStr := ""
	var unhealthyAddrs []string

	err := service.Listener.Decode(w, r, &req)
	log.Request(service.Name, &req, err)
	if err != nil {
		return
	}

	switch r.Method {
	case "POST":
		ic := service.ipamClient

		if service.state.NetworkType == "Underlay" {
			ifInfo, err := service.imdsClient.GetPrimaryInterfaceInfoFromMemory()
			if err != nil {
				returnMessage = fmt.Sprintf("[Azure CNS] Error. GetPrimaryIfaceInfo failed %v", err.Error())
				returnCode = UnexpectedError
				break
			}
			subnetStr = ifInfo.Subnet
		} else if service.state.NetworkType == "Overlay" {
			log.Printf("Network to Subnet map %v\n", service.state.NetworkInfoMap)
			subnetStr = service.state.NetworkInfoMap[req.NetworkName].Subnet
			if subnetStr == "" {
				returnMessage = fmt.Sprintf("[Azure CNS] Error. Subnet doesn't exist for this network %v", req.NetworkName)
				returnCode = UnexpectedError
				break
			}
		}

		asID, err := ic.GetAddressSpace()
		if err != nil {
			returnMessage = fmt.Sprintf("[Azure CNS] Error. GetAddressSpace failed %v", err.Error())
			returnCode = UnexpectedError
			break
		}

		poolID, err := ic.GetPoolID(asID, subnetStr, req.NetworkName)
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
		log.Printf("Capacity %v Available %v UnhealthyAddrs %v", capacity, available, unhealthyAddrs)

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

	err = service.Listener.Encode(w, &utilResponse)

	log.Response(service.Name, utilResponse, err)
}

// Handles retrieval of ip addresses that are available to be reserved from ipam driver.
func (service *httpRestService) getAvailableIPAddresses(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Azure CNS] getAvailableIPAddresses")
	log.Request(service.Name, "getAvailableIPAddresses", nil)

	switch r.Method {
	case "GET":
	default:
	}

	resp := cns.Response{ReturnCode: 0}
	ipResp := &cns.GetIPAddressesResponse{Response: resp}
	err := service.Listener.Encode(w, &ipResp)

	log.Response(service.Name, ipResp, err)
}

// Handles retrieval of reserved ip addresses from ipam driver.
func (service *httpRestService) getReservedIPAddresses(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Azure CNS] getReservedIPAddresses")
	log.Request(service.Name, "getReservedIPAddresses", nil)

	switch r.Method {
	case "GET":
	default:
	}

	resp := cns.Response{ReturnCode: 0}
	ipResp := &cns.GetIPAddressesResponse{Response: resp}
	err := service.Listener.Encode(w, &ipResp)

	log.Response(service.Name, ipResp, err)
}

// Handles retrieval of ghost ip addresses from ipam driver.
func (service *httpRestService) getUnhealthyIPAddresses(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Azure CNS] getUnhealthyIPAddresses")
	log.Request(service.Name, "getUnhealthyIPAddresses", nil)

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

		poolID, err := ic.GetPoolID(asID, ifInfo.Subnet, "")
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
		log.Printf("Capacity %v Available %v UnhealthyAddrs %v", capacity, available, unhealthyAddrs)

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

	log.Response(service.Name, ipResp, err)
}

// getAllIPAddresses retrieves all ip addresses from ipam driver.
func (service *httpRestService) getAllIPAddresses(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Azure CNS] getAllIPAddresses")
	log.Request(service.Name, "getAllIPAddresses", nil)

	switch r.Method {
	case "GET":
	default:
	}

	resp := cns.Response{ReturnCode: 0}
	ipResp := &cns.GetIPAddressesResponse{Response: resp}
	err := service.Listener.Encode(w, &ipResp)

	log.Response(service.Name, ipResp, err)
}

// Handles health report requests.
func (service *httpRestService) getHealthReport(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Azure CNS] getHealthReport")
	log.Request(service.Name, "getHealthReport", nil)

	switch r.Method {
	case "GET":
	default:
	}

	resp := &cns.Response{ReturnCode: 0}
	err := service.Listener.Encode(w, &resp)

	log.Response(service.Name, resp, err)
}

// saveState writes CNS state to persistent store.
func (service *httpRestService) saveState() error {
	log.Printf("[Azure CNS] saveState")

	// Skip if a store is not provided.
	if service.store == nil {
		log.Printf("[Azure CNS]  store not initialized.")
		return nil
	}

	// Update time stamp.
	service.state.TimeStamp = time.Now()
	err := service.store.Write(storeKey, &service.state)
	if err == nil {
		log.Printf("[Azure CNS]  State saved successfully.\n")
	} else {
		log.Printf("[Azure CNS]  Failed to save state., err:%v\n", err)
	}

	return err
}

// restoreState restores CNS state from persistent store.
func (service *httpRestService) restoreState() error {
	log.Printf("[Azure CNS] restoreState")

	// Skip if a store is not provided.
	if service.store == nil {
		log.Printf("[Azure CNS]  store not initialized.")
		return nil
	}

	// Read any persisted state.
	err := service.store.Read(storeKey, &service.state)
	if err != nil {
		if err == store.ErrKeyNotFound {
			// Nothing to restore.
			log.Printf("[Azure CNS]  No state to restore.\n")
			return nil
		}

		log.Printf("[Azure CNS]  Failed to restore state, err:%v\n", err)
		return err
	}

	log.Printf("[Azure CNS]  Restored state, %+v\n", service.state)
	return nil
}
