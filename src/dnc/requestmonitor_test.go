// +build !bmMode

package main

import (
	"bytes"
	"crypto/sha256"
	"dnc/api"
	cli "dnc/client"
	"dnc/configuration"
	"dnc/datamodel"
	"dnc/nmagentcontracts"
	"dnc/resourceupdatehandler/networkupdatehandler"
	"dnc/utils"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Azure/azure-container-networking/cns"
	"github.com/google/uuid"
)

type CreateNetworkContainerRequest struct {
	AllocationRequest          api.AllocationRequest
	AssociationInfo            api.AssociationInfo
	AllowHostToNCCommunication bool
	AllowNCToHostCommunication bool
	EndpointPolicies           []cns.NetworkContainerRequestPolicies
}

type CreateNetworkContainerRequestV2 struct {
	ReservationID              string
	AllocationRequest          api.AllocationRequest
	AssociationInfo            api.AssociationInfo
	AllowHostToNCCommunication bool
	AllowNCToHostCommunication bool
	EndpointPolicies           []cns.NetworkContainerRequestPolicies
}

type OrchestratorContext struct {
	PodName      string
	PodNamespace string
}

type AclTestPolicySetting struct {
	Protocols       string
	Action          string
	Direction       string
	LocalAddresses  string
	RemoteAddresses string
	LocalPorts      string
	RemotePorts     string
	RuleType        string
	Priority        uint16
}

// Unit Test Values
var (
	sfNodeReq = api.NodeInfoRequest{
		IPAddresses:      []string{"127.0.0.1"},
		LocationType:     "",
		OrchestratorType: cns.ServiceFabric}

	podInfo = OrchestratorContext{
		PodName:      "testpodname",
		PodNamespace: "testpodnamespace"}
	orchContext, _        = json.Marshal(podInfo)
	validEndPointPolicies = composeNetworkContainerEndpointPolicies("6", "Allow", "In", "127.0.0.1", "10.0.0.1", "1111",
		"1112", "Switch", 2000, "AclPolicy", "APIPA")
	networkContainerReq = CreateNetworkContainerRequest{
		AllocationRequest: api.AllocationRequest{
			SubnetName:     "subnet1",
			IPConstraint:   "10.0.0.15",
			NodeConstraint: ""},
		AssociationInfo: api.AssociationInfo{
			NodeID:              "testnode",
			InterfaceID:         "127.0.0.1",
			ContainerType:       cns.AzureContainerInstance,
			OrchestratorContext: orchContext}}
	nodeID  = "testnode"
	nodeReq = api.NodeInfoRequest{
		IPAddresses:      []string{"127.0.0.1"},
		LocationType:     "",
		OrchestratorType: cns.Kubernetes,
	}
	networkContainerReq6 = CreateNetworkContainerRequest{
		AllocationRequest: api.AllocationRequest{
			SubnetName:     "subnet1",
			IPConstraint:   "",
			NodeConstraint: ""},
		AssociationInfo: api.AssociationInfo{
			NodeID:              "testnode",
			InterfaceID:         "127.0.0.1",
			ContainerType:       cns.COW,
			OrchestratorContext: orchContext},
		AllowHostToNCCommunication: true,
		AllowNCToHostCommunication: true,
		EndpointPolicies:           validEndPointPolicies}
	ncID  = "0e2f40cd-8fe1-420f-af43-1c4d5c5bb83b"
	ncID2 = "0338b32f-914e-49a1-b58d-700430d63499"

	localIPConfiguaration = cns.IPConfiguration{
		IPSubnet: cns.IPSubnet{
			IPAddress:    "169.254.128.4",
			PrefixLength: utils.LocalIPSubnetPrefixLength},
		GatewayIPAddress: "169.254.128.1"}

	// client :- Client with custom transport
	tr = &http.Transport{
		MaxIdleConnsPerHost: 1024,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 0 * time.Second}
	client        = &http.Client{Transport: tr}
	nmAgentServer *MockServer
	cnsServer     *MockServer
)

const (
	defaultNumCPUCores            = 16  // Default CPU cores for general tests
	updatedNumCPUCoresForLoadTest = 300 // Higher CPU cores for simulating load test
	induceErrorPercent            = 5   // Induce 5% error for nmagent and cns responses
)

// Setup :- Set up the environment (if necessary).
func Setup() error {
	go StartDnc(false)
	time.Sleep(1 * time.Second)
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 1024
	http.DefaultTransport.(*http.Transport).TLSHandshakeTimeout = 0 * time.Second

	nmAgentServer = new(MockServer)
	if err := nmAgentServer.SetupServer("127.0.0.1:9001"); err != nil {
		return err
	}
	cnsServer = new(MockServer)
	if err := cnsServer.SetupServer("127.0.0.1:10090"); err != nil {
		return err
	}

	httpClient = http.DefaultClient
	cnsHttpClient = http.DefaultClient
	return nil
}

func TestVlanGeneration(t *testing.T) {
	fmt.Println("****TestVlanGeneration**********")
	vlanMap := make(map[string]uint16)
	vlanRefMap := make(map[string]uint16)
	vnetID := []string{"vnet1", "vnet2", "vnet3"}

	fmt.Println("Initializing vlanMap using GetNextVlanID")

	for i, id := range vnetID {
		var err *api.JsonErr
		vlanRefMap[id] = uint16(i + 1)
		vlanMap[id], err = datamodel.GetNextVlanID(vlanMap)
		fmt.Printf("vlanMap -> %+v\n", vlanMap)

		if err != nil {
			t.Fatal(err)
		}
	}

	if !reflect.DeepEqual(vlanMap, vlanRefMap) {
		t.Fatalf("vlanMap is not equal to vlanRefMap")
	}

	fmt.Println("Removing vlanID vnet2")

	temp := vlanMap[vnetID[1]]
	delete(vlanMap, vnetID[1])
	delete(vlanRefMap, vnetID[1])

	if !reflect.DeepEqual(vlanMap, vlanRefMap) {
		t.Fatalf("vlanMap is not equal to vlanRefMap")
	}

	fmt.Printf("vlanMap -> %+v\n", vlanMap)
	fmt.Printf("Confirming that next vlanID will be between vnet1 (%d) and vnet3 (%d)\n", vlanMap[vnetID[0]], vlanMap[vnetID[2]])

	var err *api.JsonErr
	vlanRefMap[vnetID[1]] = temp
	vlanMap[vnetID[1]], err = datamodel.GetNextVlanID(vlanMap)

	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(vlanMap, vlanRefMap) {
		t.Fatalf("vlanMap is not equal to vlanRefMap")
	}

	fmt.Printf("vlanMap -> %+v\n", vlanMap)
	fmt.Println("****TestVlanGeneration Succeed**********")
}

func TestGreGeneration(t *testing.T) {
	fmt.Println("****TestGreGeneration**********")
	greMap := make(map[string]uint16)
	greRefMap := make(map[string]uint16)

	fmt.Println("Initializing greMap using GetNextGreKey")

	for count := datamodel.MinGre; count <= datamodel.MaxGre; count++ {
		var err *api.JsonErr
		id, _ := uuid.NewUUID()
		greRefMap[id.String()] = count
		greMap[id.String()], err = datamodel.GetNextGreKey(greMap)

		if err != nil {
			t.Fatal(err)
		}
	}
	fmt.Printf("gre -> %+v\n", greMap)

	if !reflect.DeepEqual(greMap, greRefMap) {
		t.Fatalf("greMap is not equal to greRefMap")
	}
	fmt.Println("****TestGreGeneration Succeed**********")
}

func TestOutOfGreKeys(t *testing.T) {
	fmt.Println("****TestOutOfGreKeys**********")
	greMap := make(map[string]uint16)

	fmt.Println("Testing outOfGreKeys using GetNextGreKey")

	for count := datamodel.MinGre; count <= (datamodel.MaxGre + 1); count++ {
		var err *api.JsonErr
		id, _ := uuid.NewUUID()
		greMap[id.String()], err = datamodel.GetNextGreKey(greMap)

		if err == nil && count > datamodel.MaxGre {
			fmt.Printf("gre -> %+v\n", greMap)
			t.Fatal(fmt.Sprintf("greKeys exceeded %d", datamodel.MaxGre))
		}
	}
	fmt.Println("****TestOutOfGreKeys Succeed**********")
}

func TestSkipNodeCleanupForNetworkContainer(t *testing.T) {
	fmt.Println("****TestSkipNodeCleanupForNetworkContainer**********")
	id, _ := uuid.NewUUID()
	vnetID := id.String()

	cnsServer.SetupTest()
	ncDeploymentRequests := make([]*NCDeploymentRequest, 0)
	cnsResponderContextMap := generateDefaultCnsResponderContextMap(t, ncDeploymentRequests)
	cnsServer.Lock.Lock()
	cnsServer.ResponseFunc = CNSResponder
	cnsServer.Context = cnsResponderContextMap
	cnsServer.Lock.Unlock()

	setupNetworkAndJoinSubnet(t, dncEP, vnetID, networkContainerReq.AllocationRequest.SubnetName, apiVersion, true, 0)
	addOrUpdateNode(t, dncEP, nodeID, apiVersion, &nodeReq)

	nmAgentResponderContext := nmAgentServer.Context.(*NMAgentResponderContext)

	allocatedIP := "10.0.0.4"
	if len(networkContainerReq.AllocationRequest.IPConstraint) > 0 {
		allocatedIP = networkContainerReq.AllocationRequest.IPConstraint
	}
	var ipConfig cns.IPConfiguration
	ipConfig.DNSServers = strings.Split(nmAgentResponderContext.NetworkResponseMap.getContext(vnetID).GetNetworkResponse.NetworkResponse.DNSServers, ",")
	ipConfig.IPSubnet = cns.IPSubnet{
		IPAddress:    allocatedIP,
		PrefixLength: 24}
	ipConfig.GatewayIPAddress = "10.0.0.1"
	ncDeploymentRequests2 := []*NCDeploymentRequest{{
		NCID: ncID}}
	cnsResponderContextMap = generateDefaultCnsResponderContextMap(t, ncDeploymentRequests2)
	ncRequestValidatorFunc := func(t *testing.T, req cns.CreateNetworkContainerRequest, ncDeploymentRequests *NCDeploymentRequest) bool {
		podInfo := OrchestratorContext{
			PodName:      "testpodname",
			PodNamespace: "testpodnamespace"}
		context, _ := json.Marshal(podInfo)
		expectedReq := cns.CreateNetworkContainerRequest{
			Version:                    req.Version,
			NetworkContainerType:       networkContainerReq.AssociationInfo.ContainerType,
			NetworkContainerid:         cns.SwiftPrefix + ncID,
			PrimaryInterfaceIdentifier: networkContainerReq.AssociationInfo.InterfaceID,
			IPConfiguration:            ipConfig,
			AuthorizationToken:         "",
			LocalIPConfiguration:       localIPConfiguaration,
			CnetAddressSpace:           make([]cns.IPSubnet, 0),
			OrchestratorContext:        context,
			MultiTenancyInfo:           req.MultiTenancyInfo}
		fmt.Printf("ReceivedResponse:%+v\n", req)
		fmt.Printf("ExpectedResponse:%+v\n", expectedReq)

		if !AreCnsRequestsEqual(expectedReq, req, nil) {
			fmt.Printf("Mismatch in expected and received Cns create or update request.\nExpected:%+v\nReceived:%+v\n", expectedReq, req)
			return false
		}

		fmt.Println("Response matches with the expected request")
		return true
	}

	cnsResponderContextMap.getContext(ncID).CreateOrUpdateNCResponse.ValidateNCRequest = ncRequestValidatorFunc
	cnsServer.Lock.Lock()
	cnsServer.Context = cnsResponderContextMap
	cnsServer.Lock.Unlock()

	var body bytes.Buffer
	var respErr api.JsonErr
	ncURL := dncEP + "/networks/" + vnetID + "/networkcontainer/" + ncID + apiVersion
	ncURLStatus := dncEP + "/networks/" + vnetID + "/networkcontainer/" + ncID + "/status" + apiVersion
	json.NewEncoder(&body).Encode(networkContainerReq)
	response, err := http.Post(ncURL, "application/json", &body)
	if err != nil {
		t.Fatalf("CreateNetworkContainer failed with error: %+v\n", err)
	}

	response.Body.Close()

	if response.StatusCode != http.StatusOK {
		if err = utils.DecodeResponse(response, &respErr); err != nil || respErr.Code != 0 {
			t.Fatalf("CreateNetworkContainer failed with response %+v Err:%+v", respErr, err)
		}
	}

	fmt.Printf("Waiting for create NC Url:%s\n", ncURL)
	<-cnsResponderContextMap.getContext(ncID).CreateOrUpdateNCResponse.SignalInfo.Signal
	WaitForResourceState(t, ncURLStatus, "Completed", http.StatusOK, 2)

	networkURL := dncEP + "/networks/" + vnetID + apiVersion
	nodeURL := dncEP + "/nodes/" + nodeID + apiVersion
	getNCCleanupStateURL := dncEP + "/iswaitingfornodecleanup/networks/" + vnetID + "/networkcontainer/" + ncID + apiVersion
	promotionURL := dncEP + "/skipnodecleanupfornetworkcontainer/networks/" + vnetID + "/networkcontainer/" + ncID + apiVersion

	// stop CNS here and promote NC state after termination fails
	cnsServer.TearDownSuite()
	deleteResource(t, ncURL, http.StatusOK)
	// Wait of 10 seconds is needed because unpublish via CNS will timeout after 5 seconds and then
	// will unpublish via DNC node. 5 more seconds for disassociate to time out
	if WaitForResourceCleanup(t, ncURL, 10) {
		t.Fatalf("We should not be able to clean up NC yet")
	}

	response, err = http.Get(getNCCleanupStateURL)
	if err != nil {
		t.Fatalf("GetNCCleanupState failed with error: %+v", err)
	}

	var cleanupStateResponse api.IsWaitingForNodeCleanupResponse
	err = utils.DecodeResponse(response, &cleanupStateResponse)
	response.Body.Close()
	if err != nil {
		t.Fatalf("Decoding FailedToDisassociateNCResponse failed. Response %+v Err:%+v", cleanupStateResponse, err)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf("GetNCCleanupState failed with error: %+v {%+v}", err, response)
	}

	if cleanupStateResponse.IsWaitingForNodeCleanup != true {
		t.Fatalf("TestSkipNodeCleanupForNetworkContainer failed with unexpected error.")
	}

	response, err = http.Post(promotionURL, "application/json", &body)
	if err != nil {
		t.Fatalf("SkipNodeCleanupForNetworkContainer failed with error: %+v", err)
	}

	response.Body.Close()
	// Set failure to previous error and check if it gets reset after promotion
	var req api.NetworkContainerResponse
	req.RequestStatus.FailureDetail.Code = api.DisAssociationFailed
	for ; req.RequestStatus.FailureDetail.Code != 0; time.Sleep(1 * time.Second) {
		response, err = http.Get(ncURL)
		if err != nil {
			t.Fatalf("Failed to get NC status.")
		}

		err = utils.DecodeResponse(response, &req)
		response.Body.Close()
		if err != nil {
			t.Fatalf("Decoding RequestStatus failed. Req %+v Err:%+v", req, err)
		}
	}

	deleteResource(t, ncURL, http.StatusOK)
	if !WaitForResourceCleanup(t, ncURL, 4) {
		t.Fatalf("Failed to cleanup NC %s", ncID)
	}

	deleteResource(t, networkURL, http.StatusOK)
	deleteResource(t, nodeURL, http.StatusOK)
	// restart our CNS server for the following tests
	cnsServer.SetupServer("127.0.0.1:10090")
	fmt.Println("****TestSkipNodeCleanupForNetworkContainer Succeed**********")
}

func TestContainer(t *testing.T) {
	fmt.Println("****TestContainer**********")
	cnsServer.SetupTest()
	ncDeploymentRequests := make([]*NCDeploymentRequest, 0)
	cnsResponderContextMap := generateDefaultCnsResponderContextMap(t, ncDeploymentRequests)
	cnsServer.Lock.Lock()
	cnsServer.ResponseFunc = CNSResponder
	cnsServer.Context = cnsResponderContextMap
	cnsServer.Lock.Unlock()
	id, _ := uuid.NewUUID()
	vnetID := id.String()

	CreateNetworkContainer(t, nodeID, sfNodeReq, networkContainerReq6, dncEP, vnetID, ncID, apiVersion, false)
	CreateNetworkContainerAndUpdateCnetSpace(t, nodeID, nodeReq, networkContainerReq, dncEP, vnetID, ncID, apiVersion)
	CreateNetworkContainerAndAddCnetSpace(t, nodeID, nodeReq, networkContainerReq, dncEP, vnetID, ncID, apiVersion)
	CreateNetworkContainerAndRemoveCnetSpace(t, nodeID, nodeReq, networkContainerReq, dncEP, vnetID, ncID, apiVersion)
	CreateNetworkContainerAndAddNewDNSServer(t, nodeID, nodeReq, networkContainerReq, dncEP, vnetID, ncID, apiVersion)
	CreateNetworkContainerAndUpdateToUseAzureDNSServers(t, nodeID, nodeReq, networkContainerReq, dncEP, vnetID, ncID, apiVersion)
	CreateNetworkContainerAndUpdateToUseCustomDNSServers(t, nodeID, nodeReq, networkContainerReq, dncEP, vnetID, ncID, apiVersion)
	fmt.Println("****TestContainer Succeed**********")
}

func TestDeleteContainer(t *testing.T) {
	fmt.Println("****TestDeleteContainer**********")
	cnsServer.SetupTest()
	ncDeploymentRequests := make([]*NCDeploymentRequest, 0)
	cnsResponderContextMap := generateDefaultCnsResponderContextMap(t, ncDeploymentRequests)
	cnsServer.Lock.Lock()
	cnsServer.ResponseFunc = CNSResponder
	cnsServer.Context = cnsResponderContextMap
	cnsServer.Lock.Unlock()
	id, _ := uuid.NewUUID()
	vnetID := id.String()

	// skip delete
	CreateNetworkContainer(t, nodeID, sfNodeReq, networkContainerReq6, dncEP, vnetID, ncID, apiVersion, true)

	// cancel the background process to not process the pending queue
	cancelProcessing <- true
	time.Sleep(time.Millisecond * 2000)

	ncURL := dncEP + "/networks/" + vnetID + "/networkcontainer/" + ncID + apiVersion
	deleteResource(t, ncURL, http.StatusOK)
	t.Logf("NC %s delete will fail as RequestMonitor is not running", ncURL)
	if WaitForResourceCleanup(t, ncURL, 1) {
		t.Fatalf("Failed to cleanup network container %s", ncID)
	}

	t.Log("Now delete the entry in the PendingQueue")
	requestKey := datamodel.GetNCKey(vnetID, ncID)
	datamodel.RemovePendingRequest(requestKey)

	request, err := datamodel.TryQueryRequestContext(requestKey)
	if err != nil || request != nil {
		t.Fatalf("Unexpected NC entry in PendingQ %s %+v, err: %s,", requestKey, request, err)
	}

	// Start the background refresh monitor again.
	go monitorRequests()
	DeleteNetworkContainer(t, nodeID, "", dncEP, vnetID, ncID, apiVersion)
	fmt.Println("****TestDeleteContainer Succeed**********")
}

