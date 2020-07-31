package main

import (
	"dnc/api"
	"dnc/client"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-container-networking/cns"
	"github.com/google/uuid"
)

const (
	cnsEP = "127.0.0.1"

	nc1IP = "10.0.1.4"
	nc2IP = "10.0.1.5"

	cnsyml = "cns.yaml"
	dncyml = "dnc.yaml"

	accountName  = "acnteststprod"
	subnetID     = "delegated"
	podNamespace = "default"
	nc1          = "570ccdcc-63b4-42cd-ab88-0238ee549c6f"
	podName1     = "container1"
	nc2          = "0bf30ce0-c61e-4c88-8c1b-ac8019f42bf2"
	podName2     = "container2"
	nc3          = "7df265e1-f1ba-45df-8e2b-62b93d889032"
	podName3     = "container3mnat"

	defaultRequestTimeout   time.Duration = time.Second * 4
	defaultNCRequestTimeout time.Duration = time.Second * 20
	defaultSleep            time.Duration = time.Second * 1
	errdRequestTimeout      time.Duration = time.Second * 30

	running = "Running"

	datapathReportFile = "./datapathReport.json"

	privateLinkDomain = ".privatelink.blob.core.windows.net"
)

// Runner - Listens to Jarvis runner (client) to run tests and retrieve results
type Runner struct {
	server       *httptest.Server
	responseFunc func(http.ResponseWriter, *http.Request)
}

// SwiftDatapath - datapath report that will be serialzed to json
type SwiftDatapath struct {
	ServiceTunnel          bool
	Nc2Nc                  bool
	Nc2VM                  bool
	Vnet2VnetGW            bool
	Vnet2VnetPeering       bool
	NonRFCVnet2VnetPeering bool
	NSG                    bool
	NSGwTags               bool
	UDR                    bool
	PrivateLink            bool
	ExternalDNS            bool
	PrivateDNS             bool
	MNAT                   bool
	LastError              string
}

var (
	dncEP       string
	runner      = new(Runner)
	dnctl       *client.DncClient
	testRunning bool
	testLock    sync.Mutex
	// storage vars
	partitionKey string
	storageKey   string
	blob         string
	container    string
	// nodes
	node1 string
	node2 string
	// node interface
	iface1 string
	iface2 string
	// vnet vars
	vnetID                   string
	authenticationToken      string
	vnetWithMNATID           string
	authTokenForVnetWithMNAT string
	// NC context
	context  = []byte(`{"PodName":"` + podName1 + `","PodNamespace":"` + podNamespace + `"}`)
	context2 = []byte(`{"PodName":"` + podName2 + `","PodNamespace":"` + podNamespace + `"}`)
	context3 = []byte(`{"PodName":"` + podName3 + `","PodNamespace":"` + podNamespace + `"}`)
	// blob endpoint
	blobEP   string
	plblobEP string
	// regex for private DNS
	regPrivateDNS = regexp.MustCompile(`https://(.*).blob.core.windows.net/swiftpl/testblob.txt`)
	privateDNS    []string
	// MNAT resource public ip
	mnatPublicIP string
)

