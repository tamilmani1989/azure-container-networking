package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// JSONErr - Struct to hold err info.
type JSONErr struct {
	Code       ErrorCode `json:"ErrorCode"`
	Text       string    `json:"Text"`
	HTTPStatus int       `json:"HttpStatusCode"`
}

// AzureStorageRequest - Details to connect to azure storage account.
type AzureStorageRequest struct {
	AccountName  string `json:"AccountName"`
	Key          string `json:"Key"`
	PartitionKey string `json:"PartitionKey"`
}

// NodeInfoRequest - Node information request.
type NodeInfoRequest struct {
	IPAddresses      []string `json:"ipaddresses"`
	LocationType     string   `json:"locationtype"` // Optional.
	OrchestratorType string   `json:"orchestratortype"`
}

// NetworkInfoRequest - Network info.
type NetworkInfoRequest struct {
	NetworkType  string `json:"NetworkType"`
	NetworkSpace string `json:"NetworkSpace"` // Optional, not required for azure vnets
}

// SubnetRequest - Put subnet request body.
type SubnetRequest struct {
	AuthenticationToken string
}

// AllocationRequest - Struct to hold allocation request.
type AllocationRequest struct {
	SubnetName     string `json:"SubnetName"`
	IPConstraint   string `json:"IPConstraint"`
	NodeConstraint string `json:"NodeConstraint"`
}

// AssociationInfo - Struct to hold network container association info.
type AssociationInfo struct {
	NodeID              string          `json:"NodeID"`
	InterfaceID         string          `json:"InterfaceID"` // IPAddress of the interface.
	ContainerType       string          `json:"ContainerType"`
	OrchestratorContext json.RawMessage `json:"OrchestratorContext"`
}

// CreateNetworkContainerRequest - Struct to hold network container request.
type CreateNetworkContainerRequest struct {
	AllocationRequest AllocationRequest
	AssociationInfo   AssociationInfo
}

// IPConfiguration - Struct to hold the ip configuration.
type IPConfiguration struct {
	NetworkID  string   `json:"NetworkID"`
	SubnetName string   `json:"SubnetName"`
	IPSubnet   string   `json:"IPSubnet"`
	IPAddress  string   `json:"IPAddress"`
	GatewayIP  string   `json:"GatewayIP"`
	DNSServers []string `json:"DnsServers"`
}

// JsonErr - Struct to hold err info
type JsonErr struct {
	Code       ErrorCode `json:"ErrorCode"`
	Text       string    `json:"Text"`
	HTTPStatus int       `json:"HttpStatusCode"`
}

// RequestStatus - Struct to hold request status.
type RequestStatus struct {
	State         string  `json:"State"`
	Status        string  `json:"Status"`
	FailureDetail JsonErr `json:"FailureDetail"`
}

// NetworkContainerResponse - Struct to hold network container response.
type NetworkContainerResponse struct {
	NcID            string          `json:"NetworkContainerID"`
	IPConfiguration IPConfiguration `json:"IPConfiguration"`
	AssociationInfo AssociationInfo `json:"AssociationInfo"`
	RequestStatus   RequestStatus   `json:"RequestStatus"`
	VlanID          uint16          `json:"VlanID"`
}

// OrchestratorContext - Struct to hold orchestrator context info.
type OrchestratorContext struct {
	PodName      string
	PodNamespace string
}

// Server - simple https server to handle GET calls for retrieving dnc datapath statuses
type Server struct {
	Server       *httptest.Server
	ResponseFunc func(http.ResponseWriter, *http.Request)
	Lock         sync.Mutex
}

// DncDataPath - struct of datapath success indicators
type DncDataPath struct {
	ServiceTunnel    bool
	Nic2Nic          bool
	Nic2VM           bool
	Vnet2VnetGW      bool
	Vnet2VnetPeering bool
	LastError        string
}

// Healthy - returns true if all tests have passed
func (r *DncDataPath) Healthy() bool {
	return r.Nic2Nic && r.Nic2VM && r.ServiceTunnel
}