// Test the race scenario where Node Info gets deleted during NC create (Prepare state)
// and the delete NC still goes through (as it skips if node is null)
func TestDeleteContainerInPurgeState(t *testing.T) {
	cnsServer.SetupTest()
	ncDeploymentRequests := make([]*NCDeploymentRequest, 0)
	cnsResponderContextMap := generateDefaultCnsResponderContextMap(t, ncDeploymentRequests)
	cnsServer.Lock.Lock()
	cnsServer.ResponseFunc = CNSResponder
	cnsServer.Context = cnsResponderContextMap
	cnsServer.Lock.Unlock()
	id, _ := uuid.NewUUID()
	vnetID := id.String()

	setupNetworkAndJoinSubnet(
		t,
		dncEP,
		vnetID,
		networkContainerReq6.AllocationRequest.SubnetName,
		apiVersion,
		true,
		0)
	addOrUpdateNode(t, dncEP, nodeID, apiVersion, &sfNodeReq)

	// cancel the background process to not process the pending queue
	// adding a sleep to wait for all go routines to be cleanedup.
	cancelProcessing <- true
	time.Sleep(time.Millisecond * 2000)

	t.Logf("Disable the background RequestMonitor to skip processing Create NC")

	// Create NC
	var body bytes.Buffer
	var respErr api.JsonErr
	json.NewEncoder(&body).Encode(networkContainerReq6)

	ncURL := dncEP + "/networks/" + vnetID + "/networkcontainer/" + ncID2 + apiVersion
	response, err := http.Post(ncURL, "application/json", &body)
	if err != nil {
		t.Fatalf("CreateNetworkContainer failed with error: %+v\n", err)
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		if err = utils.DecodeResponse(response, &respErr); err != nil || respErr.Code != 0 {
			t.Fatalf("CreateNetworkContainer failed with response %+v Err:%+v", respErr, err)
		}
	}

	ncURLStatus := dncEP + "/networks/" + vnetID + "/networkcontainer/" + ncID2 + "/status" + apiVersion
	WaitForResourceState(t, ncURLStatus, "Prepare", http.StatusOK, 2)

	// Delete Node
	nodeURL := dncEP + "/nodes/" + nodeID + apiVersion
	deleteResource(t, nodeURL, http.StatusOK)

	// Delete the NC and then restart the monitor to process NC cleanup during purge state
	// Start the background refresh monitor again.
	deleteResource(t, ncURL, http.StatusOK)
	t.Logf("NC Url:%s deleted", ncURL)

	go monitorRequests()
	if !WaitForResourceCleanup(t, ncURL, 10) {
		ncStatusResponse, err := http.Get(ncURLStatus)
		if err != nil {
			t.Fatalf("Failed to get network status for resource URL:%s. Error:%+v",
				ncURLStatus,
				err)
		}

		defer ncStatusResponse.Body.Close()

		var operationStatus api.RequestStatus
		err = utils.DecodeResponse(ncStatusResponse, &operationStatus)
		if err != nil {
			t.Fatalf("Failed to decode network response for resource URL:%s. Error:%+v",
				ncURLStatus,
				err)
		}

		t.Fatalf("Failed to cleanup network container %s\n, response %s %s", ncID, operationStatus.State, operationStatus.Status)
	}

	networkURL := dncEP + "/networks/" + vnetID + apiVersion
	deleteResource(t, networkURL, http.StatusOK)
}

func TestCreateMultipleNetworks(t *testing.T) {
	fmt.Println("****TestCreateMultipleNetworks**********")
	deploymentCount := 10   // Number of vnets
	ncPerNetworkCount := 10 // Number of NCs per vnet
	nodeCount := 200
	nodeMap := make([]string, 0)
	for node := 0; node < nodeCount; node++ {
		nodeMap = append(nodeMap, fmt.Sprintf("Node-%d", node))
	}
	nodeIndex := 0
	ipIndex := 0
	deploymentInfos := make([]DeploymentInfo, deploymentCount)
	podNum := 0
	for index := 0; index < deploymentCount; index++ {
		id, _ := uuid.NewUUID()
		networkDeploymentRequest := NetworkDeploymentRequest{
			vnetID:              id.String(),
			UseCustomDnsServers: true,
			TagCount:            0,
			WaitForJoin:         true,
			WaitForGet:          true}
		ncDeploymentRequests := make([]*NCDeploymentRequest, ncPerNetworkCount)
		for ncIndex := 0; ncIndex < ncPerNetworkCount; ncIndex++ {
			ipToReserve := "10.0.0." + strconv.FormatUint(uint64(4+ipIndex), 10)
			ipIndex++
			podInfo := OrchestratorContext{
				PodName:      fmt.Sprintf("%s-%d", nodeMap[nodeIndex%nodeCount], podNum),
				PodNamespace: "testpodnamespace"}
			context, _ := json.Marshal(podInfo)
			podNum++
			ncID, _ := uuid.NewUUID()
			ncDeploymentRequest := NCDeploymentRequest{
				AllocationRequest: api.AllocationRequest{
					SubnetName:   "subnet1",
					IPConstraint: ipToReserve},
				AssociationInfo: api.AssociationInfo{NodeID: nodeMap[nodeIndex%nodeCount], InterfaceID: "127.0.0.1", ContainerType: cns.AzureContainerInstance, OrchestratorContext: context},
				NCID:            ncID.String()}

			ncDeploymentRequests[ncIndex] = &ncDeploymentRequest
			nodeIndex++
		}

		deploymentInfos[index] = DeploymentInfo{NetworkDeploymentRequest: &networkDeploymentRequest, NCDeploymentRequests: ncDeploymentRequests}
	}

	DeployNetworkContainers(t, dncEP, apiVersion, deploymentInfos, true, updatedNumCPUCoresForLoadTest)
	fmt.Println("****TestCreateMultipleNetworks Succeed**********")
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
	fmt.Printf("NodeID: %s, NodeInfo: %s", nodeID, nodeInfoReq.String())
	response, err = http.Post(dncEP+"/nodes/"+nodeID+apiVersion, "application/json", &body)
	if err != nil {
		t.Fatalf("CreateNode failed with error: %+v\n", err)
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err = utils.DecodeResponse(response, &errResponse)
		if err != nil || errResponse.Code != 0 {
			t.Fatalf("CreateNode failed with response %+v Err:%+v", errResponse, err)
		}
	}
}

func deleteResource(
	t *testing.T,
	resourceURL string,
	expectedStatusCode int) *api.JsonErr {
	var errResponse api.JsonErr
	var err error

	fmt.Printf("Resource Url %s to delete \n", resourceURL)

	// Create request
	req, err := http.NewRequest("DELETE", resourceURL, nil)
	if err != nil {
		t.Fatalf("ResourceURL %s, Create delete request failed with error: %+v\n", resourceURL, err)
	}

	// Fetch Request
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("URL:%s, Delete failed with error: %+v\n", resourceURL, err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != expectedStatusCode {
		t.Fatalf("ResourceURL %s, Delete expected to return with status:%d, Received:%d\n", resourceURL, expectedStatusCode, resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		err = utils.DecodeResponse(resp, &errResponse)
		fmt.Printf("Decode resonse error:%+v\n", err != nil)
	}

	return &errResponse
}

func validateNodeInfoResponse(
	t *testing.T,
	dncEP string,
	nodeID string,
	apiVersion string,
	nodeInfoReq *api.NodeInfoRequest) {

	response, err := http.Get(dncEP + "/nodes/" + nodeID + apiVersion)
	var nodeInfoResponse api.NodeInfoResponse
	err = utils.DecodeResponse(response, &nodeInfoResponse)
	if err != nil {
		t.Fatalf("Decoding NodeInfo failed. Response %+v Err:%+v", nodeInfoResponse, err)
	}

	defer response.Body.Close()

	fmt.Printf("Retrieved response:%s\n", nodeInfoResponse.String())
	expectedIPAddresses := make(map[uint32]bool)

	for _, ipAddressString := range nodeInfoReq.IPAddresses {
		expectedIPAddresses[utils.IP2Int(net.ParseIP(ipAddressString))] = true
	}

	if len(expectedIPAddresses) != len(nodeInfoResponse.NodeInfo.IPAddresses) {
		t.Fatalf("Mismatch in count of expceted and received IP Addresses. Expected:%d, Received:%d",
			len(expectedIPAddresses),
			len(nodeInfoResponse.NodeInfo.IPAddresses))
	}
	for _, ipAddressString := range nodeInfoResponse.NodeInfo.IPAddresses {
		receivedIPAddrUint32 := utils.IP2Int(net.ParseIP(ipAddressString))
		ok := expectedIPAddresses[receivedIPAddrUint32]
		if !ok {
			t.Fatalf("Received IP Address %s not found in the expected list", ipAddressString)
		}
	}
}

func validateNetworkContainerInfoResponse(
	t *testing.T,
	dncEP string,
	vnetID string,
	ncID string,
	apiVersion string,
	networkContainerInfoReq *CreateNetworkContainerRequest) error {
	networkContainerConfigResponse := getNetworkContainerConfiguration(t, dncEP, vnetID, ncID, apiVersion)

	if !reflect.DeepEqual(networkContainerConfigResponse.AssociationInfo,
		networkContainerInfoReq.AssociationInfo) {
		return fmt.Errorf("Mismatch for association info")
	}

	if len(networkContainerInfoReq.AllocationRequest.IPConstraint) > 0 {
		ipAddress, _, err := net.ParseCIDR(networkContainerConfigResponse.IPConfiguration.IPAddress)
		if err != nil {
			return fmt.Errorf("Failed to parse the allocated IP address: %s with error: %v",
				networkContainerConfigResponse.IPConfiguration.IPAddress, err)
		}

		if networkContainerInfoReq.AllocationRequest.IPConstraint != ipAddress.String() {
			return fmt.Errorf("Failed to allocate the requested IP address")
		}
	}

	if len(networkContainerInfoReq.AllocationRequest.SubnetName) > 0 {
		if networkContainerInfoReq.AllocationRequest.SubnetName !=
			networkContainerConfigResponse.IPConfiguration.SubnetName {
			return fmt.Errorf("Mismatch for Subnet names")
		}
	}

	return nil
}

func validateNetworkContainerEndpointPolicies(
	allocationRequest *api.AllocationRequest,
	associationInfo *api.AssociationInfo,
	dncEP string,
	vnetID string,
	ncID string,
	apiVersion string) error {

	ncURL := dncEP + "/networks/" + vnetID + "/networkcontainer/" + ncID + apiVersion

	var body bytes.Buffer

	invalidActionPolicy := composeNetworkContainerEndpointPolicies("6", "InvalidAction", "In", "127.0.0.1", "10.0.0.1", "1111",
		"1112", "Switch", 2000, "AclPolicy", "APIPA")

	networkcontainerReqInvalidAction := CreateNetworkContainerRequest{
		AllocationRequest:          *allocationRequest,
		AssociationInfo:            *associationInfo,
		AllowHostToNCCommunication: true,
		AllowNCToHostCommunication: true,
		EndpointPolicies:           invalidActionPolicy}

	invalidDirectionPolicy := composeNetworkContainerEndpointPolicies("6", "Allow", "InvalidDirection", "127.0.0.1", "10.0.0.1", "1111",
		"1112", "Switch", 2000, "AclPolicy", "APIPA")

	networkcontainerReqInvalidDirection := CreateNetworkContainerRequest{
		AllocationRequest:          *allocationRequest,
		AssociationInfo:            *associationInfo,
		AllowHostToNCCommunication: true,
		AllowNCToHostCommunication: true,
		EndpointPolicies:           invalidDirectionPolicy}

	invalidPriorityPolicy := composeNetworkContainerEndpointPolicies("6", "Allow", "In", "127.0.0.1", "10.0.0.1", "1111",
		"1112", "Switch", 0, "AclPolicy", "APIPA")

	networkcontainerreqinvalidPrior := CreateNetworkContainerRequest{
		AllocationRequest:          *allocationRequest,
		AssociationInfo:            *associationInfo,
		AllowHostToNCCommunication: true,
		AllowNCToHostCommunication: true,
		EndpointPolicies:           invalidPriorityPolicy}

	//confirm invalid action returns bad request
	json.NewEncoder(&body).Encode(networkcontainerReqInvalidAction)
	responseInvalidAction, _ := http.Post(ncURL, "application/json", &body)
	if responseInvalidAction.StatusCode != http.StatusBadRequest {
		return fmt.Errorf("Expected a 400 Bad Request when passing a invalid action but recieved %d status code", responseInvalidAction.StatusCode)
	}

	//confirm invalid direction returns bad request
	json.NewEncoder(&body).Encode(networkcontainerReqInvalidDirection)
	responseInvalidDirection, _ := http.Post(ncURL, "application/json", &body)
	if responseInvalidDirection.StatusCode != http.StatusBadRequest {
		return fmt.Errorf("Expected a 400 Bad Request when passing a invalid direction but recieved %d status code", responseInvalidDirection.StatusCode)
	}

	//confirm invalid prirority returns bad request
	json.NewEncoder(&body).Encode(networkcontainerreqinvalidPrior)
	responseInvalidPriority, _ := http.Post(ncURL, "application/json", &body)
	if responseInvalidPriority.StatusCode != http.StatusBadRequest {
		return fmt.Errorf("Expected a 400 Bad Request when passing a invalid priority but recieved %d status code", responseInvalidPriority.StatusCode)
	}
	fmt.Println("All invalid endpoint policy fields returned expected 400 bad request")
	return nil
}

func composeNetworkContainerEndpointPolicies(
	Protocols string,
	Action string,
	Direction string,
	LocalAddresses string,
	RemoteAddresses string,
	LocalPorts string,
	RemotePorts string,
	RuleType string,
	Priority uint16,
	Type string,
	EndpointType string) []cns.NetworkContainerRequestPolicies {

	aclSetting := AclTestPolicySetting{
		Action:          Action,
		Direction:       Direction,
		Protocols:       Protocols,
		LocalAddresses:  LocalAddresses,
		RemoteAddresses: RemoteAddresses,
		LocalPorts:      LocalPorts,
		RemotePorts:     RemotePorts,
		RuleType:        RuleType,
		Priority:        Priority,
	}

	endpointPolicySetting, _ := json.Marshal(aclSetting)
	endPointPolicy := cns.NetworkContainerRequestPolicies{
		Type:         Type,
		EndpointType: EndpointType,
		Settings:     endpointPolicySetting,
	}
	endPointPolicySlice := make([]cns.NetworkContainerRequestPolicies, 0)
	endPointPolicies := append(endPointPolicySlice, endPointPolicy)

	return endPointPolicies
}

func getSubnetConfiguration(
	t *testing.T,
	dncEP string,
	vnetID string,
	subnetName string,
	apiVersion string) *api.SubnetConfigurationResponse {
	response, err := http.Get(dncEP + "/networks/" + vnetID + "/subnets/" + subnetName + apiVersion)
	var subnetConfiguration api.SubnetConfigurationResponse
	err = utils.DecodeResponse(response, &subnetConfiguration)
	if err != nil {
		t.Fatalf("Decoding SubnetConfiguration failed. Response %+v Err:%+v", response, err)
	}

	defer response.Body.Close()
	fmt.Printf("Retrieved response:%+v\n", subnetConfiguration)
	return &subnetConfiguration
}

func getNetworkContainerConfiguration(
	t *testing.T,
	dncEP string,
	vnetID string,
	ncID string,
	apiVersion string) *api.NetworkContainerResponse {

	ncURL := dncEP + "/networks/" + vnetID + "/networkcontainer/" + ncID + apiVersion

	response, err := http.Get(ncURL)
	var networkContainerConfiguration api.NetworkContainerResponse
	err = utils.DecodeResponse(response, &networkContainerConfiguration)
	if err != nil {
		t.Fatalf("Decoding NetworkContainerConfiguration failed. Response %+v Err:%+v", response, err)
	}

	defer response.Body.Close()
	fmt.Printf("Retrieved response:%s\n", networkContainerConfiguration.String())
	return &networkContainerConfiguration
}

type MockServer struct {
	Server       *httptest.Server
	ResponseFunc func(http.ResponseWriter, *http.Request, interface{})
	Lock         sync.Mutex
	Context      interface{}
}

func (ts *MockServer) SetupServer(ipEP string) error {
	ts.Server = httptest.NewUnstartedServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ts.Lock.Lock()
			defer ts.Lock.Unlock()
			if ts.ResponseFunc != nil {
				ts.ResponseFunc(w, r, ts.Context)
			}
		}))
	l, err := net.Listen("tcp", ipEP)
	if err != nil {
		return err
	}
	ts.Server.Listener = l
	ts.Server.Start()
	return nil
}

func (s *MockServer) TearDownSuite() {
	s.Server.Close()
}

func (s *MockServer) SetupTest() {
	s.ResponseFunc = nil
}

type SignalInfo struct {
	Signal             chan string
	SignalCount        int
	CurrentSignalCount int
}

type JoinResponse struct {
	HTTPStatus int
	SignalInfo
}

type ValidateNCRequest func(*testing.T, cns.CreateNetworkContainerRequest, *NCDeploymentRequest) bool

type CnsCreateOrUpdateNCResponse struct {
	HTTPStatus int
	SignalInfo
	ValidateNCRequest
	CreateNCResponse *cns.CreateNetworkContainerResponse
}

type CnsDeleteNCResponse struct {
	HTTPStatus int
	SignalInfo
	ExpectedDeleteNCRequest *cns.DeleteNetworkContainerRequest
	DeleteNCResponse        *cns.DeleteNetworkContainerResponse
}

type CnsNumOfCPUCoresResponse struct {
	HTTPStatus         int
	NumOfCoresResponse *cns.NumOfCPUCoresResponse
}

type CnsPublishNetworkContainerResponse struct {
	HTTPStatus        int
	PublishNCResponse *cns.PublishNetworkContainerResponse
}

type CnsUnpublishNetworkContainerResponse struct {
	HTTPStatus          int
	UnpublishNCResponse *cns.UnpublishNetworkContainerResponse
}

type GetNetworkResponse struct {
	SignalInfo      SignalInfo
	HTTPStatus      int
	NetworkResponse *nmagentcontracts.VirtualNetworkResponse
}

type TagResponse struct {
	SignalInfo SignalInfo
	HTTPStatus int
	Response   *nmagentcontracts.TagResponse
}

type NMAgentNetworkResponse struct {
	JoinResponse       *JoinResponse
	GetNetworkResponse *GetNetworkResponse
}

type NMAgentResponderContext struct {
	TagResponse        map[string]*TagResponse
	NetworkResponseMap NMAgentNetworkResponseMap
}

type CnsResponderContext struct {
	T                        *testing.T
	CreateOrUpdateNCResponse *CnsCreateOrUpdateNCResponse
	DeleteNCResponse         *CnsDeleteNCResponse
	PublishNCResponse        *CnsPublishNetworkContainerResponse
	UnpublishNCResponse      *CnsUnpublishNetworkContainerResponse
	NumOfCoresResponse       *CnsNumOfCPUCoresResponse
	NCDeploymentRequest      *NCDeploymentRequest
}

type CnsResponderContextMap map[string]*CnsResponderContext

type NMAgentNetworkResponseMap map[string]*NMAgentNetworkResponse

var cnsContextLock = &sync.RWMutex{}
var nmagentContextLock = &sync.RWMutex{}

func (cxtMap CnsResponderContextMap) getContext(id string) *CnsResponderContext {
	cnsContextLock.RLock()
	defer cnsContextLock.RUnlock()
	return cxtMap[cns.SwiftPrefix+id]
}

func (cxtMap NMAgentNetworkResponseMap) getContext(id string) *NMAgentNetworkResponse {
	nmagentContextLock.RLock()
	defer nmagentContextLock.RUnlock()
	return cxtMap[id]
}

func signalTest(signalInfo *SignalInfo, contextID string) bool {
	if signalInfo.CurrentSignalCount < signalInfo.SignalCount {
		fmt.Printf("Signaling Context%s. Signal:%+v\n", contextID, signalInfo.Signal)
		signalInfo.Signal <- contextID
		signalInfo.CurrentSignalCount++
		fmt.Printf("Finished Signaling Context%s. Signal:%+v\n", contextID, signalInfo.Signal)
		return true
	}

	return false
}

