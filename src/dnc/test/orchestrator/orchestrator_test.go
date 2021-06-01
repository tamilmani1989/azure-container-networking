package test

import (
	"dnc/api"
	"dnc/client"
	"dnc/configuration"
	"dnc/test/utils"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-container-networking/cns"
	"github.com/google/uuid"
)

var (
	dnctl                   *client.DncClient
	defaultRequestTimeout   = time.Second * 4
	defaultNCRequestTimeout = time.Second * 10
	defaultSleep            = time.Second * 1
	errdRequestTimeout      = time.Second * 30
	devEnv                  bool
	secondaryIPCount        = 0
)

func init() {
	// determine if tests are being run on dev machine
	_, devEnv = os.LookupEnv("DNC_DEV")
	// use DNC config set with test storage settings
	configpath, found := os.LookupEnv("DNC_CONFIGURATION_PATH")
	if !found {
		configpath = "config.json"
	}

	config := configuration.Load(configpath)
	if config.ServerSettings.IPAddress == "" || config.ServerSettings.Port == "" {
		panic("DNC EP not set correctly")
	}

	dnctl = client.NewDncClient("http://" + config.ServerSettings.IPAddress + ":" + config.ServerSettings.Port)
}

func TestBasicNodeAPIs(t *testing.T) {
	var (
		nodeID           = utils.StandardNode
		ipaddr           = utils.LocalHost
		orchestratorType = cns.Kubernetes
	)

	getNodes(t, nil)
	addNode(t, nodeID, ipaddr, orchestratorType, "")
	addNode(t, nodeID, ipaddr, orchestratorType, "")
	getNodes(t, []string{nodeID})
	deleteNode(t, nodeID, http.StatusNotFound)
}

func TestBasicVnetAPIs(t *testing.T) {
	var (
		id, _  = uuid.NewUUID()
		vnetID = id.String()
	)

	addNetwork(t, vnetID, defaultRequestTimeout)
	addNetwork(t, vnetID, defaultRequestTimeout)
	getNetworks(t, []string{vnetID})
	deleteNetwork(t, vnetID, http.StatusNotFound)
}

func TestBasicSubnetAPIs(t *testing.T) {
	var (
		id, _               = uuid.NewUUID()
		vnetID              = id.String()
		id2, _              = uuid.NewUUID()
		authenticationToken = id2.String()
		subnetID            = utils.Subnet1
	)

	addNetwork(t, vnetID, defaultRequestTimeout)
	addSubnet(t, vnetID, subnetID, authenticationToken)
	addSubnet(t, vnetID, subnetID, authenticationToken)
	deleteNetwork(t, vnetID, http.StatusNotFound)
}

func TestBasicReservationAPIs(t *testing.T) {
	var (
		id, _               = uuid.NewUUID()
		reservationID       = id.String()
		id2, _              = uuid.NewUUID()
		authenticationToken = id2.String()
		nodeID              = utils.StandardNode
		vnetID              = utils.StandardVnet
		subnetID            = utils.Subnet1
	)

	addNode(t, nodeID, utils.LocalHost, cns.Kubernetes, "")
	addNetwork(t, vnetID, defaultRequestTimeout)
	addSubnet(t, vnetID, subnetID, authenticationToken)
	addReservation(t, nodeID, vnetID, subnetID, "", reservationID, "", defaultRequestTimeout)
	// validate that delete network will fail when a reservation exists
	deleteNetwork(t, vnetID, http.StatusBadRequest)
	deleteReservation(t, vnetID, reservationID)
	deleteNetwork(t, vnetID, http.StatusNotFound)
	deleteNode(t, nodeID, http.StatusNotFound)
}

