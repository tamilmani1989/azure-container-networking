package main

import (
	"bufio"
	"bytes"
	"dnc/api"
	"dnc/datamodel"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/google/uuid"
)

func main() {
	var nodeNCCount, concurrentRequests int
	flag.IntVar(&nodeNCCount, "ncs", 500, "nc count")
	flag.IntVar(&concurrentRequests, "c", 10, "number of concurrent requests")
	flag.Parse()

	logrus.Infof("testing %d ncs with %d concurrent requests", nodeNCCount, concurrentRequests)

	vnetID, vnetReq := generateVnet()
	vnetURL := fmt.Sprintf("http://localhost:9000/networks/%s?api-version=2018-03-01", vnetID)

	logrus.Infof("creating vnet %s", vnetID)

	bs, _ := json.Marshal(vnetReq)
	res, err := http.Post(vnetURL, "application/json", bytes.NewBuffer(bs))
	if err != nil {
		logrus.Fatal("post vnet", err)
	}
	_, _ = ioutil.ReadAll(res.Body)
	res.Body.Close()

	time.Sleep(time.Second)

	logrus.Infof("vnet %s posted", vnetID)

	time.Sleep(time.Second * 2)

	subnetReq := generateSubnet()
	subnetURL := fmt.Sprintf("http://localhost:9000/networks/%s/subnets/%s?api-version=2018-03-01", vnetID, subnetReq.SubnetName)

	logrus.Info("posting subnet")

	bs, _ = json.Marshal(subnetReq)
	res, err = http.Post(subnetURL, "application/json", bytes.NewBuffer(bs))
	if err != nil {
		logrus.Fatal("post subnet", err)
	}
	defer res.Body.Close()
	_, _ = ioutil.ReadAll(res.Body)

	logrus.Infof("posted subnet %s", subnetReq.SubnetName)
	time.Sleep(time.Second * 2)

	type workItem struct {
		nodeID  string
		nodeReq api.NodeInfoRequest
		ncID    string
		ncReq   api.NetworkContainerRequestV2
	}

	logrus.Infof("posting %d nodes and ncs", nodeNCCount)

	workCh := make(chan workItem)
	var workItems []workItem

	for i := 0; i < nodeNCCount; i++ {
		nodeId, nodeReq, ncID, ncReq := generateNodeAndNC()
		workItems = append(workItems, workItem{
			nodeID:  nodeId,
			nodeReq: nodeReq,
			ncID:    ncID,
			ncReq:   ncReq,
		})
	}

	go func() {
		for _, w := range workItems {
			workCh <- w
		}
		close(workCh)
	}()

	var g sync.WaitGroup
	for i := 0; i < concurrentRequests; i++ {
		g.Add(1)
		go func() {
			defer g.Done()
			for w := range workCh {
				createBs, _ := json.Marshal(w.nodeReq)
				nodeURL := fmt.Sprintf("http://localhost:9000/nodes/%s?api-version=2018-03-01", w.nodeID)
				res, err := http.Post(nodeURL, "application/json", bytes.NewBuffer(createBs))
				if err != nil {
					logrus.Error("post node", err)
				}
				readAll, _ := ioutil.ReadAll(res.Body)
				if len(readAll) > 0 {
					logrus.Infof("node response: %v", string(bs))
				}
				res.Body.Close()

				bs, _ := json.Marshal(w.ncReq)
				ncURL := fmt.Sprintf("http://localhost:9000/networks/%s/networkcontainer/%s?api-version=2018-03-01", vnetID, w.ncID)
				res, err = http.Post(ncURL, "application/json", bytes.NewBuffer(bs))
				if err != nil {
					logrus.Error("post nc", err)
				}
				bs, _ = ioutil.ReadAll(res.Body)
				if len(bs) > 0 {
					logrus.Infof("nc response: %v", string(bs))
				}
				res.Body.Close()

				logrus.Infof("nc %s on node %s", w.ncID, w.ncReq.AssociationInfo.NodeID)
			}
		}()
	}

	g.Wait()

	logrus.Info("ncs sent")

	logrus.Info("Press return when ready to send updates...")
	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')

	logrus.Info("sending scale up updates")

	updateCh := make(chan workItem)
	go func() {
		defer close(updateCh)
		for _, w := range workItems {
			updateCh <- w
		}
	}()

	for i := 0; i < concurrentRequests; i++ {
		g.Add(1)
		go func() {
			defer g.Done()
			for w := range updateCh {
				ncReq := w.ncReq
				ncReq.AllocationRequest.SecondaryIPCount += 16
				ncReq.AssociationInfo.NodeID = w.nodeID
				putBs, _ := json.Marshal(ncReq)
				ncURL := fmt.Sprintf("http://localhost:9000/networks/%s/networkcontainer/%s?api-version=2018-03-01", vnetID, w.ncID)

				req, err := http.NewRequest(http.MethodPut, ncURL, bytes.NewBuffer(putBs))
				if err != nil {
					logrus.Fatalf("new update request nc", err)
				}
				req.Header.Add("Content-Type", "application/json")

				logrus.Infof("nc %s, sending update req: %+v", w.ncID, ncReq)

				res, err = http.DefaultClient.Do(req)
				if err != nil {
					logrus.Error("put nc", err)
				}
				if res.StatusCode != http.StatusOK {
					logrus.Infof("got error code: %d. for body: %v", res.StatusCode, string(bs))
				}
				bs, _ := ioutil.ReadAll(res.Body)
				if len(bs) > 0 {
					logrus.Infof("nc response: %v", string(bs))
				}
				res.Body.Close()

				logrus.Infof("updated nc %s on node %s", w.ncID, ncReq.AssociationInfo.NodeID)
			}
		}()
	}

	g.Wait()
	logrus.Info("done sending updates")

	logrus.Info("Press return when ready to verify updates...")
	_, _ = reader.ReadString('\n')

	logrus.Info("verifying updates")

	var wg sync.WaitGroup
	var doneCount int64

	go func() {
		for range time.Tick(time.Second * 3) {
			logrus.Printf("done verifying %d updates", atomic.LoadInt64(&doneCount))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i, wi := range workItems {
			func() {
				defer atomic.AddInt64(&doneCount, 1)
				ncURL := fmt.Sprintf("http://localhost:9000/networks/%s/networkcontainer/%s?api-version=2018-03-01", vnetID, wi.ncID)
				res, err := http.Get(ncURL)
				if err != nil {
					logrus.Errorf("%d: get nc", i, err)
					return
				}
				if res.StatusCode != http.StatusOK {
					logrus.Errorf("%s: got error code: %d", wi.ncID, res.StatusCode)
				}
				bs, _ = ioutil.ReadAll(res.Body)

				var nc api.NetworkContainerResponse
				if err := json.Unmarshal(bs, &nc); err != nil {
					logrus.Errorf("%s: could not unmarshal: %v", wi.ncID, err)
				}
				res.Body.Close()

				if nc.RequestStatus.Status != datamodel.Completed.String() {
					logrus.Errorf("%s: status is not completed: %s", wi.ncID, nc.RequestStatus.Status)
				}

				if nc.RequestStatus.State != datamodel.Program.String() {
					logrus.Errorf("%s: state is not program: %s", wi.ncID, nc.RequestStatus.State)
				}

				if nc.IPConfiguration.IPAddress == "" {
					logrus.Errorf("%s: did not have primary ip", wi.ncID)
				}
				if len(nc.IPConfiguration.SecondaryIPs) != 32 {
					logrus.Errorf("%s: incomplete allocation ids, only got %d", wi.ncID, len(nc.IPConfiguration.SecondaryIPs))
					logrus.Errorf("request status: %s", nc.RequestStatus.String())
				}
				for k, v := range nc.IPConfiguration.SecondaryIPs {
					if v.IPAddress == "" {
						logrus.Errorf("%s: incomplete allocation on key %v: %+v", wi.ncID, k, nc.IPConfiguration.SecondaryIPs)
					}
				}
			}()
		}
	}()

	wg.Wait()
	logrus.Info("updates checked. see above for errors")
}

func generateVnet() (string, api.NetworkInfoRequest) {
	id := uuid.New().String()
	vnet := api.NetworkInfoRequest{
		NetworkType: "AzureNet",
	}
	return id, vnet
}

func generateNodeAndNC() (string, api.NodeInfoRequest, string, api.NetworkContainerRequestV2) {
	nodeID := uuid.New().String()
	infravnet := uuid.New().String()

	node := api.NodeInfoRequest{
		IPAddresses:           []string{"127.0.0.1"},
		InfrastructureNetwork: infravnet,
		OrchestratorType:      "Kubernetes",
	}

	ncID := uuid.New().String()
	nc := api.NetworkContainerRequestV2{
		AllocationRequest: api.AllocationRequest{
			SubnetName:       "subnet1",
			SecondaryIPCount: 16,
		},
		AssociationInfo: api.AssociationInfo{
			NodeID:        nodeID,
			ContainerType: "Docker",
		},
	}

	return nodeID, node, ncID, nc
}

func generateSubnet() api.SubnetRequest {
	return api.SubnetRequest{SubnetName: "subnet1", AuthenticationToken: "a"}
}
