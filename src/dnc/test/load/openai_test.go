//+build loadtest

package load

import (
	"context"
	"dnc/api"
	"dnc/client"
	"dnc/datamodel"
	"dnc/nmagentcontracts"
	"dnc/test/load/benchmark"
	"dnc/test/nmagentmock"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
)

var (
	b                  benchmark.DNC
	nmagentServer      *nmagentmock.Server
	defaultPercentiles = []float64{0.5, 0.9, 0.95, 0.99, 0.999}
)

func TestMain(t *testing.M) {
	b = benchmark.DNC{Client: client.DncClient{DncEP: "http://127.0.0.1:9000"}} // todo: make configurable
	nmagentServer = nmagentmock.NewServer("localhost:9001")
	nmagentServer.Start()

	defer func() {
		if err := nmagentServer.Close(); err != nil {
			fmt.Printf("could not stop mock nmagent correctly: %v\n", err)
		}
	}()

	os.Exit(t.Run())
}

/*
  todo:
   - Tests should run against a k8s cluster in order to potentially benchmark DNC with different configurations,
     partitions, databases, and cpu/memory request limits.
   - Nmagent mock should run as part of deployment, not in-memory.
   - Add assertions on percentiles once baselines have been determined.
*/

func TestOpenAI(t *testing.T) {
	ncsPerSubnet := 900
	workerCount := 50
	vnetCount := 8

	t.Logf("benchmarking %d ncs per subnet, on %d vnets, with %d concurrent requests", ncsPerSubnet, vnetCount, workerCount)

	vnets := generateOpenAITopology(vnetCount)
	nmagentServer.StoreVnetResponses(vnets)

	vnetIDs := make([]string, 0, len(vnets))
	for vnetID := range vnets {
		vnetIDs = append(vnetIDs, vnetID)
	}

	vnetToSubnets := make(map[string][]string)
	subnetCount := 0
	for vnetID, vnet := range vnets {
		for _, subnet := range vnet.Subnets {
			vnetToSubnets[vnetID] = append(vnetToSubnets[vnetID], subnet.SubnetName)
		}
		subnetCount += len(vnet.Subnets)
	}

	var (
		nodeRequests []benchmark.NodeRequest
		ncRequests   []benchmark.NCRequest
	)

	infraVnet := uuid.New().String()
	for vnetID, vnet := range vnets {
		for _, subnet := range vnet.Subnets {
			for i := 0; i < ncsPerSubnet; i++ {
				nodeId, nodeReq := generateNode(infraVnet)
				ncID, ncReq := generateNC(nodeId, subnet.SubnetName, 32)

				nodeRequests = append(nodeRequests, benchmark.NodeRequest{NodeID: nodeId, NodeReq: nodeReq})
				ncRequests = append(ncRequests, benchmark.NCRequest{NCID: ncID, NCReq: ncReq, VnetID: vnetID})
			}
		}
	}

	ctx := context.Background()

	vnetCreateTimes := b.CreateVnets(vnetIDs)
	t.Logf("%d vnets posted. avg time: %v", len(vnets), benchmark.Average(vnetCreateTimes))
	t.Logf("waiting 5s for some background operations")
	time.Sleep(time.Second * 5)

	t.Log("posting subnets")
	subnetJoinTimes := b.JoinSubnets(ctx, vnetToSubnets)
	t.Logf("%d subnets joined. avg time: %v", subnetCount, benchmark.Average(subnetJoinTimes))

	t.Logf("creating %d nodes", len(nodeRequests))
	nodeCreationTimes := b.CreateNodes(ctx, workerCount, nodeRequests)
	t.Logf("created %d nodes", len(nodeRequests))

	t.Logf("submitting %d ncs for processing", len(ncRequests))
	ncCreationTimes := b.CreateNCs(ctx, workerCount, ncRequests)
	t.Logf("submitted %d ncs for processing", len(ncRequests))

	t.Logf("verifying %d ncs", len(ncRequests))
	allStats := b.GetAllNCResponses(ctx, workerCount, ncRequests)

	var (
		timesToProgram         []time.Duration
		timesToReserveComplete []time.Duration
		programmedNCs          []benchmark.NCResponse
	)

	for _, stats := range allStats {
		if timeToProgram, ok := stats.StateDurations[datamodel.Program.String()]; ok {
			timesToProgram = append(timesToProgram, timeToProgram)
			programmedNCs = append(programmedNCs, stats)
		}

		if timeToReserveComplete, ok := stats.StateDurations[datamodel.Reserve.String()]; ok {
			timesToReserveComplete = append(timesToReserveComplete, timeToReserveComplete)
		}
	}

	logStats(t, nodeCreationTimes, "create node", defaultPercentiles)
	logStats(t, ncCreationTimes, "create (not process) NC", defaultPercentiles)
	logStats(t, timesToProgram, "reach NC Program Complete state", defaultPercentiles)
	logStats(t, timesToReserveComplete, "allocate all IPs", defaultPercentiles)

	sort.Slice(programmedNCs, func(i, j int) bool {
		return programmedNCs[i].StateDurations[datamodel.Program.String()] < programmedNCs[j].StateDurations[datamodel.Program.String()]
	})
	tailNC := programmedNCs[len(programmedNCs)-1]

	bs, _ := json.MarshalIndent(tailNC, "", "  ")
	t.Logf("tail program time: nc id %s:\n%v", tailNC.NCResponse.NcID, string(bs))
}

