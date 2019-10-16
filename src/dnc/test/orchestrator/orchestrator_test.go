package test

import (
	"dnc/api"
	"dnc/configuration"
	"dnc/dnctl/dncclient"
	"dnc/test/utils"
	"net/http"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/Azure/azure-container-networking/cns"
	"github.com/google/uuid"
)

var (
	dnctl                   *dncclient.DncClient
	defaultRequestTimeout   = time.Second * 4
	defaultNCRequestTimeout = time.Second * 10
	errdRequestTimeout      = time.Second * 30
)

func init() {
	// use DNC config set with test storage settings
	configpath, found := os.LookupEnv("DNC_CONFIGURATION_PATH")
	if !found {
		configpath = "config.json"
	}

	config := configuration.Load(configpath)
	if config.ServerSettings.IPAddress == "" || config.ServerSettings.Port == "" {
		panic("DNC EP not set correctly")
	}

	dnctl = dncclient.NewDncClient("http://" + config.ServerSettings.IPAddress + ":" + config.ServerSettings.Port)
}

func TestBasicNodeAPIs(t *testing.T) {
	var (
		nodeID           = utils.StandardNode
		ipaddr           = utils.LocalHost
		orchestratorType = cns.Kubernetes
	)

	addNode(t, nodeID, ipaddr, orchestratorType)
	addNode(t, nodeID, ipaddr, orchestratorType)
	deleteNode(t, nodeID)
}

func TestBasicVnetAPIs(t *testing.T) {
	var (
		id, _  = uuid.NewUUID()
		vnetID = id.String()
	)

	addNetwork(t, vnetID, defaultRequestTimeout)
	addNetwork(t, vnetID, defaultRequestTimeout)
	deleteNetwork(t, vnetID)
}

func TestBasicSubnetAPIs(t *testing.T) {
	var (
		id, _               = uuid.NewUUID()
		vnetID              = id.String()
		id2, _              = uuid.NewUUID()
		authenticationToken = id2.String()
		subnetID            = "subnet1"
	)

	addNetwork(t, vnetID, defaultRequestTimeout)
	addSubnet(t, vnetID, subnetID, authenticationToken)
	addSubnet(t, vnetID, subnetID, authenticationToken)
	deleteNetwork(t, vnetID)
}

func TestBasicNCAPIs(t *testing.T) {
	var (
		nodeID              = utils.StandardNode
		ipaddr              = utils.LocalHost
		orchestratorType    = cns.Kubernetes
		containerType       = cns.AzureContainerInstance
		id, _               = uuid.NewUUID()
		vnetID              = id.String()
		id2, _              = uuid.NewUUID()
		authenticationToken = id2.String()
		id3, _              = uuid.NewUUID()
		ncID                = id3.String()
		subnetID            = utils.Subnet1
		podName             = "container1"
		podNamespace        = "default"
		context             = []byte(`{"PodName":"` + podName + `","PodNamespace":"` + podNamespace + `"}`)
		recievedNcResp      *api.NetworkContainerResponse
		expectedNcResp      = &api.NetworkContainerResponse{
			NcID: ncID,
			IPConfiguration: api.IPConfiguration{
				NetworkID:  vnetID,
				SubnetName: subnetID,
				IPSubnet:   utils.Subnet1AddrSpace,
				IPAddress:  "10.0.1.4/32",
				GatewayIP:  utils.Subnet1DefaultGW,
				DNSServers: []string{"8.8.8.8", "9.9.9.9"}},
			AssociationInfo: api.AssociationInfo{
				NodeID:              nodeID,
				InterfaceID:         ipaddr,
				ContainerType:       containerType,
				OrchestratorContext: context},
			RequestStatus: api.RequestStatus{
				State:  "Program",
				Status: "Completed"},
			VlanID:                     1,
			LocalIP:                    "169.254.128.4",
			AllowHostToNCCommunication: false,
			AllowNCToHostCommunication: false}
		respErr *api.JsonErr
	)

	addNode(t, nodeID, ipaddr, orchestratorType)
	addNetwork(t, vnetID, defaultRequestTimeout)
	addSubnet(t, vnetID, subnetID, authenticationToken)
	addNc(t, nodeID, vnetID, ncID, "", subnetID, "", "", ipaddr, containerType, podName, podNamespace, false, false, defaultNCRequestTimeout)
	addNc(t, nodeID, vnetID, ncID, "", subnetID, "", "", ipaddr, containerType, podName, podNamespace, false, false, defaultNCRequestTimeout)

	if recievedNcResp, respErr = dnctl.GetNC(vnetID, ncID); respErr.HTTPStatus != http.StatusOK {
		t.Fatalf("Failed to retrieve nc response with error %+v", respErr)
	}

	if !reflect.DeepEqual(recievedNcResp, expectedNcResp) {
		expectedNcResp.RequestStatus.State = "RefreshAssociation"
		if !reflect.DeepEqual(recievedNcResp, expectedNcResp) {
			t.Fatalf("Create nc data mismatch;\nrecieved %+v\nexpected %+v", recievedNcResp, expectedNcResp)
		}
	}

	deleteNc(t, vnetID, ncID)
	deleteNetwork(t, vnetID)
	deleteNode(t, nodeID)
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
		t.Fatalf("Failed to retrieve the expected number of delegated subnets;\nrecieved %d\nexpected 1", len(vnetCfg.DelegatedSubnets))
	}

	addSubnet(t, vnetID, subnetID2, authenticationToken)
	if vnetCfg, respErr = dnctl.GetNetworkConfiguration(vnetID); respErr.HTTPStatus != http.StatusOK {
		t.Fatalf("Get network configuration failed with error %+v", respErr)
	}
	if len(vnetCfg.DelegatedSubnets) != 2 {
		t.Fatalf("Failed to retrieve the expected number of delegated subnets;\nrecieved %d\nexpected 2", len(vnetCfg.DelegatedSubnets))
	}

	deleteSubnet(t, vnetID, subnetID)
	if vnetCfg, respErr = dnctl.GetNetworkConfiguration(vnetID); respErr.HTTPStatus != http.StatusOK {
		t.Fatalf("Get network configuration failed with error %+v", respErr)
	}
	if len(vnetCfg.DelegatedSubnets) != 1 {
		t.Fatalf("Failed to retrieve the expected number of delegated subnets;\nrecieved %d\nexpected 1", len(vnetCfg.DelegatedSubnets))
	}

	deleteNetwork(t, vnetID)
}

