package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"net/http"
	"testing"

	"github.com/Azure/azure-container-networking/cns"
)

type CreateNodeRequest struct {
	IPAddresses []string
}

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

// TestMain :- main
func TestMain(m *testing.M) {
	if err := Setup(); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

// Setup :- Set up the environment (if necessary).
func Setup() error {
	return nil
}

func TestScenario(t *testing.T) {
	var scenario = struct {
		nid                  string
		nid2                 string
		dest                 string
		vnetid               string
		ncid                 string
		ncid2                string
		subnet               string
		apiversion           string
		reservationID1       string
		reservationID2       string
		nodereq              CreateNodeRequest
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
		"http://172.22.30.33:9999",
		"testvnetid",
		"testncid",
		"testncid2",
		"subnet1",
		"?api-version=2018-03-01",
		"reservationID1",
		"reservationID2",
		CreateNodeRequest{[]string{"127.0.0.1", "10.0.0.9"}},
		CreateNetworkRequest{"AzureNet"},
		JoinSubnetRequest{"0bb4132b-5932-4ec4-a47f-1b24f0eeb426"},
		_AllocationRequest{"subnet1", "10.0.0.15", ""},
		CreateReservationWithNodeConstraintRequest{"testnode", "subnet1"},
		CreateNetworkContainerRequest{_AllocationRequest{"subnet1", "10.0.0.7", ""}, _AssociationInfo{"testnode", "10.1.1.2"}},
		CreateNetworkContainerUsingReservationRequest{"reservationID2", _AssociationInfo{"testnode", "10.1.1.2"}},
		CreateNetworkContainerUsingReservationRequest{"reservationID2", _AssociationInfo{"testnode2", "10.1.1.2"}},
	}

	CreateNode(t, scenario.nodereq, scenario.dest, scenario.nid, scenario.apiversion)
	CreateNetwork(t, scenario.networkreq, scenario.dest, scenario.vnetid, scenario.apiversion)
	JoinSubnet(t, scenario.networkreq, scenario.dest, scenario.vnetid, scenario.subnet, scenario.apiversion)
	CreateReservation(t, scenario.reservationreq1, scenario.dest, scenario.vnetid, scenario.reservationID1, scenario.apiversion)
	CreateReservationNodeConstraint(t, scenario.reservationreq2, scenario.dest, scenario.vnetid, scenario.reservationID2, scenario.apiversion)
	CreateNetworkContainer(t, scenario.networkcontainerreq, scenario.dest, scenario.vnetid, scenario.ncid, scenario.apiversion)
	DeleteNetworkContainer(t, scenario.dest, scenario.vnetid, scenario.ncid, scenario.apiversion)
	CreateNetworkContainerUsingReservation(t, scenario.networkcontainerreq2, scenario.dest, scenario.vnetid, scenario.ncid2, scenario.apiversion)
	DeleteNetworkContainer(t, scenario.dest, scenario.vnetid, scenario.ncid2, scenario.apiversion)
}

func CreateNode(t *testing.T, nodereq CreateNodeRequest, dest, nid, apiversion string) {
	var body bytes.Buffer
	var response cns.Response
	json.NewEncoder(&body).Encode(nodereq)
	request, err := http.Post(dest+"/nodes/"+nid+apiversion, "application/json", &body)

	if err != nil {
		t.Errorf("CreateNode failed with error: %+v\n", err)
		t.Fatal(err)
	}

	if err = decodeResponse(request, &response); !strings.Contains(err.Error(), "201") || response.ReturnCode != 0 {
		t.Errorf("CreateNode failed with response %+v Err:%+v", response, err)
		t.Fatal(err)
	}

	fmt.Printf("Successfully created node [%s]\n", nid)
}

func CreateNetwork(t *testing.T, networkreq CreateNetworkRequest, dest, vnetid, apiversion string) {
	var body bytes.Buffer
	var response cns.Response
	json.NewEncoder(&body).Encode(networkreq)
	request, err := http.Post(dest+"/networks/"+vnetid+apiversion, "application/json", &body)

	if err != nil {
		t.Errorf("CreateNetwork failed with error: %+v\n", err)
		t.Fatal(err)
	}

	if err = decodeResponse(request, &response); !strings.Contains(err.Error(), "201") || response.ReturnCode != 0 {
		t.Errorf("CreateNetwork failed with response %+v Err:%+v", response, err)
		t.Fatal(err)
	}
	// Sleep for 5s for processing to complete
	time.Sleep(5000 * time.Millisecond)
	fmt.Printf("Successfully created network [%s]\n", vnetid)
}

func JoinSubnet(t *testing.T, networkreq CreateNetworkRequest, dest, vnetid, subnetname, apiversion string) {
	var body bytes.Buffer
	var response cns.Response
	json.NewEncoder(&body).Encode(networkreq)
	request, err := http.Post(dest+"/networks/"+vnetid+"/subnets/"+subnetname+apiversion, "application/json", &body)

	if err != nil {
		t.Errorf("JoinSubnet failed with error: %+v\n", err)
		t.Fatal(err)
	}

	if err = decodeResponse(request, &response); !strings.Contains(err.Error(), "201") || response.ReturnCode != 0 {
		t.Errorf("JoinSubnet failed with response %+v Err:%+v", response, err)
		t.Fatal(err)
	}
	// Sleep for 5s for processing to complete
	time.Sleep(5000 * time.Millisecond)
	fmt.Printf("Successfully joined subnet [%s]\n", subnetname)
}

func CreateReservation(t *testing.T, reservationreq _AllocationRequest, dest, vnetid, reservationID, apiversion string) {
	var body bytes.Buffer
	var response cns.Response
	json.NewEncoder(&body).Encode(reservationreq)
	request, err := http.Post(dest+"/networks/"+vnetid+"/reservations/"+reservationID+apiversion, "application/json", &body)

	if err != nil {
		t.Errorf("CreateReservation failed with error: %+v\n", err)
		t.Fatal(err)
	}

	if err = decodeResponse(request, &response); !strings.Contains(err.Error(), "201") || response.ReturnCode != 0 {
		t.Errorf("CreateReservation failed with response %+v Err:%+v", response, err)
		t.Fatal(err)
	}

	fmt.Printf("Successfully created reservation [%s]\n", reservationID)
}

func CreateReservationNodeConstraint(t *testing.T, reservationreq CreateReservationWithNodeConstraintRequest, dest, vnetid, reservationID, apiversion string) {
	var body bytes.Buffer
	var response cns.Response
	json.NewEncoder(&body).Encode(reservationreq)
	request, err := http.Post(dest+"/networks/"+vnetid+"/reservations/"+reservationID+apiversion, "application/json", &body)

	if err != nil {
		t.Errorf("CreateReservationNodeConstraint failed with error: %+v\n", err)
		t.Fatal(err)
	}

	if err = decodeResponse(request, &response); !strings.Contains(err.Error(), "201") || response.ReturnCode != 0 {
		t.Errorf("CreateReservationNodeConstraint failed with response %+v Err:%+v", response, err)
		t.Fatal(err)
	}
	// Sleep for 10s for processing to complete
	time.Sleep(10000 * time.Millisecond)
	fmt.Printf("Successfully created reservation [%s] with node constraint [%s]\n", reservationID, reservationreq.NodeConstraint)
}

func CreateNetworkContainer(t *testing.T, networkcontainerreq CreateNetworkContainerRequest, dest, vnetid, ncid, apiversion string) {
	var body bytes.Buffer
	var response cns.Response
	json.NewEncoder(&body).Encode(networkcontainerreq)
	request, err := http.Post(dest+"/networks/"+vnetid+"/networkcontainer/"+ncid+apiversion, "application/json", &body)

	if err != nil {
		t.Errorf("CreateNetworkContainer failed with error: %+v\n", err)
		t.Fatal(err)
	}

	if err = decodeResponse(request, &response); !strings.Contains(err.Error(), "201") || response.ReturnCode != 0 {
		t.Errorf("CreateNetworkContainer failed with response %+v Err:%+v", response, err)
		t.Fatal(err)
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
	var response cns.Response
	// this should fail since reservation was made with testnode instead of testnode2
	json.NewEncoder(&body).Encode(networkcontainerreq)
	request, err := http.Post(dest+"/networks/"+vnetid+"/networkcontainer/"+ncid+apiversion, "application/json", &body)

	if err != nil {
		t.Errorf("CreateNetworkContainerUsingReservation failed with error: %+v\n", err)
		t.Fatal(err)
	}

	if err = decodeResponse(request, &response); !strings.Contains(err.Error(), "201") || response.ReturnCode != 0 {
		t.Errorf("CreateNetworkContainerUsingReservation failed with response %+v Err:%+v", response, err)
		t.Fatal(err)
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
		t.Errorf("DeleteeNetworkContainer failed with Err:%+v", resperr)
		t.Fatal(resperr)
	}

	if response != nil && response.StatusCode != 200 {
		t.Errorf("DeleteNetworkContainer failed with response %+v", response)
	}

	fmt.Printf("Deleted network container with response %+v\n", response)
}