func TestBasicNCAPIs(t *testing.T) {
	var (
		nodeID              = utils.StandardNode
		ipaddr              = utils.LocalHost
		nodeIface           = utils.NodeIface
		orchestratorType    = cns.Kubernetes
		containerType       = cns.AzureContainerInstance
		id, _               = uuid.NewUUID()
		vnetID              = id.String()
		id2, _              = uuid.NewUUID()
		authenticationToken = id2.String()
		id3, _              = uuid.NewUUID()
		ncID                = id3.String()
		id4, _              = uuid.NewUUID()
		ncID2               = id4.String()
		id5, _              = uuid.NewUUID()
		reservationID       = id5.String()
		subnetID            = utils.Subnet1
		reservationForLBIP  = "39d810c2-669a-11eb-ae93-0242ac130002"
		podName             = "container1"
		podNamespace        = "default"
		context             = []byte(`{"PodName":"` + podName + `","PodNamespace":"` + podNamespace + `"}`)
		secondaryIPCount    = 0
		receivedNcResp      *api.NetworkContainerResponse
		expectedNcResp      = &api.NetworkContainerResponse{
			NcID: ncID,
			IPConfiguration: api.IPConfiguration{
				NetworkID:    vnetID,
				SubnetName:   subnetID,
				IPSubnet:     utils.Subnet1AddrSpace,
				IPAddress:    "10.0.1.4/32",
				GatewayIP:    utils.Subnet1DefaultGW,
				DNSServers:   []string{"8.8.8.8", "9.9.9.9"},
				SecondaryIPs: make(map[string]api.IPInfo),
			},
			AssociationInfo: api.AssociationInfo{
				NodeID:              nodeID,
				InterfaceID:         nodeIface,
				ContainerType:       containerType,
				OrchestratorContext: context},
			RequestStatus: api.RequestStatus{
				State:  "Program",
				Status: "Completed"},
			VlanID:                     1,
			LocalIP:                    "169.254.128.4",
			NCVersion:                  0,
			AllowHostToNCCommunication: false,
			AllowNCToHostCommunication: false}
		respErr *api.JsonErr
	)

	addNode(t, nodeID, ipaddr, orchestratorType, "")
	addNetwork(t, vnetID, defaultRequestTimeout)
	addSubnet(t, vnetID, subnetID, authenticationToken)
	addNc(t, nodeID, vnetID, ncID, "", subnetID, "", "", nodeIface, containerType, podName, podNamespace, false, false, defaultNCRequestTimeout, secondaryIPCount)
	addNc(t, nodeID, vnetID, ncID, "", subnetID, "", "", nodeIface, containerType, podName, podNamespace, false, false, defaultNCRequestTimeout, secondaryIPCount)
	// Adding another nc with the same orchestrator context on the same node should fail (v2)
	// using GenericFailureNC here but this call will never reach NMAgent and is expected to fail in networkmanager.go
	if respErr := dnctl.AddNc(nodeID, vnetID, utils.GenericFailureNC, "", subnetID, "", "", nodeIface, cns.AzureContainerInstance, podName, podNamespace, "", false, false, secondaryIPCount); respErr == nil {
		t.Fatalf("Failed to block duplicate orchestrator contexts with different NC IDs")
	}
	// create nc with reservation
	addReservation(t, nodeID, vnetID, subnetID, "", reservationID, "", defaultRequestTimeout)
	addNc(t, nodeID, vnetID, ncID2, reservationID, subnetID, "", nodeID, nodeIface, containerType, podName+"2", podNamespace, false, false, defaultNCRequestTimeout, secondaryIPCount)
	// Adding another nc with the same orchestrator context on the same node should fail (v1)
	// Using the same reservationID over here but this is expected to fail before we persist nc configuration
	if respErr := dnctl.AddNc(nodeID, vnetID, utils.GenericFailureNC, reservationID, subnetID, "", nodeID, nodeIface, cns.AzureContainerInstance, podName+"2", podNamespace, "", false, false, secondaryIPCount); respErr == nil {
		t.Fatalf("Failed to block duplicate orchestrator contexts with different NC IDs")
	}

	if receivedNcResp, respErr = dnctl.GetNC(vnetID, ncID); respErr.HTTPStatus != http.StatusOK {
		t.Fatalf("Failed to retrieve nc response with error %+v", respErr)
	}

	// todo: refactor way of checking NC response
	if receivedNcResp != nil {
		receivedNcResp.StateTransitionLog = nil
		receivedNcResp.CreatedDateTime = time.Time{}
	}

	if !reflect.DeepEqual(receivedNcResp, expectedNcResp) {
		expectedNcResp.RequestStatus.State = "RefreshAssociation"
		if !reflect.DeepEqual(receivedNcResp, expectedNcResp) {
			t.Fatalf("Create nc data mismatch;\nreceived %+v\nexpected %+v", receivedNcResp, expectedNcResp)
		}
	}

	//create NC with reservation which has been assigned to front end IP, creation NC should fail
	addReservation(t, nodeID, vnetID, subnetID, "", reservationForLBIP, "frontendIP", defaultRequestTimeout)
	if respErr := dnctl.AddNc(nodeID, vnetID, utils.FrontendIPNC, reservationForLBIP, subnetID, "", nodeID, nodeIface, cns.AzureContainerInstance, podName+"3", podNamespace, "", false, false, secondaryIPCount); respErr == nil {
		t.Fatalf("Failed to block NC creation on front end IP reservation")
	}

	if !devEnv {
		// usage: docker-run.sh <container-name> <namespace> <image>
		cmd := "/bin/sh"

		args := []string{"-c", "./docker-run.sh " + podName + " " + podNamespace + " nginx:alpine"}
		logExecCmd(t, cmd, args...)
		output, err := exec.Command(cmd, args...).CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to start container with error code %+v: %s", err, string(output))
		}

		args = []string{"-c", "docker exec " + podName + " ip a | grep \"global eth0\" | awk '{print $2}'"}
		logExecCmd(t, cmd, args...)
		output, err = exec.Command(cmd, args...).CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to get container ip config with error code %+v: %s", err, string(output))
		}

		ncIP := strings.TrimRight(string(output), "\n")
		if ncIP != "10.0.1.4/24" {
			t.Fatalf("Created container IP mismatch;\nreceived %s\nexpected %s", ncIP, "10.0.1.4/24")
		}

		args = []string{"-c", "./docker-del.sh " + podName + " " + podNamespace}
		logExecCmd(t, cmd, args...)
		output, err = exec.Command(cmd, args...).CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to delete container with error code %+v: %s", err, output)
		}
	}

	deleteNode(t, nodeID, http.StatusBadRequest)
	deleteNetwork(t, vnetID, http.StatusBadRequest)
	t.Log("deleting first nc")
	deleteNc(t, vnetID, ncID)
	t.Log("deleting second nc")
	deleteNc(t, vnetID, ncID2)
	deleteNc(t, vnetID, utils.FrontendIPNC)
	deleteReservation(t, vnetID, reservationID)
	deleteReservation(t, vnetID, reservationForLBIP)
	deleteNetwork(t, vnetID, http.StatusNotFound)
	deleteNode(t, nodeID, http.StatusNotFound)

	ncCleanupResp, respErr := dnctl.GetNetworkContainerCleanupStatus(vnetID, ncID)
	if respErr != nil && respErr.HTTPStatus != http.StatusOK {
		t.Fatalf("Failed to ensure NC %s was cleaned up with error: %+v", ncID, respErr)
	}
	if ncCleanupResp.Completed != true {
		t.Fatalf("Failed to cleanup NC %s;\nreceived %+v\nexpected %s", ncID, ncCleanupResp, "{\"Completed\": true}")
	}
}

