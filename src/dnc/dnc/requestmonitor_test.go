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

	"github.com/azure-container-networking-dnc-apis/cns"
	//"github.com/stretchr/testify/assert"
)

type testVariables struct {
	dest   string
	vnetID string
	ncID   string
	apiVer string
	addr   addresses
	ninfo  netinfo
	subreq subnetreq
	reqv2  requestv2
}

type addresses struct {
	IPAddresses []string
}

type netinfo struct {
	NetworkType string
}

type subnetreq struct {
	AuthenticationToken string
}

type allocationrequest struct {
	SubnetName     string
	IPConstraint   string
	NodeConstraint string
}

type associationinfo struct {
	NodeID      string
	InterfaceID string
}

type requestv2 struct {
	AllocationRequest allocationrequest
	AssociationInfo   associationinfo
}

// client :- default client
var client = &http.Client{}

// testcase :- default test cases/parameters
var testcase = testVariables{
	"http://10.30.76.53:8085",
	"test-vnetID", /// or func to create network
	"test-ncID",   /// or func to create ncID
	"?api-version=2018-03-01",
	addresses{[]string{"127.0.0.1", "10.0.0.9"}},
	netinfo{"AzureNet"},
	subnetreq{"0bb4132b-5932-4ec4-a47f-1b24f0eeb426"},
	requestv2{allocationrequest{"subnet1", "10.0.0.7", ""}, associationinfo{"testnode", "10.1.1.2"}},
}

// TestMain :- main
func TestMain(m *testing.M) {
	/// Create Node and Network for testing
	if err := Setup(); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

// Setup :- Setup testing environment by creating node and subnet
//		and joining a network.
func Setup() error {
	fmt.Println("Setting up test environment.")
	var response cns.Response
	// create node...
	var body bytes.Buffer
	json.NewEncoder(&body).Encode(testcase.addr)
	request, err := http.Post(testcase.dest+"/nodes/testnode"+testcase.apiVer, "application/json", &body)

	if err != nil {
		fmt.Printf("Create node failed: %s\n", err.Error())
		return err
	}

	if err = decodeResponse(request, &response); !strings.Contains(err.Error(), "201") || response.ReturnCode != 0 {
		fmt.Printf("Create node failed: %s -> %d\n", err.Error(), response.ReturnCode)
		return err
	}

	fmt.Println("testnode created successfully.")
	// create network...
	json.NewEncoder(&body).Encode(testcase.ninfo)
	request, err = http.Post(testcase.dest+"/networks/"+testcase.vnetID+testcase.apiVer, "application/json", &body)

	if err != nil {
		fmt.Printf("Create network failed: %s\n", err.Error())
		return err
	}

	if err = decodeResponse(request, &response); !strings.Contains(err.Error(), "201") || response.ReturnCode != 0 {
		fmt.Printf("Create network failed: %s\n", err.Error())
		return err
	}
	// Wait for create network completion?
	time.Sleep(5000 * time.Millisecond)
	fmt.Println("test-vnetID created successfully.")
	// join subnet...
	json.NewEncoder(&body).Encode(testcase.ninfo)
	request, err = http.Post(testcase.dest+"/networks/"+testcase.vnetID+"/subnets/subnet1"+testcase.apiVer, "application/json", &body)

	if err != nil {
		fmt.Printf("Join subnet failed: %s\n", err.Error())
		return err
	}

	if err = decodeResponse(request, &response); !strings.Contains(err.Error(), "201") || response.ReturnCode != 0 {
		fmt.Printf("Join subnet failed: %s\n", err.Error())
		return err
	}

	fmt.Println("Joined subnet1.")
	return nil
}

func TestCreateNetworkContainer(t *testing.T) {
	fmt.Println("Testing CreateNetworkContainer")
	var body bytes.Buffer
	json.NewEncoder(&body).Encode(testcase.reqv2)
	request, err := http.NewRequest(http.MethodPost, testcase.dest+"/networks/"+testcase.vnetID+
		"/networkcontainer/"+testcase.ncID+testcase.apiVer, &body)

	if err != nil {
		t.Fatal(err)
	}

	response, resperr := client.Do(request)

	/// Check if err status code = 201???
	if response.StatusCode != 201 || resperr != nil {
		t.Errorf("TestCreateNetworkContainer failed with response %+v Err:%+v", response, resperr)
		t.Fatal(resperr)
	}

	//assert.NotNil(t, resperr, "Response error should not be nil.")
	//assert.Equal(t, 201, response.StatusCode, "Expecting status code 201 for success.")
	fmt.Printf("CreateNetworkContainerRequest succeeded with response %+v\n", response)
}

func TestDeleteNetworkContainer(t *testing.T) {
	time.Sleep(30000 * time.Millisecond)
	fmt.Println("Testing DeleteNetworkContainer")
	fmt.Println("IPCONFIG now for results before deletion...")
	time.Sleep(10000 * time.Millisecond)
	request, err := http.NewRequest(http.MethodDelete, testcase.dest+"/networks/"+testcase.vnetID+
		"/networkcontainer/"+testcase.ncID+testcase.apiVer, nil)

	if err != nil {
		t.Fatal(err)
	}

	response, resperr := client.Do(request)

	/// Check if err status code = 200???
	if response.StatusCode != 200 || resperr != nil {
		t.Errorf("TestDeleteNetworkContainer failed with response %+v Err:%+v", response, resperr)
		t.Fatal(err)
	}

	//assert.NotNil(t, resperr, "Response error should not be nil.")
	//assert.Equal(t, 200, response.StatusCode, "Expecting status code 200 for success.")

	fmt.Printf("DeleteNetworkContainerRequest succeeded with response %+v\n", response)
}