func init() {
	var found bool
	// clean up old services & start them again
	exec.Command("kubectl", "delete", "-f", cnsyml).Run()
	exec.Command("kubectl", "delete", "-f", dncyml).Run()
	time.Sleep(10 * time.Second)
	exec.Command("kubectl", "apply", "-f", cnsyml).Run()
	exec.Command("kubectl", "apply", "-f", dncyml).Run()

	// retrieve necessary values
	blobEP, found = os.LookupEnv("SERVICE_TUNNEL_EP")
	if !found {
		os.Exit(1)
	}

	blob, found = os.LookupEnv("BLOB")
	if !found {
		os.Exit(2)
	}

	container, found = os.LookupEnv("CONTAINER")
	if !found {
		os.Exit(3)
	}

	node1, found = os.LookupEnv("NODE1")
	if !found {
		os.Exit(4)
	}
	jsonPath := "jsonpath=\"{.items[?(@.metadata.name==\"" + node1 + "\")].status.addresses[?(@.type==\"InternalIP\")].address}\""
	output, _ := exec.Command("kubectl", "get", "nodes", "-o", jsonPath).CombinedOutput()
	iface1 = string(output[1 : len(output)-1])

	dncEP = "http://" + iface1 + ":9000"

	node2, found = os.LookupEnv("NODE2")
	if !found {
		os.Exit(5)
	}
	jsonPath = "jsonpath=\"{.items[?(@.metadata.name==\"" + node2 + "\")].status.addresses[?(@.type==\"InternalIP\")].address}\""
	output, _ = exec.Command("kubectl", "get", "nodes", "-o", jsonPath).CombinedOutput()
	iface2 = string(output[1 : len(output)-1])

	vnetID, found = os.LookupEnv("CUST_VNET")
	if !found {
		os.Exit(6)
	}

	partitionKey, found = os.LookupEnv("PARTITION_KEY")
	if !found {
		os.Exit(7)
	}

	storageKey, found = os.LookupEnv("STORAGE_KEY")
	if !found {
		os.Exit(8)
	}

	authenticationToken, found = os.LookupEnv("AUTHENTICATION_TOKEN")
	if !found {
		os.Exit(9)
	}

	plblobEP, found = os.LookupEnv("PRIVATELINK_EP")
	if !found {
		os.Exit(10)
	}

	privateDNS = regPrivateDNS.FindStringSubmatch(plblobEP)
	if len(privateDNS) < 2 {
		os.Exit(11)
	}

	vnetWithMNATID, found = os.LookupEnv("CUST_VNET_WITH_MNAT")
	if !found || len(vnetWithMNATID) <= 0 {
		os.Exit(12)
	}

	// Ensure it is a valid GUID
	_, parseErr := uuid.Parse(vnetWithMNATID)
	if parseErr != nil {
		os.Exit(13)
	}

	authTokenForVnetWithMNAT, found = os.LookupEnv("AUTHENTICATION_TOKEN_CUST_VNET_WITH_MNAT")
	if !found || len(authTokenForVnetWithMNAT) <= 0 {
		os.Exit(14)
	}

	// Ensure it is a valid GUID
	_, parseErr = uuid.Parse(authTokenForVnetWithMNAT)
	if parseErr != nil {
		os.Exit(15)
	}

	mnatPublicIP, found = os.LookupEnv("MNAT_PUBLIC_IP")
	if !found || len(net.ParseIP(mnatPublicIP)) <= 0 {
		os.Exit(16)
	}

	runner.responseFunc = responder
	runner.setupServer(":9009")
	dnctl = client.NewDncClient(dncEP)
}

func main() {
	runner.server.Start()
	select {}
}

func (s *Runner) setupServer(endpoint string) {
	s.server = httptest.NewUnstartedServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if s.responseFunc != nil {
				s.responseFunc(w, r)
			}
		}))
	l, _ := net.Listen("tcp", endpoint)
	s.server.Listener = l
}

func responder(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.RequestURI, "swiftRunner") {
		switch r.Method {
		case "POST":
			handlePost(w, r)
		case "GET":
			handleGet(w, r)
		case "DELETE":
			handleDelete(w, r)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func handlePost(w http.ResponseWriter, r *http.Request) {
	testLock.Lock()
	if !testRunning {
		testRunning = true
	}
	testLock.Unlock()
	w.WriteHeader(http.StatusAccepted)
	go e2eTests()
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	testLock.Lock()
	if testRunning {
		testLock.Unlock()
		w.WriteHeader(http.StatusAccepted)
	} else {
		testLock.Unlock()
		report, err := retrieveDatapathReport()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(report)
		}
	}
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusAccepted)
	go func() {
		for start := time.Now(); time.Since(start) < time.Minute*2; {
			testLock.Lock()
			if testRunning {
				testLock.Unlock()
				time.Sleep(10 * time.Second)
			} else {
				break
			}
		}
		swiftTeardown()
		testLock.Unlock()
	}()
}

