package main

import (
	"bytes"
	"dnc/api"
	"dnc/nmagentcontracts"
	"dnc/utils"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Azure/azure-container-networking/cns"
)

type CreateNetworkRequest struct {
	NetworkType string
}

type JoinSubnetRequest struct {
	AuthenticationToken string
}

type CreateReservationWithNodeConstraintRequest struct {
	NodeConstraint string
	SubnetName     string
}

type _AllocationRequest struct {
	SubnetName     string
	IPConstraint   string
	NodeConstraint string
}

type _AssociationInfo struct {
	NodeID      string
	InterfaceID string
}

type CreateNetworkContainerRequest struct {
	AllocationRequest _AllocationRequest
	AssociationInfo   _AssociationInfo
}

type CreateNetworkContainerUsingReservationRequest struct {
	ReservationID   string
	AssociationInfo _AssociationInfo
}

// client :- default client
var client = &http.Client{}
var nmAgentServer *MockServer
var cnsServer *MockServer

// TestMain :- main
func TestMain(m *testing.M) {
	if err := Setup(); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
	nmAgentServer.TearDownSuite()
	cnsServer.TearDownSuite()
}

// Setup :- Set up the environment (if necessary).
func Setup() error {
	nmAgentServer = new(MockServer)
	nmAgentServer.SetupServer("127.0.0.1:9001")
	cnsServer = new(MockServer)
	cnsServer.SetupServer("127.0.0.1:10090")
	return nil
}

func TestScenario(t *testing.T) {
	var scenario = struct {
		nid                  string
		nid2                 string
		dest                 string
		vnetid               string
		vnetid1              string
		ncid                 string
		ncid2                string
		subnet               string
		apiversion           string
		reservationID1       string
		reservationID2       string
		nodereq              api.NodeInfoRequest
		networkreq           CreateNetworkRequest
		subnetreq            JoinSubnetRequest
		reservationreq1      _AllocationRequest
		reservationreq2      CreateReservationWithNodeConstraintRequest
		networkcontainerreq  CreateNetworkContainerRequest
		networkcontainerreq2 CreateNetworkContainerUsingReservationRequest
		networkcontainerreq3 CreateNetworkContainerUsingReservationRequest
	}{
		"testnode",
		"testnode2",
		"http://192.168.1.3:9000",
		"testvnetid",
		"testvnetid1",
		"testncid",
		"testncid2",
		"subnet1",
		"?api-version=2018-03-01",
		"reservationID1",
		"reservationID2",
		api.NodeInfoRequest{IPAddresses: []string{"127.0.0.1"}},
		CreateNetworkRequest{"AzureNet"},
		JoinSubnetRequest{"0bb4132b-5932-4ec4-a47f-1b24f0eeb426"},
		_AllocationRequest{"subnet1", "10.0.0.15", ""},
		CreateReservationWithNodeConstraintRequest{"testnode", "subnet1"},
		CreateNetworkContainerRequest{_AllocationRequest{"subnet1", "10.0.2.7", ""}, _AssociationInfo{"testnode", "127.0.0.1"}},
		CreateNetworkContainerUsingReservationRequest{"reservationID2", _AssociationInfo{"testnode", "127.0.0.1"}},
		CreateNetworkContainerUsingReservationRequest{"reservationID2", _AssociationInfo{"testnode2", "127.0.0.1"}},
	}

	CreateNode(t, scenario.nodereq, scenario.dest, scenario.nid, scenario.apiversion)
	CreateNetwork(t, scenario.networkreq, scenario.dest, scenario.vnetid, scenario.apiversion)
	CreateNetworkAuthFailure(t, scenario.networkreq, scenario.dest, scenario.vnetid1, scenario.apiversion)
	JoinSubnetWithNoNetwork(t, scenario.dest, scenario.vnetid, scenario.subnet, scenario.apiversion)
	JoinSubnet(t, scenario.nid, scenario.nodereq, scenario.networkreq, scenario.dest, scenario.vnetid, scenario.subnet, scenario.apiversion)
	CreateReservation(t, scenario.reservationreq1, scenario.dest, scenario.vnetid, scenario.reservationID1, scenario.apiversion)
	CreateReservationNodeConstraint(t, scenario.reservationreq2, scenario.dest, scenario.vnetid, scenario.reservationID2, scenario.apiversion)
	CreateNetworkContainer(t, scenario.networkcontainerreq, scenario.dest, scenario.vnetid, scenario.ncid, scenario.apiversion)
	DeleteNetworkContainer(t, scenario.dest, scenario.vnetid, scenario.ncid, scenario.apiversion)
	CreateNetworkContainerUsingReservation(t, scenario.networkcontainerreq2, scenario.dest, scenario.vnetid, scenario.ncid2, scenario.apiversion)
	DeleteNetworkContainer(t, scenario.dest, scenario.vnetid, scenario.ncid2, scenario.apiversion)
}