const (
	running = "Running"
	cnsyml  = "cns.yaml"
	dncyml  = "dnc.yaml"
	pod0IP  = "10.0.1.4"
	pod1IP  = "10.0.1.5"
)

var (
	dncEP           = "http://127.0.0.1:9000"
	blobEP          = ""
	delegatedSubnet = "delegated"
	apiVersion      = "?api-version=2018-03-01"
	ncURL           = ""
	ncURL2          = ""
	ncID            = "570ccdcc-63b4-42cd-ab88-0238ee549c6f"
	ncID2           = "0bf30ce0-c61e-4c88-8c1b-ac8019f42bf2"
	nodeReq         = NodeInfoRequest{
		IPAddresses:      []string{"127.0.0.1"},
		LocationType:     "",
		OrchestratorType: "Kubernetes"}
	networkReq = NetworkInfoRequest{NetworkType: "AzureNet"}
	// container0 specs
	podInfo = OrchestratorContext{
		PodName:      "container0",
		PodNamespace: "default"}
	context, _           = json.Marshal(podInfo)
	networkContainerReq0 = CreateNetworkContainerRequest{
		AllocationRequest: AllocationRequest{
			SubnetName:     delegatedSubnet,
			IPConstraint:   "",
			NodeConstraint: ""},
		AssociationInfo: AssociationInfo{
			NodeID:              "node0",
			InterfaceID:         "10.240.0.4",
			ContainerType:       "AzureContainerInstance",
			OrchestratorContext: context}}
	// container1 specs
	podInfo1 = OrchestratorContext{
		PodName:      "container1",
		PodNamespace: "default"}
	context1, _          = json.Marshal(podInfo1)
	networkContainerReq1 = CreateNetworkContainerRequest{
		AllocationRequest: AllocationRequest{
			SubnetName:     delegatedSubnet,
			IPConstraint:   "",
			NodeConstraint: ""},
		AssociationInfo: AssociationInfo{
			NodeID:              "node0",
			InterfaceID:         "10.240.0.4",
			ContainerType:       "AzureContainerInstance",
			OrchestratorContext: context1}}
	storageURL = ""
	nodeURL    = dncEP + "/nodes/" + networkContainerReq0.AssociationInfo.NodeID + apiVersion
	networkURL = ""
	subnetURL  = ""
	tr         = &http.Transport{
		MaxIdleConnsPerHost: 1024,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 300 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 0 * time.Second}
	client      = &http.Client{Transport: tr}
	server      = new(Server)
	testLock    sync.Mutex
	testRunning = false
)

func init() {
	exec.Command("kubectl", "delete", "-f", cnsyml).Run()
	exec.Command("kubectl", "delete", "-f", dncyml).Run()
	time.Sleep(10 * time.Second)
	server.setupServer(":9009")
	server.ResponseFunc = responder
	if err := createPod(dncyml); err != nil {
		os.Exit(1)
	}
	if err := createPod(cnsyml); err != nil {
		os.Exit(2)
	}

	custVnet, found := os.LookupEnv("CUST_VNET")
	if !found {
		os.Exit(3)
	}

	blobEP, found = os.LookupEnv("SERVICE_TUNNEL_EP")
	if !found {
		os.Exit(4)
	}

	blobName, found := os.LookupEnv("BLOB")
	if !found {
		os.Exit(5)
	}

	containerName, found := os.LookupEnv("CONTAINER")
	if !found {
		os.Exit(6)
	}

	nodeName, found := os.LookupEnv("NODE")
	if found {
		jsonPath := "jsonpath=\"{.items[?(@.metadata.name==\"" + nodeName + "\")].status.addresses[?(@.type==\"InternalIP\")].address}\""
		output, _ := exec.Command("kubectl", "get", "nodes", "-o", jsonPath).CombinedOutput()
		nodeIP := string(output[1 : len(output)-1])
		networkContainerReq0.AssociationInfo.InterfaceID = nodeIP
		networkContainerReq1.AssociationInfo.InterfaceID = nodeIP
	}

	storageURL = dncEP + "/azurestoragesettingsv3/blob/" + blobName + "/container/" + containerName + apiVersion
	ncURL = dncEP + "/networks/" + custVnet + "/networkcontainer/" + ncID + apiVersion
	ncURL2 = dncEP + "/networks/" + custVnet + "/networkcontainer/" + ncID2 + apiVersion
	networkURL = dncEP + "/networks/" + custVnet + apiVersion
	subnetURL = dncEP + "/networks/" + custVnet + "/subnets/" + delegatedSubnet + apiVersion
}