func testGenericFailureVnet(t *testing.T) {
	t.Parallel()
	addNetwork(t, utils.GenericFailureVnet, errdRequestTimeout)
	deleteNetwork(t, utils.GenericFailureVnet)
}

func testThrottledVnet(t *testing.T) {
	t.Parallel()
	addNetwork(t, utils.ThrottledVnet, errdRequestTimeout)
	deleteNetwork(t, utils.ThrottledVnet)
}

func testPendingVnet(t *testing.T) {
	t.Parallel()
	addNetwork(t, utils.PendingVnet, errdRequestTimeout)
	deleteNetwork(t, utils.PendingVnet)
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

	addNc(t, nodeID, vnetID, ncID, "", utils.Subnet1, "", "", utils.LocalHost, containerType, podName, podNamespace, false, false, errdRequestTimeout)
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

	addNc(t, nodeID, vnetID, ncID, "", utils.Subnet1, "", "", utils.LocalHost, containerType, podName, podNamespace, false, false, errdRequestTimeout)
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

	addNc(t, nodeID, vnetID, ncID, "", utils.Subnet1, "", "", utils.LocalHost, containerType, podName, podNamespace, false, false, errdRequestTimeout)
	deleteNc(t, vnetID, ncID)
}

func TestNcFailures(t *testing.T) {
	var (
		nodeID              = utils.StandardNode
		id, _               = uuid.NewUUID()
		authenticationToken = id.String()
		subnetID            = utils.Subnet1
	)

	addNode(t, nodeID, utils.LocalHost, cns.Kubernetes)
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

	deleteNetwork(t, utils.StandardVnet)
	deleteNetwork(t, utils.StandardVnet2)
	deleteNetwork(t, utils.StandardVnet3)
	deleteNode(t, nodeID)
}

func addNode(t *testing.T, nodeID, ipaddr, orchestratorType string) {
	var (
		respErr *api.JsonErr
		nodeCfg *api.NodeInfoResponse
	)

	if respErr = dnctl.AddNode(nodeID, ipaddr, orchestratorType); respErr.HTTPStatus != http.StatusOK {
		t.Fatalf("Register node failed with error %+v", respErr)
	}

	if nodeCfg, respErr = dnctl.GetNode(nodeID); respErr.HTTPStatus != http.StatusOK {
		t.Fatalf("Register node failed with error %+v", respErr)
	} else {
		recievedNodeCfg := nodeCfg.NodeInfo
		expectedNodeCfg := api.NodeInfoRequest{
			IPAddresses:      []string{ipaddr},
			LocationType:     "",
			OrchestratorType: orchestratorType}
		if recievedNodeCfg.String() != expectedNodeCfg.String() {
			t.Fatalf("Register node failed;\nrecieved %+v\nexpected %+v", recievedNodeCfg, expectedNodeCfg)
		}
	}
}