func addOrUpdateNode(
	t *testing.T,
	dncEP string,
	nodeID string,
	apiVersion string,
	nodeInfoReq *api.NodeInfoRequest) {
	var body bytes.Buffer
	var errResponse api.JsonErr
	var err error
	var response *http.Response
	json.NewEncoder(&body).Encode(nodeInfoReq)

	response, err = http.Post(dncEP+"/nodes/"+nodeID+apiVersion, "application/json", &body)

	if err != nil {
		t.Errorf("CreateNode failed with error: %+v\n", err)
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		err = decodeResponse(response, &errResponse)
		fmt.Printf("%v", err != nil)
		if err != nil || errResponse.Code != 0 {
			t.Errorf("CreateNode failed with response %+v Err:%+v", errResponse, err)
			t.Fatal(err)
		}
	}

}

func deleteResource(
	t *testing.T,
	resourceURL string) {
	var errResponse api.JsonErr
	var err error

	fmt.Printf("Resource Url %s to delete \n", resourceURL)
	// Create client
	client := &http.Client{}
	// Create request
	req, err := http.NewRequest("DELETE", resourceURL, nil)
	if err != nil {
		t.Errorf("ResourceURL %s, Create delete request failed with error: %v\n", resourceURL, err)
		t.Fatal(err)
	}

	// Fetch Request
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("URL:%s, Delete failed with error: %+v\n", resourceURL, err)
		t.Fatal(err)
	}
	if resp != nil {
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		err = decodeResponse(resp, &errResponse)
		fmt.Printf("%v", err != nil)
		if err != nil || errResponse.Code != 0 {
			t.Errorf("Delete for resoucre URL:%s failed with response %+v Err:%v\n", resourceURL, errResponse, err)
			t.Fatal(err)
		}
	}
}

func validateNodeInfoResponse(
	t *testing.T,
	dncEP string,
	nodeID string,
	apiVersion string,
	nodeInfoReq *api.NodeInfoRequest) {

	response, err := http.Get(dncEP + "/nodes/" + nodeID + apiVersion)
	var nodeInfoResponse api.NodeInfoResponse
	err = decodeResponse(response, &nodeInfoResponse)
	if err != nil {
		t.Errorf("Decoding NodeInfo failed. Response %v Err:%v", nodeInfoResponse, err)
		t.Fatal(err)
	}
	fmt.Printf("Retrieved response:%v\n", nodeInfoResponse)
	expectedIPAddresses := make(map[uint32]bool)

	for _, ipAddressString := range nodeInfoReq.IPAddresses {
		expectedIPAddresses[utils.IP2Int(net.ParseIP(ipAddressString))] = true
	}

	if len(expectedIPAddresses) != len(nodeInfoResponse.NodeInfo.IPAddresses) {
		t.Errorf("Mismatch in count of expceted and received IP Addresses. Expected:%d, Received:%d",
			len(expectedIPAddresses),
			len(nodeInfoResponse.NodeInfo.IPAddresses))
		t.Fatal(err)
	}
	for _, ipAddressString := range nodeInfoResponse.NodeInfo.IPAddresses {
		receivedIPAddrUint32 := utils.IP2Int(net.ParseIP(ipAddressString))
		ok := expectedIPAddresses[receivedIPAddrUint32]
		if !ok {
			t.Errorf("Received IP Address %s not found in the expected list",
				ipAddressString)
			t.Fatal(err)
		}
	}
}

func getSubnetConfiguration(
	t *testing.T,
	dncEP string,
	networkID string,
	subnetName string,
	apiVersion string) *api.SubnetConfigurationResponse {
	response, err := http.Get(dncEP + "/networks/" + networkID + "/subnets/" + subnetName + apiVersion)
	var subnetConfiguration api.SubnetConfigurationResponse
	err = decodeResponse(response, &subnetConfiguration)
	if err != nil {
		t.Errorf("Decoding SubnetConfiguration failed. Response %v Err:%v", response, err)
		t.Fatal(err)
	}
	fmt.Printf("Retrieved response:%v\n", subnetConfiguration)
	return &subnetConfiguration
}