func main() {
	select {}
}

func (s *Server) setupServer(endpoint string) {
	s.Server = httptest.NewUnstartedServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if s.ResponseFunc != nil {
				s.ResponseFunc(w, r)
			}
		}))
	l, _ := net.Listen("tcp", endpoint)
	s.Server.Listener = l
	s.Server.Start()
}

func responder(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.RequestURI, "swiftRunner") {
		switch r.Method {
		case "POST":
			testLock.Lock()
			if !testRunning {
				testRunning = true
			}
			testLock.Unlock()
			go func() {
				initAzStorage()
				createnode()
				joinnetwork()
				time.Sleep(1 * time.Second)
				putsubnet()
				createnc(true)
				triggerDatapathTests(w)
			}()
			w.WriteHeader(http.StatusAccepted)
		case "GET":
			testLock.Lock()
			if testRunning {
				testLock.Unlock()
				w.WriteHeader(http.StatusAccepted)
			} else {
				testLock.Unlock()
				report, err := retrieveDatapathReport()
				if err != nil {
					log.Printf("%+v", err)
					w.WriteHeader(http.StatusInternalServerError)
				} else {
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(report)
				}
			}
		case "DELETE":
			w.WriteHeader(http.StatusAccepted)
			go func() {
				for {
					testLock.Lock()
					if testRunning {
						testLock.Unlock()
						time.Sleep(10 * time.Second)
					} else {
						break
					}
				}
				deletenc(true)
				deletenc(false)
				testLock.Unlock()
			}()
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func createPod(yaml string) (err error) {
	err = exec.Command("kubectl", "apply", "-f", yaml).Run()
	log.Printf("%+v", err)
	return
}

func initAzStorage() (err error) {
	var req AzureStorageRequest
	jsonBytes, err := load("storage.json")
	if err == nil {
		if err = json.Unmarshal(jsonBytes, &req); err == nil {
			var body bytes.Buffer
			var respErr JSONErr

			json.NewEncoder(&body).Encode(req)
			response, err := http.Post(storageURL, "application/json", &body)
			if err == nil {
				defer response.Body.Close()

				if response.StatusCode != http.StatusOK {
					if err = decodeResponse(response, &respErr); err != nil {
						err = fmt.Errorf("Initialize storage failed with message %s [error code: %d]", respErr.Text, respErr.Code)
					}
				}
			}
		}
	}

	log.Printf("%+v", err)
	return
}

func createnode() (err error) {
	var body bytes.Buffer
	var respErr JSONErr

	json.NewEncoder(&body).Encode(nodeReq)
	response, err := http.Post(nodeURL, "application/json", &body)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = decodeResponse(response, &respErr); err != nil {
				err = fmt.Errorf("Create node failed with message %s [error code: %d]", respErr.Text, respErr.Code)
			}
		}
	}

	log.Printf("%+v", err)
	return
}

func joinnetwork() (err error) {
	var body bytes.Buffer
	var respErr JSONErr

	json.NewEncoder(&body).Encode(networkReq)
	response, err := http.Post(networkURL, "application/json", &body)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = decodeResponse(response, &respErr); err != nil {
				err = fmt.Errorf("Join network failed with message %s [error code: %d]", respErr.Text, respErr.Code)
			}
		} else {
			// wait for network join to complete
			time.Sleep(5 * time.Second)
		}
	}

	log.Printf("%+v", err)
	return
}