func deleteNode(t *testing.T, nodeID string) {
	var respErr *api.JsonErr

	for start := time.Now(); ; {
		respErr = dnctl.DeleteNode(nodeID)
		if respErr.HTTPStatus == http.StatusOK {
			break
		}

		if respErr.HTTPStatus == http.StatusAccepted && time.Since(start) < defaultRequestTimeout {
			continue
		}

		t.Fatalf("Unregister node failed with error %+v", respErr)
	}

	for start := time.Now(); ; {
		_, respErr = dnctl.GetNode(nodeID)
		if respErr.HTTPStatus == http.StatusNotFound {
			break
		}

		if (respErr.HTTPStatus == http.StatusOK || respErr.HTTPStatus == http.StatusAccepted) && time.Since(start) < defaultRequestTimeout {
			continue
		}

		t.Fatalf("Unregister node failed with error %+v", respErr)
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
			continue
		}

		t.Fatalf("Register network failed with error %+v", respErr)
	}

	if !reflect.DeepEqual(vnetStatus, expectedNetworkStatus) {
		t.Fatalf("Register network failed;\nrecieved %+v\nexpected %+v", vnetStatus, expectedNetworkStatus)
	}
}

func deleteNetwork(t *testing.T, vnetID string) {
	var respErr *api.JsonErr

	for start := time.Now(); ; {
		respErr = dnctl.DeleteNetwork(vnetID)
		if respErr.HTTPStatus == http.StatusOK {
			break
		}

		if respErr.HTTPStatus == http.StatusAccepted && time.Since(start) < defaultRequestTimeout {
			continue
		}

		t.Fatalf("Unregister network failed with error %+v", respErr)
	}

	for start := time.Now(); ; {
		_, respErr = dnctl.GetNetworkStatus(vnetID)
		if respErr.HTTPStatus == http.StatusNotFound {
			break
		}

		if (respErr.HTTPStatus == http.StatusOK || respErr.HTTPStatus == http.StatusAccepted) && time.Since(start) < defaultRequestTimeout {
			continue
		}

		t.Fatalf("Unregister network failed with error %+v", respErr)
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

	if respErr = dnctl.AddSubnet(vnetID, subnetID, authToken); respErr.HTTPStatus != http.StatusOK {
		t.Fatalf("Register subnet failed with error %+v", respErr)
	}

	for start := time.Now(); ; {
		subnetStatus, respErr = dnctl.GetSubnetStatus(vnetID, subnetID)
		if respErr.HTTPStatus == http.StatusOK {
			break
		}

		if respErr.HTTPStatus == http.StatusAccepted && time.Since(start) < defaultRequestTimeout {
			continue
		}

		t.Fatalf("Register subnet failed with error %+v", respErr)
	}

	if !reflect.DeepEqual(subnetStatus, expectedSubnetStatus) {
		t.Fatalf("Register subnet failed;\nrecieved %+v\nexpected %+v", subnetStatus, expectedSubnetStatus)
	}
}

func deleteSubnet(t *testing.T, vnetID, subnetID string) {
	var respErr *api.JsonErr

	for start := time.Now(); ; {
		respErr = dnctl.DeleteSubnet(vnetID, subnetID)
		if respErr.HTTPStatus == http.StatusOK {
			break
		}

		if respErr.HTTPStatus == http.StatusAccepted && time.Since(start) < defaultRequestTimeout {
			continue
		}

		t.Fatalf("Unregister subnet failed with error %+v", respErr)
	}
}

func addNc(t *testing.T, nodeID, vnetID, ncID, reservationID, subnetID, ipConstraint, nodeConstraint, ifaceID, containerType, podName, podNamespace string, host2NC, NC2Host bool, timeout time.Duration) {
	var (
		respErr          *api.JsonErr
		ncStatus         *api.RequestStatus
		expectedNcStatus = &api.RequestStatus{
			State:  "Program",
			Status: "Completed"}
	)

	if respErr = dnctl.AddNc(nodeID, vnetID, ncID, "", subnetID, "", "", ifaceID, cns.AzureContainerInstance, podName, podNamespace, false, false); respErr.HTTPStatus != http.StatusOK {
		t.Fatalf("Create nc failed with error %+v", respErr)
	}

	for start := time.Now(); ; {
		ncStatus, respErr = dnctl.GetNCStatus(vnetID, ncID)
		if respErr.HTTPStatus == http.StatusOK {
			break
		}

		if respErr.HTTPStatus == http.StatusAccepted && time.Since(start) < timeout {
			continue
		}

		t.Fatalf("Create nc failed with error %+v", respErr)
	}

	if !reflect.DeepEqual(ncStatus, expectedNcStatus) {
		expectedNcStatus.State = "RefreshAssociation"
		if !reflect.DeepEqual(ncStatus, expectedNcStatus) {
			t.Fatalf("Create nc failed;\nrecieved %+v\nexpected %+v", ncStatus, expectedNcStatus)
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
			continue
		}

		t.Fatalf("Delete nc failed with error %+v", respErr)
	}
}