func CreateNode(t *testing.T, nodeInfoReq api.NodeInfoRequest, dncEP, nodeID, apiVersion string) {
	fmt.Printf("Created node %s with info %v\n", nodeID, nodeInfoReq)
	addOrUpdateNode(t, dncEP, nodeID, apiVersion, &nodeInfoReq)
	fmt.Printf("Successfully created node [%s]\n", nodeID)

	fmt.Printf("Validate persisted node info [%s]\n", nodeID)
	validateNodeInfoResponse(t, dncEP, nodeID, apiVersion, &nodeInfoReq)
	fmt.Println("Validate idempotency")
	addOrUpdateNode(t, dncEP, nodeID, apiVersion, &nodeInfoReq)
	fmt.Printf("Validate persisted node info [%s]\n", nodeID)
	validateNodeInfoResponse(t, dncEP, nodeID, apiVersion, &nodeInfoReq)

	updatedNodeInfoReq := api.NodeInfoRequest{
		IPAddresses: []string{"10.1.1.2"}}
	fmt.Printf("Update node %s with info %v\n", nodeID, updatedNodeInfoReq)
	addOrUpdateNode(t, dncEP, nodeID, apiVersion, &updatedNodeInfoReq)
	fmt.Printf("Successfully updated node [%s]\n", nodeID)
	fmt.Printf("Validate persisted node info [%s]\n", nodeID)
	validateNodeInfoResponse(t, dncEP, nodeID, apiVersion, &updatedNodeInfoReq)

	fmt.Printf("Deleting node %s\n", nodeID)
	nodeURL := dncEP + "/nodes/" + nodeID + apiVersion

	deleteResource(t, nodeURL)
	fmt.Printf("Node deleted successfully %s\n", nodeID)

	fmt.Println("Validate delete node idempotency")
	deleteResource(t, nodeURL)

	fmt.Println("Validate that appropriate error is thrown when querying for node that does not exist")
	response, err := http.Get(dncEP + "/nodes/" + nodeID + apiVersion)
	if response.StatusCode != http.StatusNotFound {
		t.Errorf("Received Invalid status code %d, Expected:%d",
			response.StatusCode,
			http.StatusNotFound)
		t.Fatal(err)
	}

	var respErr api.JsonErr
	decodeResponse(response, &respErr)
	fmt.Printf("Received error %s\n", respErr.String())
	if respErr.Code != api.NodeNotFound {
		t.Errorf("Invalid error code %d, Expected:%d",
			respErr.Code,
			api.NodeNotFound)
		t.Fatal(err)
	}

	addOrUpdateNode(t, dncEP, nodeID, apiVersion, &nodeInfoReq)
	fmt.Printf("Successfully created node [%s]\n", nodeID)
}

type MockServer struct {
	Server       *httptest.Server
	ResponseFunc func(http.ResponseWriter, *http.Request, interface{})
	Lock         sync.Mutex
	Context      interface{}
}

func (ts *MockServer) SetupServer(ipEP string) {
	ts.Server = httptest.NewUnstartedServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ts.Lock.Lock()
			defer ts.Lock.Unlock()
			if ts.ResponseFunc != nil {
				ts.ResponseFunc(w, r, ts.Context)
			}
		}))
	l, _ := net.Listen("tcp", ipEP)
	ts.Server.Listener = l
	ts.Server.Start()
}

func (s *MockServer) TearDownSuite() {
	s.Server.Close()
}

func (s *MockServer) SetupTest() {
	s.ResponseFunc = nil
}

type SignalInfo struct {
	Signal             chan bool
	SignalCount        int
	CurrentSignalCount int
}

type JoinResponse struct {
	HTTPStatus int
	SignalInfo
}

type GetNetworkResponse struct {
	SignalInfo      SignalInfo
	HTTPStatus      int
	NetworkResponse *nmagentcontracts.VirtualNetworkResponse
}

type NMAgentResponderContext struct {
	JoinResponse       *JoinResponse
	GetNetworkResponse *GetNetworkResponse
}

func signalTest(signalInfo *SignalInfo) {
	if signalInfo.CurrentSignalCount < signalInfo.SignalCount {
		signalInfo.Signal <- true
		signalInfo.CurrentSignalCount++
	}
}