func e2eTests() {
	var (
		err     error
		lastErr error
		report  = SwiftDatapath{
			ServiceTunnel:          true,
			Nc2Nc:                  true,
			Nc2VM:                  true,
			Vnet2VnetGW:            true,
			Vnet2VnetPeering:       true,
			NonRFCVnet2VnetPeering: true,
			NSG:                    true,
			NSGwTags:               true,
			UDR:                    true,
			PrivateLink:            true,
			ExternalDNS:            true,
			PrivateDNS:             true,
			MNAT:                   true}
		srcPod, pod2, pod3, podIP1, podIP2 string
		output                             []byte
	)

	if err = swiftSetup(); err != nil {
		goto WriteJsonReport
	}

	// sleep and wait for pod creation (specifically ip allocation)
	time.Sleep(defaultNCRequestTimeout)

	// get details to exec into pods and ping each other
	srcPod, podIP1, err = getPodExecInfo(podName1)
	if err != nil {
		goto WriteJsonReport
	}

	pod2, podIP2, err = getPodExecInfo(podName2)
	if err != nil {
		goto WriteJsonReport
	}

	pod3, _, err = getPodExecInfo(podName3)
	if err != nil {
		goto WriteJsonReport
	}

	if podIP1 != nc1IP {
		temp := srcPod
		srcPod = pod2
		pod2 = temp
	}

	// test storage endpoint (service tunneling)
	_, err = exec.Command("kubectl", "exec", srcPod, "--", "wget", blobEP, "-O", "testblob.txt").CombinedOutput()
	if err != nil {
		report.ServiceTunnel = false
		lastErr = fmt.Errorf("ServiceTunnel failed: %+v", err)
	}
	// test nic to nic connection
	_, err = exec.Command("kubectl", "exec", srcPod, "--", "ping", "-c", "3", podIP2).CombinedOutput()
	if err != nil {
		report.Nc2Nc = false
		lastErr = fmt.Errorf("Nc2Nc failed: %+v", err)
	}
	// test nic to vm connection
	_, err = exec.Command("kubectl", "exec", srcPod, "--", "ping", "-c", "3", "10.0.0.4").CombinedOutput()
	if err != nil {
		report.Nc2VM = false
		lastErr = fmt.Errorf("Nc2VM failed: %+v", err)
	}
	// test gateway (vnet to vnet)
	_, err = exec.Command("kubectl", "exec", srcPod, "--", "ping", "-c", "3", "20.0.0.4").CombinedOutput()
	if err != nil {
		report.Vnet2VnetGW = false
		lastErr = fmt.Errorf("Vnet2VnetGW failed: %+v", err)
	}
	// test peering (vnet to vnet) RFC1918
	_, err = exec.Command("kubectl", "exec", srcPod, "--", "ping", "-c", "3", "172.16.0.4").CombinedOutput()
	if err != nil {
		report.Vnet2VnetPeering = false
		lastErr = fmt.Errorf("Vnet2VnetPeering failed: %+v", err)
	}
	// test peering (vnet to vnet) Non-RFC1918
	_, err = exec.Command("kubectl", "exec", srcPod, "--", "ping", "-c", "3", "30.0.0.4").CombinedOutput()
	if err != nil {
		report.NonRFCVnet2VnetPeering = false
		lastErr = fmt.Errorf("NonRFCVnet2VnetPeering failed: %+v", err)
	}
	// test UDR (pod1->10.0.0.5 should fail because the next hop is 10.0.1.5 and 10.0.1.5->10.0.0.5 is blocked by nsg)
	_, err = exec.Command("kubectl", "exec", srcPod, "--", "ping", "-c", "3", "10.0.0.5").CombinedOutput()
	if err == nil {
		report.UDR = false
		lastErr = fmt.Errorf("UDR failed")
	}
	// test Private Link
	_, err = exec.Command("kubectl", "exec", srcPod, "--", "wget", plblobEP, "-O", "testblob.txt").CombinedOutput()
	if err != nil {
		report.PrivateLink = false
		lastErr = fmt.Errorf("Private Link failed: %+v", err)
	}
	// test NSG (ping to default subnet from 10.0.1.5 should be blocked/expected to fail)
	_, err = exec.Command("kubectl", "exec", pod2, "--", "ping", "-c", "3", "10.0.0.5").CombinedOutput()
	if err == nil {
		report.NSG = false
		lastErr = fmt.Errorf("NSG failed: %+v", err)
	}
	// test NSG with service tags
	_, err = exec.Command("kubectl", "exec", pod2, "--", "wget", "google.com", "-O", "index.html").CombinedOutput()
	if err == nil {
		report.NSGwTags = false
		lastErr = fmt.Errorf("NSGwTags failed: pod2 should not have outbound internet access")
	}
	// test external DNS resolution
	output, err = exec.Command("kubectl", "exec", srcPod, "--", "dig", "+short", "google.com").CombinedOutput()
	if err != nil || net.ParseIP(string(output[1:len(output)-1])) == nil {
		report.ExternalDNS = false
		lastErr = fmt.Errorf("ExternalDNS failed: unable to resolve google.com")
	}
	// test private link DNS resolution
	output, err = exec.Command("kubectl", "exec", srcPod, "--", "dig", "+short", privateDNS[1]+privateLinkDomain).CombinedOutput()
	if err != nil || net.ParseIP(string(output[1:len(output)-1])) == nil {
		report.PrivateDNS = false
		lastErr = fmt.Errorf("PrivateDNS failed: unable resolve %+v with err %+v", privateDNS[1]+privateLinkDomain, err)
	}

	// test MNAT, the link below basically gives the src ip of the outbound packet
	output, err = exec.Command("kubectl", "exec", pod3, "--", "curl", "https://api.ipify.org?format=json").CombinedOutput()
	if err != nil {
		report.MNAT = false
		lastErr = fmt.Errorf("Managed NAT failed: MNAT verification command failed with error %+v", err)
	}

	// Check if the command output is non-empty
	if report.MNAT && len(output) == 0 {
		report.MNAT = false
		lastErr = fmt.Errorf("Managed NAT failed: MNAT verification command output is empty.")
	}

	// Check if the command output has MNAT public IP
	if report.MNAT && !strings.Contains(string(output[1:len(output)-1]), mnatPublicIP) {
		report.MNAT = false
		lastErr = fmt.Errorf("Managed NAT failed: MNAT verification command output: %s does not contain MNAT public IP %s",
			string(output),
			mnatPublicIP)
	}

	// clean up all the NC states for next test (we only clean up when datapath is healthy)
	if lastErr == nil {
		err = swiftTeardown()
	} else {
		report.LastError = lastErr.Error()
	}

WriteJsonReport:
	// write failure to json
	if err != nil {
		log.Printf("e2eTests failed: %+v", err)
		report.LastError = err.Error()
	}

	jsonData, err := json.Marshal(report)
	if err == nil {
		jsonFile, err := os.Create(datapathReportFile)
		if err == nil {
			jsonFile.Write(jsonData)
			jsonFile.Close()
		}
	}

	testLock.Lock()
	testRunning = false
	testLock.Unlock()
}

