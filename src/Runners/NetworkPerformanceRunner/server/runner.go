package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Iperf3Received - Struct that contains the throughput in bits_per_second.
type Iperf3Received struct {
	Throughput float64 `json:"bits_per_second"`
}

// Iperf3CPU - Struct that holds iperf3 CPU utilization info.
type Iperf3CPU struct {
	Sender   float64 `json:"host_total"`
	Receiver float64 `json:"remote_total"`
}

// Iperf3Summary - Struct that holds iperf3 summary.
type Iperf3Summary struct {
	SumReceived    Iperf3Received `json:"sum_received"`
	CPUUtilization Iperf3CPU      `json:"cpu_utilization_percent"`
}

// Iperf3JSON - Struct that holds iperf3 info.
type Iperf3JSON struct {
	End   Iperf3Summary `json:"end"`
	Error string        `json:"error"`
}

// Server - simple https server to handle GET calls for retrieving dnc datapath statuses
type Server struct {
	Server       *httptest.Server
	ResponseFunc func(http.ResponseWriter, *http.Request)
	Lock         sync.Mutex
}

// NetworkPerformanceResults - struct of datapath success indicators
type NetworkPerformanceResults struct {
	CPUSender   int
	CPUReceiver int
	Throughput  int64
	LatencyMin  int
	LatencyMax  int
	LatencyAvg  int
}

const (
	running   = "Running"
	completed = "Succeeded"
	clientyml = "iperfc.yaml"
	serveryml = "iperfs.yaml"
)

var (
	tr = &http.Transport{
		MaxIdleConnsPerHost: 1024,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 300 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 0 * time.Second}
	client               = &http.Client{Transport: tr}
	server               = new(Server)
	testLock             sync.Mutex
	testRunning          = false
	compiledLatencyRegex = regexp.MustCompile(`min\/avg\/max\/mdev = (\S+)`)
	compiledIPRegex      = regexp.MustCompile(`[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}`)
)

func init() {
	exec.Command("kubectl", "delete", "-f", clientyml).Run()
	exec.Command("kubectl", "delete", "-f", serveryml).Run()
	time.Sleep(10 * time.Second)
	server.setupServer(":9009")
	server.ResponseFunc = responder
	if err := createPod(serveryml); err != nil {
		os.Exit(1)
	}

	time.Sleep(10 * time.Second)
	output, _ := exec.Command("kubectl", "get", "pods", "-l", "app=iperfs", "-o", "jsonpath=\"{.items[0].status.podIP}\"").CombinedOutput()
	if result := string(output[1 : len(output)-1]); len(result) > 0 {
		log.Printf("%+v", result)
		b, err := ioutil.ReadFile(clientyml)
		if err != nil {
			panic(err)
		}
		ioutil.WriteFile(clientyml, compiledIPRegex.ReplaceAll(b, []byte(result)), 0644)
	} else {
		os.Exit(2)
	}
}

func main() {
	select {}
}

func (s *Server) setupServer(endpoint string) {
	s.Server = httptest.NewUnstartedServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// s.Lock.Lock()
			// defer s.Lock.Unlock()
			if s.ResponseFunc != nil {
				s.ResponseFunc(w, r)
			}
		}))
	l, _ := net.Listen("tcp", endpoint)
	s.Server.Listener = l
	s.Server.Start()
}

func responder(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.RequestURI, "networkPerformanceRunner") {
		switch r.Method {
		case "POST":
			testLock.Lock()
			if !testRunning {
				testRunning = true
				go triggerNetworkPerformanceTests(w)
			}
			testLock.Unlock()
			w.WriteHeader(http.StatusAccepted)
		case "GET":
			testLock.Lock()
			if testRunning {
				testLock.Unlock()
				w.WriteHeader(http.StatusAccepted)
			} else {
				testLock.Unlock()
				report, err := retrieveNetworkPerformance()
				if err != nil {
					log.Printf("%+v", err)
					w.WriteHeader(http.StatusInternalServerError)
				} else {
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(report)
				}
			}
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

func triggerNetworkPerformanceTests(w http.ResponseWriter) (err error) {
	var report NetworkPerformanceResults
	// wait for previous container to finish deletion (up to 2 minutes)
	for start := time.Now(); time.Since(start) < time.Minute*2; time.Sleep(5 * time.Second) {
		output, _ := exec.Command("kubectl", "get", "pod", "iperfc-pod", "-o", "jsonpath=\"{.metadata.name}\"").CombinedOutput()
		if result := string(output); strings.Contains(strings.ToLower(result), "error") {
			break
		}
	}

	// create container and run datapath tests
	if err = createPod(clientyml); err == nil {
		// sleep and wait for pod creation
		time.Sleep(15 * time.Second)
		// wait for container to become ready (up to 2 minutes)
		log.Printf("Waiting for pod to start running...")
		for start := time.Now(); time.Since(start) < time.Minute*2; time.Sleep(5 * time.Second) {
			output, _ := exec.Command("kubectl", "get", "pods", "iperfc-pod", "-o", "jsonpath=\"{.status.phase}\"").CombinedOutput()
			if result := string(output[1 : len(output)-1]); result == running {
				break
			}
		}
		log.Printf("Waiting for pod to be in succeeded/completed state...")
		// wait for container to be in completed state (up to 3 minutes)
		for start := time.Now(); time.Since(start) < time.Minute*3; time.Sleep(30 * time.Second) {
			output, _ := exec.Command("kubectl", "get", "pods", "iperfc-pod", "-o", "jsonpath=\"{.status.phase}\"").CombinedOutput()
			if result := string(output[1 : len(output)-1]); result == completed {
				break
			}
		}
		log.Printf("Getting logs from iperf client pod...")
		// get logs from pod
		output, err := exec.Command("kubectl", "logs", "iperfc-pod").CombinedOutput()
		if err == nil {
			// parse the ping results
			err = parseLogs(output, &report)
			log.Printf("report: %+v\t%+v", report, err)
			// create json report
			jsonData, err := json.Marshal(report)
			if err == nil {
				jsonFile, err := os.Create("./networkPerformance.json")
				if err == nil {
					jsonFile.Write(jsonData)
					jsonFile.Close()
				}
			}
		}

		log.Printf("Error getting logs: %+v", err)
	}
	// delete container if all tests have passed
	exec.Command("kubectl", "delete", "-f", clientyml).Run()
	testLock.Lock()
	testRunning = false
	testLock.Unlock()

	log.Printf("Error in triggerNetworkPerformanceTests: %+v", err)
	return
}

func retrieveNetworkPerformance() (report *NetworkPerformanceResults, err error) {
	jsonFile, err := os.Open("./networkPerformance.json")
	if err != nil {
		return nil, err
	}

	defer jsonFile.Close()
	bytes, err := ioutil.ReadAll(jsonFile)
	json.Unmarshal(bytes, &report)
	log.Printf("%+v", err)
	return
}