func generateDefaultNMAgentResponderContext() *NMAgentResponderContext {
	context := new(NMAgentResponderContext)
	subnets := make([]nmagentcontracts.Subnet, 3)
	for index := 0; index < 3; index++ {
		subnets[index] = nmagentcontracts.Subnet{
			SubnetName:     fmt.Sprintf("subnet%v", index+1),
			AddressPrefix:  fmt.Sprintf("10.0.%v.0/24", index),
			DefaultGateway: fmt.Sprintf("10.0.%v.1", index)}
	}

	vnetResponse := nmagentcontracts.VirtualNetworkResponse{
		ResponseCode: "200",
		CNetSpace:    "10.0.0.0/8,172.16.0.0/16,192.168.1.0/24",
		DNSServers:   "8.8.8.8,9.9.9.9",
		VNetSpace:    "10.0.0.0/16",
		VnetVersion:  "4",
		Subnets:      subnets}
	context.GetNetworkResponse = &GetNetworkResponse{
		NetworkResponse: &vnetResponse,
		HTTPStatus:      http.StatusOK,
		SignalInfo: SignalInfo{
			Signal:             make(chan bool),
			SignalCount:        1,
			CurrentSignalCount: 0}}

	context.JoinResponse = &JoinResponse{
		HTTPStatus: http.StatusOK,
		SignalInfo: SignalInfo{
			Signal:             make(chan bool),
			SignalCount:        1,
			CurrentSignalCount: 0}}
	return context
}

