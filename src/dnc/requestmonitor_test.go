package main

import (
	"bytes"
	"crypto/sha256"
	"dnc/api"
	"dnc/configuration"
	"dnc/controller"
	"dnc/datamodel"
	"dnc/logger/loggershim"
	"dnc/masterelection/masterelectionshim"
	"dnc/nmagentcontracts"
	"dnc/replica"
	"dnc/server"
	"dnc/utils"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
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

type CreateNetworkRequest struct {
	NetworkType string
}

type JoinSubnetRequest struct {
	AuthenticationToken string
}

type CreateNetworkContainerRequest struct {
	AllocationRequest api.AllocationRequest
	AssociationInfo   api.AssociationInfo
}

type CreateNetworkContainerRequestV2 struct {
	ReservationID     string
	AllocationRequest api.AllocationRequest
	AssociationInfo   api.AssociationInfo
}

type OrchestratorContext struct {
	PodName      string
	PodNamespace string
}

// Unit Test Values
var (
	dncEP = "http://127.0.0.1:9000"

	nodeReq = api.NodeInfoRequest{
		IPAddresses:      []string{"127.0.0.1"},
		LocationType:     "",
		OrchestratorType: cns.Kubernetes}
	nodeID = "testnode"

	networkReq = CreateNetworkRequest{"AzureNet"}
	subnetReq  = JoinSubnetRequest{"0bb4132b-5932-4ec4-a47f-1b24f0eeb426"}
	subnetName = "subnet1"

	reservationReq = api.AllocationRequest{
		SubnetName:   "subnet1",
		IPConstraint: "10.0.0.15"}
	reservationReq2 = api.AllocationRequest{
		NodeConstraint: "testnode",
		SubnetName:     "subnet1"}
	reservationID = "testreservation"

	podInfo = OrchestratorContext{
		PodName:      "testpodname",
		PodNamespace: "testpodnamespace"}
	context, _          = json.Marshal(podInfo)
	networkContainerReq = CreateNetworkContainerRequest{
		AllocationRequest: api.AllocationRequest{
			SubnetName:     "subnet1",
			IPConstraint:   "10.0.0.15",
			NodeConstraint: ""},
		AssociationInfo: api.AssociationInfo{
			NodeID:              "testnode",
			InterfaceID:         "127.0.0.1",
			ContainerType:       cns.AzureContainerInstance,
			OrchestratorContext: context}}
	networkContainerReq2 = CreateNetworkContainerRequest{
		AllocationRequest: api.AllocationRequest{
			SubnetName:     "subnet1",
			IPConstraint:   "",
			NodeConstraint: ""},
		AssociationInfo: api.AssociationInfo{
			NodeID:              "testnode",
			InterfaceID:         "127.0.0.1",
			ContainerType:       cns.AzureContainerInstance,
			OrchestratorContext: context}}
	networkContainerReq3 = CreateNetworkContainerRequestV2{
		ReservationID: "testreservation",
		AllocationRequest: api.AllocationRequest{
			SubnetName:     "subnet1",
			IPConstraint:   "",
			NodeConstraint: ""},
		AssociationInfo: api.AssociationInfo{
			NodeID:              "testnode",
			InterfaceID:         "127.0.0.1",
			ContainerType:       cns.AzureContainerInstance,
			OrchestratorContext: context}}
	ncID  = "0e2f40cd-8fe1-420f-af43-1c4d5c5bb83b"
	ncID2 = "b8be30e0-7b79-42cb-bdb1-8c0ecd77d52a"

	apiVersion = "?api-version=2018-03-01"

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
	go StartDnc()
	time.Sleep(1 * time.Second)
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 1024
	http.DefaultTransport.(*http.Transport).TLSHandshakeTimeout = 0 * time.Second
	nmAgentServer = new(MockServer)
	nmAgentServer.SetupServer("127.0.0.1:9001")
	cnsServer = new(MockServer)
	cnsServer.SetupServer("127.0.0.1:10090")
	return nil
}

func StartDnc() {
	configpath, found := os.LookupEnv("DNC_CONFIGURATION_PATH")
	if !found {
		configpath = "config.json"
	}

	config = configuration.Load(configpath)
	setDncConfigDefaults()
	logger = loggershim.NewLoggerShim(config)
	logger.Init(stderr, reports)
	replicaInstance = replica.NewReplica(
		config.ServerSettings,
		GetNetworkRefreshMonitorStatus)
	sessionManager = utils.NewSessionManager([]string{datamodel.NodeRepositoryName, datamodel.NetworkRepositoryName, datamodel.TagRepositoryName})
	masterElectionShim = masterelectionshim.NewMasterElectionShim(
		config,
		replicaInstance.UpdateReplicaStatus,
		replicaInstance.SetPrimaryIPEndpoint,
		startChan,
		doneChan,
		demoteChan)
	ctrler = controller.NewController(replicaInstance, sessionManager, config.CNSSettings.ConnectionStrings.SetOrchestratorTypeURLFmt, masterElectionShim, config)
	dncServer = server.Server{
		ServerSettings: config.ServerSettings,
		Controller:     ctrler}
	dncServer.Init()
	requestContextRetryInterval = time.Duration(config.DNCSettings.RequestRetryBaseInterval) * time.Millisecond
	maxRequestBackoffInterval = time.Duration(config.DNCSettings.MaxRequestBackoffInterval) * time.Millisecond
	sleepIntervalBetweenNetworkBatches = time.Duration(config.DNCSettings.SleepIntervalBetweenNetworkBatches) * time.Millisecond
	sleepIntervalBetweenNetworkRefresh = time.Duration(config.DNCSettings.SleepIntervalBetweenNetworkRefresh) * time.Millisecond
	reportToHostInterval = time.Duration(config.DNCSettings.ReportToHostInterval) * time.Millisecond

	if config.DataStoreSettings != (configuration.DataStoreSettings{}) {
		masterElectionShim.Init()
		datamodel.Init(config.DataStoreSettings, config.DNCSettings.RequestRetryBaseInterval, config.VlanSettings.Min, config.VlanSettings.Max)
		ctrler.UpdateReplicaStatus(true)
	}

	go startReplica(ctrler)
	dncServer.Start()
}

func TestVlanGeneration(t *testing.T) {
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
}

func TestSkipNodeCleanupForNetworkContainer(t *testing.T) {
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

	cnetSpace := nmAgentResponderContext.NetworkResponseMap.getContext(vnetID).GetNetworkResponse.NetworkResponse.CNetSpace
	if nmAgentResponderContext.TagResponse != nil {
		for _, tagResponse := range nmAgentResponderContext.TagResponse {
			cnetSpace = cnetSpace + "," + strings.Join(tagResponse.Response.Values, ",")
		}
	}

	fmt.Println("CnetSpace;", cnetSpace)
	fmt.Println("Started CNS Server", cnsServer.Server.URL)
	cnsServer.SetupTest()
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
	ncDeploymentRequests2 := []*NCDeploymentRequest{&NCDeploymentRequest{
		NCID: ncID}}
	cnsResponderContextMap = generateDefaultCnsResponderContextMap(t, ncDeploymentRequests2)
	ncRequestValidatorFunc := func(t *testing.T, req cns.CreateNetworkContainerRequest) bool {
		podInfo := OrchestratorContext{
			PodName:      "testpodname",
			PodNamespace: "testpodnamespace"}
		context, _ := json.Marshal(podInfo)
		expectedReq := cns.CreateNetworkContainerRequest{
			Version:                    "1",
			NetworkContainerType:       cns.AzureContainerInstance,
			NetworkContainerid:         "Swift_" + ncID,
			PrimaryInterfaceIdentifier: networkContainerReq.AssociationInfo.InterfaceID,
			IPConfiguration:            ipConfig,
			AuthorizationToken:         "",
			CnetAddressSpace:           convertAddressStringsToCnsIPSubnets(cnetSpace),
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
	cnsServer.ResponseFunc = CNSResponder
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
	<-cnsResponderContextMap.getContext(ncID).CreateOrUpdateNCResponse.SignalInfo.Signal
	WaitForResourceState(t, ncURLStatus, "Completed", http.StatusOK, 2)

	networkURL := dncEP + "/networks/" + vnetID + apiVersion
	nodeURL := dncEP + "/nodes/" + nodeID + apiVersion
	getNCCleanupStateURL := dncEP + "/iswaitingfornodecleanup/networks/" + vnetID + "/networkcontainer/" + ncID + apiVersion
	promotionURL := dncEP + "/skipnodecleanupfornetworkcontainer/networks/" + vnetID + "/networkcontainer/" + ncID + apiVersion

	// stop CNS here and promote NC state after termination fails
	cnsServer.TearDownSuite()
	deleteResource(t, ncURL, http.StatusOK)
	if WaitForResourceCleanup(t, ncURL, 4) {
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
	var req api.NetworkContainerResponse
	for ; req.RequestStatus.FailureDetail.Code != api.DisAssociationFailed; time.Sleep(1 * time.Second) {
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
}

func TestNode(t *testing.T) {
	cnsServer.SetupTest()
	ncDeploymentRequests := make([]*NCDeploymentRequest, 0)
	cnsResponderContextMap := generateDefaultCnsResponderContextMap(t, ncDeploymentRequests)
	cnsServer.Lock.Lock()
	cnsServer.ResponseFunc = CNSResponder
	cnsServer.Context = cnsResponderContextMap
	cnsServer.Lock.Unlock()

	fmt.Printf("Created node %s with info %+v\n", nodeID, nodeReq)
	addOrUpdateNode(t, dncEP, nodeID, apiVersion, &nodeReq)
	fmt.Printf("Successfully created node [%s]\n", nodeID)

	fmt.Printf("Validate persisted node info [%s]\n", nodeID)
	validateNodeInfoResponse(t, dncEP, nodeID, apiVersion, &nodeReq)
	fmt.Println("Validate idempotency")
	addOrUpdateNode(t, dncEP, nodeID, apiVersion, &nodeReq)
	fmt.Printf("Validate persisted node info [%s]\n", nodeID)
	validateNodeInfoResponse(t, dncEP, nodeID, apiVersion, &nodeReq)

	updatedNodeInfoReq := api.NodeInfoRequest{
		IPAddresses: []string{"127.0.0.1", "10.1.1.2"}, OrchestratorType: cns.Kubernetes}
	fmt.Printf("Update node %s with info %+v\n", nodeID, updatedNodeInfoReq)
	addOrUpdateNode(t, dncEP, nodeID, apiVersion, &updatedNodeInfoReq)
	fmt.Printf("Successfully updated node [%s]\n", nodeID)
	fmt.Printf("Validate persisted node info [%s]\n", nodeID)
	validateNodeInfoResponse(t, dncEP, nodeID, apiVersion, &updatedNodeInfoReq)

	fmt.Printf("Deleting node %s\n", nodeID)
	nodeURL := dncEP + "/nodes/" + nodeID + apiVersion

	deleteResource(t, nodeURL, http.StatusOK)
	fmt.Printf("Node deleted successfully %s\n", nodeID)

	fmt.Println("Validate delete node idempotency")
	deleteResource(t, nodeURL, http.StatusOK)

	fmt.Println("Validate that appropriate error is thrown when querying for node that does not exist")
	response, err := http.Get(dncEP + "/nodes/" + nodeID + apiVersion)
	if err != nil {
		t.Fatal(err)
	}

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("Received Invalid status code %d, Expected:%d",
			response.StatusCode,
			http.StatusNotFound)
	}

	defer response.Body.Close()

	var respErr api.JsonErr
	utils.DecodeResponse(response, &respErr)

	if respErr.Code != api.NodeNotFound {
		t.Fatalf("Invalid error code %d, Expected:%d",
			respErr.Code,
			api.NodeNotFound)
	}
}

func TestNetwork(t *testing.T) {
	var body bytes.Buffer
	var respErr api.JsonErr
	json.NewEncoder(&body).Encode(networkReq)
	fmt.Println("Started NMAgent Server", nmAgentServer.Server.URL)
	nmAgentServer.SetupTest()
	id, _ := uuid.NewUUID()
	vnetID := id.String()

	networkDeploymentRequests := []*NetworkDeploymentRequest{
		&NetworkDeploymentRequest{vnetID: vnetID, UseCustomDnsServers: true, TagCount: 0, WaitForJoin: true, WaitForGet: true}}

	nmAgentResponderContext := generateDefaultNMAgentResponderContext(
		networkDeploymentRequests,
		false)
	nmAgentServer.Lock.Lock()
	nmAgentResponderContext.NetworkResponseMap.getContext(vnetID).GetNetworkResponse.NetworkResponse.ResponseCode = strconv.Itoa(http.StatusProcessing)
	nmAgentServer.ResponseFunc = NMAgentResponder
	nmAgentServer.Context = nmAgentResponderContext
	nmAgentServer.Lock.Unlock()

	response, err := http.Post(dncEP+"/networks/"+vnetID+apiVersion, "application/json", &body)
	if err != nil {
		t.Fatalf("CreateNetwork failed with error: %+v\n", err)
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err = utils.DecodeResponse(response, &respErr)
		if err != nil || respErr.Code != 0 {
			t.Fatalf("CreateNetwork failed with response %+v Err:%+v", response, err)
		}
	}

	fmt.Printf("Waiting for Join signal for vnet %s\n", vnetID)
	<-nmAgentResponderContext.NetworkResponseMap.getContext(vnetID).JoinResponse.Signal
	fmt.Printf("Received Join Network request signal for vnet %s\n", vnetID)

	fmt.Printf("Waiting for get network signal for vnet %s\n", vnetID)
	<-nmAgentResponderContext.NetworkResponseMap.getContext(vnetID).GetNetworkResponse.SignalInfo.Signal
	fmt.Printf("Received Get Network request signal for context:%+v\n",
		nmAgentResponderContext.NetworkResponseMap.getContext(vnetID).GetNetworkResponse)
	resourceStatusURL := dncEP + "/networks/" + vnetID + "/status" + apiVersion
	WaitForResourceState(t, resourceStatusURL, "Pending", http.StatusProcessing, 2)
	fmt.Printf("Setting the get virtual network response error code to completed from pending state\n")
	nmAgentServer.Lock.Lock()
	nmAgentResponderContext.NetworkResponseMap.getContext(vnetID).GetNetworkResponse.NetworkResponse.ResponseCode = strconv.Itoa(http.StatusOK)
	nmAgentResponderContext.NetworkResponseMap.getContext(vnetID).GetNetworkResponse.SignalInfo.CurrentSignalCount = 0
	nmAgentServer.Lock.Unlock()

	fmt.Println("Waiting for get network signal")
	<-nmAgentResponderContext.NetworkResponseMap.getContext(vnetID).GetNetworkResponse.SignalInfo.Signal
	fmt.Printf("Received Get Network request signal for context:%+v\n", nmAgentResponderContext.NetworkResponseMap.getContext(vnetID).GetNetworkResponse)
	WaitForResourceState(t, resourceStatusURL, "Completed", http.StatusOK, 2)
	fmt.Printf("Successfully created network [%s]\n", vnetID)
	networkURL := dncEP + "/networks/" + vnetID + apiVersion
	deleteResource(t, networkURL, http.StatusOK)

	// Test create network with authentication failure
	var body2 bytes.Buffer
	var respErr2 api.JsonErr
	json.NewEncoder(&body2).Encode(networkReq)
	networkDeploymentRequests2 := []*NetworkDeploymentRequest{
		&NetworkDeploymentRequest{vnetID: vnetID, UseCustomDnsServers: true, TagCount: 0, WaitForJoin: true, WaitForGet: true}}
	nmAgentResponderContext2 := generateDefaultNMAgentResponderContext(
		networkDeploymentRequests2,
		false)
	nmAgentServer.Lock.Lock()
	nmAgentResponderContext2.NetworkResponseMap.getContext(vnetID).JoinResponse.HTTPStatus = http.StatusUnauthorized
	nmAgentServer.ResponseFunc = NMAgentResponder
	nmAgentServer.Context = nmAgentResponderContext2
	nmAgentServer.Lock.Unlock()
	response2, err := http.Post(networkURL, "application/json", &body2)
	if err != nil {
		t.Fatalf("CreateNetwork failed with error: %+v\n", err)
	}

	defer response2.Body.Close()

	if response2.StatusCode != http.StatusOK {
		err = utils.DecodeResponse(response2, &respErr2)
		if err != nil || respErr2.Code != 0 {
			t.Fatalf("CreateNetwork failed with response %+v Err:%+v", response2, err)
		}
	}

	fmt.Println("Waiting for Join signal")
	<-nmAgentResponderContext2.NetworkResponseMap.getContext(vnetID).JoinResponse.Signal
	fmt.Printf("Received Join Network request signal")

	WaitForResourceState(t, resourceStatusURL, "Failed", http.StatusUnauthorized, 2)
	fmt.Printf("Setting the get virtual network response error code to completed from failed state\n")
	deleteResource(t, networkURL, http.StatusOK)
}

func TestSubnet(t *testing.T) {
	// join subnet with NO network
	var body bytes.Buffer
	var responseErr api.JsonErr
	json.NewEncoder(&body).Encode(subnetReq)
	id, _ := uuid.NewUUID()
	vnetID := id.String()

	response, err := http.Post(dncEP+"/networks/"+vnetID+"/subnets/"+subnetName+apiVersion, "application/json", &body)
	if err != nil {
		t.Fatalf("JoinSubnet failed with error: %+v\n", err)
	}

	defer response.Body.Close()

	if response.StatusCode == http.StatusOK {
		t.Fatalf("JoinSubnet expected to fail")
	}

	err = utils.DecodeResponse(response, &responseErr)
	fmt.Printf("JoinSubnet failed with response %+v Err:%+v\n", responseErr, err)
	if err != nil && !strings.Contains(err.Error(), "Request failed with HTTP error 404") {
		t.Fatalf("JoinSubnet decode failed with response %+v Err:%+v", responseErr, err)
	}

	if responseErr.Code != api.NetworkNotFound {
		t.Fatalf("JoinSubnet expected to fail with error code %s, Received response %s", responseErr.Code.String(), api.NetworkNotFound.String())
	}

	// join subnet with network
	cnsServer.SetupTest()
	ncDeploymentRequests := make([]*NCDeploymentRequest, 0)
	cnsResponderContextMap := generateDefaultCnsResponderContextMap(t, ncDeploymentRequests)
	cnsServer.Lock.Lock()
	cnsServer.ResponseFunc = CNSResponder
	cnsServer.Context = cnsResponderContextMap
	cnsServer.Lock.Unlock()
	resourceURL := dncEP + "/networks/" + vnetID + "/subnets/" + subnetName + apiVersion
	addOrUpdateNode(t, dncEP, nodeID, apiVersion, &nodeReq)
	networkDeploymentRequests := []*NetworkDeploymentRequest{
		&NetworkDeploymentRequest{vnetID: vnetID, UseCustomDnsServers: true, TagCount: 0, WaitForJoin: true, WaitForGet: true}}

	setupNetwork(t, dncEP, apiVersion, networkDeploymentRequests)
	json.NewEncoder(&body).Encode(subnetReq)
	fmt.Printf("Joining subnet %s, network %s. AccessToken:%s\n", vnetID, subnetName, subnetReq.AuthenticationToken)
	response, err = http.Post(resourceURL, "application/json", &body)
	if err != nil {
		t.Fatalf("JoinSubnet failed with error: %+v\n", err)
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("JoinSubnet failed with status code %d\n", response.StatusCode)
	}

	subnetConfiguration := getSubnetConfiguration(t, dncEP, vnetID, subnetName, apiVersion)
	if subnetConfiguration.AccessToken != fmt.Sprintf("%x", sha256.Sum256([]byte(subnetReq.AuthenticationToken))) {
		t.Fatalf("AccessToken mismatch. Received:%s , Expected:%s\n",
			subnetConfiguration.AccessToken,
			subnetReq.AuthenticationToken)
	}

	subnetReq2 := api.SubnetRequest{
		AuthenticationToken: "Key2-" + vnetID}
	json.NewEncoder(&body).Encode(subnetReq2)
	resourceStatusURL := dncEP + "/networks/" + vnetID + "/subnets/" + subnetName + "/status" + apiVersion

	WaitForResourceState(t, resourceStatusURL, "Completed", http.StatusOK, 2)

	fmt.Printf("Updating subnet %s, network %s with accessToken:%s\n", vnetID, subnetName, subnetReq2.AuthenticationToken)
	response, err = http.Post(resourceURL, "application/json", &body)
	if err != nil {
		t.Fatalf("JoinSubnet update failed with error: %+v\n", err)
	}

	response.Body.Close()

	fmt.Printf("Deleting network %s\n", vnetID)
	networkURL := dncEP + "/networks/" + vnetID + apiVersion
	deleteResource(t, networkURL, http.StatusOK)
}

func TestReservation(t *testing.T) {
	id, _ := uuid.NewUUID()
	vnetID := id.String()
	setupNetworkAndJoinSubnet(t, dncEP, vnetID, reservationReq.SubnetName, apiVersion, true, 0)
	reservationURL := dncEP + "/networks/" + vnetID + "/reservations/" + reservationID + apiVersion
	reservationStatusURL := dncEP + "/networks/" + vnetID + "/reservations/" + reservationID + "/status" + apiVersion
	createReservation(t, reservationReq, dncEP, vnetID, reservationID, apiVersion)
	networkURL := dncEP + "/networks/" + vnetID + apiVersion
	WaitForResourceState(t, reservationStatusURL, "Completed", http.StatusOK, 4)
	deleteErrResp := deleteResource(t, networkURL, http.StatusBadRequest)
	if deleteErrResp.Code != api.NetworkInUse {
		t.Fatalf("Delete network expected to fail with error %s, Error response %s \n", api.NetworkInUse.String(), deleteErrResp.String())
	}

	deleteResource(t, reservationURL, http.StatusOK)
	deleteResource(t, networkURL, http.StatusOK)

	// test create reservation with node constraint
	cnsServer.SetupTest()
	ncDeploymentRequests := make([]*NCDeploymentRequest, 0)
	cnsResponderContextMap := generateDefaultCnsResponderContextMap(t, ncDeploymentRequests)
	cnsServer.Lock.Lock()
	cnsServer.ResponseFunc = CNSResponder
	cnsServer.Context = cnsResponderContextMap
	cnsServer.Lock.Unlock()

	setupNetworkAndJoinSubnet(t, dncEP, vnetID, reservationReq2.SubnetName, apiVersion, true, 0)
	addOrUpdateNode(t, dncEP, nodeID, apiVersion, &nodeReq)
	createReservation(t, reservationReq2, dncEP, vnetID, reservationID, apiVersion)
	WaitForResourceState(t, reservationStatusURL, "Completed", http.StatusOK, 4)
	nodeURL := dncEP + "/nodes/" + nodeID + apiVersion
	fmt.Printf("Delete resource %s when allocations exist. Expected to fail\n", networkURL)
	deleteErrResp = deleteResource(t, networkURL, http.StatusBadRequest)
	if deleteErrResp.Code != api.NetworkInUse {
		t.Fatalf("Delete network expected to fail with error %s, Error response %s \n", api.NetworkInUse.String(), deleteErrResp.String())
	}

	fmt.Printf("Delete resource %s when reservations exist. Expected to fail\n", nodeURL)
	deleteErrResp = deleteResource(t, nodeURL, http.StatusBadRequest)
	if deleteErrResp.Code != api.NodeInUse {
		t.Fatalf("Delete node expected to fail with error %s, Error response %s \n", api.NodeInUse.String(), deleteErrResp.String())
	}

	deleteResource(t, reservationURL, http.StatusOK)
	deleteResource(t, networkURL, http.StatusOK)
	deleteResource(t, nodeURL, http.StatusOK)
}

func TestContainer(t *testing.T) {
	cnsServer.SetupTest()
	ncDeploymentRequests := make([]*NCDeploymentRequest, 0)
	cnsResponderContextMap := generateDefaultCnsResponderContextMap(t, ncDeploymentRequests)
	cnsServer.Lock.Lock()
	cnsServer.ResponseFunc = CNSResponder
	cnsServer.Context = cnsResponderContextMap
	cnsServer.Lock.Unlock()
	id, _ := uuid.NewUUID()
	vnetID := id.String()

	CreateNetworkContainer(t, nodeID, nodeReq, networkContainerReq, dncEP, vnetID, ncID, apiVersion)
	CreateNetworkContainerUsingReservation(t, nodeID, nodeReq, networkContainerReq3, dncEP, vnetID, ncID2, apiVersion)
	CreateNetworkContainerDeployedInTaggedSubnets(t, nodeID, nodeReq, networkContainerReq, dncEP, vnetID, ncID, apiVersion)
	CreateNetworkContainerAndUpdateCnetSpace(t, nodeID, nodeReq, networkContainerReq, dncEP, vnetID, ncID, apiVersion)
	CreateNetworkContainerAndAddCnetSpace(t, nodeID, nodeReq, networkContainerReq, dncEP, vnetID, ncID, apiVersion)
	CreateNetworkContainerAndRemoveCnetSpace(t, nodeID, nodeReq, networkContainerReq, dncEP, vnetID, ncID, apiVersion)
	CreateNetworkContainerAndAddNewDNSServer(t, nodeID, nodeReq, networkContainerReq, dncEP, vnetID, ncID, apiVersion)
	CreateNetworkContainerAndUpdateToUseAzureDNSServers(t, nodeID, nodeReq, networkContainerReq, dncEP, vnetID, ncID, apiVersion)
	CreateNetworkContainerAndUpdateToUseCustomDNSServers(t, nodeID, nodeReq, networkContainerReq, dncEP, vnetID, ncID, apiVersion)
	CreateNetworkContainerAndUpdateToUseTags(t, nodeID, nodeReq, networkContainerReq, dncEP, vnetID, ncID, apiVersion)
	CreateNetworkContainerDeployedInTaggedSubnetsAndRemoveTags(t, nodeID, nodeReq, networkContainerReq, dncEP, vnetID, ncID, apiVersion)
	CreateNetworkContainerDeployedInTaggedSubnetsAndAddNewTag(t, nodeID, nodeReq, networkContainerReq, dncEP, vnetID, ncID, apiVersion)
	CreateNetworkContainerDeployedInTaggedSubnetsAndRemoveTag(t, nodeID, nodeReq, networkContainerReq, dncEP, vnetID, ncID, apiVersion)
}

func TestCreateMultipleNetworks(t *testing.T) {
	deploymentCount := 2
	ncPerNetworkCount := 5
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
				AssociationInfo: api.AssociationInfo{NodeID: nodeID, InterfaceID: "127.0.0.1", ContainerType: cns.AzureContainerInstance, OrchestratorContext: context},
				NCID:            ncID.String()}

			ncDeploymentRequests[ncIndex] = &ncDeploymentRequest
		}

		deploymentInfos[index] = DeploymentInfo{NetworkDeploymentRequest: &networkDeploymentRequest, NCDeploymentRequests: ncDeploymentRequests}
	}

	DeployNetworkContainers(t, dncEP, apiVersion, deploymentInfos, true)
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
	fmt.Printf("NodeID:%s, NodeInfo%s", nodeID, nodeInfoReq.String())
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
	Signal             chan string
	SignalCount        int
	CurrentSignalCount int
}

type JoinResponse struct {
	HTTPStatus int
	SignalInfo
}

type ValidateNCRequest func(*testing.T, cns.CreateNetworkContainerRequest) bool

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
}