func logExecCmd(t *testing.T, cmd string, args ...string) {
	t.Logf(" executing command '%s %s'", cmd, strings.Join(args, " "))
}

func TestRetrieveDelegatedSubnets(t *testing.T) {
	t.Parallel()
	var (
		vnetCfg             *api.NetworkConfigurationResponse
		respErr             *api.JsonErr
		id, _               = uuid.NewUUID()
		vnetID              = id.String()
		id2, _              = uuid.NewUUID()
		authenticationToken = id2.String()
		subnetID            = "subnet1"
		subnetID2           = "subnet2"
	)

	addNetwork(t, vnetID, defaultRequestTimeout)
	addSubnet(t, vnetID, subnetID, authenticationToken)
	if vnetCfg, respErr = dnctl.GetNetworkConfiguration(vnetID); respErr.HTTPStatus != http.StatusOK {
		t.Fatalf("Get network configuration failed with error %+v", respErr)
	}
	if len(vnetCfg.DelegatedSubnets) != 1 {
		t.Fatalf("Failed to retrieve the expected number of delegated subnets;\nreceived %d\nexpected 1", len(vnetCfg.DelegatedSubnets))
	}

	addSubnet(t, vnetID, subnetID2, authenticationToken)
	if vnetCfg, respErr = dnctl.GetNetworkConfiguration(vnetID); respErr.HTTPStatus != http.StatusOK {
		t.Fatalf("Get network configuration failed with error %+v", respErr)
	}
	if len(vnetCfg.DelegatedSubnets) != 2 {
		t.Fatalf("Failed to retrieve the expected number of delegated subnets;\nreceived %d\nexpected 2", len(vnetCfg.DelegatedSubnets))
	}

	deleteSubnet(t, vnetID, subnetID)
	if vnetCfg, respErr = dnctl.GetNetworkConfiguration(vnetID); respErr.HTTPStatus != http.StatusOK {
		t.Fatalf("Get network configuration failed with error %+v", respErr)
	}
	if len(vnetCfg.DelegatedSubnets) != 1 {
		t.Fatalf("Failed to retrieve the expected number of delegated subnets;\nreceived %d\nexpected 1", len(vnetCfg.DelegatedSubnets))
	}

	deleteNetwork(t, vnetID, http.StatusNotFound)
}