func putsubnet() (err error) {
	var req SubnetRequest
	jsonBytes, err := load("subnet.json")
	if err == nil {
		if err = json.Unmarshal(jsonBytes, &req); err == nil {
			var body bytes.Buffer
			var respErr JSONErr

			json.NewEncoder(&body).Encode(req)
			response, err := http.Post(subnetURL, "application/json", &body)
			if err == nil {
				defer response.Body.Close()

				if response.StatusCode != http.StatusOK {
					if err = decodeResponse(response, &respErr); err != nil {
						err = fmt.Errorf("Put subnet failed with message %s [error code: %d]", respErr.Text, respErr.Code)
					}
				}
			}
		}
	}

	log.Printf("%+v", err)
	return
}

func createnc(original bool) (err error) {
	var body bytes.Buffer
	var respErr JSONErr
	var tmpURL string
	if original {
		tmpURL = ncURL
		json.NewEncoder(&body).Encode(networkContainerReq0)
	} else {
		tmpURL = ncURL2
		json.NewEncoder(&body).Encode(networkContainerReq1)
	}

	response, err := http.Post(tmpURL, "application/json", &body)
	if err == nil {
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			if err = decodeResponse(response, &respErr); err != nil {
				err = fmt.Errorf("Createnc failed with message %s [error code: %d]", respErr.Text, respErr.Code)
			}
		} else {
			// sleep for 5s NC goal state to be persisted
			time.Sleep(5 * time.Second)

			response, err = http.Get(tmpURL)
			if err == nil {
				var ncresponse NetworkContainerResponse
				if err = decodeResponse(response, ncresponse); err == nil {
					if ncresponse.RequestStatus.FailureDetail.Code == 0 {
						// create container
						var config string
						if original {
							config = "container0.yaml"
						} else {
							config = "container1.yaml"
						}
						exec.Command("kubectl", "apply", "-f", config).Run()
					} else {
						err = fmt.Errorf("%+v", ncresponse.RequestStatus.FailureDetail)
					}
				}
			}
		}
	}

	log.Printf("%+v", err)
	return
}

func deletenc(original bool) (err error) {
	var config string
	if original {
		config = "container0.yaml"
	} else {
		config = "container1.yaml"
	}
	exec.Command("kubectl", "delete", "-f", config).Run()

	var respErr JSONErr
	var tmpURL string
	if original {
		tmpURL = ncURL
	} else {
		tmpURL = ncURL2
	}
	req, _ := http.NewRequest("DELETE", tmpURL, nil)
	response, err := client.Do(req)
	if err == nil {
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			if err = decodeResponse(response, &respErr); err != nil {
				err = fmt.Errorf("Deletenc failed with message %s [error code: %d]", respErr.Text, respErr.Code)
			}
		}
	}

	log.Printf("%+v", err)
	return
}

func retrieveDatapathReport() (report *DncDataPath, err error) {
	jsonFile, err := os.Open("./datapathReport.json")
	if err != nil {
		return nil, err
	}

	defer jsonFile.Close()
	bytes, err := ioutil.ReadAll(jsonFile)
	json.Unmarshal(bytes, &report)
	log.Printf("%+v", err)
	return
}

func decodeResponse(w *http.Response, response interface{}) (err error) {
	if w.Body != nil {
		err = json.NewDecoder(w.Body).Decode(&response)
	} else {
		err = fmt.Errorf("Response body is empty")
	}

	if err != nil {
		return err
	}

	if w.StatusCode != http.StatusOK {
		err = fmt.Errorf("Request failed with HTTP error %d", w.StatusCode)
	}

	log.Printf("%+v", err)
	return
}

// Load the JSON config file
func load(jsonFile string) (jsonBytes []byte, err error) {
	var absPath string
	var input = io.ReadCloser(os.Stdin)
	if absPath, err = filepath.Abs(jsonFile); err == nil {
		if input, err = os.Open(absPath); err == nil {
			// Read the config file
			jsonBytes, err = ioutil.ReadAll(input)
			input.Close()
		}
	}

	return
}