type CnsResponderContextMap map[string]*CnsResponderContext

type NMAgentNetworkResponseMap map[string]*NMAgentNetworkResponse

var cnsContextLock = &sync.RWMutex{}
var nmagentContextLock = &sync.RWMutex{}

func (cxtMap CnsResponderContextMap) getContext(id string) *CnsResponderContext {
	cnsContextLock.RLock()
	defer cnsContextLock.RUnlock()
	return cxtMap["Swift_"+id]
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
				Message:    "Successfully delelted NC"}}
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

		cnsRespoonderContextMap["Swift_"+ncDeploymentRequest.NCID] = context
	}
	return cnsRespoonderContextMap
}

var tagDefinitions = map[string]nmagentcontracts.TagResponse{
	"XStore":   nmagentcontracts.TagResponse{ResponseCode: "200", Name: "XStore", Values: []string{"168.10.10.0/24", "144.10.12.12/24"}},
	"AzureSql": nmagentcontracts.TagResponse{ResponseCode: "200", Name: "AzureSql", Values: []string{"132.10.10.0/24", "134.10.12.12/24"}}}

var updatedTagDefinitions = map[string]nmagentcontracts.TagResponse{
	"XStore":   nmagentcontracts.TagResponse{ResponseCode: "200", Name: "XStore", Values: []string{"167.10.10.0/24", "168.10.10.0/24", "144.10.12.12/24"}},
	"AzureSql": nmagentcontracts.TagResponse{ResponseCode: "200", Name: "AzureSql", Values: []string{"132.10.10.0/24", "133.10.10.0/24", "134.10.12.12/24"}}}