func swiftSetup() error {
	var (
		orchestratorType = cns.Kubernetes
		containerType    = cns.AzureContainerInstance
		respErr          *api.JsonErr
		err              error
	)

	if respErr = dnctl.InitAzStorage(accountName, storageKey, partitionKey, blob, container); respErr.HTTPStatus != http.StatusOK {
		return fmt.Errorf(respErr.String())
	}

	if err = addNode(node1, iface1, orchestratorType, vnetID); err != nil {
		return err
	}

	if err = addNode(node2, iface2, orchestratorType, vnetID); err != nil {
		return err
	}

	if err = addNetwork(vnetID, defaultNCRequestTimeout); err != nil {
		return err
	}

	if err = addNetwork(vnetWithMNATID, defaultNCRequestTimeout); err != nil {
		return err
	}

	if err = addSubnet(vnetID, subnetID, authenticationToken); err != nil {
		return err
	}

	if err = addSubnet(vnetWithMNATID, subnetID, authTokenForVnetWithMNAT); err != nil {
		return err
	}

	if err = addNc(node1, vnetID, nc1, "", subnetID, nc1IP, "", iface1, containerType, podName1, podNamespace, false, false, defaultNCRequestTimeout); err != nil {
		return err
	}

	if err = addNc(node2, vnetID, nc2, "", subnetID, nc2IP, "", iface2, containerType, podName2, podNamespace, false, false, defaultNCRequestTimeout); err != nil {
		return err
	}

	// NC for MNAT on node1 (10.0.1.4)
	if err = addNc(node1, vnetWithMNATID, nc3, "", subnetID, nc1IP, "", iface1, containerType, podName3, podNamespace, false, false, defaultNCRequestTimeout); err != nil {
		return err
	}

	if err = createPod(podName1); err != nil {
		return err
	}

	if err = createPod(podName2); err != nil {
		return err
	}

	if err = createPod(podName3); err != nil {
		return err
	}

	return nil
}

func swiftTeardown() (err error) {
	var lastErr error

	deletePod(podName1)
	deletePod(podName2)
	deletePod(podName3)

	if lastErr = deleteNc(vnetID, nc1); err != nil {
		err = lastErr
	}

	if lastErr = deleteNc(vnetID, nc2); err != nil {
		err = lastErr
	}

	if lastErr = deleteNc(vnetWithMNATID, nc3); err != nil {
		err = lastErr
	}

	return err
}