func testGenericFailureVnet(t *testing.T) {
	t.Parallel()
	addNetwork(t, utils.GenericFailureVnet, errdRequestTimeout)
	deleteNetwork(t, utils.GenericFailureVnet, http.StatusNotFound)
}

func testThrottledVnet(t *testing.T) {
	t.Parallel()
	addNetwork(t, utils.ThrottledVnet, errdRequestTimeout)
	deleteNetwork(t, utils.ThrottledVnet, http.StatusNotFound)
}

func testPendingVnet(t *testing.T) {
	t.Parallel()
	addNetwork(t, utils.PendingVnet, errdRequestTimeout)
	deleteNetwork(t, utils.PendingVnet, http.StatusNotFound)
}

func TestVnetFailures(t *testing.T) {
	t.Parallel()
	t.Run("VnetFailureGroup", func(t *testing.T) {
		t.Run("TestGenericFailureVnet", testGenericFailureVnet)
		t.Run("TestThrottledVnet", testThrottledVnet)
		t.Run("TestPendingVnet", testPendingVnet)
	})
}

func testGenericFailureNc(t *testing.T) {
	t.Parallel()
	var (
		nodeID        = utils.StandardNode
		vnetID        = utils.StandardVnet
		ncID          = utils.GenericFailureNC
		containerType = cns.AzureContainerInstance
		podName       = "container1"
		podNamespace  = "default"
	)

	addNc(t, nodeID, vnetID, ncID, "", utils.Subnet1, "", "", utils.NodeIface, containerType, podName, podNamespace, false, false, errdRequestTimeout, secondaryIPCount)
	deleteNc(t, vnetID, ncID)
}