func logStats(t *testing.T, samples []time.Duration, action string, ths []float64) {
	for _, th := range ths {
		pTh := benchmark.Percentile(samples, th)
		t.Logf("%vth percentile time to %s (%d samples): %v", th*100, action, len(samples), pTh)
		pAvgTh := benchmark.PercentileAvg(samples, th)
		t.Logf("%vth percentile average time to %s (%d samples): %v", th*100, action, len(samples), pAvgTh)
	}
}

func generateOpenAITopology(vnetCount int) map[string]nmagentcontracts.VirtualNetworkResponse {
	allVnets := make(map[string]nmagentcontracts.VirtualNetworkResponse)

	// todo: configurable subnet mask
	for i := 0; i < vnetCount; i++ {
		vnetID, vnet := generateVnet()
		vnet.Subnets = append(vnet.Subnets, nmagentcontracts.Subnet{SubnetName: "subnet1", AddressPrefix: "10.1.0.0/17", DefaultGateway: "10.1.0.1"})
		allVnets[vnetID] = vnet
	}

	return allVnets
}

func generateVnet() (string, nmagentcontracts.VirtualNetworkResponse) {
	vnetID := uuid.New().String()
	vnet := nmagentcontracts.VirtualNetworkResponse{
		ResponseCode: strconv.Itoa(http.StatusOK),
		CNetSpace:    "10.0.0.0/8,172.16.0.0/16,192.168.1.0/24",
		DNSServers:   "8.8.8.8,9.9.9.9",
		VNetSpace:    "10.0.0.0/8",
		VnetVersion:  "1",
	}
	return vnetID, vnet
}

func generateNode(infraVnet string) (string, api.NodeInfoRequest) {
	nodeID := uuid.New().String()
	node := api.NodeInfoRequest{
		IPAddresses:           []string{"127.0.0.1"},
		InfrastructureNetwork: infraVnet,
		OrchestratorType:      "Kubernetes",
	}

	return nodeID, node
}

func generateNC(nodeID, subnetName string, secondaryIPCount int) (string, api.NetworkContainerRequestV2) {
	ncID := uuid.New().String()
	nc := api.NetworkContainerRequestV2{
		AllocationRequest: api.AllocationRequest{
			SubnetName:       subnetName,
			SecondaryIPCount: secondaryIPCount,
		},
		AssociationInfo: api.AssociationInfo{
			NodeID:        nodeID,
			ContainerType: "Docker",
		},
	}

	return ncID, nc
}