func addNode(nodeID, ipaddr, orchestratorType, vnetID string) error {
	var (
		respErr *api.JsonErr
		nodeCfg *api.NodeInfoResponse
	)

	if respErr = dnctl.AddNode(nodeID, ipaddr, orchestratorType, vnetID); respErr.HTTPStatus != http.StatusOK {
		return fmt.Errorf("Register node (create) failed with error %+v", respErr)
	}

	for start := time.Now(); ; {
		nodeCfg, respErr = dnctl.GetNode(nodeID)
		if respErr.HTTPStatus == http.StatusOK {
			break
		}

		if respErr.HTTPStatus == http.StatusAccepted && time.Since(start) < defaultRequestTimeout {
			time.Sleep(defaultSleep)
			continue
		}

		return fmt.Errorf("Register node (get) failed with error %+v", respErr)
	}

	receivedNodeCfg := nodeCfg.NodeInfo
	expectedNodeCfg := api.NodeInfoRequest{
		IPAddresses:      []string{ipaddr},
		LocationType:     "",
		OrchestratorType: orchestratorType}
	if receivedNodeCfg.String() != expectedNodeCfg.String() {
		return fmt.Errorf("Register node failed;\nreceived %+v\nexpected %+v", receivedNodeCfg, expectedNodeCfg)
	}

	return nil
}

func deleteNode(nodeID string, status int) error {
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

		return fmt.Errorf("Unregister node failed with error %+v", respErr)
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

			return fmt.Errorf("Unregister node failed with error %+v", respErr)
		}
	}

	return nil
}

func addNetwork(vnetID string, timeout time.Duration) error {
	var (
		respErr               *api.JsonErr
		vnetStatus            *api.RequestStatus
		expectedNetworkStatus = &api.RequestStatus{
			State:  "Join",
			Status: "Completed"}
	)

	if respErr = dnctl.AddNetwork(vnetID); respErr.HTTPStatus != http.StatusOK {
		return fmt.Errorf("Register network failed with error %+v", respErr)
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

		return fmt.Errorf("Register network failed with error %+v", respErr)
	}

	if !reflect.DeepEqual(vnetStatus, expectedNetworkStatus) {
		return fmt.Errorf("Register network failed;\nreceived %+v\nexpected %+v", vnetStatus, expectedNetworkStatus)
	}

	return nil
}

func deleteNetwork(vnetID string, status int) error {
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

		return fmt.Errorf("Unregister network failed with error %+v", respErr)
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

			return fmt.Errorf("Unregister network failed with error %+v", respErr)
		}
	}

	return nil
}

func addSubnet(vnetID, subnetID, authToken string) error {
	var (
		respErr              *api.JsonErr
		subnetStatus         *api.RequestStatus
		expectedSubnetStatus = &api.RequestStatus{
			State:  "Join",
			Status: "Completed"}
	)

	if respErr = dnctl.AddSubnet(vnetID, subnetID, authToken); respErr.HTTPStatus != http.StatusOK {
		return fmt.Errorf("Register subnet failed with error %+v", respErr)
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

		return fmt.Errorf("Register subnet failed with error %+v", respErr)
	}

	if !reflect.DeepEqual(subnetStatus, expectedSubnetStatus) {
		return fmt.Errorf("Register subnet failed;\nreceived %+v\nexpected %+v", subnetStatus, expectedSubnetStatus)
	}

	return nil
}

func deleteSubnet(vnetID, subnetID string) error {
	var respErr *api.JsonErr

	for start := time.Now(); ; {
		respErr = dnctl.DeleteSubnet(vnetID, subnetID)
		if respErr.HTTPStatus == http.StatusOK {
			break
		}

		if respErr.HTTPStatus == http.StatusAccepted && time.Since(start) < defaultRequestTimeout {
			time.Sleep(defaultSleep)
			continue
		}

		return fmt.Errorf("Unregister subnet failed with error %+v", respErr)
	}

	return nil
}