func generateDefaultCnsResponderContextMap(t *testing.T, ncDeploymentRequests []*NCDeploymentRequest) CnsResponderContextMap {
	cnsRespoonderContextMap := make(CnsResponderContextMap)
	for _, ncDeploymentRequest := range ncDeploymentRequests {
		context := new(CnsResponderContext)
		context.T = t
		cnsCreateNCResponse := &cns.CreateNetworkContainerResponse{
			Response: cns.Response{
				ReturnCode: 0,
				Message:    "Successfully created NC"}}
		cnsDeleteNCResponse := &cns.DeleteNetworkContainerResponse{
			Response: cns.Response{
				ReturnCode: 0,
				Message:    "Successfully deleted NC"}}
		cnsPublishNCResponse := &cns.PublishNetworkContainerResponse{
			Response: cns.Response{
				ReturnCode: 0,
				Message:    "",
			},
			PublishErrorStr:     "",
			PublishStatusCode:   http.StatusOK,
			PublishResponseBody: make([]byte, 1),
		}
		cnsUnpublishNCResponse := &cns.UnpublishNetworkContainerResponse{
			Response: cns.Response{
				ReturnCode: 0,
				Message:    "",
			},
			UnpublishErrorStr:     "",
			UnpublishStatusCode:   http.StatusOK,
			UnpublishResponseBody: make([]byte, 1),
		}

		context.CreateOrUpdateNCResponse = &CnsCreateOrUpdateNCResponse{
			HTTPStatus: http.StatusOK,
			SignalInfo: SignalInfo{
				Signal:             make(chan string, 5),
				SignalCount:        1,
				CurrentSignalCount: 0},
			CreateNCResponse:  cnsCreateNCResponse,
			ValidateNCRequest: nil}

		context.DeleteNCResponse = &CnsDeleteNCResponse{
			HTTPStatus: http.StatusOK,
			SignalInfo: SignalInfo{
				Signal:             make(chan string, 5),
				SignalCount:        1,
				CurrentSignalCount: 0},
			DeleteNCResponse:        cnsDeleteNCResponse,
			ExpectedDeleteNCRequest: nil}

		context.PublishNCResponse = &CnsPublishNetworkContainerResponse{
			HTTPStatus:        http.StatusOK,
			PublishNCResponse: cnsPublishNCResponse,
		}

		context.UnpublishNCResponse = &CnsUnpublishNetworkContainerResponse{
			HTTPStatus:          http.StatusOK,
			UnpublishNCResponse: cnsUnpublishNCResponse,
		}

		cnsRespoonderContextMap[cns.SwiftPrefix+ncDeploymentRequest.NCID] = context
	}
	return cnsRespoonderContextMap
}

type NetworkDeploymentRequest struct {
	vnetID              string
	TagCount            int
	UseCustomDnsServers bool
	SubnetSize          int
	WaitForJoin         bool
	WaitForGet          bool
}

type DeployedNetworkContainersInfo struct {
	NetworkDeploymentRequests        []*NetworkDeploymentRequest
	NCDeploymentRequestsMapByNetwork map[string][]*NCDeploymentRequest
	CNSResponderContextMap           CnsResponderContextMap
	NodesMap                         map[string]bool
}

func generateDefaultNMAgentResponderContext(
	networkDeploymentRequests []*NetworkDeploymentRequest,
	useUpdatedTags bool) *NMAgentResponderContext {
	context := new(NMAgentResponderContext)
	context.NetworkResponseMap = make(NMAgentNetworkResponseMap)
	for _, networkDeploymentRequest := range networkDeploymentRequests {
		subnets := make([]nmagentcontracts.Subnet, 3)
		subnetSize := 24
		if networkDeploymentRequest.SubnetSize > 0 {
			subnetSize = networkDeploymentRequest.SubnetSize
		}

		for index := 0; index < 3; index++ {
			subnet := nmagentcontracts.Subnet{
				SubnetName:     fmt.Sprintf("subnet%+v", index+1),
				AddressPrefix:  fmt.Sprintf("10.0.%+v.0/%d", index, subnetSize),
				DefaultGateway: fmt.Sprintf("10.0.%+v.1", index)}

			subnets[index] = subnet
		}

		dnsServers := ""
		if networkDeploymentRequest.UseCustomDnsServers {
			dnsServers = "8.8.8.8,9.9.9.9"
		}
		var response NMAgentNetworkResponse
		vnetResponse := nmagentcontracts.VirtualNetworkResponse{
			ResponseCode: "200",
			CNetSpace:    "10.0.0.0/8,172.16.0.0/16,192.168.1.0/24",
			DNSServers:   dnsServers,
			VNetSpace:    "10.0.0.0/16",
			VnetVersion:  "4",
			Subnets:      subnets}
		getSignalCount := 0
		joinSignalCount := 0
		if networkDeploymentRequest.WaitForGet {
			getSignalCount = 1
		}
		if networkDeploymentRequest.WaitForJoin {
			joinSignalCount = 1
		}

		response.GetNetworkResponse = &GetNetworkResponse{
			NetworkResponse: &vnetResponse,
			HTTPStatus:      http.StatusOK,
			SignalInfo: SignalInfo{
				Signal:             make(chan string, 5),
				SignalCount:        getSignalCount,
				CurrentSignalCount: 0}}

		response.JoinResponse = &JoinResponse{
			HTTPStatus: http.StatusOK,
			SignalInfo: SignalInfo{
				Signal:             make(chan string, 5),
				SignalCount:        joinSignalCount,
				CurrentSignalCount: 0}}

		nmagentContextLock.Lock()
		context.NetworkResponseMap[networkDeploymentRequest.vnetID] = &response
		nmagentContextLock.Unlock()
	}

	return context
}