func testThrottledNc(t *testing.T) {
	t.Parallel()
	var (
		nodeID        = utils.StandardNode
		vnetID        = utils.StandardVnet2
		ncID          = utils.ThrottledNC
		containerType = cns.AzureContainerInstance
		podName       = "container2"
		podNamespace  = "default"
	)

	addNc(t, nodeID, vnetID, ncID, "", utils.Subnet1, "", "", utils.NodeIface, containerType, podName, podNamespace, false, false, errdRequestTimeout, secondaryIPCount)
	deleteNc(t, vnetID, ncID)
}

func testUnauthorizedNc(t *testing.T) {
	t.Parallel()
	var (
		nodeID        = utils.StandardNode
		vnetID        = utils.StandardVnet3
		ncID          = utils.UnauthorizedNC
		containerType = cns.AzureContainerInstance
		podName       = "container3"
		podNamespace  = "default"
	)

	addNc(t, nodeID, vnetID, ncID, "", utils.Subnet1, "", "", utils.NodeIface, containerType, podName, podNamespace, false, false, errdRequestTimeout, secondaryIPCount)
	deleteNc(t, vnetID, ncID)
}

func TestNcFailures(t *testing.T) {
	var (
		nodeID              = utils.StandardNode
		id, _               = uuid.NewUUID()
		authenticationToken = id.String()
		subnetID            = utils.Subnet1
	)

	addNode(t, nodeID, utils.LocalHost, cns.Kubernetes, "")
	addNetwork(t, utils.StandardVnet, defaultRequestTimeout)
	addSubnet(t, utils.StandardVnet, subnetID, authenticationToken)
	addNetwork(t, utils.StandardVnet2, defaultRequestTimeout)
	addSubnet(t, utils.StandardVnet2, subnetID, authenticationToken)
	addNetwork(t, utils.StandardVnet3, defaultRequestTimeout)
	addSubnet(t, utils.StandardVnet3, subnetID, authenticationToken)
	t.Run("NcFailureGroup", func(t *testing.T) {
		t.Run("TestGenericFailureNc", testGenericFailureNc)
		t.Run("TestThrottledNc", testThrottledNc)
		t.Run("TestUnauthorizedNc", testUnauthorizedNc)
	})

	deleteNetwork(t, utils.StandardVnet, http.StatusNotFound)
	deleteNetwork(t, utils.StandardVnet2, http.StatusNotFound)
	deleteNetwork(t, utils.StandardVnet3, http.StatusNotFound)
	deleteNode(t, nodeID, http.StatusNotFound)
}

func addNode(t *testing.T, nodeID, ipaddr, orchestratorType, infraVnet string) {
	var (
		respErr *api.JsonErr
		nodeCfg *api.NodeInfoResponse
	)

	if respErr = dnctl.AddNode(nodeID, ipaddr, orchestratorType, infraVnet, "", "", "", 0); respErr.HTTPStatus != http.StatusOK {
		t.Fatalf("Register node failed with error %+v", respErr)
	}

	if nodeCfg, respErr = dnctl.GetNode(nodeID); respErr.HTTPStatus != http.StatusOK {
		t.Fatalf("Register node failed with error %+v", respErr)
	} else {
		receivedNodeCfg := nodeCfg.NodeInfo
		expectedNodeCfg := api.NodeInfoRequest{
			IPAddresses:      []string{ipaddr},
			LocationType:     "",
			OrchestratorType: orchestratorType}
		if !reflect.DeepEqual(receivedNodeCfg, expectedNodeCfg) {
			t.Fatalf("Register node failed;\nreceived %+v\nexpected %+v", receivedNodeCfg, expectedNodeCfg)
		}
	}
}