func NMAgentResponder(w http.ResponseWriter, r *http.Request, context interface{}) {
	fmt.Println("Received URL:", r.RequestURI)
	fmt.Println("Method:", r.Method)
	nmagentResponderContext := context.(*NMAgentResponderContext)
	fmt.Printf("Received URL:%s, Method:%s\n", r.RequestURI, r.Method)
	if strings.Contains(r.RequestURI, "joinedVirtualNetworks") {
		if r.Method == "GET" {
			if nmagentResponderContext.GetNetworkResponse != nil {
				w.WriteHeader(nmagentResponderContext.GetNetworkResponse.HTTPStatus)
				if nmagentResponderContext.GetNetworkResponse.NetworkResponse != nil {
					json.NewEncoder(w).Encode(nmagentResponderContext.GetNetworkResponse.NetworkResponse)
				}
				signalTest(&nmagentResponderContext.GetNetworkResponse.SignalInfo)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
		} else if r.Method == "POST" {
			if nmagentResponderContext.JoinResponse != nil {
				w.WriteHeader(nmagentResponderContext.JoinResponse.HTTPStatus)
				signalTest(&nmagentResponderContext.JoinResponse.SignalInfo)
				fmt.Println("Responding to joined network")
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}

		} else {
			w.WriteHeader(http.StatusOK)
		}

	} else if strings.Contains(r.RequestURI, "networkContainers") {
		w.WriteHeader(http.StatusOK)
		if r.Method == "POST" {
			if strings.Contains(r.RequestURI, "DELETE") {
				deleteResponse := nmagentcontracts.DeleteNetworkContainerResponse{
					ResponseCode: "200",
					ErrorMessage: "Successfully unpublished"}
				json.NewEncoder(w).Encode(deleteResponse)
			} else {
				var ncReq nmagentcontracts.CreateOrUpdateNetworkContainerRequest
				json.NewDecoder(r.Body).Decode(&ncReq)
				fmt.Printf("NC Publishing request:%v\n", ncReq)
			}
		}
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func CNSResponder(w http.ResponseWriter, r *http.Request, context interface{}) {
	fmt.Println("Received URL:", r.RequestURI)
	fmt.Println("Method:", r.Method)
	w.WriteHeader(http.StatusOK)
	fmt.Printf("Request:%v \n", r)
	if strings.Contains(r.RequestURI, "createorupdatenetworkcontainer") &&
		r.Method == "POST" {
		var ncRequest cns.CreateNetworkContainerRequest
		body, _ := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
		json.Unmarshal(body, &ncRequest)
		fmt.Printf("==========CreateNCRequest:%v\n", ncRequest)
		cnsResponse := cns.CreateNetworkContainerResponse{
			Response: cns.Response{
				ReturnCode: 0,
				Message:    "Successfully created network container"}}
		json.NewEncoder(w).Encode(cnsResponse)
	}
	if strings.Contains(r.RequestURI, "deletenetworkcontainer") &&
		r.Method == "POST" {
		var ncRequest cns.DeleteNetworkContainerRequest
		json.NewDecoder(r.Body).Decode(&ncRequest)
		fmt.Printf("NC Request:%v \n", ncRequest)
		cnsResponse := cns.DeleteNetworkContainerResponse{
			Response: cns.Response{
				ReturnCode: 0,
				Message:    "Successfully deleted network container"}}
		json.NewEncoder(w).Encode(cnsResponse)
	}
}

func GetNetworkStatus(t *testing.T, resourceStatusURL string, expectedStatusCode int) string {
	fmt.Printf("Resource status URL:%s\n", resourceStatusURL)
	response, err := http.Get(resourceStatusURL)
	if err != nil {
		t.Errorf("Failed to get network status for resource URL:%s. Error:%v",
			resourceStatusURL,
			err)
		t.Fatal(err)
	}

	if response.StatusCode != expectedStatusCode && response.StatusCode != http.StatusProcessing {
		t.Errorf("Unexpected response code %d received when trying to get network status for resource URL:%s.",
			response.StatusCode,
			resourceStatusURL)
		t.Fatal(err)
	}
	fmt.Printf("Status:%v", response)
	if response.StatusCode == http.StatusProcessing {
		return "Pending"
	}
	var operationStatus api.RequestStatus
	err = decodeResponse(response, &operationStatus)
	if err != nil &&
		(response.StatusCode != expectedStatusCode) {
		t.Errorf("Failed to decode network response for resource URL:%s. Error:%v",
			resourceStatusURL,
			err)
		t.Fatal(err)
	}
	fmt.Printf("Resource URL:%s Status:%s\n", resourceStatusURL, operationStatus.String())

	return operationStatus.Status
}

func IsJoinInState(t *testing.T, resourceStatusURL string, state string, expectedStatusCode int) bool {
	status := GetNetworkStatus(t, resourceStatusURL, expectedStatusCode)
	return status == state
}

func WaitForJoinState(t *testing.T, resourceStatusURL string, status string, expectedStatusCode int, timeToWait int) bool {
	for {
		select {
		case <-time.After(1 * time.Microsecond):
			if IsJoinInState(t, resourceStatusURL, status, expectedStatusCode) {
				return true
			}

		case <-time.After(time.Duration(timeToWait) * time.Second):
			return false
		}
	}
}
func CreateNetworkAuthFailure(t *testing.T, networkreq CreateNetworkRequest, dest, vnetid, apiversion string) {
	var body bytes.Buffer
	var respErr api.JsonErr
	json.NewEncoder(&body).Encode(networkreq)
	fmt.Println("Started NMAgent Server", nmAgentServer.Server.URL)
	nmAgentServer.SetupTest()
	nmAgentResponderContext := generateDefaultNMAgentResponderContext()
	nmAgentResponderContext.JoinResponse.HTTPStatus = http.StatusUnauthorized
	nmAgentServer.Lock.Lock()
	nmAgentServer.ResponseFunc = NMAgentResponder
	nmAgentServer.Context = nmAgentResponderContext
	nmAgentServer.Lock.Unlock()
	resourceStatusURL := dest + "/networks/" + vnetid + "/status" + apiversion
	networkURL := dest + "/networks/" + vnetid + apiversion
	response, err := http.Post(networkURL, "application/json", &body)
	if err != nil {
		t.Errorf("CreateNetwork failed with error: %+v\n", err)
		t.Fatal(err)
	}

	if response.StatusCode != http.StatusOK {
		err = decodeResponse(response, &respErr)
		if err != nil || respErr.Code != 0 {
			t.Errorf("CreateNetwork failed with response %+v Err:%+v", response, err)
			t.Fatal(err)
		}
	}

	fmt.Println("Waiting for Join signal")
	<-nmAgentResponderContext.JoinResponse.Signal
	fmt.Printf("Received Join Network request signal")

	WaitForJoinState(t, resourceStatusURL, "Failed", http.StatusUnauthorized, 30)
	fmt.Printf("Setting the get virtual network response error code to completed from failed state\n")
	deleteResource(t, networkURL)
}

func setupNetwork(t *testing.T, dncEP, networkID, apiVersion string) {
	var body bytes.Buffer
	var respErr api.JsonErr
	networkReq := api.NetworkInfoRequest{
		NetworkType: api.AzureNet}

	json.NewEncoder(&body).Encode(networkReq)
	fmt.Println("Started NMAgent Server", nmAgentServer.Server.URL)
	nmAgentServer.SetupTest()
	nmAgentResponderContext := generateDefaultNMAgentResponderContext()
	nmAgentResponderContext.GetNetworkResponse.NetworkResponse.ResponseCode = strconv.Itoa(http.StatusOK)
	nmAgentServer.Lock.Lock()
	nmAgentServer.ResponseFunc = NMAgentResponder
	nmAgentServer.Context = nmAgentResponderContext
	nmAgentServer.Lock.Unlock()

	response, err := http.Post(dncEP+"/networks/"+networkID+apiVersion, "application/json", &body)
	if err != nil {
		t.Errorf("CreateNetwork failed with error: %+v\n", err)
		t.Fatal(err)
	}

	if response.StatusCode != http.StatusOK {
		err = decodeResponse(response, &respErr)
		if err != nil || respErr.Code != 0 {
			t.Errorf("CreateNetwork failed with response %+v Err:%+v", response, err)
			t.Fatal(err)
		}
	}

	fmt.Println("Waiting for Join signal")
	<-nmAgentResponderContext.JoinResponse.Signal
	fmt.Printf("Received Join Network request signal")
	<-nmAgentResponderContext.GetNetworkResponse.SignalInfo.Signal
	fmt.Printf("Received Get Network request signal for context:%v\n", nmAgentResponderContext.GetNetworkResponse)
	resourceStatusURL := dncEP + "/networks/" + networkID + "/status" + apiVersion
	WaitForJoinState(t, resourceStatusURL, "Completed", http.StatusOK, 30)
	fmt.Printf("Successfully created network [%s]\n", networkID)
}

func CreateNetwork(t *testing.T, networkreq CreateNetworkRequest, dest, vnetid, apiversion string) {
	var body bytes.Buffer
	var respErr api.JsonErr
	json.NewEncoder(&body).Encode(networkreq)
	fmt.Println("Started NMAgent Server", nmAgentServer.Server.URL)
	nmAgentServer.SetupTest()
	nmAgentResponderContext := generateDefaultNMAgentResponderContext()
	nmAgentResponderContext.GetNetworkResponse.NetworkResponse.ResponseCode = strconv.Itoa(http.StatusProcessing)
	nmAgentServer.Lock.Lock()
	nmAgentServer.ResponseFunc = NMAgentResponder
	nmAgentServer.Context = nmAgentResponderContext
	nmAgentServer.Lock.Unlock()

	response, err := http.Post(dest+"/networks/"+vnetid+apiversion, "application/json", &body)
	if err != nil {
		t.Errorf("CreateNetwork failed with error: %+v\n", err)
		t.Fatal(err)
	}

	if response.StatusCode != http.StatusOK {
		err = decodeResponse(response, &respErr)
		if err != nil || respErr.Code != 0 {
			t.Errorf("CreateNetwork failed with response %+v Err:%+v", response, err)
			t.Fatal(err)
		}
	}

	fmt.Println("Waiting for Join signal")
	<-nmAgentResponderContext.JoinResponse.Signal
	fmt.Printf("Received Join Network request signal")
	<-nmAgentResponderContext.GetNetworkResponse.SignalInfo.Signal
	fmt.Printf("Received Get Network request signal for context:%v\n", nmAgentResponderContext.GetNetworkResponse)
	resourceStatusURL := dest + "/networks/" + vnetid + "/status" + apiversion
	WaitForJoinState(t, resourceStatusURL, "Pending", http.StatusProcessing, 30)
	fmt.Printf("Setting the get virtual network response error code to completed from pending state\n")
	nmAgentServer.Lock.Lock()
	nmAgentResponderContext.GetNetworkResponse.NetworkResponse.ResponseCode = strconv.Itoa(http.StatusOK)
	nmAgentResponderContext.GetNetworkResponse.SignalInfo.CurrentSignalCount = 0
	nmAgentServer.Lock.Unlock()
	<-nmAgentResponderContext.GetNetworkResponse.SignalInfo.Signal
	fmt.Printf("Received Get Network request signal for context:%v\n", nmAgentResponderContext.GetNetworkResponse)
	WaitForJoinState(t, resourceStatusURL, "Completed", http.StatusOK, 30)
	fmt.Printf("Successfully created network [%s]\n", vnetid)
	networkURL := dest + "/networks/" + vnetid + apiversion
	deleteResource(t, networkURL)
}

func JoinSubnetWithNoNetwork(t *testing.T, dest, vnetid, subnetname, apiversion string) {
	var body bytes.Buffer
	var responseErr api.JsonErr
	subnetReq := api.SubnetRequest{
		AuthenticationToken: "Key1"}
	json.NewEncoder(&body).Encode(subnetReq)

	response, err := http.Post(dest+"/networks/"+vnetid+"/subnets/"+subnetname+apiversion, "application/json", &body)
	if err != nil {
		t.Errorf("JoinSubnet failed with error: %+v\n", err)
		t.Fatal(err)
	}
	if response.StatusCode == http.StatusOK {
		t.Errorf("JoinSubnet expected to fail")
		t.Fatal(err)
	}

	err = decodeResponse(response, &responseErr)
	fmt.Printf("JoinSubnet failed with response %+v Err:%v", responseErr, err)
	if err != nil && !strings.Contains(err.Error(), "Request failed with HTTP error 404") {
		t.Errorf("JoinSubnet decode failed with response %v Err:%v", responseErr, err)
		t.Fatal(err)
	}

	if responseErr.Code != api.NetworkNotFound {
		t.Errorf("JoinSubnet expected to fail with error code %s, Received response %s", responseErr.Code.String(), api.NetworkNotFound.String())
		t.Fatal(err)
	}
}

func JoinSubnet(t *testing.T, nodeID string, nodeInfo api.NodeInfoRequest, networkReq CreateNetworkRequest, dest, vnetid, subnetname, apiversion string) {
	var body bytes.Buffer
	resourceURL := dest + "/networks/" + vnetid + "/subnets/" + subnetname + apiversion
	addOrUpdateNode(t, dest, nodeID, apiversion, &nodeInfo)
	setupNetwork(t, dest, vnetid, apiversion)
	subnetReq := api.SubnetRequest{
		AuthenticationToken: "Key1"}
	json.NewEncoder(&body).Encode(subnetReq)
	fmt.Printf("Joining subnet %s, network %s. AccessToken:%s", vnetid, subnetname, subnetReq.AuthenticationToken)
	response, err := http.Post(resourceURL, "application/json", &body)
	if err != nil {
		t.Errorf("JoinSubnet failed with error: %+v\n", err)
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Errorf("JoinSubnet failed with status code %d\n", response.StatusCode)
		t.Fatal(err)
	}

	subnetConfiguration := getSubnetConfiguration(t, dest, vnetid, subnetname, apiversion)
	if subnetConfiguration.AccessToken != subnetReq.AuthenticationToken {
		t.Errorf("AccessToken mismatch. Received:%s , Expected:%s\n",
			subnetConfiguration.AccessToken,
			subnetReq.AuthenticationToken)
		t.Fatal(err)
	}

	subnetReq = api.SubnetRequest{
		AuthenticationToken: "Key2"}
	json.NewEncoder(&body).Encode(subnetReq)
	resourceStatusURL := dest + "/networks/" + vnetid + "/subnets/" + subnetname + "/status" + apiversion

	WaitForJoinState(t, resourceStatusURL, "Completed", http.StatusOK, 30)

	fmt.Printf("Updating subnet %s, network %s with accessToken:%s", vnetid, subnetname, subnetReq.AuthenticationToken)
	response, err = http.Post(resourceURL, "application/json", &body)
	if err != nil {
		t.Errorf("JoinSubnet update failed with error: %+v\n", err)
		t.Fatal(err)
	}
	fmt.Printf("Deleting network %s\n", vnetid)
	networkURL := dest + "/networks/" + vnetid + apiversion
	deleteResource(t, networkURL)
}

func setupNetworkAndJoinSubnet(t *testing.T, dncEP, networkID, subnetName, apiVersion string) {
	var body bytes.Buffer
	resourceURL := dncEP + "/networks/" + networkID + "/subnets/" + subnetName + apiVersion
	setupNetwork(t, dncEP, networkID, apiVersion)
	subnetReq := api.SubnetRequest{
		AuthenticationToken: "Key1"}
	json.NewEncoder(&body).Encode(subnetReq)
	fmt.Printf("Joining subnet %s, network %s. AccessToken:%s", networkID, subnetName, subnetReq.AuthenticationToken)
	response, err := http.Post(resourceURL, "application/json", &body)
	if err != nil {
		t.Errorf("JoinSubnet failed with error: %+v\n", err)
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Errorf("JoinSubnet failed with status code %d\n", response.StatusCode)
		t.Fatal(err)
	}
}

func CreateReservation(t *testing.T, reservationreq _AllocationRequest, dest, vnetid, reservationID, apiversion string) {
	var body bytes.Buffer
	var respErr api.JsonErr
	setupNetworkAndJoinSubnet(t, dest, vnetid, reservationreq.SubnetName, apiversion)
	json.NewEncoder(&body).Encode(reservationreq)
	reservationURL := dest + "/networks/" + vnetid + "/reservations/" + reservationID + apiversion
	response, err := http.Post(reservationURL, "application/json", &body)

	if err != nil {
		t.Errorf("CreateReservation failed with error: %+v\n", err)
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		if err = decodeResponse(response, &respErr); err != nil || respErr.Code != 0 {
			t.Errorf("CreateReservation failed with response %+v Err:%+v", respErr, err)
			t.Fatal(err)
		}
	}
	reservationStatusURL := dest + "/networks/" + vnetid + "/reservations/" + reservationID + "/status" + apiversion
	WaitForJoinState(t, reservationStatusURL, "Completed", http.StatusOK, 30)

	fmt.Printf("Successfully created reservation [%s]\n", reservationID)
}

func CreateReservationNodeConstraint(t *testing.T, reservationreq CreateReservationWithNodeConstraintRequest, dest, vnetid, reservationID, apiversion string) {
	var body bytes.Buffer
	var respErr api.JsonErr
	json.NewEncoder(&body).Encode(reservationreq)
	response, err := http.Post(dest+"/networks/"+vnetid+"/reservations/"+reservationID+apiversion, "application/json", &body)

	if response.StatusCode != http.StatusOK {
		if err != nil {
			t.Errorf("CreateReservationNodeConstraint failed with error: %+v\n", err)
			t.Fatal(err)
		}

		if err = decodeResponse(response, &respErr); err != nil || respErr.Code != 0 {
			t.Errorf("CreateReservationNodeConstraint failed with response %+v Err:%+v", respErr, err)
			t.Fatal(err)
		}
	}
	// Sleep for 10s for processing to complete
	time.Sleep(10000 * time.Millisecond)
	fmt.Printf("Successfully created reservation [%s] with node constraint [%s]\n", reservationID, reservationreq.NodeConstraint)
}

func CreateNetworkContainer(t *testing.T, networkcontainerreq CreateNetworkContainerRequest, dest, vnetid, ncid, apiversion string) {
	var body bytes.Buffer
	var respErr api.JsonErr
	json.NewEncoder(&body).Encode(networkcontainerreq)
	cnsServer.SetupTest()
	cnsServer.ResponseFunc = CNSResponder

	response, err := http.Post(dest+"/networks/"+vnetid+"/networkcontainer/"+ncid+apiversion, "application/json", &body)

	if err != nil {
		t.Errorf("CreateNetworkContainer failed with error: %+v\n", err)
		t.Fatal(err)
	}

	if response.StatusCode != http.StatusOK {
		if err = decodeResponse(response, &respErr); err != nil || respErr.Code != 0 {
			t.Errorf("CreateNetworkContainer failed with response %+v Err:%+v", respErr, err)
			t.Fatal(err)
		}
	}

	// Sleep for 30s for processing to complete
	time.Sleep(30000 * time.Millisecond)

	if networkcontainerreq.AllocationRequest.IPConstraint == "" {
		fmt.Printf("Successfully created network container using dynamic IP [%s]\n", ncid)
	} else {
		fmt.Printf("Successfully created network container using specific IP [%s]\n", ncid)
	}
}

func CreateNetworkContainerUsingReservation(t *testing.T, networkcontainerreq CreateNetworkContainerUsingReservationRequest, dest, vnetid, ncid, apiversion string) {
	var body bytes.Buffer
	var respErr api.JsonErr
	// this should fail since reservation was made with testnode instead of testnode2
	json.NewEncoder(&body).Encode(networkcontainerreq)
	response, err := http.Post(dest+"/networks/"+vnetid+"/networkcontainer/"+ncid+apiversion, "application/json", &body)

	if err != nil {
		t.Errorf("CreateNetworkContainerUsingReservation failed with error: %+v\n", err)
		t.Fatal(err)
	}

	if response.StatusCode != http.StatusOK {

		if err = decodeResponse(response, &respErr); err != nil || respErr.Code != 0 {
			t.Errorf("CreateNetworkContainerUsingReservation failed with response %+v Err:%+v", respErr, err)
			t.Fatal(err)
		}
	}
	// Sleep for 30s for processing to complete
	time.Sleep(30000 * time.Millisecond)
	fmt.Printf("Successfully created network container using reservation [%s]\n", ncid)
}

func DeleteNetworkContainer(t *testing.T, dest, vnetid, nc, apiversion string) {
	fmt.Printf("DeleteNetworkContainer: Deleting nc with id [%s]...\n", nc)
	fmt.Println("IPCONFIG now for results before deletion...")
	// Sleep to check whether container was created
	time.Sleep(10000 * time.Millisecond)
	request, err := http.NewRequest(http.MethodDelete, dest+"/networks/"+vnetid+"/networkcontainer/"+nc+apiversion, nil)

	if err != nil {
		t.Fatal(err)
	}

	response, resperr := client.Do(request)

	if resperr != nil {
		t.Errorf("DeleteNetworkContainer failed with Err:%+v", resperr)
		t.Fatal(resperr)
	}

	if response != nil && response.StatusCode != 200 {
		t.Errorf("DeleteNetworkContainer failed with response %+v", response)
	}

	fmt.Printf("Deleted network container with response %+v\n", response)
}