func NMAgentResponder(w http.ResponseWriter, r *http.Request, context interface{}) {
	nmagentResponderContext := context.(*NMAgentResponderContext)
	//fmt.Printf("Received URL:%s, Method:%s\n", r.RequestURI, r.Method)
	if strings.Contains(r.RequestURI, "joinedVirtualNetworks") {
		if r.Method == "GET" {
			vnetID := strings.Split(r.URL.Path, "/")[2]
			//fmt.Printf("Get network response for network:%s, %+v\n", vnetID, r)
			networkResponseContext := nmagentResponderContext.NetworkResponseMap.getContext(vnetID)
			if networkResponseContext == nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if networkResponseContext.GetNetworkResponse != nil {
				w.WriteHeader(networkResponseContext.GetNetworkResponse.HTTPStatus)
				if networkResponseContext.GetNetworkResponse.NetworkResponse != nil {
					json.NewEncoder(w).Encode(networkResponseContext.GetNetworkResponse.NetworkResponse)
				}
				shouldSignal := true
				if networkResponseContext.JoinResponse.SignalInfo.CurrentSignalCount <
					networkResponseContext.JoinResponse.SignalInfo.SignalCount {
					shouldSignal = false
				}
				nmagentContextLock.Lock()
				fmt.Printf("Responded with Config:%s\n", networkResponseContext.GetNetworkResponse.NetworkResponse.String())
				if shouldSignal && signalTest(&networkResponseContext.GetNetworkResponse.SignalInfo, vnetID) {
					fmt.Printf("Responded with Config:%s\n", networkResponseContext.GetNetworkResponse.NetworkResponse.String())
					fmt.Println("Set signal for get virtual network")
				}
				nmagentContextLock.Unlock()
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
		} else if r.Method == "POST" {
			vnetID := strings.Split(r.URL.Path, "/")[2]
			//fmt.Printf("Post join network for network:%s\n", vnetID)
			networkResponseContext := nmagentResponderContext.NetworkResponseMap.getContext(vnetID)
			if networkResponseContext == nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if networkResponseContext.JoinResponse != nil {
				w.WriteHeader(networkResponseContext.JoinResponse.HTTPStatus)
				if signalTest(&networkResponseContext.JoinResponse.SignalInfo, vnetID) {
					fmt.Println("Set signal for post join virtual network")
				} else {
					//fmt.Println("Responding to joined network")
				}
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
				// Induce error in response
				if induceError(induceErrorPercent) {
					fmt.Println("Setting response status with internal server error")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				deleteResponse := nmagentcontracts.DeleteNetworkContainerResponse{
					ResponseCode: "200"}
				json.NewEncoder(w).Encode(deleteResponse)
			} else {
				// Induce error in response
				if induceError(induceErrorPercent) {
					fmt.Println("Setting response status with internal server error")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				var ncReq nmagentcontracts.CreateOrUpdateNetworkContainerRequest
				json.NewDecoder(r.Body).Decode(&ncReq)
				putResponse := nmagentcontracts.CreateNetworkContainerResponse{ResponseCode: "200", Version: ncReq.Version}
				json.NewEncoder(w).Encode(putResponse)
				fmt.Printf("NC Publishing request:%+v, RequestURI:%+v\n", ncReq, r.RequestURI)
			}
		}
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func CNSResponder(w http.ResponseWriter, r *http.Request, context interface{}) {
	cnsResponderContextMap := context.(CnsResponderContextMap)

	if strings.Contains(r.RequestURI, "createorupdatenetworkcontainer") &&
		r.Method == "POST" {
		var ncRequest cns.CreateNetworkContainerRequest
		body, _ := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
		json.Unmarshal(body, &ncRequest)
		fmt.Printf("NCRequest:%+v\n", ncRequest)
		cnsContextLock.RLock()
		cnsResponderContext := cnsResponderContextMap[ncRequest.NetworkContainerid]

		cnsContextLock.RUnlock()
		if cnsResponderContext == nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Induce error in response
		if induceError(induceErrorPercent) {
			fmt.Println("Setting response status with internal server error")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if cnsResponderContext.CreateOrUpdateNCResponse.ValidateNCRequest != nil {
			deploymentRequest := cnsResponderContext.NCDeploymentRequest
			if !cnsResponderContext.CreateOrUpdateNCResponse.ValidateNCRequest(cnsResponderContext.T, ncRequest, deploymentRequest) {
				fmt.Println("Setting response status with internal server error")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		w.WriteHeader(cnsResponderContext.CreateOrUpdateNCResponse.HTTPStatus)
		json.NewEncoder(w).Encode(cnsResponderContext.CreateOrUpdateNCResponse.CreateNCResponse)
		if signalTest(&cnsResponderContext.CreateOrUpdateNCResponse.SignalInfo, ncRequest.NetworkContainerid) {
			fmt.Println("Set signal for CreateOrUpdateNC")
		}
	}
	if strings.Contains(r.RequestURI, "deletenetworkcontainer") &&
		r.Method == "POST" {
		var ncRequest cns.DeleteNetworkContainerRequest
		json.NewDecoder(r.Body).Decode(&ncRequest)
		fmt.Printf("NC Request:%+v \n", ncRequest)
		cnsContextLock.RLock()
		cnsResponderContext, ok := cnsResponderContextMap[ncRequest.NetworkContainerid]
		cnsContextLock.RUnlock()
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(cnsResponderContext.DeleteNCResponse.HTTPStatus)
		json.NewEncoder(w).Encode(cnsResponderContext.DeleteNCResponse.DeleteNCResponse)
		if signalTest(&cnsResponderContext.DeleteNCResponse.SignalInfo, ncRequest.NetworkContainerid) {
			fmt.Println("Set signal for delete NC")
		}
	}

	if strings.Contains(r.RequestURI, "setorchestratortype") &&
		r.Method == "POST" {
		var otRequest cns.SetOrchestratorTypeRequest
		json.NewDecoder(r.Body).Decode(&otRequest)
		fmt.Printf("Set Orchestrator Type Request:%+v \n", otRequest)
		cnsResponse := cns.Response{
			ReturnCode: 0,
			Message:    "Successfully set orchestrator type"}
		json.NewEncoder(w).Encode(cnsResponse)
	}

	if strings.Contains(r.RequestURI, "hostcpucores") &&
		r.Method == "GET" {
		cnsContextLock.RLock()
		cnsResponderContext, ok := cnsResponderContextMap["usecustomcpucorresponse"]
		cnsContextLock.RUnlock()
		if !ok {
			CnsNumOfCPUCoresResp := cns.NumOfCPUCoresResponse{
				Response: cns.Response{
					ReturnCode: 0,
					Message:    "Successfully retrieved CPU cores"},
				NumOfCPUCores: defaultNumCPUCores}
			json.NewEncoder(w).Encode(CnsNumOfCPUCoresResp)
		} else {
			w.WriteHeader(cnsResponderContext.NumOfCoresResponse.HTTPStatus)
			json.NewEncoder(w).Encode(cnsResponderContext.NumOfCoresResponse.NumOfCoresResponse)
		}
	}

	if strings.Contains(r.RequestURI, "unpublishnetworkcontainer") &&
		r.Method == "POST" {
		var ncRequest cns.UnpublishNetworkContainerRequest
		json.NewDecoder(r.Body).Decode(&ncRequest)
		fmt.Printf("NC Request:%+v \n", ncRequest)
		cnsContextLock.RLock()
		cnsResponderContext, ok := cnsResponderContextMap[cns.SwiftPrefix+ncRequest.NetworkContainerID]
		cnsContextLock.RUnlock()
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(cnsResponderContext.UnpublishNCResponse.HTTPStatus)
		// Generate UnpublishNCResponse
		var deleteNCResponse nmagentcontracts.DeleteNetworkContainerResponse
		// Induce error in response
		if induceError(induceErrorPercent) {
			fmt.Println("Inducing error for deleteNC from nmagent")
			deleteNCResponse.ResponseCode = "401" // Authentication error
		} else {
			deleteNCResponse.ResponseCode = "200"
		}
		cnsResponderContext.UnpublishNCResponse.UnpublishNCResponse.UnpublishStatusCode = http.StatusOK
		unpublishResponseBody, _ := json.Marshal(deleteNCResponse)
		cnsResponderContext.UnpublishNCResponse.UnpublishNCResponse.UnpublishResponseBody = unpublishResponseBody
		json.NewEncoder(w).Encode(cnsResponderContext.UnpublishNCResponse.UnpublishNCResponse)
	} else if strings.Contains(r.RequestURI, "publishnetworkcontainer") &&
		r.Method == "POST" {
		var ncRequest cns.PublishNetworkContainerRequest
		json.NewDecoder(r.Body).Decode(&ncRequest)
		fmt.Printf("NC Request:%+v \n", ncRequest)
		cnsContextLock.RLock()
		cnsResponderContext, ok := cnsResponderContextMap[cns.SwiftPrefix+ncRequest.NetworkContainerID]
		cnsContextLock.RUnlock()
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(cnsResponderContext.PublishNCResponse.HTTPStatus)
		// Generate PublishNCResponse
		var putNCResponse nmagentcontracts.CreateNetworkContainerResponse
		// Induce error in response
		if induceError(induceErrorPercent) {
			fmt.Println("Inducing error for putNC from nmagent")
			putNCResponse.ResponseCode = "401" // Authentication error
		} else {
			putNCResponse.ResponseCode = "200"
		}
		putNCResponse.Version = "1"
		cnsResponderContext.PublishNCResponse.PublishNCResponse.PublishStatusCode = http.StatusOK
		publishResponseBody, _ := json.Marshal(putNCResponse)
		cnsResponderContext.PublishNCResponse.PublishNCResponse.PublishResponseBody = publishResponseBody
		json.NewEncoder(w).Encode(cnsResponderContext.PublishNCResponse.PublishNCResponse)
	}
}

func GetNetworkStatus(t *testing.T, resourceStatusURL string, expectedStatusCode int) string {
	fmt.Printf("Resource status URL:%s\n", resourceStatusURL)
	response, err := http.Get(resourceStatusURL)
	if err != nil {
		t.Fatalf("Failed to get network status for resource URL:%s. Error:%+v",
			resourceStatusURL,
			err)
	}

	defer response.Body.Close()

	var operationStatus api.RequestStatus
	err = utils.DecodeResponse(response, &operationStatus)
	if err != nil {
		t.Fatalf("Failed to decode network response for resource URL:%s. Error:%+v",
			resourceStatusURL,
			err)
	}

	if response.StatusCode != expectedStatusCode && response.StatusCode != http.StatusAccepted {
		t.Fatalf("Unexpected response code %d received when trying to get network status for resource URL:%s.",
			response.StatusCode,
			resourceStatusURL)
	}

	fmt.Printf("Resource URL:%s Status:%s\n", resourceStatusURL, operationStatus.String())

	if response.StatusCode == http.StatusAccepted {
		return "Pending"
	}

	return operationStatus.Status
}

func IsJoinInState(t *testing.T, resourceStatusURL string, state string, expectedStatusCode int) bool {
	status := GetNetworkStatus(t, resourceStatusURL, expectedStatusCode)
	return status == state
}

func WaitForResourceCleanup(t *testing.T, resourceStatusURL string, timeToWait int) bool {
	ticker := time.NewTicker(1 * time.Second).C
	timer := time.NewTimer(time.Duration(timeToWait) * time.Second).C
	for {
		select {
		case <-timer:
			return false

		case <-ticker:
			response, err := http.Get(resourceStatusURL)
			if err == nil {
				response.Body.Close()
				if response.StatusCode == http.StatusNotFound {
					return true
				}
			}
		}
	}
}

func WaitForResourceState(t *testing.T, resourceStatusURL string, status string, expectedStatusCode int, timeToWait int) bool {
	ticker := time.NewTicker(1 * time.Second).C
	timer := time.NewTimer(time.Duration(timeToWait) * time.Second).C
	for {
		select {
		case <-timer:
			return false

		case <-ticker:
			if IsJoinInState(t, resourceStatusURL, status, expectedStatusCode) {
				return true
			}
		}
	}
}

func setupNetwork(t *testing.T, dncEP, apiVersion string, networkDeploymentRequests []*NetworkDeploymentRequest) {
	fmt.Println("Started NMAgent Server\n", nmAgentServer.Server.URL)
	nmAgentServer.SetupTest()
	nmAgentResponderContext := generateDefaultNMAgentResponderContext(
		networkDeploymentRequests,
		false)
	nmAgentServer.Lock.Lock()
	nmAgentServer.ResponseFunc = NMAgentResponder
	nmAgentServer.Context = nmAgentResponderContext
	nmAgentServer.Lock.Unlock()

	for _, networkDeploymentRequest := range networkDeploymentRequests {
		var body bytes.Buffer
		var respErr api.JsonErr
		networkReq := api.NetworkInfoRequest{
			NetworkType: api.AzureNet}

		json.NewEncoder(&body).Encode(networkReq)

		response, err := http.Post(dncEP+"/networks/"+networkDeploymentRequest.vnetID+apiVersion, "application/json", &body)
		if err != nil {
			t.Fatalf("CreateNetwork failed with error: %+v\n", err)
		}

		if response.StatusCode != http.StatusOK {
			err = utils.DecodeResponse(response, &respErr)
			if err != nil || respErr.Code != 0 {
				t.Fatalf("CreateNetwork failed with response %+v Err:%+v", response, err)
			}
		}

		response.Body.Close()
	}

	fmt.Println("Waiting for Join and Get network signals")

	var chans = []chan string{}
	nmagentContextLock.RLock()
	for _, networkDeploymentRequest := range networkDeploymentRequests {
		response, ok := nmAgentResponderContext.NetworkResponseMap[networkDeploymentRequest.vnetID]
		if !ok {
			continue
		}

		if response.GetNetworkResponse.SignalInfo.CurrentSignalCount < response.GetNetworkResponse.SignalInfo.SignalCount {
			fmt.Printf("Waiting for get network response for network %+v \n", networkDeploymentRequest)
			chans = append(chans, response.GetNetworkResponse.SignalInfo.Signal)
		}

		if response.JoinResponse.SignalInfo.CurrentSignalCount < response.JoinResponse.SignalInfo.SignalCount {
			fmt.Printf("Waiting for join network response for network %+v \n", networkDeploymentRequest)
			chans = append(chans, response.JoinResponse.SignalInfo.Signal)
		}
	}
	nmagentContextLock.RUnlock()

	cases := make([]reflect.SelectCase, len(chans))
	for i, ch := range chans {
		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
	}
	remaining := len(chans)
	for remaining > 0 {
		chosen, value, _ := reflect.Select(cases)
		fmt.Printf("Received value:%+v\n", value)
		cases[chosen].Chan = reflect.ValueOf(nil)
		remaining--
		continue
	}
	fmt.Println("Received all join and get signals")

	for _, networkDeploymentRequest := range networkDeploymentRequests {
		resourceStatusURL := dncEP + "/networks/" + networkDeploymentRequest.vnetID + "/status" + apiVersion
		WaitForResourceState(t, resourceStatusURL, "Completed", http.StatusOK, 2)
		fmt.Printf("Successfully created network [%s]\n", networkDeploymentRequest.vnetID)
	}
}

func setupNetworkAndJoinSubnet(t *testing.T, dncEP, vnetID, subnetName, apiVersion string, addDNSServers bool, tagCount int) {
	var body bytes.Buffer
	resourceURL := dncEP + "/networks/" + vnetID + "/subnets/" + subnetName + apiVersion
	networkDeploymentRequests := []*NetworkDeploymentRequest{{vnetID: vnetID, UseCustomDnsServers: addDNSServers, TagCount: tagCount, WaitForJoin: true, WaitForGet: true}}

	setupNetwork(t, dncEP, apiVersion, networkDeploymentRequests)
	subnetReq := api.SubnetRequest{
		AuthenticationToken: vnetID}
	json.NewEncoder(&body).Encode(subnetReq)
	fmt.Printf("Joining subnet %s, network %s. AccessToken:%s\n", subnetName, vnetID, subnetReq.AuthenticationToken)
	response, err := http.Post(resourceURL, "application/json", &body)
	if err != nil {
		t.Fatalf("JoinSubnet failed with error: %+v\n", err)
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("JoinSubnet failed with status code %d\n", response.StatusCode)
	}
}

func createReservation(t *testing.T, reservationreq api.AllocationRequest, dncEP, vnetID, reservationID, apiVersion string) {
	var body bytes.Buffer
	var respErr api.JsonErr
	json.NewEncoder(&body).Encode(reservationreq)
	reservationURL := dncEP + "/networks/" + vnetID + "/reservations/" + reservationID + apiVersion
	response, err := http.Post(reservationURL, "application/json", &body)
	if err != nil {
		t.Fatalf("CreateReservation failed with error: %+v\n", err)
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		if err = utils.DecodeResponse(response, &respErr); err != nil || respErr.Code != 0 {
			t.Fatalf("CreateReservation failed with response %+v Err:%+v", respErr, err)
		}
	}

	reservationStatusURL := dncEP + "/networks/" + vnetID + "/reservations/" + reservationID + "/status" + apiVersion
	WaitForResourceState(t, reservationStatusURL, "Completed", http.StatusOK, 2)
	fmt.Printf("Successfully created reservation [%s]\n", reservationID)
}

func AreCnsRequestsEqual(expected, received cns.CreateNetworkContainerRequest, vnetIDMap map[string]string) bool {
	if expected.Version != received.Version {
		fmt.Printf("Expected.Version %s Received.Version:%s", expected.Version, received.Version)
		return false
	}

	if expected.NetworkContainerType != received.NetworkContainerType {
		fmt.Printf("Expected.NetworkContainerType %s Received.NetworkContainerType:%s",
			expected.NetworkContainerType, received.NetworkContainerType)
		return false
	}

	if expected.NetworkContainerid != received.NetworkContainerid {
		fmt.Printf("Expected.NetworkContainerid %s Received.NetworkContainerid:%s",
			expected.NetworkContainerid, received.NetworkContainerid)
		return false
	}

	if expected.PrimaryInterfaceIdentifier != received.PrimaryInterfaceIdentifier {
		fmt.Printf("Expected.PrimaryInterfaceIdentifier %s Received.PrimaryInterfaceIdentifier:%s",
			expected.PrimaryInterfaceIdentifier, received.PrimaryInterfaceIdentifier)
		return false
	}

	if expected.AuthorizationToken != received.AuthorizationToken {
		fmt.Printf("Expected.AuthorizationToken %s Received.AuthorizationToken:%s",
			expected.AuthorizationToken, received.AuthorizationToken)
		return false
	}

	if !reflect.DeepEqual(expected.OrchestratorContext, received.OrchestratorContext) {
		fmt.Printf("Expected.OrchestratorContext %+v Received.OrchestratorContext:%+v",
			expected.OrchestratorContext, received.OrchestratorContext)
		return false
	}

	if vnetIDMap != nil {
		associatedNCID, ok := vnetIDMap[received.IPConfiguration.IPSubnet.IPAddress]
		if !ok {
			fmt.Printf("Allocated IP %s expected to be found in the allocated list", received.IPConfiguration.IPSubnet.IPAddress)
			return false
		}
		if associatedNCID == "" {
			vnetIDMap[received.IPConfiguration.IPSubnet.IPAddress] = received.NetworkContainerid
		} else {
			if associatedNCID != received.NetworkContainerid {
				fmt.Printf("Allocated IP %s received in the request NC %s has already been allcoated to different NC %s",
					received.IPConfiguration.IPSubnet.IPAddress,
					associatedNCID,
					received.NetworkContainerid)
				return false
			}
		}
	}

	if len(expected.IPConfiguration.IPSubnet.IPAddress) != 0 {
		if !reflect.DeepEqual(expected.IPConfiguration, received.IPConfiguration) {
			fmt.Printf("Expected.IPConfiguration %+v Received.IPConfiguration:%+v",
				expected.IPConfiguration, received.IPConfiguration)

			return false
		}
	}

	if !reflect.DeepEqual(expected.MultiTenancyInfo, received.MultiTenancyInfo) {
		fmt.Printf("Expected.MultiTenancyInfo %+v Received.MultiTenancyInfo:%+v",
			expected.MultiTenancyInfo, received.MultiTenancyInfo)
		return false
	}

	if expected.AllowHostToNCCommunication != received.AllowHostToNCCommunication {
		fmt.Printf("Expected.AllowHostToNCCommunication %t Received.AllowHostToNCCommunication:%t",
			expected.AllowHostToNCCommunication, received.AllowHostToNCCommunication)
		return false
	}

	if expected.AllowNCToHostCommunication != received.AllowNCToHostCommunication {
		fmt.Printf("Expected.AllowNCToHostCommunication %t Received.AllowNCToHostCommunication:%t",
			expected.AllowNCToHostCommunication, received.AllowNCToHostCommunication)
		return false
	}

	return true
}

func createNetworkContainer(
	t *testing.T,
	nodeID string,
	nodeInfoReq api.NodeInfoRequest,
	allocationRequest *api.AllocationRequest,
	associationInfo *api.AssociationInfo,
	reservationID, dncEP, vnetID, ncid, apiVersion string,
	tagCount int,
	allowHostToNCCommunication bool,
	allowNCToHostCommunication bool,
	endPointPolicies []cns.NetworkContainerRequestPolicies,
	skipDelete bool) {
	fmt.Println("****CreateNetworkContainer Test**********")

	setupNetworkAndJoinSubnet(t, dncEP, vnetID, allocationRequest.SubnetName, apiVersion, true, tagCount)
	addOrUpdateNode(t, dncEP, nodeID, apiVersion, &nodeInfoReq)

	if len(reservationID) > 0 {
		createReservation(t, *allocationRequest, dncEP, vnetID, reservationID, apiVersion)
	}

	nmAgentResponderContext := nmAgentServer.Context.(*NMAgentResponderContext)

	cnetSpace := nmAgentResponderContext.NetworkResponseMap.getContext(vnetID).GetNetworkResponse.NetworkResponse.CNetSpace
	if nmAgentResponderContext.TagResponse != nil {
		for _, tagResponse := range nmAgentResponderContext.TagResponse {
			cnetSpace = cnetSpace + "," + strings.Join(tagResponse.Response.Values, ",")
		}
	}
	fmt.Println("CnetSpace;", cnetSpace)
	allocatedIP := "10.0.0.4"
	if len(allocationRequest.IPConstraint) > 0 {
		allocatedIP = allocationRequest.IPConstraint
	}
	var ipConfig cns.IPConfiguration
	ipConfig.DNSServers = strings.Split(nmAgentResponderContext.NetworkResponseMap.getContext(vnetID).GetNetworkResponse.NetworkResponse.DNSServers, ",")
	ipConfig.IPSubnet = cns.IPSubnet{
		IPAddress:    allocatedIP,
		PrefixLength: 24}
	ipConfig.GatewayIPAddress = "10.0.0.1"
	ncDeploymentRequests := []*NCDeploymentRequest{&NCDeploymentRequest{
		NCID: ncid}}
	cnsResponderContextMap := generateDefaultCnsResponderContextMap(t, ncDeploymentRequests)
	ncRequestValidatorFunc := func(t *testing.T, req cns.CreateNetworkContainerRequest, deploymentRequest *NCDeploymentRequest) bool {
		podInfo := OrchestratorContext{
			PodName:      "testpodname",
			PodNamespace: "testpodnamespace"}
		context, _ := json.Marshal(podInfo)
		expectedReq := cns.CreateNetworkContainerRequest{
			Version:                    req.Version,
			NetworkContainerType:       associationInfo.ContainerType,
			NetworkContainerid:         cns.SwiftPrefix + ncid,
			PrimaryInterfaceIdentifier: associationInfo.InterfaceID,
			IPConfiguration:            ipConfig,
			AuthorizationToken:         "",
			LocalIPConfiguration:       localIPConfiguaration,
			CnetAddressSpace:           make([]cns.IPSubnet, 0),
			OrchestratorContext:        context,
			MultiTenancyInfo:           req.MultiTenancyInfo,
			AllowHostToNCCommunication: allowHostToNCCommunication,
			AllowNCToHostCommunication: allowNCToHostCommunication,
			EndpointPolicies:           endPointPolicies}
		fmt.Printf("ReceivedResponse:%+v\n", req)
		fmt.Printf("ExpectedResponse:%+v\n", expectedReq)

		if !AreCnsRequestsEqual(expectedReq, req, nil) {
			fmt.Printf("Mismatch in expected and received Cns create or update request.\nExpected:%+v\nReceived:%+v\n", expectedReq, req)
			return false
		}

		fmt.Println("Response matches with the expected request")
		return true
	}

	cnsResponderContextMap.getContext(ncid).CreateOrUpdateNCResponse.ValidateNCRequest = ncRequestValidatorFunc
	cnsServer.Lock.Lock()
	cnsServer.Context = cnsResponderContextMap
	cnsServer.Lock.Unlock()

	var body bytes.Buffer
	var respErr api.JsonErr
	if len(reservationID) > 0 {
		networkcontainerreq := CreateNetworkContainerRequestV2{
			AllocationRequest:          *allocationRequest,
			AssociationInfo:            *associationInfo,
			ReservationID:              reservationID,
			AllowHostToNCCommunication: allowHostToNCCommunication,
			AllowNCToHostCommunication: allowNCToHostCommunication,
			EndpointPolicies:           endPointPolicies}
		json.NewEncoder(&body).Encode(networkcontainerreq)
	} else {
		networkcontainerreq := CreateNetworkContainerRequest{
			AllocationRequest:          *allocationRequest,
			AssociationInfo:            *associationInfo,
			AllowHostToNCCommunication: allowHostToNCCommunication,
			AllowNCToHostCommunication: allowNCToHostCommunication,
			EndpointPolicies:           endPointPolicies}
		json.NewEncoder(&body).Encode(networkcontainerreq)
	}
	ncURL := dncEP + "/networks/" + vnetID + "/networkcontainer/" + ncid + apiVersion
	ncURLStatus := dncEP + "/networks/" + vnetID + "/networkcontainer/" + ncid + "/status" + apiVersion
	response, err := http.Post(ncURL, "application/json", &body)
	if err != nil {
		t.Fatalf("CreateNetworkContainer failed with error: %+v\n", err)
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		if err = utils.DecodeResponse(response, &respErr); err != nil || respErr.Code != 0 {
			t.Fatalf("CreateNetworkContainer failed with response %+v Err:%+v", respErr, err)
		}
	}
	if err := validateNetworkContainerEndpointPolicies(allocationRequest, associationInfo, dncEP, vnetID, ncid, apiVersion); err != nil {
		t.Fatalf("Network container endpoint policy failed to handle validation with error: %+v", err)
	}

	var chans = []chan string{}
	if nmAgentResponderContext.TagResponse != nil {
		for _, tagResponse := range nmAgentResponderContext.TagResponse {
			fmt.Printf("Waiting for get %s tag definition\n", tagResponse.Response.Name)
			chans = append(chans, tagResponse.SignalInfo.Signal)
		}
		cases := make([]reflect.SelectCase, len(chans))
		for i, ch := range chans {
			cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
		}
		remaining := len(chans)
		for remaining > 0 {
			chosen, value, _ := reflect.Select(cases)
			fmt.Printf("Received signal value:%+v\n", value)
			cases[chosen].Chan = reflect.ValueOf(nil)
			remaining--
			continue
		}
	}

	fmt.Printf("Waiting for create NC Url:%s\n", ncURL)
	<-cnsResponderContextMap.getContext(ncid).CreateOrUpdateNCResponse.SignalInfo.Signal
	WaitForResourceState(t, ncURLStatus, "Completed", http.StatusOK, 2)

	if allocationRequest.IPConstraint == "" {
		fmt.Printf("Successfully created network container using dynamic IP [%s]\n", ncid)
	} else {
		fmt.Printf("Successfully created network container using specific IP [%s]\n", ncid)
	}

	networkcontainerreq := CreateNetworkContainerRequest{
		AllocationRequest:          *allocationRequest,
		AssociationInfo:            *associationInfo,
		AllowHostToNCCommunication: allowHostToNCCommunication}

	if err := validateNetworkContainerInfoResponse(t, dncEP, vnetID, ncid, apiVersion, &networkcontainerreq); err != nil {
		t.Errorf("Network container validation failed for network container due to error: %v\n", err)
		t.Fatal(err)
	}

	networkURL := dncEP + "/networks/" + vnetID + apiVersion
	nodeURL := dncEP + "/nodes/" + nodeID + apiVersion

	fmt.Printf("Delete resource %s when allocations exist. Expected to fail\n", networkURL)
	deleteErrResp := deleteResource(t, networkURL, http.StatusBadRequest)
	if deleteErrResp.Code != api.NetworkInUse {
		t.Fatalf("Delete network expected to fail with error %s, Error response %s \n", api.NetworkInUse.String(), deleteErrResp.String())
	}

	fmt.Printf("Delete resource %s when node has NC's deployed exist. Expected to fail\n", nodeURL)
	deleteErrResp = deleteResource(t, nodeURL, http.StatusBadRequest)
	if deleteErrResp.Code != api.NodeInUse {
		t.Fatalf("Delete node expected to fail with error %s, Error response %s \n", api.NodeInUse.String(), deleteErrResp.String())
	}

	savedSubnets := nmAgentResponderContext.NetworkResponseMap.getContext(vnetID).GetNetworkResponse.NetworkResponse.Subnets

	// delete subnets from NMAgent before cleanup up Dnc
	nmAgentServer.Lock.Lock()
	nmAgentResponderContext.NetworkResponseMap.getContext(vnetID).GetNetworkResponse.NetworkResponse.Subnets = make([]nmagentcontracts.Subnet, 0)
	nmAgentServer.Lock.Unlock()

	networkupdatehandler.RefreshNetworkConfiguration(datamodel.GetNetworkKey(vnetID), true)

	fmt.Printf("Delete subnets %+v when allocations exist. Expected to fail\n", savedSubnets)
	subnetConfig := getSubnetConfiguration(t, dncEP, vnetID, savedSubnets[0].SubnetName, apiVersion)
	if subnetConfig.SubnetName != savedSubnets[0].SubnetName {
		t.Fatalf("Subnet %s deleted even though NC %s exists", savedSubnets[0], ncid)
	}

	nmAgentServer.Lock.Lock()
	nmAgentResponderContext.NetworkResponseMap.getContext(vnetID).GetNetworkResponse.NetworkResponse.Subnets = savedSubnets
	nmAgentServer.Lock.Unlock()

	reservationURL := dncEP + "/networks/" + vnetID + "/reservations/" + reservationID + apiVersion
	if len(reservationID) > 0 {
		fmt.Printf("Delete resource %s when reservation has NCassociated with it. Expected to fail\n", nodeURL)
		deleteErrResp = deleteResource(t, reservationURL, http.StatusBadRequest)
		if deleteErrResp.Code != api.ReservationInUse {
			t.Fatalf("Delete reservation expected to fail with error %s as there is an association, Error response %s \n", api.ReservationInUse.String(), deleteErrResp.String())
		}
	}

	if !skipDelete {
		DeleteNetworkContainer(t, nodeID, reservationID, dncEP, vnetID, ncid, apiVersion)
	}
}

func CreateNetworkContainer(
	t *testing.T,
	nodeID string,
	nodeInfoReq api.NodeInfoRequest, networkcontainerreq CreateNetworkContainerRequest, dncEP, vnetID, ncid, apiVersion string, skipDelete bool) {
	fmt.Println("****CreateNetworkContainer Test**********")
	createNetworkContainer(
		t,
		nodeID,
		nodeInfoReq,
		&networkcontainerreq.AllocationRequest,
		&networkcontainerreq.AssociationInfo,
		"",
		dncEP,
		vnetID,
		ncid,
		apiVersion,
		0,
		networkcontainerreq.AllowHostToNCCommunication,
		networkcontainerreq.AllowNCToHostCommunication,
		networkcontainerreq.EndpointPolicies,
		skipDelete)
}

func createNetworkContainerAndUpdateNetworkConfig(
	t *testing.T,
	nodeID string,
	nodeInfoReq api.NodeInfoRequest,
	networkcontainerreq CreateNetworkContainerRequest,
	dncEP, vnetID, ncid, apiVersion string,
	updatedNMAgentResponsderContext *NMAgentResponderContext,
	addDNSServersInConfig bool,
	tagCount int) {
	setupNetworkAndJoinSubnet(
		t,
		dncEP,
		vnetID,
		networkcontainerreq.AllocationRequest.SubnetName,
		apiVersion,
		addDNSServersInConfig,
		tagCount)
	addOrUpdateNode(t, dncEP, nodeID, apiVersion, &nodeInfoReq)
	fmt.Println("Started CNS Server", cnsServer.Server.URL)
	ncDeploymentRequests := []*NCDeploymentRequest{&NCDeploymentRequest{
		NCID: ncid}}
	cnsResponderContextMap := generateDefaultCnsResponderContextMap(t, ncDeploymentRequests)
	cnsServer.Lock.Lock()
	cnsServer.Context = cnsResponderContextMap
	cnsServer.Lock.Unlock()

	var body bytes.Buffer
	var respErr api.JsonErr
	json.NewEncoder(&body).Encode(networkcontainerreq)
	ncURL := dncEP + "/networks/" + vnetID + "/networkcontainer/" + ncid + apiVersion
	ncURLStatus := dncEP + "/networks/" + vnetID + "/networkcontainer/" + ncid + "/status" + apiVersion
	response, err := http.Post(ncURL, "application/json", &body)
	if err != nil {
		t.Fatalf("CreateNetworkContainer failed with error: %+v\n", err)
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		if err = utils.DecodeResponse(response, &respErr); err != nil || respErr.Code != 0 {
			t.Fatalf("CreateNetworkContainer failed with response %+v Err:%+v", respErr, err)
		}
	}

	nmAgentResponderContext := nmAgentServer.Context.(*NMAgentResponderContext)

	if nmAgentResponderContext.TagResponse != nil {
		var chans = []chan string{}
		for _, tagResponse := range nmAgentResponderContext.TagResponse {
			fmt.Printf("Waiting for get %s tag definition\n", tagResponse.Response.Name)
			chans = append(chans, tagResponse.SignalInfo.Signal)
		}
		cases := make([]reflect.SelectCase, len(chans))
		for i, ch := range chans {
			cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
		}
		remaining := len(chans)
		for remaining > 0 {
			chosen, value, _ := reflect.Select(cases)
			fmt.Printf("Received signal value:%+v", value)
			cases[chosen].Chan = reflect.ValueOf(nil)
			remaining--
			continue
		}
	}

	fmt.Printf("Waiting for create NC Url:%s\n", ncURL)
	<-cnsResponderContextMap.getContext(ncid).CreateOrUpdateNCResponse.SignalInfo.Signal
	WaitForResourceState(t, ncURLStatus, "Completed", http.StatusOK, 2)

	var ipConfig cns.IPConfiguration
	var dnsServers []string

	if len(updatedNMAgentResponsderContext.NetworkResponseMap.getContext(vnetID).GetNetworkResponse.NetworkResponse.DNSServers) != 0 {
		dnsServers = strings.Split(updatedNMAgentResponsderContext.NetworkResponseMap.getContext(vnetID).GetNetworkResponse.NetworkResponse.DNSServers, ",")
	} else {
		dnsServers = []string{"168.63.129.16"}
	}

	cnetSpace := updatedNMAgentResponsderContext.NetworkResponseMap.getContext(vnetID).GetNetworkResponse.NetworkResponse.CNetSpace
	if updatedNMAgentResponsderContext.TagResponse != nil {
		for _, tagResponse := range updatedNMAgentResponsderContext.TagResponse {
			cnetSpace = cnetSpace + "," + strings.Join(tagResponse.Response.Values, ",")
		}
	}

	ipConfig.DNSServers = dnsServers
	ipConfig.IPSubnet = cns.IPSubnet{
		IPAddress:    "10.0.0.15",
		PrefixLength: 24}
	ipConfig.GatewayIPAddress = "10.0.0.1"
	updatedCnsResponderContextMap := generateDefaultCnsResponderContextMap(t, ncDeploymentRequests)
	updateNCRequestValidatorFunc := func(t *testing.T, req cns.CreateNetworkContainerRequest, deploymentRequest *NCDeploymentRequest) bool {
		podInfo := OrchestratorContext{
			PodName:      "testpodname",
			PodNamespace: "testpodnamespace"}
		context, _ := json.Marshal(podInfo)
		expectedReq := cns.CreateNetworkContainerRequest{
			Version:                    req.Version,
			NetworkContainerType:       cns.AzureContainerInstance,
			NetworkContainerid:         cns.SwiftPrefix + ncid,
			PrimaryInterfaceIdentifier: networkcontainerreq.AssociationInfo.InterfaceID,
			IPConfiguration:            ipConfig,
			AuthorizationToken:         "",
			LocalIPConfiguration:       localIPConfiguaration,
			CnetAddressSpace:           make([]cns.IPSubnet, 0),
			OrchestratorContext:        context,
			MultiTenancyInfo:           req.MultiTenancyInfo}
		fmt.Printf("ReceivedResponse:%+v\n", req)
		fmt.Printf("ExpectedResponse:%+v\n", expectedReq)

		if !AreCnsRequestsEqual(expectedReq, req, nil) {
			fmt.Printf("Mismatch in expected and received Cns create or update request.\nExpected:%+v\nReceived:%+v\n", expectedReq, req)
			return false
		}
		fmt.Println("Response matches with the expected request")
		return true
	}

	updatedCnsResponderContextMap.getContext(ncid).CreateOrUpdateNCResponse.ValidateNCRequest = updateNCRequestValidatorFunc
	cnsServer.Lock.Lock()
	cnsServer.Context = updatedCnsResponderContextMap
	cnsServer.Lock.Unlock()

	nmAgentServer.Lock.Lock()
	nmAgentServer.ResponseFunc = NMAgentResponder
	nmAgentServer.Context = updatedNMAgentResponsderContext
	nmAgentServer.Lock.Unlock()

	fmt.Println("Waiting for get network reponse post network update")
	<-updatedNMAgentResponsderContext.NetworkResponseMap.getContext(vnetID).GetNetworkResponse.SignalInfo.Signal
	fmt.Println("Received signal for get network reponse post network update")

	if updatedNMAgentResponsderContext.TagResponse != nil {
		var chans = []chan string{}
		for _, tagResponse := range updatedNMAgentResponsderContext.TagResponse {
			fmt.Printf("Waiting for get %s tag definition\n", tagResponse.Response.Name)
			chans = append(chans, tagResponse.SignalInfo.Signal)
		}
		cases := make([]reflect.SelectCase, len(chans))
		for i, ch := range chans {
			cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
		}
		remaining := len(chans)
		for remaining > 0 {
			chosen, value, _ := reflect.Select(cases)
			fmt.Printf("Received signal value:%+v", value)
			cases[chosen].Chan = reflect.ValueOf(nil)
			remaining--
			continue
		}
	}

	fmt.Printf("Waiting for update NC post network update")
	<-updatedCnsResponderContextMap.getContext(ncid).CreateOrUpdateNCResponse.SignalInfo.Signal
	fmt.Printf("Recieved signal for update NC post network update")
	deleteResource(t, ncURL, http.StatusOK)
	fmt.Printf("Waiting for delete NC Url:%s", ncURL)

	<-updatedCnsResponderContextMap.getContext(ncid).DeleteNCResponse.SignalInfo.Signal

	if !WaitForResourceCleanup(t, ncURL, 4) {
		t.Fatalf("Failed to cleanup network container %s\n", ncid)
	}

	networkURL := dncEP + "/networks/" + vnetID + apiVersion
	nodeURL := dncEP + "/nodes/" + nodeID + apiVersion

	deleteResource(t, networkURL, http.StatusOK)
	deleteResource(t, nodeURL, http.StatusOK)
}

func CreateNetworkContainerAndUpdateCnetSpace(t *testing.T, nodeID string, nodeInfoReq api.NodeInfoRequest, networkcontainerreq CreateNetworkContainerRequest, dncEP, vnetID, ncid, apiVersion string) {
	fmt.Println("****CreateNetworkContainerAndUpdateCnetSpace Test**********")
	networkDeploymentRequests := []*NetworkDeploymentRequest{
		&NetworkDeploymentRequest{vnetID: vnetID, UseCustomDnsServers: true, TagCount: 0, WaitForJoin: true, WaitForGet: true}}

	nmAgentResponderContext := generateDefaultNMAgentResponderContext(
		networkDeploymentRequests,
		false)
	networkResponseContext := nmAgentResponderContext.NetworkResponseMap.getContext(vnetID)

	networkResponseContext.GetNetworkResponse.NetworkResponse.CNetSpace = "10.0.0.0/16,172.16.0.0/16,192.168.1.0/24"
	networkResponseContext.GetNetworkResponse.NetworkResponse.VnetVersion = "5"
	networkResponseContext.JoinResponse.HTTPStatus = http.StatusOK
	networkResponseContext.JoinResponse.SignalInfo.SignalCount = 0
	networkResponseContext.GetNetworkResponse.SignalInfo.SignalCount = 1
	networkResponseContext.GetNetworkResponse.SignalInfo.CurrentSignalCount = 0
	createNetworkContainerAndUpdateNetworkConfig(
		t,
		nodeID,
		nodeInfoReq,
		networkcontainerreq,
		dncEP,
		vnetID,
		ncid,
		apiVersion,
		nmAgentResponderContext,
		true,
		0)
}

func CreateNetworkContainerAndAddCnetSpace(t *testing.T, nodeID string, nodeInfoReq api.NodeInfoRequest, networkcontainerreq CreateNetworkContainerRequest, dncEP, vnetID, ncid, apiVersion string) {
	fmt.Println("****CreateNetworkContainerAndAddCnetSpace Test**********")
	networkDeploymentRequests := []*NetworkDeploymentRequest{
		&NetworkDeploymentRequest{vnetID: vnetID, UseCustomDnsServers: true, TagCount: 0, WaitForJoin: true, WaitForGet: true}}

	nmAgentResponderContext := generateDefaultNMAgentResponderContext(
		networkDeploymentRequests,
		false)
	networkResponseContext := nmAgentResponderContext.NetworkResponseMap.getContext(vnetID)
	networkResponseContext.GetNetworkResponse.NetworkResponse.CNetSpace =
		networkResponseContext.GetNetworkResponse.NetworkResponse.CNetSpace + ",1.0.0.0/8"
	networkResponseContext.GetNetworkResponse.NetworkResponse.VnetVersion = "5"
	networkResponseContext.JoinResponse.HTTPStatus = http.StatusOK
	networkResponseContext.JoinResponse.SignalInfo.SignalCount = 0
	networkResponseContext.GetNetworkResponse.SignalInfo.SignalCount = 1
	networkResponseContext.GetNetworkResponse.SignalInfo.CurrentSignalCount = 0
	createNetworkContainerAndUpdateNetworkConfig(
		t,
		nodeID,
		nodeInfoReq,
		networkcontainerreq,
		dncEP,
		vnetID,
		ncid,
		apiVersion,
		nmAgentResponderContext,
		true,
		0)
}

func CreateNetworkContainerAndRemoveCnetSpace(t *testing.T, nodeID string, nodeInfoReq api.NodeInfoRequest, networkcontainerreq CreateNetworkContainerRequest, dncEP, vnetID, ncid, apiVersion string) {
	fmt.Println("****CreateNetworkContainerAndRemoveCnetSpace Test**********")
	networkDeploymentRequests := []*NetworkDeploymentRequest{
		&NetworkDeploymentRequest{vnetID: vnetID, UseCustomDnsServers: true, TagCount: 0, WaitForJoin: true, WaitForGet: true}}

	nmAgentResponderContext := generateDefaultNMAgentResponderContext(
		networkDeploymentRequests,
		false)
	networkResponseContext := nmAgentResponderContext.NetworkResponseMap.getContext(vnetID)

	networkResponseContext.GetNetworkResponse.NetworkResponse.CNetSpace = "10.0.0.0/8,172.16.0.0/16"
	networkResponseContext.GetNetworkResponse.NetworkResponse.VnetVersion = "5"
	networkResponseContext.JoinResponse.HTTPStatus = http.StatusOK
	networkResponseContext.JoinResponse.SignalInfo.SignalCount = 0
	networkResponseContext.GetNetworkResponse.SignalInfo.SignalCount = 1
	networkResponseContext.GetNetworkResponse.SignalInfo.CurrentSignalCount = 0
	createNetworkContainerAndUpdateNetworkConfig(
		t,
		nodeID,
		nodeInfoReq,
		networkcontainerreq,
		dncEP,
		vnetID,
		ncid,
		apiVersion,
		nmAgentResponderContext,
		true,
		0)
}

func CreateNetworkContainerAndAddNewDNSServer(t *testing.T, nodeID string, nodeInfoReq api.NodeInfoRequest, networkcontainerreq CreateNetworkContainerRequest, dncEP, vnetID, ncid, apiVersion string) {
	fmt.Println("****CreateNetworkContainerAndAddNewDNSServer Test**********")
	networkDeploymentRequests := []*NetworkDeploymentRequest{
		&NetworkDeploymentRequest{vnetID: vnetID, UseCustomDnsServers: true, TagCount: 0, WaitForJoin: true, WaitForGet: true}}

	nmAgentResponderContext := generateDefaultNMAgentResponderContext(
		networkDeploymentRequests,
		false)
	networkResponseContext := nmAgentResponderContext.NetworkResponseMap.getContext(vnetID)

	networkResponseContext.GetNetworkResponse.NetworkResponse.DNSServers =
		networkResponseContext.GetNetworkResponse.NetworkResponse.DNSServers + ",10.10.10.10"
	networkResponseContext.GetNetworkResponse.NetworkResponse.VnetVersion = "5"
	networkResponseContext.JoinResponse.HTTPStatus = http.StatusOK
	networkResponseContext.JoinResponse.SignalInfo.SignalCount = 0
	networkResponseContext.GetNetworkResponse.SignalInfo.SignalCount = 1
	networkResponseContext.GetNetworkResponse.SignalInfo.CurrentSignalCount = 0
	createNetworkContainerAndUpdateNetworkConfig(
		t,
		nodeID,
		nodeInfoReq,
		networkcontainerreq,
		dncEP,
		vnetID,
		ncid,
		apiVersion,
		nmAgentResponderContext,
		true,
		0)
}

func CreateNetworkContainerAndRemoveDNSServerInNetworkConfig(t *testing.T, nodeID string, nodeInfoReq api.NodeInfoRequest, networkcontainerreq CreateNetworkContainerRequest, dncEP, vnetID, ncid, apiVersion string) {
	fmt.Println("****CreateNetworkContainerAndUpdateToUseAzureDNSServers Test**********")
	networkDeploymentRequests := []*NetworkDeploymentRequest{
		&NetworkDeploymentRequest{vnetID: vnetID, UseCustomDnsServers: true, TagCount: 0, WaitForJoin: true, WaitForGet: true}}

	nmAgentResponderContext := generateDefaultNMAgentResponderContext(
		networkDeploymentRequests,
		false)
	networkResponseContext := nmAgentResponderContext.NetworkResponseMap.getContext(vnetID)

	networkResponseContext.GetNetworkResponse.NetworkResponse.DNSServers = "8.8.8.8"
	networkResponseContext.GetNetworkResponse.NetworkResponse.VnetVersion = "5"
	networkResponseContext.JoinResponse.HTTPStatus = http.StatusOK
	networkResponseContext.JoinResponse.SignalInfo.SignalCount = 0
	networkResponseContext.GetNetworkResponse.SignalInfo.SignalCount = 1
	networkResponseContext.GetNetworkResponse.SignalInfo.CurrentSignalCount = 0
	createNetworkContainerAndUpdateNetworkConfig(
		t,
		nodeID,
		nodeInfoReq,
		networkcontainerreq,
		dncEP,
		vnetID,
		ncid,
		apiVersion,
		nmAgentResponderContext,
		true,
		0)
}

func CreateNetworkContainerAndUpdateToUseAzureDNSServers(t *testing.T, nodeID string, nodeInfoReq api.NodeInfoRequest, networkcontainerreq CreateNetworkContainerRequest, dncEP, vnetID, ncid, apiVersion string) {
	fmt.Println("****CreateNetworkContainerAndUpdateToUseAzureDNSServers Test**********")
	networkDeploymentRequests := []*NetworkDeploymentRequest{
		&NetworkDeploymentRequest{vnetID: vnetID, UseCustomDnsServers: false, TagCount: 0, WaitForJoin: true, WaitForGet: true}}

	nmAgentResponderContext := generateDefaultNMAgentResponderContext(
		networkDeploymentRequests,
		false)
	networkResponseContext := nmAgentResponderContext.NetworkResponseMap.getContext(vnetID)

	networkResponseContext.GetNetworkResponse.NetworkResponse.VnetVersion = "5"
	networkResponseContext.JoinResponse.HTTPStatus = http.StatusOK
	networkResponseContext.JoinResponse.SignalInfo.SignalCount = 0
	networkResponseContext.GetNetworkResponse.SignalInfo.SignalCount = 1
	networkResponseContext.GetNetworkResponse.SignalInfo.CurrentSignalCount = 0
	createNetworkContainerAndUpdateNetworkConfig(
		t,
		nodeID,
		nodeInfoReq,
		networkcontainerreq,
		dncEP,
		vnetID,
		ncid,
		apiVersion,
		nmAgentResponderContext,
		true,
		0)
}

func CreateNetworkContainerAndUpdateToUseCustomDNSServers(t *testing.T, nodeID string, nodeInfoReq api.NodeInfoRequest, networkcontainerreq CreateNetworkContainerRequest, dncEP, vnetID, ncid, apiVersion string) {
	fmt.Println("****CreateNetworkContainerAndUpdateToUseCustomDNSServers Test**********")
	networkDeploymentRequests := []*NetworkDeploymentRequest{
		&NetworkDeploymentRequest{vnetID: vnetID, UseCustomDnsServers: true, TagCount: 0, WaitForJoin: true, WaitForGet: true}}

	nmAgentResponderContext := generateDefaultNMAgentResponderContext(
		networkDeploymentRequests,
		false)
	networkResponseContext := nmAgentResponderContext.NetworkResponseMap.getContext(vnetID)

	networkResponseContext.GetNetworkResponse.NetworkResponse.VnetVersion = "6"
	networkResponseContext.JoinResponse.HTTPStatus = http.StatusOK
	networkResponseContext.JoinResponse.SignalInfo.SignalCount = 0
	networkResponseContext.GetNetworkResponse.SignalInfo.SignalCount = 1
	networkResponseContext.GetNetworkResponse.SignalInfo.CurrentSignalCount = 0
	createNetworkContainerAndUpdateNetworkConfig(
		t,
		nodeID,
		nodeInfoReq,
		networkcontainerreq,
		dncEP,
		vnetID,
		ncid,
		apiVersion,
		nmAgentResponderContext,
		false,
		0)
}

func DeleteNetworkContainer(t *testing.T,
	nodeID string,
	reservationID, dncEP, vnetID, ncid, apiVersion string) {
	networkURL := dncEP + "/networks/" + vnetID + apiVersion
	nodeURL := dncEP + "/nodes/" + nodeID + apiVersion
	ncURL := dncEP + "/networks/" + vnetID + "/networkcontainer/" + ncid + apiVersion
	reservationURL := dncEP + "/networks/" + vnetID + "/reservations/" + reservationID + apiVersion

	deleteResource(t, ncURL, http.StatusOK)
	t.Logf("Waiting for delete NC Url:%s", ncURL)

	if !WaitForResourceCleanup(t, ncURL, 4) {
		t.Fatalf("Failed to cleanup network container %s\n", ncid)
	}

	if len(reservationID) > 0 {
		t.Logf("Delete resource %s when reservation exist. Expected to fail", networkURL)
		deleteErrResp := deleteResource(t, networkURL, http.StatusBadRequest)
		if deleteErrResp.Code != api.NetworkInUse {
			t.Fatalf("Delete network expected to fail with error %s, Error response %s \n", api.NetworkInUse.String(), deleteErrResp.String())
		}

		deleteResource(t, reservationURL, http.StatusOK)
	}

	deleteResource(t, networkURL, http.StatusOK)
	deleteResource(t, nodeURL, http.StatusOK)
}

type NCDeploymentRequest struct {
	NCID              string
	AllocationRequest api.AllocationRequest
	ReservationID     string
	AssociationInfo   api.AssociationInfo
}

type DeploymentInfo struct {
	NetworkDeploymentRequest *NetworkDeploymentRequest
	NCDeploymentRequests     []*NCDeploymentRequest
}

//  http://play.golang.org/p/m8TNTtygK0
func getNextIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func DeployNetworkContainers(
	t *testing.T,
	dncEP, apiVersion string,
	deploymentInfos []DeploymentInfo,
	deleteDeployments bool,
	updatedCPUCores int) *DeployedNetworkContainersInfo {
	networkDeploymentRequests := make([]*NetworkDeploymentRequest, 0)
	ncDeploymentRequestsMapByNetwork := make(map[string][]*NCDeploymentRequest)
	networkSubnetsMap := make(map[string]map[string]bool)
	nodesMap := make(map[string]bool)
	totalNCs := 0
	vnetSubnetMap := make(map[string]net.IP)

	for _, deploymentInfo := range deploymentInfos {
		index := 0
		networkDeploymentRequests = append(networkDeploymentRequests, deploymentInfo.NetworkDeploymentRequest)
		networkNCDeploymentRequests := make([]*NCDeploymentRequest, len(deploymentInfo.NCDeploymentRequests))

		ncDeploymentRequestsMapByNetwork[deploymentInfo.NetworkDeploymentRequest.vnetID] = networkNCDeploymentRequests
		for _, ncDeploymentRequest := range deploymentInfo.NCDeploymentRequests {
			t.Logf("NC request:%+v", ncDeploymentRequest)
			networkNCDeploymentRequests[index] = ncDeploymentRequest
			index++
			subnetMap, ok := networkSubnetsMap[deploymentInfo.NetworkDeploymentRequest.vnetID]
			if !ok {
				subnetMap = make(map[string]bool)
				networkSubnetsMap[deploymentInfo.NetworkDeploymentRequest.vnetID] = subnetMap
			}
			subnetMap[ncDeploymentRequest.AllocationRequest.SubnetName] = true
			nodesMap[ncDeploymentRequest.AssociationInfo.NodeID] = true
			totalNCs++
		}
	}

	vnetsIPMap := make(map[string]map[string]string)
	setupNetwork(t, dncEP, apiVersion, networkDeploymentRequests)
	for vnetID, subnetMap := range networkSubnetsMap {
		_, ok := vnetsIPMap[vnetID]
		if !ok {
			vnetsIPMap[vnetID] = make(map[string]string)
		}
		for subnetName := range subnetMap {
			var body bytes.Buffer
			subnetReq := api.SubnetRequest{
				AuthenticationToken: vnetID}
			json.NewEncoder(&body).Encode(subnetReq)
			subnetURL := dncEP + "/networks/" + vnetID + "/subnets/" + subnetName + apiVersion
			fmt.Printf("Joining subnet %s, network %s. AccessToken:%s\n", vnetID, subnetName, subnetReq.AuthenticationToken)
			response, err := http.Post(subnetURL, "application/json", &body)
			if err != nil {
				t.Fatalf("JoinSubnet failed with error: %+v\n", err)
			}

			if response.StatusCode != http.StatusOK {
				t.Fatalf("JoinSubnet failed with status code %d\n", response.StatusCode)
			}

			subnetConfiguration := getSubnetConfiguration(t, dncEP, vnetID, subnetName, apiVersion)
			if subnetConfiguration.AccessToken != fmt.Sprintf("%x", sha256.Sum256([]byte(subnetReq.AuthenticationToken))) {
				t.Fatalf("AccessToken mismatch. Received:%s , Expected:%s\n",
					subnetConfiguration.AccessToken,
					subnetReq.AuthenticationToken)
			}

			response.Body.Close()
		}
	}

	ncDeploymentRequests := make([]*NCDeploymentRequest, totalNCs)
	index := 0
	fmt.Printf("Total NCs:%d\n", totalNCs)
	for _, ncDeploymentRequestList := range ncDeploymentRequestsMapByNetwork {
		for _, ncDeploymentRequest := range ncDeploymentRequestList {
			ncDeploymentRequests[index] = ncDeploymentRequest
			index++
		}
	}

	cnsResponderContextMap := generateDefaultCnsResponderContextMap(t, ncDeploymentRequests)

	if updatedCPUCores > 0 {
		contextWithCPUCores := new(CnsResponderContext)
		CnsNumOfCPUCoresResp := &cns.NumOfCPUCoresResponse{
			Response: cns.Response{
				ReturnCode: 0,
				Message:    "Successfully retrieved CPU cores"},
			NumOfCPUCores: updatedCPUCores}
		contextWithCPUCores.NumOfCoresResponse = &CnsNumOfCPUCoresResponse{
			HTTPStatus:         http.StatusOK,
			NumOfCoresResponse: CnsNumOfCPUCoresResp}
		cnsResponderContextMap["usecustomcpucorresponse"] = contextWithCPUCores
	}

	cnsServer.Lock.Lock()
	cnsServer.Context = cnsResponderContextMap
	cnsServer.ResponseFunc = CNSResponder
	cnsServer.Lock.Unlock()

	for nodeID := range nodesMap {
		nodeInfoReq := api.NodeInfoRequest{IPAddresses: []string{"127.0.0.1"}, LocationType: "", OrchestratorType: cns.Kubernetes}
		addOrUpdateNode(t, dncEP, nodeID, apiVersion, &nodeInfoReq)
	}

	nodeLocalIPMap := make(map[string]map[string]string)
	for nodeID := range nodesMap {
		nodeLocalIPMap[nodeID] = make(map[string]string)
	}

	for vnetID, ncDeploymentRequests := range ncDeploymentRequestsMapByNetwork {
		fmt.Printf("vnetID:%s, ncRequests:%+v", vnetID, ncDeploymentRequests)
		for _, ncDeploymentRequest := range ncDeploymentRequests {
			if len(ncDeploymentRequest.ReservationID) > 0 {
				createReservation(t, ncDeploymentRequest.AllocationRequest, dncEP, vnetID, ncDeploymentRequest.ReservationID, apiVersion)
			}

			nmAgentResponderContext := nmAgentServer.Context.(*NMAgentResponderContext)

			cnetSpace := nmAgentResponderContext.NetworkResponseMap.getContext(vnetID).GetNetworkResponse.NetworkResponse.CNetSpace
			if nmAgentResponderContext.TagResponse != nil {
				for _, tagResponse := range nmAgentResponderContext.TagResponse {
					cnetSpace = cnetSpace + "," + strings.Join(tagResponse.Response.Values, ",")
				}
			}
			fmt.Println("CnetSpace;", cnetSpace)
			var allocatedIP string
			if len(ncDeploymentRequest.AllocationRequest.IPConstraint) > 0 {
				allocatedIP = ncDeploymentRequest.AllocationRequest.IPConstraint
			} else {
				lastAllocatedIP, ok := vnetSubnetMap[vnetID+ncDeploymentRequest.AllocationRequest.SubnetName]
				if !ok {
					allocatedIP = "10.0.0.4"
					lastAllocatedIP = net.ParseIP(allocatedIP)

				} else {
					getNextIP(lastAllocatedIP)
					allocatedIP = lastAllocatedIP.String()
				}
				vnetSubnetMap[vnetID+ncDeploymentRequest.AllocationRequest.SubnetName] = lastAllocatedIP
			}
			vnetIPMap, ok := vnetsIPMap[vnetID]
			if !ok {
				vnetIPMap = make(map[string]string)
				vnetsIPMap[vnetID] = vnetIPMap
			}
			vnetIPMap[allocatedIP] = ""

			accessToken := ""
			var ipConfig cns.IPConfiguration
			ipConfig.DNSServers = strings.Split(nmAgentResponderContext.NetworkResponseMap.getContext(vnetID).GetNetworkResponse.NetworkResponse.DNSServers, ",")
			if len(ncDeploymentRequest.ReservationID) > 0 {
				ipConfig.IPSubnet = cns.IPSubnet{
					IPAddress:    allocatedIP,
					PrefixLength: 24}
			}
			ipConfig.GatewayIPAddress = "10.0.0.1"
			ncRequestValidatorFunc := func(t *testing.T, req cns.CreateNetworkContainerRequest, deploymentRequest *NCDeploymentRequest) bool {
				expectedReq := cns.CreateNetworkContainerRequest{
					Version:                    req.Version,
					NetworkContainerType:       cns.AzureContainerInstance,
					NetworkContainerid:         cns.SwiftPrefix + deploymentRequest.NCID,
					PrimaryInterfaceIdentifier: deploymentRequest.AssociationInfo.InterfaceID,
					IPConfiguration:            ipConfig,
					AuthorizationToken:         accessToken,
					LocalIPConfiguration:       localIPConfiguaration,
					CnetAddressSpace:           make([]cns.IPSubnet, 0),
					OrchestratorContext:        deploymentRequest.AssociationInfo.OrchestratorContext,
					MultiTenancyInfo:           req.MultiTenancyInfo}
				fmt.Printf("ReceivedResponse:%+v\n", req)
				fmt.Printf("ExpectedResponse:%+v\n", expectedReq)
				var expectedIP string
				if len(deploymentRequest.ReservationID) > 0 {
					expectedIP = expectedReq.IPConfiguration.IPSubnet.IPAddress + "/" + strconv.FormatInt(int64(expectedReq.IPConfiguration.IPSubnet.PrefixLength), 10)
				} else {
					expectedIP = ""
				}
				receivedIP := req.IPConfiguration.IPSubnet.IPAddress + "/" + strconv.FormatInt(int64(req.IPConfiguration.IPSubnet.PrefixLength), 10)
				fmt.Printf("%+v, %+v\n", expectedIP, receivedIP)
				fmt.Printf("AllocatedIP:%+v, ReceivedIP:%+v", expectedReq.IPConfiguration.IPSubnet, req.IPConfiguration.IPSubnet)

				if req.LocalIPConfiguration.IPSubnet.IPAddress == "" || req.LocalIPConfiguration.GatewayIPAddress == "" {
					return false
				}

				if req.LocalIPConfiguration.GatewayIPAddress != "169.254.128.1" || req.LocalIPConfiguration.IPSubnet.PrefixLength != utils.LocalIPSubnetPrefixLength {
					return false
				}

				if len(req.IPConfiguration.IPSubnet.IPAddress) == 0 {
					fmt.Printf("Allocated subnet in the request is empty")
					return false
				}

				if associatedncID, ok :=
					nodeLocalIPMap[deploymentRequest.AssociationInfo.NodeID][req.LocalIPConfiguration.IPSubnet.IPAddress]; ok {
					if associatedncID != deploymentRequest.NCID {
						fmt.Printf("Local IP %s received as part of NC %s is already associated with %s on node: %s\n",
							req.LocalIPConfiguration.IPSubnet.IPAddress,
							deploymentRequest.NCID,
							associatedncID,
							deploymentRequest.AssociationInfo.NodeID)
						return false
					}
				} else {
					nodeLocalIPMap[deploymentRequest.AssociationInfo.NodeID][req.LocalIPConfiguration.IPSubnet.IPAddress] = deploymentRequest.NCID
				}

				if !AreCnsRequestsEqual(expectedReq, req, vnetIPMap) {
					fmt.Printf("Mismatch in expected and received Cns create or update request. "+
						"\nExpected:%+v\nReceived:%+v\n",
						expectedReq,
						req)
					return false
				}

				fmt.Println("Response matches with the expected request")
				return true
			}

			cnsResponderContextMap.getContext(ncDeploymentRequest.NCID).CreateOrUpdateNCResponse.ValidateNCRequest = ncRequestValidatorFunc
			cnsResponderContextMap.getContext(ncDeploymentRequest.NCID).NCDeploymentRequest = ncDeploymentRequest
		}

	}

	cnsServer.Lock.Lock()
	cnsServer.Context = cnsResponderContextMap
	cnsServer.Lock.Unlock()

	ncIndex := 0
	for vnetID, ncDeploymentRequests := range ncDeploymentRequestsMapByNetwork {
		for _, ncDeploymentRequest := range ncDeploymentRequests {
			var body bytes.Buffer
			var respErr api.JsonErr
			if len(ncDeploymentRequest.ReservationID) > 0 {
				networkcontainerreq := CreateNetworkContainerRequestV2{
					AllocationRequest: ncDeploymentRequest.AllocationRequest,
					AssociationInfo:   ncDeploymentRequest.AssociationInfo,
					ReservationID:     ncDeploymentRequest.ReservationID}
				json.NewEncoder(&body).Encode(networkcontainerreq)
			} else {
				networkcontainerreq := CreateNetworkContainerRequest{
					AllocationRequest: ncDeploymentRequest.AllocationRequest,
					AssociationInfo:   ncDeploymentRequest.AssociationInfo}
				json.NewEncoder(&body).Encode(networkcontainerreq)
			}

			ncURL := dncEP + "/networks/" + vnetID + "/networkcontainer/" + ncDeploymentRequest.NCID + apiVersion
			response, err := http.Post(ncURL, "application/json", &body)
			fmt.Printf("%d: Creating resource:%s\n", ncIndex, ncURL)
			ncIndex++
			if err != nil {
				t.Fatalf("CreateNetworkContainer failed with error: %+v\n", err)
			}

			if response.StatusCode != http.StatusOK {
				if err = utils.DecodeResponse(response, &respErr); err != nil || respErr.Code != 0 {
					t.Fatalf("CreateNetworkContainer failed with response %+v Err:%+v", respErr, err)
				}
			}

			response.Body.Close()
		}
	}

	var chans = []chan string{}
	for _, ncDeploymentRequests := range ncDeploymentRequestsMapByNetwork {
		for _, ncDeploymentRequest := range ncDeploymentRequests {
			fmt.Printf("Waiting for create NC:%s\n", ncDeploymentRequest.NCID)
			chans = append(chans, cnsResponderContextMap.getContext(ncDeploymentRequest.NCID).CreateOrUpdateNCResponse.Signal)
		}
	}
	cases := make([]reflect.SelectCase, len(chans))
	for i, ch := range chans {
		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
	}
	remaining := len(chans)
	for remaining > 0 {
		chosen, value, _ := reflect.Select(cases)
		fmt.Printf("Received signal for CreateOrUpdate NC:%+v\n", value)
		cases[chosen].Chan = reflect.ValueOf(nil)
		remaining--
		continue
	}

	for vnetID, ncDeploymentRequests := range ncDeploymentRequestsMapByNetwork {
		for _, ncDeploymentRequest := range ncDeploymentRequests {
			fmt.Printf("Waiting for NC creation to complete: %s\n", ncDeploymentRequest.NCID)
			ncURLStatus := dncEP + "/networks/" + vnetID + "/networkcontainer/" + ncDeploymentRequest.NCID + "/status" + apiVersion
			WaitForResourceState(t, ncURLStatus, "Completed", http.StatusOK, 2)
		}
	}

	deployedNetworkContainersInfo := &DeployedNetworkContainersInfo{
		NetworkDeploymentRequests:        networkDeploymentRequests,
		NCDeploymentRequestsMapByNetwork: ncDeploymentRequestsMapByNetwork,
		CNSResponderContextMap:           cnsResponderContextMap,
		NodesMap:                         nodesMap}

	if deleteDeployments {
		DeleteDeployments(t, deployedNetworkContainersInfo)
	}

	return deployedNetworkContainersInfo
}

func DeleteDeployments(t *testing.T, deployedNetworkContainersInfo *DeployedNetworkContainersInfo) {

	for vnetID, ncDeploymentRequests := range deployedNetworkContainersInfo.NCDeploymentRequestsMapByNetwork {
		for _, ncDeploymentRequest := range ncDeploymentRequests {
			fmt.Printf("Deleting NC:%+v\n", ncDeploymentRequest)
			ncURL := dncEP + "/networks/" + vnetID + "/networkcontainer/" + ncDeploymentRequest.NCID + apiVersion
			deleteResource(t, ncURL, http.StatusOK)
		}
	}

	var chans = []chan string{}
	for _, ncDeploymentRequests := range deployedNetworkContainersInfo.NCDeploymentRequestsMapByNetwork {
		for _, ncDeploymentRequest := range ncDeploymentRequests {
			fmt.Printf("Waiting for delete NC:%s\n", ncDeploymentRequest.NCID)
			chans = append(chans,
				deployedNetworkContainersInfo.CNSResponderContextMap.getContext(ncDeploymentRequest.NCID).DeleteNCResponse.SignalInfo.Signal)
		}
	}
	cases := make([]reflect.SelectCase, len(chans))
	for i, ch := range chans {
		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
	}
	remaining := len(chans)
	for remaining > 0 {
		chosen, value, _ := reflect.Select(cases)
		fmt.Printf("Received signal for Delete NC:%+v\n", value)
		cases[chosen].Chan = reflect.ValueOf(nil)
		remaining--
		continue
	}

	for vnetID, ncDeploymentRequests := range deployedNetworkContainersInfo.NCDeploymentRequestsMapByNetwork {
		for _, ncDeploymentRequest := range ncDeploymentRequests {
			fmt.Printf("Waiting for NC deletion to complete: %s\n", ncDeploymentRequest.NCID)
			ncURL := dncEP + "/networks/" + vnetID + "/networkcontainer/" + ncDeploymentRequest.NCID + apiVersion

			if !WaitForResourceCleanup(t, ncURL, 4) {
				t.Fatalf("Failed to cleanup network container %s", ncDeploymentRequest.NCID)
			}

			if len(ncDeploymentRequest.ReservationID) > 0 {
				reservationURL := dncEP + "/networks/" + vnetID + "/reservations/" + ncDeploymentRequest.ReservationID + apiVersion
				fmt.Printf("Deleting reservation%s\n", ncDeploymentRequest.ReservationID)
				deleteResource(t, reservationURL, http.StatusOK)
			}
		}
	}

	for _, networkDeploymentRequest := range deployedNetworkContainersInfo.NetworkDeploymentRequests {
		networkURL := dncEP + "/networks/" + networkDeploymentRequest.vnetID + apiVersion
		deleteResource(t, networkURL, http.StatusOK)
	}

	for nodeID := range deployedNetworkContainersInfo.NodesMap {
		networkURL := dncEP + "/nodes/" + nodeID + apiVersion
		deleteResource(t, networkURL, http.StatusOK)
	}
}

func TestValidateSubnetExhaustion(t *testing.T) {
	fmt.Println("****ValidateSubnetExhaustion**********")

	id, _ := uuid.NewUUID()
	vnetID := id.String()

	networkDeploymentRequest := NetworkDeploymentRequest{
		vnetID:              vnetID,
		UseCustomDnsServers: true,
		TagCount:            0,
		SubnetSize:          29,
		WaitForJoin:         true,
		WaitForGet:          true,
	}

	ncPerNetworkCount := 3
	ncDeploymentRequests := make([]*NCDeploymentRequest, ncPerNetworkCount)

	for ncIndex := 0; ncIndex < ncPerNetworkCount; ncIndex++ {
		ncID, _ := uuid.NewUUID()
		podInfo := OrchestratorContext{
			PodName:      fmt.Sprintf("%s-%d", nodeID, ncIndex),
			PodNamespace: "testpodnamespace",
		}
		context, _ := json.Marshal(podInfo)
		ncDeploymentRequest := NCDeploymentRequest{
			AllocationRequest: api.AllocationRequest{SubnetName: "subnet1"},
			AssociationInfo:   api.AssociationInfo{NodeID: nodeID, InterfaceID: "127.0.0.1", ContainerType: cns.AzureContainerInstance, OrchestratorContext: context},
			NCID:              ncID.String(),
		}
		ncDeploymentRequests[ncIndex] = &ncDeploymentRequest
	}

	deploymentInfos := []DeploymentInfo{{NetworkDeploymentRequest: &networkDeploymentRequest, NCDeploymentRequests: ncDeploymentRequests}}

	deployedNetworkContainersInfo := DeployNetworkContainers(t, dncEP, apiVersion, deploymentInfos, false, defaultNumCPUCores)

	id, _ = uuid.NewUUID()
	ncIDToFail := id.String()
	deploymentRequestToFail := CreateNetworkContainerRequest{
		AllocationRequest: api.AllocationRequest{SubnetName: "subnet1"},
		AssociationInfo:   api.AssociationInfo{NodeID: nodeID, InterfaceID: "127.0.0.1", ContainerType: cns.AzureContainerInstance, OrchestratorContext: orchContext},
	}
	var body bytes.Buffer
	_ = json.NewEncoder(&body).Encode(deploymentRequestToFail)
	ncURL := dncEP + "/networks/" + vnetID + "/networkcontainer/" + ncIDToFail + apiVersion
	response, err := http.Post(ncURL, "application/json", &body)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		var respErr api.JsonErr
		if err := utils.DecodeResponse(response, &respErr); err != nil || respErr.Code != 0 {
			t.Fatalf("CreateNetworkContainer call failed with response %+v Err:%+v", respErr, err)
		}
	}
	ncURLStatus := dncEP + "/networks/" + vnetID + "/networkcontainer/" + ncIDToFail + "/status" + apiVersion
	WaitForResourceState(t, ncURLStatus, "Failed", http.StatusBadRequest, 2)
	if err := deleteResource(t, ncURL, http.StatusOK); err != nil {
		t.Log(err)
	}
	fmt.Println("Cleaned up failed deployment")
	DeleteDeployments(t, deployedNetworkContainersInfo)
	fmt.Println("****ValidateSubnetExhaustion Succeed**********")
}

func TestValidateThrottlingOnGetNetwork(t *testing.T) {
	fmt.Println("****ValidateThrottlingOnGetNetwork**********")
	vnetsCount := 5
	networkDeploymentRequests := make([]*NetworkDeploymentRequest, 0)
	for index := 0; index < vnetsCount; index++ {
		id, _ := uuid.NewUUID()
		vnetID := id.String()
		networkDeploymentRequest := &NetworkDeploymentRequest{
			vnetID:              vnetID,
			UseCustomDnsServers: true,
			TagCount:            0,
			SubnetSize:          29,
			WaitForJoin:         true,
			WaitForGet:          true}
		networkDeploymentRequests = append(networkDeploymentRequests, networkDeploymentRequest)
	}

	setupNetwork(t, dncEP, apiVersion, networkDeploymentRequests)

	vnetIDToThrottle := networkDeploymentRequests[0].vnetID
	nmAgentResponderContext := generateDefaultNMAgentResponderContext(
		networkDeploymentRequests,
		false)
	fmt.Printf("Throttling get of vnet %s \n", vnetIDToThrottle)
	nmAgentServer.Lock.Lock()
	nmAgentResponderContext.NetworkResponseMap.getContext(
		vnetIDToThrottle).GetNetworkResponse.HTTPStatus = http.StatusForbidden
	nmAgentServer.ResponseFunc = NMAgentResponder
	nmAgentServer.Context = nmAgentResponderContext
	nmAgentServer.Lock.Unlock()
	startTime := time.Now().UTC()
	var refreshStatus api.RefreshMonitorStatus
	layout := "2006-01-02 15:04:05 -0700 MST"
	for {
		response, err := http.Get(dncEP + "/refreshmonitorstatus" + apiVersion)
		if err != nil {
			t.Fatal(err)
		}

		if response.StatusCode != http.StatusOK {
			t.Fatalf("Received Invalid status code %d, Expected:%d",
				response.StatusCode,
				http.StatusOK)
		}

		utils.DecodeResponse(response, &refreshStatus)
		response.Body.Close()
		receivedStartTime, _ := time.Parse(layout, refreshStatus.StartTime)
		if receivedStartTime.After(startTime) && refreshStatus.IsThrottled {
			break
		}
	}

	fmt.Printf("RefreshStatus after verifying throttling:%s\n", refreshStatus.String())
	fmt.Printf("Unthrottling get of vnet %s \n", vnetIDToThrottle)
	nmAgentServer.Lock.Lock()
	nmAgentResponderContext.NetworkResponseMap.getContext(
		vnetIDToThrottle).GetNetworkResponse.HTTPStatus = http.StatusOK
	nmAgentServer.ResponseFunc = NMAgentResponder
	nmAgentServer.Context = nmAgentResponderContext
	nmAgentServer.Lock.Unlock()
	for {
		response, err := http.Get(dncEP + "/refreshmonitorstatus" + apiVersion)
		if err != nil {
			t.Fatal(err)
		}

		if response.StatusCode != http.StatusOK {
			t.Fatalf("Received Invalid status code %d, Expected:%d",
				response.StatusCode,
				http.StatusOK)
		}

		utils.DecodeResponse(response, &refreshStatus)
		response.Body.Close()
		receivedStartTime, _ := time.Parse(layout, refreshStatus.StartTime)
		if receivedStartTime.After(startTime) && !refreshStatus.IsRefreshInProgress {
			break
		}
	}

	fmt.Printf("RefreshStatus after verifying unthrottling:%s\n", refreshStatus.String())

	fmt.Printf("Wait for all vnet gets to succeed \n")
	for {
		response, err := http.Get(dncEP + "/refreshmonitorstatus" + apiVersion)
		if err != nil {
			t.Fatal(err)
		}

		if response.StatusCode != http.StatusOK {
			t.Fatalf("Received Invalid status code %d, Expected:%d",
				response.StatusCode,
				http.StatusOK)
		}

		utils.DecodeResponse(response, &refreshStatus)
		response.Body.Close()
		receivedStartTime, _ := time.Parse(layout, refreshStatus.StartTime)
		if receivedStartTime.After(startTime) && !refreshStatus.IsRefreshInProgress && refreshStatus.FailedItems == 0 {
			break
		}

		time.Sleep(2 * time.Second)
	}

	fmt.Printf("RefreshStatus after all vnet processing succeeds:%s\n", refreshStatus.String())

	for _, networkDeploymentRequest := range networkDeploymentRequests {
		networkURL := dncEP + "/networks/" + networkDeploymentRequest.vnetID + apiVersion
		deleteResource(t, networkURL, http.StatusOK)
	}
	fmt.Println("****ValidateThrottlingOnGetNetwork Succeed**********")
}

func TestValidateThrottlingOnJoin(t *testing.T) {
	fmt.Println("****ValidateThrottlingOnJoin**********")
	vnetsCount := 5
	networkDeploymentRequests := make([]*NetworkDeploymentRequest, 0)
	for index := 0; index < vnetsCount; index++ {
		id, _ := uuid.NewUUID()
		vnetID := id.String()
		networkDeploymentRequest := &NetworkDeploymentRequest{
			vnetID:              vnetID,
			UseCustomDnsServers: true,
			TagCount:            0,
			SubnetSize:          29,
			WaitForJoin:         true,
			WaitForGet:          true}
		networkDeploymentRequests = append(networkDeploymentRequests, networkDeploymentRequest)
	}

	setupNetwork(t, dncEP, apiVersion, networkDeploymentRequests)

	vnetIDToThrottle := networkDeploymentRequests[0].vnetID
	nmAgentResponderContext := generateDefaultNMAgentResponderContext(
		networkDeploymentRequests,
		false)
	fmt.Printf("Throttling join of vnet %s \n", vnetIDToThrottle)
	nmAgentServer.Lock.Lock()
	nmAgentResponderContext.NetworkResponseMap.getContext(
		vnetIDToThrottle).GetNetworkResponse.NetworkResponse.ResponseCode =
		strconv.Itoa(http.StatusProcessing)
	nmAgentResponderContext.NetworkResponseMap.getContext(
		vnetIDToThrottle).JoinResponse.HTTPStatus = http.StatusForbidden

	nmAgentServer.ResponseFunc = NMAgentResponder
	nmAgentServer.Context = nmAgentResponderContext
	nmAgentServer.Lock.Unlock()
	startTime := time.Now().UTC()
	var refreshStatus api.RefreshMonitorStatus
	layout := "2006-01-02 15:04:05 -0700 MST"
	for {
		response, err := http.Get(dncEP + "/refreshmonitorstatus" + apiVersion)
		if err != nil {
			t.Fatal(err)
		}

		if response.StatusCode != http.StatusOK {
			t.Fatalf("Received Invalid status code %d, Expected:%d",
				response.StatusCode,
				http.StatusOK)
		}

		utils.DecodeResponse(response, &refreshStatus)
		response.Body.Close()
		receivedStartTime, _ := time.Parse(layout, refreshStatus.StartTime)
		if receivedStartTime.After(startTime) && refreshStatus.IsThrottled {
			break
		}

		time.Sleep(2 * time.Second)
	}

	fmt.Printf("RefreshStatus after verifying throttling:%s\n", refreshStatus.String())
	fmt.Printf("Unthrottling join of vnet %s \n", vnetIDToThrottle)
	nmAgentServer.Lock.Lock()
	nmAgentResponderContext.NetworkResponseMap.getContext(
		vnetIDToThrottle).GetNetworkResponse.NetworkResponse.ResponseCode =
		strconv.Itoa(http.StatusOK)
	nmAgentResponderContext.NetworkResponseMap.getContext(
		vnetIDToThrottle).JoinResponse.HTTPStatus = http.StatusForbidden
	nmAgentServer.ResponseFunc = NMAgentResponder
	nmAgentServer.Context = nmAgentResponderContext
	nmAgentServer.Lock.Unlock()
	for {
		response, err := http.Get(dncEP + "/refreshmonitorstatus" + apiVersion)
		if err != nil {
			t.Fatal(err)
		}

		if response.StatusCode != http.StatusOK {
			t.Fatalf("Received Invalid status code %d, Expected:%d",
				response.StatusCode,
				http.StatusOK)
		}

		utils.DecodeResponse(response, &refreshStatus)
		response.Body.Close()
		receivedStartTime, _ := time.Parse(layout, refreshStatus.StartTime)
		if receivedStartTime.After(startTime) && !refreshStatus.IsRefreshInProgress {
			break
		}

		time.Sleep(2 * time.Second)
	}

	fmt.Printf("RefreshStatus after verifying unthrottling:%s\n", refreshStatus.String())
	fmt.Printf("Wait for all vnet gets to succeed \n")
	for {
		response, err := http.Get(dncEP + "/refreshmonitorstatus" + apiVersion)
		if err != nil {
			t.Fatal(err)
		}

		if response.StatusCode != http.StatusOK {
			t.Fatalf("Received Invalid status code %d, Expected:%d",
				response.StatusCode,
				http.StatusOK)
		}

		utils.DecodeResponse(response, &refreshStatus)
		response.Body.Close()
		receivedStartTime, _ := time.Parse(layout, refreshStatus.StartTime)
		if receivedStartTime.After(startTime) && !refreshStatus.IsRefreshInProgress && refreshStatus.FailedItems == 0 {
			break
		}

		time.Sleep(2 * time.Second)
	}

	fmt.Printf("RefreshStatus after all vnet processing succeeds:%s\n", refreshStatus.String())
	for _, networkDeploymentRequest := range networkDeploymentRequests {
		networkURL := dncEP + "/networks/" + networkDeploymentRequest.vnetID + apiVersion
		deleteResource(t, networkURL, http.StatusOK)
	}
	fmt.Println("****ValidateThrottlingOnJoin Succeed**********")
}

func TestValidateBatching(t *testing.T) {
	fmt.Println("****ValidateBatching**********")
	layout := "2006-01-02 15:04:05 -0700 MST"
	configpath := "config_test.json"
	config := configuration.Load(configpath)
	vnetsCount := config.DNCSettings.NetworkRefreshBatchSize
	backoffInterval := config.DNCSettings.SleepIntervalBetweenNetworkBatches
	sleepBetweenNetworks := config.DNCSettings.SleepIntervalBetweenNetworkRefresh
	fmt.Printf("BackoffInterval:%v\n", backoffInterval)
	fmt.Printf("SleepBetweenNetworks:%v\n", sleepBetweenNetworks)
	fmt.Printf("BatchVnetCount:%v\n", vnetsCount)
	networkDeploymentRequests := make([]*NetworkDeploymentRequest, 0)
	for index := 0; index < vnetsCount; index++ {
		id, _ := uuid.NewUUID()
		vnetID := id.String()
		networkDeploymentRequest := &NetworkDeploymentRequest{
			vnetID:              vnetID,
			UseCustomDnsServers: true,
			TagCount:            0,
			SubnetSize:          29,
			WaitForJoin:         true,
			WaitForGet:          true}
		networkDeploymentRequests = append(networkDeploymentRequests, networkDeploymentRequest)
	}

	setupNetwork(t, dncEP, apiVersion, networkDeploymentRequests)
	startTime := time.Now().UTC()
	var refreshStatus api.RefreshMonitorStatus
	for {
		response, err := http.Get(dncEP + "/refreshmonitorstatus" + apiVersion)
		if err != nil {
			t.Fatal(err)
		}

		if response.StatusCode != http.StatusOK {
			t.Fatalf("Received Invalid status code %d, Expected:%d",
				response.StatusCode,
				http.StatusOK)
		}

		utils.DecodeResponse(response, &refreshStatus)
		response.Body.Close()
		receivedStartTime, _ := time.Parse(layout, refreshStatus.StartTime)
		if receivedStartTime.After(startTime) &&
			!refreshStatus.IsRefreshInProgress &&
			refreshStatus.FailedItems == 0 &&
			len(refreshStatus.TimeToRefresh) != 0 {
			break
		}

		time.Sleep(2 * time.Second)
	}

	fmt.Printf("RefreshStatus after verifying throttling:%s\n", refreshStatus.String())
	fmt.Printf("Elapsed Time %v for refreshing Vnets:%v\n", refreshStatus.TimeToRefresh, len(networkDeploymentRequests))
	for _, networkDeploymentRequest := range networkDeploymentRequests {
		networkDeploymentRequest.WaitForJoin = false
	}

	elapsedDuration, _ := time.ParseDuration(refreshStatus.TimeToRefresh)
	timeElapsedBetweenNetworkRefresh := (sleepBetweenNetworks * (len(networkDeploymentRequests) - 1)) // -1 since we do single backoff
	lowerLimit := time.Duration(backoffInterval+timeElapsedBetweenNetworkRefresh) * time.Millisecond
	if elapsedDuration > lowerLimit {
		t.Fatalf("Time to refresh Vnets %d expected to be less than backoff time, Elapsed time:%v Expected:[0, %v]",
			len(networkDeploymentRequests),
			elapsedDuration,
			lowerLimit)
	}

	fmt.Printf("Add one more Vnet which should result in throttling\n")
	id, _ := uuid.NewUUID()
	vnetID := id.String()
	networkDeploymentRequest := &NetworkDeploymentRequest{
		vnetID:              vnetID,
		UseCustomDnsServers: true,
		TagCount:            0,
		SubnetSize:          29}
	networkDeploymentRequests = append(networkDeploymentRequests, networkDeploymentRequest)
	setupNetwork(t, dncEP, apiVersion, networkDeploymentRequests)

	fmt.Printf("Validate that calls are backed off\n")
	startTime = time.Now().UTC()
	for {
		response, err := http.Get(dncEP + "/refreshmonitorstatus" + apiVersion)
		if err != nil {
			t.Fatal(err)
		}

		if response.StatusCode != http.StatusOK {
			t.Fatalf("Received Invalid status code %d, Expected:%d",
				response.StatusCode,
				http.StatusOK)
		}

		utils.DecodeResponse(response, &refreshStatus)
		response.Body.Close()

		receivedStartTime, _ := time.Parse(layout, refreshStatus.StartTime)
		if receivedStartTime.After(startTime) &&
			!refreshStatus.IsRefreshInProgress &&
			refreshStatus.FailedItems == 0 &&
			len(refreshStatus.TimeToRefresh) != 0 {
			break
		}

		time.Sleep(2 * time.Second)
	}

	fmt.Printf("RefreshStatus after all vnet processing succeeds:%s\n", refreshStatus.String())
	fmt.Printf("Elapsed Time %v for refreshing Vnets:%v\n", refreshStatus.TimeToRefresh, len(networkDeploymentRequests))
	elapsedDuration, _ = time.ParseDuration(refreshStatus.TimeToRefresh)
	timeElapsedBetweenNetworkRefresh = (sleepBetweenNetworks * (len(networkDeploymentRequests) - 1))
	lowerLimit = time.Duration(backoffInterval+timeElapsedBetweenNetworkRefresh) * time.Millisecond
	upperLimit := time.Duration((backoffInterval*2)+timeElapsedBetweenNetworkRefresh) * time.Millisecond
	if elapsedDuration < lowerLimit ||
		elapsedDuration > upperLimit {
		t.Fatalf("Time to refresh Vnets %d as there should be a single backoff, Elapsed time:%v Expected:[%v, %v]",
			len(networkDeploymentRequests),
			elapsedDuration,
			lowerLimit,
			upperLimit)
	}

	fmt.Printf("Add %d more Vnet which should result in double throttling\n", vnetsCount+2)
	for _, networkDeploymentRequest := range networkDeploymentRequests {
		networkDeploymentRequest.WaitForJoin = false
	}

	for index := 0; index < vnetsCount+1; index++ {
		id, _ := uuid.NewUUID()
		vnetID := id.String()
		networkDeploymentRequest := &NetworkDeploymentRequest{
			vnetID:              vnetID,
			UseCustomDnsServers: true,
			TagCount:            0,
			SubnetSize:          29,
			WaitForJoin:         true,
			WaitForGet:          true}
		networkDeploymentRequests = append(networkDeploymentRequests, networkDeploymentRequest)
	}

	setupNetwork(t, dncEP, apiVersion, networkDeploymentRequests)

	fmt.Printf("Validate that calls are backed off\n")
	startTime = time.Now().UTC()
	for {
		response, err := http.Get(dncEP + "/refreshmonitorstatus" + apiVersion)
		if err != nil {
			t.Fatal(err)
		}

		if response.StatusCode != http.StatusOK {
			t.Fatalf("Received Invalid status code %d, Expected:%d",
				response.StatusCode,
				http.StatusOK)
		}

		utils.DecodeResponse(response, &refreshStatus)
		response.Body.Close()

		receivedStartTime, _ := time.Parse(layout, refreshStatus.StartTime)
		if receivedStartTime.After(startTime) &&
			!refreshStatus.IsRefreshInProgress &&
			refreshStatus.FailedItems == 0 &&
			len(refreshStatus.TimeToRefresh) != 0 {
			break
		}

		time.Sleep(2 * time.Second)
	}

	fmt.Printf("RefreshStatus after all vnet processing succeeds:%s\n", refreshStatus.String())
	fmt.Printf("Elapsed Time %v for refreshing Vnets:%v\n", refreshStatus.TimeToRefresh, len(networkDeploymentRequests))
	elapsedDuration, _ = time.ParseDuration(refreshStatus.TimeToRefresh)
	timeElapsedBetweenNetworkRefresh = (sleepBetweenNetworks * (len(networkDeploymentRequests) - 2)) // -2 is that we will do two back offs
	lowerLimit = time.Duration((backoffInterval*2)+timeElapsedBetweenNetworkRefresh) * time.Millisecond
	upperLimit = time.Duration((backoffInterval*3)+timeElapsedBetweenNetworkRefresh) * time.Millisecond

	if elapsedDuration < lowerLimit ||
		elapsedDuration > upperLimit {
		t.Fatalf("Time to refresh Vnets %d as there should be a double backoff, Elapsed time:%v Expected:[%v, %v]",
			len(networkDeploymentRequests),
			elapsedDuration,
			lowerLimit,
			upperLimit)
	}

	for _, networkDeploymentRequest := range networkDeploymentRequests {
		networkURL := dncEP + "/networks/" + networkDeploymentRequest.vnetID + apiVersion
		deleteResource(t, networkURL, http.StatusOK)
	}
	fmt.Println("****ValidateBatching Succeed**********")
}

func TestRefreshNMAgentGoalState(t *testing.T) {
	fmt.Println("****RefreshNMAgentGoalState**********")
	deploymentCount := 1
	ncPerNetworkCount := 1
	ipIndex := 0
	deploymentInfos := make([]DeploymentInfo, deploymentCount)
	for index := 0; index < deploymentCount; index++ {
		id, _ := uuid.NewUUID()
		networkDeploymentRequest := NetworkDeploymentRequest{
			vnetID:              id.String(),
			UseCustomDnsServers: true,
			TagCount:            0,
			WaitForJoin:         true,
			WaitForGet:          true}
		ncDeploymentRequests := make([]*NCDeploymentRequest, ncPerNetworkCount)
		for ncIndex := 0; ncIndex < ncPerNetworkCount; ncIndex++ {
			ipToReserve := "10.0.0." + strconv.FormatUint(uint64(4+ipIndex), 10)
			ipIndex++

			ncID, _ := uuid.NewUUID()
			ncDeploymentRequest := NCDeploymentRequest{
				AllocationRequest: api.AllocationRequest{
					SubnetName:   "subnet1",
					IPConstraint: ipToReserve},
				AssociationInfo: api.AssociationInfo{NodeID: nodeID, InterfaceID: "127.0.0.1", ContainerType: cns.AzureContainerInstance, OrchestratorContext: orchContext},
				NCID:            ncID.String()}

			ncDeploymentRequests[ncIndex] = &ncDeploymentRequest
		}

		deploymentInfos[index] = DeploymentInfo{NetworkDeploymentRequest: &networkDeploymentRequest, NCDeploymentRequests: ncDeploymentRequests}
	}

	deployedNetworkContainersInfo := DeployNetworkContainers(t, dncEP, apiVersion, deploymentInfos, false, defaultNumCPUCores)

	var body bytes.Buffer
	refreshURL := dncEP + "/refreshnmagentgoalstate" + apiVersion
	json.NewEncoder(&body).Encode("")
	http.Post(refreshURL, "application/json", &body)

	// TODO: Validate that the refresh calls are received by NMAgent server
	time.Sleep(100 * time.Second)

	DeleteDeployments(t, deployedNetworkContainersInfo)
	fmt.Println("****RefreshNMAgentGoalState Succeed**********")
}

func TestNodeNicLimit(t *testing.T) {
	fmt.Println("****ValidateNodeNicLimit**********")
	ncDeploymentRequests := []*NCDeploymentRequest{&NCDeploymentRequest{
		NCID: ncID}}

	fmt.Println("Started CNS Server", cnsServer.Server.URL)
	cnsServer.SetupTest()

	cnsResponderContextMap := generateDefaultCnsResponderContextMap(t, ncDeploymentRequests)

	contextWithCPUCores := new(CnsResponderContext)
	CnsNumOfCPUCoresResp := &cns.NumOfCPUCoresResponse{
		Response: cns.Response{
			ReturnCode: 0,
			Message:    "Successfully retrieved CPU cores"},
		NumOfCPUCores: 1}
	contextWithCPUCores.NumOfCoresResponse = &CnsNumOfCPUCoresResponse{
		HTTPStatus:         http.StatusOK,
		NumOfCoresResponse: CnsNumOfCPUCoresResp}

	cnsResponderContextMap["usecustomcpucorresponse"] = contextWithCPUCores
	cnsServer.Lock.Lock()
	cnsServer.ResponseFunc = CNSResponder
	cnsServer.Context = cnsResponderContextMap
	cnsServer.Lock.Unlock()

	id, _ := uuid.NewUUID()
	vnetID := id.String()
	setupNetworkAndJoinSubnet(
		t, dncEP, vnetID, networkContainerReq.AllocationRequest.SubnetName, apiVersion, true, 0)
	fmt.Printf("Created node %s with info %+v\n", nodeID, nodeReq)
	addOrUpdateNode(t, dncEP, nodeID, apiVersion, &nodeReq)
	fmt.Printf("Successfully created node [%s]\n", nodeID)

	var (
		body    bytes.Buffer
		respErr api.JsonErr
	)

	json.NewEncoder(&body).Encode(networkContainerReq)

	ncURL := dncEP + "/networks/" + vnetID + "/networkcontainer/" + ncID + apiVersion
	response, err := http.Post(ncURL, "application/json", &body)
	if err != nil {
		t.Fatalf("CreateNetworkContainer was failed with error: %+v\n", err)
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		if err = utils.DecodeResponse(response, &respErr); err != nil || respErr.Code != 0 {
			t.Fatalf("CreateNetworkContainer POST request failed with response %+v Err:%+v", respErr, err)
		}
	}

	ncURLStatus := dncEP + "/networks/" + vnetID + "/networkcontainer/" + ncID + "/status" + apiVersion
	reqStatus := WaitForResourceState(t, ncURLStatus, "Failed", http.StatusBadRequest, 2)

	if !reqStatus {
		t.Fatalf("CreateNetworkContainer request didn't reach the expected status (Failed) in 2 seconds")
	}

	// Clean up
	// delete NC
	deleteResource(t, ncURL, http.StatusOK)
	if !WaitForResourceCleanup(t, ncURL, 4) {
		t.Fatalf("Failed to cleanup network container %s\n", ncID)
	}

	networkURL := dncEP + "/networks/" + vnetID + apiVersion
	deleteResource(t, networkURL, http.StatusOK)

	nodeURL := dncEP + "/nodes/" + nodeID + apiVersion
	deleteResource(t, nodeURL, http.StatusOK)
	fmt.Println("****ValidateNodeNicLimit Succeed**********")
}

func TestNodeNicLimitReservationDelete(t *testing.T) {
	fmt.Println("****TestNodeNicLimitReservationDelete**********")
	ncDeploymentRequests := []*NCDeploymentRequest{&NCDeploymentRequest{
		NCID: ncID}}

	fmt.Println("Started CNS Server", cnsServer.Server.URL)
	cnsServer.SetupTest()

	cnsResponderContextMap := generateDefaultCnsResponderContextMap(t, ncDeploymentRequests)

	contextWithCPUCores := new(CnsResponderContext)
	CnsNumOfCPUCoresResp := &cns.NumOfCPUCoresResponse{
		Response: cns.Response{
			ReturnCode: 0,
			Message:    "Successfully retrieved CPU cores"},
		NumOfCPUCores: 1}
	contextWithCPUCores.NumOfCoresResponse = &CnsNumOfCPUCoresResponse{
		HTTPStatus:         http.StatusOK,
		NumOfCoresResponse: CnsNumOfCPUCoresResp}

	cnsResponderContextMap["usecustomcpucorresponse"] = contextWithCPUCores
	cnsServer.Lock.Lock()
	cnsServer.ResponseFunc = CNSResponder
	cnsServer.Context = cnsResponderContextMap
	cnsServer.Lock.Unlock()

	id, _ := uuid.NewUUID()
	vnetID := id.String()
	setupNetworkAndJoinSubnet(
		t, dncEP, vnetID, networkContainerReq.AllocationRequest.SubnetName, apiVersion, true, 0)
	fmt.Printf("Created node %s with info %+v\n", nodeID, nodeReq)
	addOrUpdateNode(t, dncEP, nodeID, apiVersion, &nodeReq)
	fmt.Printf("Successfully created node [%s]\n", nodeID)

	var (
		body    bytes.Buffer
		respErr api.JsonErr
	)

	json.NewEncoder(&body).Encode(networkContainerReq)

	ncURL := dncEP + "/networks/" + vnetID + "/networkcontainer/" + ncID + apiVersion
	response, err := http.Post(ncURL, "application/json", &body)
	if err != nil {
		t.Fatalf("CreateNetworkContainer was failed with error: %+v\n", err)
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		if err = utils.DecodeResponse(response, &respErr); err != nil || respErr.Code != 0 {
			t.Fatalf("CreateNetworkContainer POST request failed with response %+v Err:%+v", respErr, err)
		}
	}

	ncURLStatus := dncEP + "/networks/" + vnetID + "/networkcontainer/" + ncID + "/status" + apiVersion
	reqStatus := WaitForResourceState(t, ncURLStatus, "Failed", http.StatusBadRequest, 2)

	if !reqStatus {
		t.Fatalf("CreateNetworkContainer request didn't reach the expected status (Failed) in 2 seconds")
	}

	ncURLConfig := dncEP + "/networks/" + vnetID + "/networkcontainer/" + ncID + "/configuration" + apiVersion
	configResponse, err := http.Get(ncURLConfig)
	if err != nil {
		t.Fatalf("Get NetworkContainerConfguration was failed with error: %+v\n", err)
	}

	defer configResponse.Body.Close()

	var ncConfig datamodel.NCConfiguration
	decodeRespErr := utils.DecodeResponse(configResponse, &ncConfig)
	if decodeRespErr != nil {
		t.Fatalf("Failed to decode NetworkContainerConfgurationresponse for: %+v\n", decodeRespErr)
	}

	reservationIDStr := ncConfig.ReservationID
	reservationStatusURL := dncEP + "/networks/" + vnetID + "/reservations/" + reservationIDStr + "/status" + apiVersion

	// Clean up
	// delete NC
	deleteResource(t, ncURL, http.StatusOK)
	if !WaitForResourceCleanup(t, ncURL, 4) {
		t.Fatalf("Failed to cleanup network container %s\n", ncID)
	}

	// verify reservation doesn't exist, after NC has been deleted.
	response, err = http.Get(reservationStatusURL)
	if err != nil {
		t.Fatal(err)
	}

	// StatusNotFound gets thrown if the reservation doesn't exist.
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("Received Invalid status code %d, Expected:%d",
			response.StatusCode,
			http.StatusNotFound)
	}

	networkURL := dncEP + "/networks/" + vnetID + apiVersion
	deleteResource(t, networkURL, http.StatusOK)

	nodeURL := dncEP + "/nodes/" + nodeID + apiVersion
	deleteResource(t, nodeURL, http.StatusOK)
	fmt.Println("****TestNodeNicLimitReservationDelete Succeed**********")
}

func TestPublishingRequestMarshallingLogicForSuccess(t *testing.T) {
	fmt.Println("****TestPublishingRequestMarshallingLogicForSuccess**********")
	var ncRequest nmagentcontracts.CreateOrUpdateNetworkContainerRequest
	ncRequest.IPV4Addresses = make([]string, 1)
	ncRequest.IPV4Addresses[0] = "10.0.0.0"
	ncRequest.Policies = make([]string, 0)
	ncRequest.SubnetName = "test"
	ncRequest.Version = strconv.FormatUint(5, 10)
	ncRequest.VirtualNetworkID = "048622d9-49c2-4cad-94ff-7112e6f2ab5a"
	ncRequest.VlanID = 1
	ncRequest.GreKey = 257

	var body bytes.Buffer
	err := json.NewEncoder(&body).Encode(ncRequest)

	if err != nil {
		t.Fatalf("Marshalling was not supposed to fail, observerd error [%s]", err.Error())
	}

	fmt.Printf("Encoded string [%s]\n", body.String())
	fmt.Println("****TestPublishingRequestMarshallingLogicForSuccess Succeed**********")
}

func TestPublishingRequestMarshallingLogicForFailure(t *testing.T) {
	fmt.Println("****TestPublishingRequestMarshallingLogicForFailure**********")
	var ncRequest nmagentcontracts.CreateOrUpdateNetworkContainerRequest
	ncRequest.IPV4Addresses = make([]string, 1)
	ncRequest.IPV4Addresses[0] = "" // Empty IPAddress
	ncRequest.Policies = make([]string, 0)
	ncRequest.SubnetName = "test"
	ncRequest.Version = strconv.FormatUint(5, 10)
	ncRequest.VirtualNetworkID = "048622d9-49c2-4cad-94ff-7112e6f2ab5a"
	ncRequest.VlanID = 1

	var body bytes.Buffer
	err := json.NewEncoder(&body).Encode(ncRequest)

	if err == nil {
		t.Fatalf("Marshalling was supposed to fail")
	}
	fmt.Println("****TestPublishingRequestMarshallingLogicForFailure Succeed**********")
}

func TestManagedDncAPIs(t *testing.T) {
	fmt.Println("****TestManagedDncAPIs**********")
	processExitCallback()

	// start dnc in managed mode
	go StartDnc(true)
	time.Sleep(2 * time.Second)

	fmt.Println("Started CNS Server", cnsServer.Server.URL)
	cnsServer.SetupTest()
	cnsServer.Lock.Lock()
	cnsServer.ResponseFunc = CNSResponder
	cnsServer.Lock.Unlock()

	var (
		mdncEP       = "http://127.0.0.1:9002"
		dnctl        = cli.NewDncClient(mdncEP)
		iface        = "127.0.0.1"
		infraVnet    = "048622d9-49c2-4cad-94ff-7112e6f2ab5a"
		ipConstraint = "10.0.0.4"
		id, _        = uuid.NewUUID()
		vnetID       = id.String()
		body         bytes.Buffer
	)

	// Register network/subnet and add node
	setupNetworkAndJoinSubnet(t, mdncEP, vnetID, networkContainerReq.AllocationRequest.SubnetName, apiVersion, true, 0)
	if respErr := dnctl.AddNode(nodeID, iface, cns.Kubernetes, infraVnet); respErr.HTTPStatus != http.StatusOK {
		t.Fatalf("Register node failed with error %+v", respErr)
	}

	// Test RegisterNode API
	nodeRegistrationRequest := api.NodeRegistrationRequest{
		NumCores:             2,
		NmAgentSupportedApis: []string{},
	}
	json.NewEncoder(&body).Encode(nodeRegistrationRequest)

	registerNodeURL := fmt.Sprintf("%s/networks/%s/node/%s%s", mdncEP, infraVnet, nodeID, apiVersion)
	response, err := http.Post(registerNodeURL, "application/json", &body)
	if err == nil {
		if response.StatusCode == http.StatusOK {
			var req cns.SetOrchestratorTypeRequest
			expectedReq := cns.SetOrchestratorTypeRequest{OrchestratorType: cns.Kubernetes, DncPartitionKey: "", NodeID: nodeID}
			json.NewDecoder(response.Body).Decode(&req)
			if !reflect.DeepEqual(req, expectedReq) {
				t.Fatalf("RegisterNode API is broken: expected %+v\treceived %+v", expectedReq, req)
			}
		} else {
			t.Fatalf("Failed to register node with http status code %s", strconv.Itoa(response.StatusCode))
		}

		response.Body.Close()
	} else {
		t.Fatalf("Failed to register node with err: %+v", err)
	}

	// test SyncNodeNetworkContainers API
	if respErr := dnctl.AddNc(nodeID, vnetID, ncID, "", networkContainerReq.AllocationRequest.SubnetName, ipConstraint, "", iface, networkContainerReq.AssociationInfo.ContainerType, podInfo.PodName, podInfo.PodNamespace, false, false, 0); respErr.HTTPStatus != http.StatusOK {
		t.Fatalf("Create nc failed with error %+v", respErr)
	}

	time.Sleep(4 * time.Second)

	programmedNC, respErr := dnctl.GetNC(vnetID, ncID)
	if respErr.HTTPStatus != http.StatusOK || programmedNC.NCVersion != 0 {
		t.Fatalf("Get nc failed with error %+v or NC version is not equal to expected. Expected version is 0 but got %d", respErr, programmedNC.NCVersion)
	}

	syncNodeStatusURL := fmt.Sprintf("%s/networks/%s/node/%s/networkcontainers%s", mdncEP, infraVnet, nodeID, apiVersion)
	response, err = http.Get(syncNodeStatusURL)
	if err == nil {
		if response.StatusCode == http.StatusOK {
			var (
				req         cns.NodeInfoResponse
				expectedReq = cns.NodeInfoResponse{
					NetworkContainers: []cns.CreateNetworkContainerRequest{{
						Version:                    "0",
						NetworkContainerType:       networkContainerReq.AssociationInfo.ContainerType,
						NetworkContainerid:         ncID,
						PrimaryInterfaceIdentifier: iface,
						AuthorizationToken:         vnetID,
						LocalIPConfiguration: cns.IPConfiguration{
							IPSubnet: cns.IPSubnet{
								IPAddress:    "169.254.128.4",
								PrefixLength: 17},
							GatewayIPAddress: "169.254.128.1"},
						OrchestratorContext: orchContext,
						IPConfiguration: cns.IPConfiguration{
							DNSServers:       programmedNC.IPConfiguration.DNSServers,
							GatewayIPAddress: programmedNC.IPConfiguration.GatewayIP,
							IPSubnet: cns.IPSubnet{
								IPAddress:    ipConstraint,
								PrefixLength: 24}},
						MultiTenancyInfo:           cns.MultiTenancyInfo{EncapType: cns.Vlan, ID: 1},
						CnetAddressSpace:           make([]cns.IPSubnet, 0),
						AllowHostToNCCommunication: false,
						AllowNCToHostCommunication: false},
					}}
			)

			json.NewDecoder(response.Body).Decode(&req)
			if !reflect.DeepEqual(req, expectedReq) {
				t.Fatalf("SyncNodeNetworkContainers API is broken: expected %+v\treceived %+v", expectedReq, req)
			}
		}

		response.Body.Close()
	}
	fmt.Println("****TestManagedDncAPIs Succeed**********")

	// TODO:Move this test into a separate test case.
	// Currently separation got blocked by TestCreateMultipleNetworks timeout. Needs test case refactor when move it out.
	fmt.Println("****TestNCUpdate**********")
	context, _ := json.Marshal(podInfo)
	associationInfo := api.AssociationInfo{
		NodeID:              nodeID,
		InterfaceID:         iface,
		ContainerType:       networkContainerReq.AssociationInfo.ContainerType,
		OrchestratorContext: context}
	// Update secondary IP count from 0 to 1 to trigger NC update.
	allocationReq := api.AllocationRequest{
		SubnetName:       networkContainerReq.AllocationRequest.SubnetName,
		IPConstraint:     ipConstraint,
		NodeConstraint:   "",
		SecondaryIPCount: 1}
	ncReq := api.NetworkContainerRequestV2{
		AllocationRequest:          allocationReq,
		AssociationInfo:            associationInfo,
		AllowHostToNCCommunication: false,
		AllowNCToHostCommunication: false}
	json.NewEncoder(&body).Encode(ncReq)

	// Test Update NC
	if respErr = dnctl.UpsertNcV2(vnetID, ncID, ncReq); respErr.HTTPStatus != http.StatusOK {
		t.Fatalf("Update nc failed with error %+v", respErr)
	}

	// Update NC will result NC version changed from 0 to 1
	if receivedNcResp, respErr := dnctl.GetNC(vnetID, ncID); respErr.HTTPStatus != http.StatusOK || receivedNcResp.NCVersion != 1 {
		t.Fatalf("Update NC failed with error %v or NC version is not equal to expected. Expected version is 1 but got %d", respErr, receivedNcResp.NCVersion)
	}
	fmt.Println("****TestNCUpdate Succeed**********")

	// todo: add following checks once mDNC cns interface delete is implemented:

	// ncURL := mdncEP + "/networks/" + vnetID + "/networkcontainer/" + ncID + apiVersion
	// networkURL := mdncEP + "/networks/" + vnetID + apiVersion
	// nodeURL := mdncEP + "/nodes/" + nodeID + apiVersion

	// deleteResource(t, ncURL, http.StatusOK)
	// deleteResource(t, networkURL, http.StatusOK)
	// deleteResource(t, nodeURL, http.StatusOK)
}

// Induce error in the processing
// This function induces error with provided percent probability
func induceError(percentProbability int) bool {
	induceError := false
	// Create a random num generator with a new seed every time to
	// avoid deterministic number generation
	randNumGen := rand.New(rand.NewSource(time.Now().UnixNano()))

	if randNumGen.Intn(100) < percentProbability {
		induceError = true
	}

	return induceError
}