func getNodes(t *testing.T, expectedNodeNames []string) {
	nodesResp, respErr := dnctl.GetNodes("")
	if respErr.HTTPStatus != http.StatusOK {
		// Should always return 200 even if empty
		t.Fatalf("GetNodes failed with error %+v", respErr)
	}

	// Check for matching size
	if len(expectedNodeNames) != len(nodesResp.Nodes) {
		t.Fatalf("GetNodes failed, received %d nodes but expected %d nodes: Resp:%+v", len(nodesResp.Nodes), len(expectedNodeNames), nodesResp)
	}

	// Check that each node is present in the list
	for _, nodeName := range expectedNodeNames {
		found := false
		for _, nodeInfo := range nodesResp.Nodes {
			if nodeName == nodeInfo.Name {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("GetNodes failed, expected to find Node %s but it was not found: Resp: %+v", nodeName, nodesResp)
		}
	}

}

func deleteNode(t *testing.T, nodeID string, status int) {
	var (
		respErr                      *api.JsonErr
		deleteStatus, validateDelete = deleteHelper(status)
	)

	for start := time.Now(); ; {
		respErr = dnctl.DeleteNode(nodeID)
		if respErr.HTTPStatus == deleteStatus {
			break
		}

		if respErr.HTTPStatus == http.StatusAccepted && time.Since(start) < defaultRequestTimeout {
			time.Sleep(defaultSleep)
			continue
		}

		t.Fatalf("Unregister node failed with error %+v", respErr)
	}

	if validateDelete {
		for start := time.Now(); ; {
			_, respErr = dnctl.GetNode(nodeID)
			if respErr.HTTPStatus == http.StatusNotFound {
				break
			}

			if (respErr.HTTPStatus == http.StatusOK || respErr.HTTPStatus == http.StatusAccepted) && time.Since(start) < defaultRequestTimeout {
				time.Sleep(defaultSleep)
				continue
			}

			t.Fatalf("Unregister node failed with error %+v", respErr)
		}
	}
}

func addNetwork(t *testing.T, vnetID string, timeout time.Duration) {
	var (
		respErr               *api.JsonErr
		vnetStatus            *api.RequestStatus
		expectedNetworkStatus = &api.RequestStatus{
			State:  "Join",
			Status: "Completed"}
	)

	if respErr = dnctl.AddNetwork(vnetID); respErr.HTTPStatus != http.StatusOK {
		t.Fatalf("Register network failed with error %+v", respErr)
	}

	for start := time.Now(); ; {
		vnetStatus, respErr = dnctl.GetNetworkStatus(vnetID)
		if respErr.HTTPStatus == http.StatusOK {
			break
		}

		if respErr.HTTPStatus == http.StatusAccepted && time.Since(start) < timeout {
			time.Sleep(defaultSleep)
			continue
		}

		t.Fatalf("Register network failed with error %+v", respErr)
	}

	if !reflect.DeepEqual(vnetStatus, expectedNetworkStatus) {
		t.Fatalf("Register network failed;\nreceived %+v\nexpected %+v", vnetStatus, expectedNetworkStatus)
	}
}

func getNetworks(t *testing.T, expectedVnetIDs []string) {
	var (
		respErr          *api.JsonErr
		networksResponse *api.NetworksResponse
	)

	if networksResponse, respErr = dnctl.GetNetworks(); respErr.HTTPStatus != http.StatusOK {
		t.Fatalf("Failed to get all networks with error %+v", respErr)
	}

	for _, expectedVnetID := range expectedVnetIDs {
		if !containsNetwork(networksResponse.NetworkInfos, expectedVnetID) {
			t.Fatalf("Get Networks failed, expected %s in response but was not present", expectedVnetID)
		}
	}
}

func deleteNetwork(t *testing.T, vnetID string, status int) {
	var (
		respErr                      *api.JsonErr
		deleteStatus, validateDelete = deleteHelper(status)
	)

	for start := time.Now(); ; {
		respErr = dnctl.DeleteNetwork(vnetID)
		if respErr.HTTPStatus == deleteStatus {
			break
		}

		if respErr.HTTPStatus == http.StatusAccepted && time.Since(start) < defaultRequestTimeout {
			time.Sleep(defaultSleep)
			continue
		}

		t.Fatalf("Unregister network failed with error %+v", respErr)
	}

	// if expected status is 404, check to see if resource has been deleted
	if validateDelete {
		for start := time.Now(); ; {
			_, respErr = dnctl.GetNetworkStatus(vnetID)
			if respErr.HTTPStatus == http.StatusNotFound {
				break
			}

			if (respErr.HTTPStatus == http.StatusOK || respErr.HTTPStatus == http.StatusAccepted) && time.Since(start) < defaultRequestTimeout {
				time.Sleep(defaultSleep)
				continue
			}

			t.Fatalf("Unregister network failed with error %+v", respErr)
		}
	}
}

func addSubnet(t *testing.T, vnetID, subnetID, authToken string) {
	var (
		respErr              *api.JsonErr
		subnetStatus         *api.RequestStatus
		expectedSubnetStatus = &api.RequestStatus{
			State:  "Join",
			Status: "Completed"}
	)

	if respErr = dnctl.AddSubnet(vnetID, subnetID, authToken, ""); respErr.HTTPStatus != http.StatusOK {
		t.Fatalf("Register subnet failed with error %+v", respErr)
	}

	for start := time.Now(); ; {
		subnetStatus, respErr = dnctl.GetSubnetStatus(vnetID, subnetID)
		if respErr.HTTPStatus == http.StatusOK {
			break
		}

		if respErr.HTTPStatus == http.StatusAccepted && time.Since(start) < defaultRequestTimeout {
			time.Sleep(defaultSleep)
			continue
		}

		t.Fatalf("Register subnet failed with error %+v", respErr)
	}

	if !reflect.DeepEqual(subnetStatus, expectedSubnetStatus) {
		t.Fatalf("Register subnet failed;\nreceived %+v\nexpected %+v", subnetStatus, expectedSubnetStatus)
	}
}

func deleteSubnet(t *testing.T, vnetID, subnetID string) {
	var respErr *api.JsonErr

	for start := time.Now(); ; {
		respErr = dnctl.DeleteSubnet(vnetID, subnetID, "")
		if respErr.HTTPStatus == http.StatusOK {
			break
		}

		if respErr.HTTPStatus == http.StatusAccepted && time.Since(start) < defaultRequestTimeout {
			time.Sleep(defaultSleep)
			continue
		}

		t.Fatalf("Unregister subnet failed with error %+v", respErr)
	}
}

func addReservation(t *testing.T, nodeID, vnetID, subnetID, ipConstraint, reservationID, requestType string, timeout time.Duration) {
	var (
		respErr                   *api.JsonErr
		reservationStatus         *api.RequestStatus
		expectedReservationStatus = &api.RequestStatus{
			State:  "Process",
			Status: "Completed"}
	)

	if respErr = dnctl.AddReservation(vnetID, subnetID, ipConstraint, nodeID, reservationID, requestType); respErr.HTTPStatus != http.StatusOK {
		t.Fatalf("Register reservation failed with error %+v", respErr)
	}

	for start := time.Now(); ; {
		reservationStatus, respErr = dnctl.GetReservationStatus(vnetID, reservationID)
		if respErr.HTTPStatus == http.StatusOK {
			break
		}

		if respErr.HTTPStatus == http.StatusAccepted && time.Since(start) < timeout {
			time.Sleep(defaultSleep)
			continue
		}

		t.Fatalf("Register reservation failed with error %+v", respErr)
	}

	if !reflect.DeepEqual(reservationStatus, expectedReservationStatus) {
		t.Fatalf("Register reservation failed;\nrecieved %+v\nexpected %+v", reservationStatus, expectedReservationStatus)
	}
}

func deleteReservation(t *testing.T, vnetID, reservationID string) {
	var respErr *api.JsonErr

	for start := time.Now(); ; {
		respErr = dnctl.DeleteReservation(vnetID, reservationID)
		if respErr.HTTPStatus == http.StatusOK {
			break
		}

		if respErr.HTTPStatus == http.StatusAccepted && time.Since(start) < defaultRequestTimeout {
			time.Sleep(defaultSleep)
			continue
		}

		t.Fatalf("Unregister reservation failed with error %+v", respErr)
	}

	for start := time.Now(); ; {
		_, respErr = dnctl.GetReservationStatus(vnetID, reservationID)
		if respErr.HTTPStatus == http.StatusNotFound {
			break
		}

		if (respErr.HTTPStatus == http.StatusOK || respErr.HTTPStatus == http.StatusAccepted) && time.Since(start) < defaultRequestTimeout {
			time.Sleep(defaultSleep)
			continue
		}

		t.Fatalf("Unregister reservation failed with error %+v", respErr)
	}
}

func addNc(t *testing.T, nodeID, vnetID, ncID, reservationID, subnetID, ipConstraint, nodeConstraint, ifaceID, containerType, podName, podNamespace string, host2NC, NC2Host bool, timeout time.Duration, secondaryIPCount int) {
	var (
		respErr          *api.JsonErr
		ncStatus         *api.NCRequestStatus
		expectedNcStatus = &api.NCRequestStatus{
			State:     "Program",
			Status:    "Completed",
			NICStatus: "N/A",
		}
	)

	if respErr = dnctl.AddNc(nodeID, vnetID, ncID, reservationID, subnetID, ipConstraint, nodeConstraint, ifaceID, containerType, podName, podNamespace, "", host2NC, NC2Host, secondaryIPCount); respErr.HTTPStatus != http.StatusOK {
		t.Fatalf("Create nc failed with error %+v", respErr)
	}

	for start := time.Now(); ; {
		ncStatus, respErr = dnctl.GetNCStatus(vnetID, ncID)
		if respErr.HTTPStatus == http.StatusOK {
			break
		}

		if respErr.HTTPStatus == http.StatusAccepted && time.Since(start) < timeout {
			time.Sleep(defaultSleep)
			continue
		}

		t.Fatalf("Create nc failed with error %+v", respErr)
	}

	if !reflect.DeepEqual(ncStatus, expectedNcStatus) {
		expectedNcStatus.State = "RefreshAssociation"
		if !reflect.DeepEqual(ncStatus, expectedNcStatus) {
			t.Fatalf("Create nc failed;\nreceived %+v\nexpected %+v", ncStatus, expectedNcStatus)
		}
	}
}

func deleteNc(t *testing.T, vnetID, ncID string) {
	var respErr *api.JsonErr

	for start := time.Now(); ; {
		respErr = dnctl.DeleteNC(vnetID, ncID)
		if respErr.HTTPStatus == http.StatusOK {
			break
		}

		if respErr.HTTPStatus == http.StatusAccepted && time.Since(start) < defaultRequestTimeout {
			time.Sleep(defaultSleep)
			continue
		}

		t.Fatalf("Delete nc failed with error %+v", respErr)
	}

	for start := time.Now(); ; {
		_, respErr = dnctl.GetNCStatus(vnetID, ncID)
		if respErr.HTTPStatus == http.StatusNotFound {
			break
		}

		if (respErr.HTTPStatus == http.StatusOK || respErr.HTTPStatus == http.StatusAccepted) && time.Since(start) < defaultNCRequestTimeout {
			time.Sleep(defaultSleep)
			continue
		}

		t.Fatalf("Delete nc id %s in vnet %s failed with error %+v", vnetID, ncID, respErr)
	}
}

func deleteHelper(status int) (expectedStatus int, validateDelete bool) {
	expectedStatus = status
	if status == http.StatusNotFound {
		expectedStatus, validateDelete = http.StatusOK, true
	}
	return
}

func containsNetwork(networkInfos []api.NetworkInfo, s string) bool {
	for _, t := range networkInfos {
		if t.GUID == s {
			return true
		}
	}

	return false
}