func addReservation(nodeID, vnetID, subnetID, ipConstraint, reservationID string, timeout time.Duration) error {
	var (
		respErr                   *api.JsonErr
		reservationStatus         *api.RequestStatus
		expectedReservationStatus = &api.RequestStatus{
			State:  "Process",
			Status: "Completed"}
	)

	if respErr = dnctl.AddReservation(vnetID, subnetID, ipConstraint, nodeID, reservationID); respErr.HTTPStatus != http.StatusOK {
		return fmt.Errorf("Register reservation failed with error %+v", respErr)
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

		return fmt.Errorf("Register reservation failed with error %+v", respErr)
	}

	if !reflect.DeepEqual(reservationStatus, expectedReservationStatus) {
		return fmt.Errorf("Register reservation failed;\nrecieved %+v\nexpected %+v", reservationStatus, expectedReservationStatus)
	}

	return nil
}

func deleteReservation(vnetID, reservationID string) error {
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

		return fmt.Errorf("Unregister reservation failed with error %+v", respErr)
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

		return fmt.Errorf("Unregister reservation failed with error %+v", respErr)
	}

	return nil
}

func addNc(nodeID, vnetID, ncID, reservationID, subnetID, ipConstraint, nodeConstraint, ifaceID, containerType, podName, podNamespace string, host2NC, NC2Host bool, timeout time.Duration) error {
	var (
		respErr          *api.JsonErr
		ncStatus         *api.RequestStatus
		expectedNcStatus = &api.RequestStatus{
			State:  "Program",
			Status: "Completed"}
	)

	if respErr = dnctl.AddNc(nodeID, vnetID, ncID, "", subnetID, "", "", ifaceID, cns.AzureContainerInstance, podName, podNamespace, false, false); respErr.HTTPStatus != http.StatusOK {
		return fmt.Errorf("Create nc failed with error %+v", respErr)
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

		return fmt.Errorf("Create nc failed with error %+v", respErr)
	}

	if !reflect.DeepEqual(ncStatus, expectedNcStatus) {
		expectedNcStatus.State = "RefreshAssociation"
		if !reflect.DeepEqual(ncStatus, expectedNcStatus) {
			return fmt.Errorf("Create nc failed;\nreceived %+v\nexpected %+v", ncStatus, expectedNcStatus)
		}
	}

	return nil
}

func deleteNc(vnetID, ncID string) error {
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

		return fmt.Errorf("Delete nc failed with error %+v", respErr)
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

		return fmt.Errorf("Delete nc failed with error %+v", respErr)
	}

	return nil
}

func deleteHelper(status int) (expectedStatus int, validateDelete bool) {
	expectedStatus = status
	if status == http.StatusNotFound {
		expectedStatus, validateDelete = http.StatusOK, true
	}
	return
}

func retrieveDatapathReport() (report *SwiftDatapath, err error) {
	jsonFile, err := os.Open(datapathReportFile)
	if err != nil {
		log.Printf("Failed to retrieve datapath report with error: %+v", err)
		return nil, err
	}

	defer jsonFile.Close()
	bytes, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Printf("Failed to retrieve datapath report with error: %+v", err)
		return nil, err
	}
	if err = json.Unmarshal(bytes, &report); err != nil {
		log.Printf("Failed to retrieve datapath report with error: %+v", err)
		return nil, err
	}

	return
}

func createPod(podName string) error {
	// podName is either container1/container2
	return exec.Command("kubectl", "apply", "-f", podName+".yaml").Run()
}

func deletePod(podName string) error {
	// podName is either container1/container2
	return exec.Command("kubectl", "delete", "-f", podName+".yaml").Run()
}

func getPodExecInfo(podName string) (containerID string, podIP string, err error) {
	// get container ID and remove wrapping quotes
	output, _ := exec.Command("kubectl", "get", "pods", "-l", "app="+podName, "-o", "jsonpath=\"{.items[0].metadata.name}\"").CombinedOutput()
	containerID = string(output[1 : len(output)-1])
	// wait for container to become ready (up to 2 minutes)
	for start := time.Now(); time.Since(start) < time.Minute*2; time.Sleep(5 * time.Second) {
		output, _ = exec.Command("kubectl", "get", "pods", containerID, "-o", "jsonpath=\"{.status.phase}\"").CombinedOutput()
		if result := string(output[1 : len(output)-1]); result == running {
			output, _ = exec.Command("kubectl", "get", "pods", containerID, "-o", "jsonpath=\"{.status.podIP}\"").CombinedOutput()
			result = string(output[1 : len(output)-1])
			if net.ParseIP(result).To4() != nil {
				podIP = result
				break
			}
		}
	}

	if podIP == "" {
		return "", "", fmt.Errorf("Failed to retrieve pod IP")
	}

	return containerID, podIP, nil
}