func getTags(tagCount int, isUpdate bool) []nmagentcontracts.Tag {
	tagsArray := make([]nmagentcontracts.Tag, tagCount)
	index := 0
	tagDefinitionsToUse := tagDefinitions
	if isUpdate {
		tagDefinitionsToUse = updatedTagDefinitions
	}
	for tagName := range tagDefinitionsToUse {
		if index < tagCount {
			tagsArray[index] = nmagentcontracts.Tag{Name: tagName, Type: "System"}
			index++
		} else {
			break
		}
	}

	return tagsArray
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
	tagList := make(map[string]nmagentcontracts.Tag)
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
			if networkDeploymentRequest.TagCount > 0 {
				if index == 0 {
					fmt.Printf("Generating tags for subnet%s\n", subnet)
					subnet.Tags = getTags(networkDeploymentRequest.TagCount, useUpdatedTags)
					for _, tag := range subnet.Tags {
						tagList[tag.Name] = tag
					}
					fmt.Printf("Generated tags for subnet%s\n", subnet)
				}
			}

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

	if len(tagList) > 0 {
		responderTagsResponse := make(map[string]*TagResponse)
		for _, tag := range tagList {
			tagResponse := tagDefinitions[tag.Name]
			responderTagsResponse[tag.Name] = &TagResponse{
				HTTPStatus: http.StatusOK,
				Response:   &nmagentcontracts.TagResponse{Name: tagResponse.Name, Values: tagResponse.Values, ResponseCode: tagResponse.ResponseCode},
				SignalInfo: SignalInfo{
					Signal:             make(chan string, 5),
					SignalCount:        1,
					CurrentSignalCount: 0}}
		}
		context.TagResponse = responderTagsResponse
	} else {
		context.TagResponse = nil
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
				deleteResponse := nmagentcontracts.DeleteNetworkContainerResponse{
					ResponseCode: "200"}
				json.NewEncoder(w).Encode(deleteResponse)
			} else {
				var ncReq nmagentcontracts.CreateOrUpdateNetworkContainerRequest
				json.NewDecoder(r.Body).Decode(&ncReq)
				putResponse := nmagentcontracts.CreateNetworkContainerResponse{ResponseCode: "200", Version: ncReq.Version}
				json.NewEncoder(w).Encode(putResponse)
				fmt.Printf("NC Publishing request:%+v, RequestURI:%+v\n", ncReq, r.RequestURI)
			}
		}
	} else if strings.Contains(r.RequestURI, "systemTags") {
		shouldSignal := true

		nmagentContextLock.RLock()
		for _, networkResponseContext := range nmagentResponderContext.NetworkResponseMap {
			if networkResponseContext.GetNetworkResponse != nil {
				if networkResponseContext.GetNetworkResponse.SignalInfo.CurrentSignalCount <
					networkResponseContext.GetNetworkResponse.SignalInfo.SignalCount {
					shouldSignal = false
				}
			}
		}
		nmagentContextLock.RUnlock()

		if r.Method != "GET" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if nmagentResponderContext.TagResponse == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		tagName := strings.Split(r.URL.Path, "/")[2]
		tagResponse, ok := nmagentResponderContext.TagResponse[tagName]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(tagResponse.HTTPStatus)
		json.NewEncoder(w).Encode(tagResponse.Response)
		if shouldSignal && signalTest(&tagResponse.SignalInfo, tagName) {
			fmt.Printf("Set signal for tag request: %v\n", tagResponse.Response)
		}
		fmt.Printf("Responding to tag request, Name:%s\n", tagName)

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

		if cnsResponderContext.CreateOrUpdateNCResponse.ValidateNCRequest != nil {
			if !cnsResponderContext.CreateOrUpdateNCResponse.ValidateNCRequest(cnsResponderContext.T, ncRequest) {
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

	if response.StatusCode != expectedStatusCode && response.StatusCode != http.StatusProcessing {
		t.Fatalf("Unexpected response code %d received when trying to get network status for resource URL:%s.",
			response.StatusCode,
			resourceStatusURL)
	}
	// fmt.Printf("Received Response:%+v\n", response)
	if response.StatusCode == http.StatusProcessing {
		return "Pending"
	}
	var operationStatus api.RequestStatus
	err = utils.DecodeResponse(response, &operationStatus)
	if err != nil &&
		(response.StatusCode != expectedStatusCode) {
		t.Fatalf("Failed to decode network response for resource URL:%s. Error:%+v",
			resourceStatusURL,
			err)
	}
	fmt.Printf("Resource URL:%s Status:%s\n", resourceStatusURL, operationStatus.String())

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
	networkDeploymentRequests := []*NetworkDeploymentRequest{
		&NetworkDeploymentRequest{vnetID: vnetID, UseCustomDnsServers: addDNSServers, TagCount: tagCount, WaitForJoin: true, WaitForGet: true}}

	setupNetwork(t, dncEP, apiVersion, networkDeploymentRequests)
	subnetReq := api.SubnetRequest{
		AuthenticationToken: "Key1-" + vnetID}
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

	if !reflect.DeepEqual(expected.MultiTenancyInfo, expected.MultiTenancyInfo) {
		fmt.Printf("Expected.MultiTenancyInfo %+v Received.MultiTenancyInfo:%+v",
			expected.MultiTenancyInfo, received.MultiTenancyInfo)
		return false
	}

	if len(expected.CnetAddressSpace) != len(received.CnetAddressSpace) {
		fmt.Printf("Expected.CnetAddressSpace %+v Received.CnetAddressSpace:%+v",
			expected.CnetAddressSpace, received.CnetAddressSpace)
		return false
	}

	receivedCnetAddressSpace := make([]string, len(expected.CnetAddressSpace))
	expectedCnetAddressSpace := make([]string, len(expected.CnetAddressSpace))

	for index, cnetAddressSpace := range expected.CnetAddressSpace {
		expectedCnetAddressSpace[index] =
			cnetAddressSpace.IPAddress + "/" +
				strconv.Itoa(int(cnetAddressSpace.PrefixLength))
	}

	for index, cnetAddressSpace := range received.CnetAddressSpace {
		receivedCnetAddressSpace[index] =
			cnetAddressSpace.IPAddress + "/" +
				strconv.Itoa(int(cnetAddressSpace.PrefixLength))
	}

	if !utils.AreSetsEqual(receivedCnetAddressSpace, receivedCnetAddressSpace) {
		fmt.Printf("Expected.CnetAddressSpace %+v Received.CnetAddressSpace:%+v",
			expected.CnetAddressSpace, received.CnetAddressSpace)
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
	tagCount int) {
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
	fmt.Println("Started CNS Server", cnsServer.Server.URL)
	cnsServer.SetupTest()
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
	ncRequestValidatorFunc := func(t *testing.T, req cns.CreateNetworkContainerRequest) bool {
		podInfo := OrchestratorContext{
			PodName:      "testpodname",
			PodNamespace: "testpodnamespace"}
		context, _ := json.Marshal(podInfo)
		expectedReq := cns.CreateNetworkContainerRequest{
			Version:                    "1",
			NetworkContainerType:       cns.AzureContainerInstance,
			NetworkContainerid:         "Swift_" + ncid,
			PrimaryInterfaceIdentifier: associationInfo.InterfaceID,
			IPConfiguration:            ipConfig,
			AuthorizationToken:         "",
			CnetAddressSpace:           convertAddressStringsToCnsIPSubnets(cnetSpace),
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

	cnsResponderContextMap.getContext(ncid).CreateOrUpdateNCResponse.ValidateNCRequest = ncRequestValidatorFunc
	cnsServer.Lock.Lock()
	cnsServer.ResponseFunc = CNSResponder
	cnsServer.Context = cnsResponderContextMap
	cnsServer.Lock.Unlock()

	var body bytes.Buffer
	var respErr api.JsonErr
	if len(reservationID) > 0 {
		networkcontainerreq := CreateNetworkContainerRequestV2{
			AllocationRequest: *allocationRequest,
			AssociationInfo:   *associationInfo,
			ReservationID:     reservationID}
		json.NewEncoder(&body).Encode(networkcontainerreq)
	} else {
		networkcontainerreq := CreateNetworkContainerRequest{
			AllocationRequest: *allocationRequest,
			AssociationInfo:   *associationInfo}
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

	reservationURL := dncEP + "/networks/" + vnetID + "/reservations/" + reservationID + apiVersion
	if len(reservationID) > 0 {
		fmt.Printf("Delete resource %s when reservation has NCassociated with it. Expected to fail\n", nodeURL)
		deleteErrResp = deleteResource(t, reservationURL, http.StatusBadRequest)
		if deleteErrResp.Code != api.ReservationInUse {
			t.Fatalf("Delete reservation expected to fail with error %s as there is an association, Error response %s \n", api.ReservationInUse.String(), deleteErrResp.String())
		}
	}

	deleteResource(t, ncURL, http.StatusOK)
	fmt.Printf("Waiting for delete NC Url:%s\n", ncURL)
	<-cnsResponderContextMap.getContext(ncid).DeleteNCResponse.SignalInfo.Signal

	if !WaitForResourceCleanup(t, ncURL, 4) {
		t.Fatalf("Failed to cleanup network container%s\n", ncid)
	}

	if len(reservationID) > 0 {
		fmt.Printf("Delete resource %s when reservation exist. Expected to fail\n", networkURL)
		deleteErrResp := deleteResource(t, networkURL, http.StatusBadRequest)
		if deleteErrResp.Code != api.NetworkInUse {
			t.Fatalf("Delete network expected to fail with error %s, Error response %s \n", api.NetworkInUse.String(), deleteErrResp.String())
		}

		deleteResource(t, reservationURL, http.StatusOK)
	}

	deleteResource(t, networkURL, http.StatusOK)
	deleteResource(t, nodeURL, http.StatusOK)
}

func CreateNetworkContainer(t *testing.T, nodeID string, nodeInfoReq api.NodeInfoRequest, networkcontainerreq CreateNetworkContainerRequest, dncEP, vnetID, ncid, apiVersion string) {
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
		0)
}

func CreateNetworkContainerUsingReservation(t *testing.T, nodeID string, nodeInfoReq api.NodeInfoRequest, networkcontainerreq CreateNetworkContainerRequestV2, dncEP, vnetID, ncid, apiVersion string) {
	fmt.Println("****CreateNetworkContainerUsingReservation Test**********")
	createNetworkContainer(
		t,
		nodeID,
		nodeInfoReq,
		&networkcontainerreq.AllocationRequest,
		&networkcontainerreq.AssociationInfo,
		networkcontainerreq.ReservationID,
		dncEP,
		vnetID,
		ncid,
		apiVersion,
		0)
}

func CreateNetworkContainerDeployedInTaggedSubnets(t *testing.T, nodeID string, nodeInfoReq api.NodeInfoRequest, networkcontainerreq CreateNetworkContainerRequest, dncEP, vnetID, ncid, apiVersion string) {
	fmt.Println("****CreateNetworkContainerDeployedInTaggedSubnets Test**********")
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
		2)
}

func convertAddressStringsToCnsIPSubnets(networksString string) []cns.IPSubnet {

	netAddresses := strings.Split(networksString, ",")
	cnsAddresses := make([]cns.IPSubnet, len(netAddresses))
	for index, netAddress := range netAddresses {
		ipAddressString, prefix, _ := utils.GetIPAddressStringAndPrefix(netAddress)

		cnsIPSubnet := cns.IPSubnet{IPAddress: ipAddressString, PrefixLength: uint8(prefix)}
		cnsAddresses[index] = cnsIPSubnet
	}

	return cnsAddresses
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
	cnsServer.SetupTest()
	cnsResponderContextMap := generateDefaultCnsResponderContextMap(t, ncDeploymentRequests)
	cnsServer.Lock.Lock()
	cnsServer.ResponseFunc = CNSResponder
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
		dnsServers = []string{"10.0.0.2", "10.0.0.3"}
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
	updateNCRequestValidatorFunc := func(t *testing.T, req cns.CreateNetworkContainerRequest) bool {
		podInfo := OrchestratorContext{
			PodName:      "testpodname",
			PodNamespace: "testpodnamespace"}
		context, _ := json.Marshal(podInfo)
		expectedReq := cns.CreateNetworkContainerRequest{
			Version:                    "2",
			NetworkContainerType:       cns.AzureContainerInstance,
			NetworkContainerid:         "Swift_" + ncid,
			PrimaryInterfaceIdentifier: networkcontainerreq.AssociationInfo.InterfaceID,
			IPConfiguration:            ipConfig,
			AuthorizationToken:         "",
			CnetAddressSpace:           convertAddressStringsToCnsIPSubnets(cnetSpace),
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
	cnsServer.ResponseFunc = CNSResponder
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

	if !WaitForResourceCleanup(t, ncURL, 2) {
		t.Fatalf("Failed to cleanup network container%s\n", ncid)
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

func CreateNetworkContainerAndUpdateToUseTags(t *testing.T, nodeID string, nodeInfoReq api.NodeInfoRequest, networkcontainerreq CreateNetworkContainerRequest, dncEP, vnetID, ncid, apiVersion string) {
	fmt.Println("****CreateNetworkContainerAndUpdateToUseTags Test**********")
	networkDeploymentRequests := []*NetworkDeploymentRequest{
		&NetworkDeploymentRequest{vnetID: vnetID, UseCustomDnsServers: true, TagCount: 2, WaitForJoin: true, WaitForGet: true}}

	nmAgentResponderContext := generateDefaultNMAgentResponderContext(
		networkDeploymentRequests,
		false)
	networkResponseContext := nmAgentResponderContext.NetworkResponseMap.getContext(vnetID)

	networkResponseContext.GetNetworkResponse.NetworkResponse.VnetVersion = "6"
	networkResponseContext.JoinResponse.HTTPStatus = http.StatusOK
	networkResponseContext.JoinResponse.SignalInfo.SignalCount = 0
	networkResponseContext.GetNetworkResponse.SignalInfo.SignalCount = 1
	networkResponseContext.GetNetworkResponse.SignalInfo.CurrentSignalCount = 0
	for _, tagResponse := range nmAgentResponderContext.TagResponse {
		tagResponse.SignalInfo.SignalCount = 1
		tagResponse.SignalInfo.CurrentSignalCount = 0
	}

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

func CreateNetworkContainerDeployedInTaggedSubnetsAndRemoveTags(t *testing.T, nodeID string, nodeInfoReq api.NodeInfoRequest, networkcontainerreq CreateNetworkContainerRequest, dncEP, vnetID, ncid, apiVersion string) {
	fmt.Println("****CreateNetworkContainerDeployedInTaggedSubnetsAndRemoveTags Test**********")
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
		true,
		2)
}

func CreateNetworkContainerDeployedInTaggedSubnetsAndAddNewTag(t *testing.T, nodeID string, nodeInfoReq api.NodeInfoRequest, networkcontainerreq CreateNetworkContainerRequest, dncEP, vnetID, ncid, apiVersion string) {
	fmt.Println("****CreateNetworkContainerDeployedInTaggedSubnetsAndAddNewTag Test**********")
	networkDeploymentRequests := []*NetworkDeploymentRequest{
		&NetworkDeploymentRequest{vnetID: vnetID, UseCustomDnsServers: true, TagCount: 2, WaitForJoin: true, WaitForGet: true}}

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
		true,
		1)
}

func CreateNetworkContainerDeployedInTaggedSubnetsAndRemoveTag(t *testing.T, nodeID string, nodeInfoReq api.NodeInfoRequest, networkcontainerreq CreateNetworkContainerRequest, dncEP, vnetID, ncid, apiVersion string) {
	fmt.Println("****CreateNetworkContainerDeployedInTaggedSubnetsAndRemoveTag Test**********")
	networkDeploymentRequests := []*NetworkDeploymentRequest{
		&NetworkDeploymentRequest{vnetID: vnetID, UseCustomDnsServers: true, TagCount: 1, WaitForJoin: true, WaitForGet: true}}

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
		true,
		2)
}

func TestCreateNetworkContainerDeployedInTaggedSubnetsAndUpdateTags(t *testing.T) {
	fmt.Println("****CreateNetworkContainerDeployedInTaggedSubnetsAndUpdateTags Test**********")
	id, _ := uuid.NewUUID()
	vnetID := id.String()

	cnsServer.SetupTest()
	ncDeploymentRequests := make([]*NCDeploymentRequest, 0)
	cnsResponderContextMap := generateDefaultCnsResponderContextMap(t, ncDeploymentRequests)
	cnsServer.Lock.Lock()
	cnsServer.ResponseFunc = CNSResponder
	cnsServer.Context = cnsResponderContextMap
	cnsServer.Lock.Unlock()

	networkDeploymentRequests := []*NetworkDeploymentRequest{
		&NetworkDeploymentRequest{vnetID: vnetID, UseCustomDnsServers: true, TagCount: 2, WaitForJoin: true, WaitForGet: true}}

	nmAgentResponderContext := generateDefaultNMAgentResponderContext(
		networkDeploymentRequests,
		true)
	networkResponseContext := nmAgentResponderContext.NetworkResponseMap.getContext(vnetID)

	networkResponseContext.GetNetworkResponse.NetworkResponse.VnetVersion = "6"
	networkResponseContext.JoinResponse.HTTPStatus = http.StatusOK
	networkResponseContext.JoinResponse.SignalInfo.SignalCount = 0
	networkResponseContext.GetNetworkResponse.SignalInfo.SignalCount = 1
	networkResponseContext.GetNetworkResponse.SignalInfo.CurrentSignalCount = 0

	responderTagsResponse := make(map[string]*TagResponse)
	index := 1
	for tagName := range updatedTagDefinitions {
		if index > networkDeploymentRequests[0].TagCount {
			break
		}
		tagResponse := updatedTagDefinitions[tagName]
		responderTagsResponse[tagName] = &TagResponse{
			HTTPStatus: http.StatusOK,
			Response:   &nmagentcontracts.TagResponse{Name: tagResponse.Name, Values: tagResponse.Values, ResponseCode: tagResponse.ResponseCode},
			SignalInfo: SignalInfo{
				Signal:             make(chan string),
				SignalCount:        1,
				CurrentSignalCount: 0}}
		index++
	}
	nmAgentResponderContext.TagResponse = responderTagsResponse
	createNetworkContainerAndUpdateNetworkConfig(
		t,
		nodeID,
		nodeReq,
		networkContainerReq,
		dncEP,
		vnetID,
		ncID,
		apiVersion,
		nmAgentResponderContext,
		true,
		2)
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

func DeployNetworkContainers(t *testing.T, dncEP, apiVersion string, deploymentInfos []DeploymentInfo, deleteDeployments bool) *DeployedNetworkContainersInfo {
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
			fmt.Printf("NC request:%+v", ncDeploymentRequest)
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
				AuthenticationToken: "Key1-" + vnetID}
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
	fmt.Printf("Total NCs:%d", totalNCs)
	for _, ncDeploymentRequestList := range ncDeploymentRequestsMapByNetwork {
		for _, ncDeploymentRequest := range ncDeploymentRequestList {
			ncDeploymentRequests[index] = ncDeploymentRequest
			index++
		}
	}

	fmt.Println("Started CNS Server", cnsServer.Server.URL)
	cnsServer.SetupTest()
	cnsResponderContextMap := generateDefaultCnsResponderContextMap(t, ncDeploymentRequests)
	cnsServer.Lock.Lock()
	cnsServer.ResponseFunc = CNSResponder
	cnsServer.Context = cnsResponderContextMap
	cnsServer.Lock.Unlock()

	for nodeID := range nodesMap {
		nodeInfoReq := api.NodeInfoRequest{IPAddresses: []string{"127.0.0.1"}, LocationType: "", OrchestratorType: cns.Kubernetes}
		addOrUpdateNode(t, dncEP, nodeID, apiVersion, &nodeInfoReq)
	}

	nodeLocalIPMap := make(map[string]string)

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
			networkContainerID := "Swift_" + ncDeploymentRequest.NCID
			podInfo := OrchestratorContext{
				PodName:      "testpodname",
				PodNamespace: "testpodnamespace"}
			context, _ := json.Marshal(podInfo)
			ncRequestValidatorFunc := func(t *testing.T, req cns.CreateNetworkContainerRequest) bool {
				expectedReq := cns.CreateNetworkContainerRequest{
					Version:                    "1",
					NetworkContainerType:       cns.AzureContainerInstance,
					NetworkContainerid:         networkContainerID,
					PrimaryInterfaceIdentifier: ncDeploymentRequest.AssociationInfo.InterfaceID,
					IPConfiguration:            ipConfig,
					AuthorizationToken:         accessToken,
					CnetAddressSpace:           convertAddressStringsToCnsIPSubnets(cnetSpace),
					OrchestratorContext:        context,
					MultiTenancyInfo:           req.MultiTenancyInfo}
				fmt.Printf("ReceivedResponse:%+v\n", req)
				fmt.Printf("ExpectedResponse:%+v\n", expectedReq)
				var expectedIP string
				if len(ncDeploymentRequest.ReservationID) > 0 {
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

				if req.LocalIPConfiguration.GatewayIPAddress != localIPGatewayString || req.LocalIPConfiguration.IPSubnet.PrefixLength != localIPSubnetPrefixLength {
					return false
				}

				if len(req.IPConfiguration.IPSubnet.IPAddress) == 0 {
					fmt.Printf("Allocated subnet in the request is empty")
					return false
				}

				if associatedncID, ok := nodeLocalIPMap[req.LocalIPConfiguration.IPSubnet.IPAddress]; ok {
					if associatedncID != networkContainerID {
						fmt.Printf("Local IP %s received as part of NC %s is already associated with %s",
							req.LocalIPConfiguration.IPSubnet.IPAddress,
							networkContainerID,
							associatedncID)
						return false
					}
				}

				nodeLocalIPMap[req.LocalIPConfiguration.IPSubnet.IPAddress] = networkContainerID

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
		}

	}

	cnsServer.Lock.Lock()
	cnsServer.ResponseFunc = CNSResponder
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
			fmt.Printf("Waiting for create  NC:%s\n", ncDeploymentRequest.NCID)
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
			fmt.Printf("Deleting NC:%s\n", ncDeploymentRequest)
			ncURL := dncEP + "/networks/" + vnetID + "/networkcontainer/" + ncDeploymentRequest.NCID + apiVersion
			deleteResource(t, ncURL, http.StatusOK)
			fmt.Printf("Waiting for delete NC Url:%s\n", ncURL)
			<-deployedNetworkContainersInfo.CNSResponderContextMap.getContext(ncDeploymentRequest.NCID).DeleteNCResponse.SignalInfo.Signal

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
	deploymentCount := 1
	ncPerNetworkCount := 3
	deploymentInfos := make([]DeploymentInfo, deploymentCount)
	id, _ := uuid.NewUUID()
	vnetID := id.String()
	for index := 0; index < deploymentCount; index++ {
		networkDeploymentRequest := NetworkDeploymentRequest{
			vnetID:              vnetID,
			UseCustomDnsServers: true,
			TagCount:            0,
			SubnetSize:          29,
			WaitForJoin:         true,
			WaitForGet:          true}
		ncDeploymentRequests := make([]*NCDeploymentRequest, ncPerNetworkCount)
		for ncIndex := 0; ncIndex < ncPerNetworkCount; ncIndex++ {
			ncID, _ := uuid.NewUUID()
			ncDeploymentRequest := NCDeploymentRequest{
				AllocationRequest: api.AllocationRequest{
					SubnetName: "subnet1"},
				AssociationInfo: api.AssociationInfo{NodeID: nodeID, InterfaceID: "127.0.0.1", ContainerType: cns.AzureContainerInstance, OrchestratorContext: context},
				NCID:            ncID.String()}

			ncDeploymentRequests[ncIndex] = &ncDeploymentRequest
		}

		deploymentInfos[index] = DeploymentInfo{NetworkDeploymentRequest: &networkDeploymentRequest, NCDeploymentRequests: ncDeploymentRequests}
	}

	deployedNetworkContainersInfo := DeployNetworkContainers(t, dncEP, apiVersion, deploymentInfos, false)
	id, _ = uuid.NewUUID()
	ncIDToFail := id.String()
	deploymentRequestToFail := CreateNetworkContainerRequest{
		AllocationRequest: api.AllocationRequest{
			SubnetName: "subnet1"},
		AssociationInfo: api.AssociationInfo{NodeID: nodeID, InterfaceID: "127.0.0.1", ContainerType: cns.AzureContainerInstance, OrchestratorContext: context}}
	var body bytes.Buffer
	json.NewEncoder(&body).Encode(deploymentRequestToFail)
	ncURL := dncEP + "/networks/" + vnetID + "/networkcontainer/" + ncIDToFail + apiVersion
	response, err := http.Post(ncURL, "application/json", &body)
	if err != nil {
		t.Fatal(err)
	}

	defer response.Body.Close()
	var respErr api.JsonErr
	if response.StatusCode != http.StatusOK {
		if err = utils.DecodeResponse(response, &respErr); err != nil || respErr.Code != 0 {
			t.Fatalf("CreateNetworkContainer call failed with response %+v Err:%+v", respErr, err)
		}
	}
	ncURLStatus := dncEP + "/networks/" + vnetID + "/networkcontainer/" + ncIDToFail + "/status" + apiVersion
	WaitForResourceState(t, ncURLStatus, "Failed", http.StatusBadRequest, 2)
	deleteResource(t, ncURL, http.StatusOK)
	fmt.Printf("Cleaned up failed deployment")
	DeleteDeployments(t, deployedNetworkContainersInfo)
}

func TestValidateThrottlingOnGetNetwork(t *testing.T) {
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
}

func TestValidateThrottlingOnJoin(t *testing.T) {
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
}

func TestValidateBatching(t *testing.T) {
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
}
