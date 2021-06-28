//+build loadtest

package load

import (
	"bytes"
	"context"
	"dnc/api"
	"dnc/client"
	"dnc/configuration"
	"dnc/datamodel"
	"dnc/nmagentcontracts"
	"dnc/retry"
	"dnc/test/junit"
	"dnc/test/k8s"
	"dnc/test/load/benchmark"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
)

var (
	defaultPercentiles = []float64{0.5, 0.9, 0.95, 0.99, 0.999}
	defaultRetrier     = retry.Retrier{Attempts: 10, Delay: time.Second}
)

func TestOpenAI(t *testing.T) {
	dncPort := 9000
	nmagentPort := 9001
	partition := generateRunPartition()
	ctx := context.Background()
	start := time.Now()

	openAISuite := junit.TestSuite{Name: "DNCLoadTest/OpenAI"}

	// func to mutate the base configuration as needed by this test
	cfgFunc := func(dncCfg *configuration.Configuration) {
		dncCfg.DataStoreSettings = configuration.DataStoreSettings{
			AzureStorageInfo: configuration.AzureStorageInfo{
				AccountName:     cosmosAccountName,
				Key:             cosmosAccountKey,
				PartitionKey:    partition,
				IsUsingCosmosDb: true,
			},
		}
		dncCfg.ServerSettings.Port = strconv.Itoa(dncPort)
		dncCfg.HostAgentSettings.WireServerIP = fmt.Sprintf("localhost:%d", nmagentPort)
		dncCfg.MasterElectionSettings.TestDataElectionSettings.Test = "TestStore"
		dncCfg.DataStoreSettings.MaxBatchSize = 1000
		dncCfg.FeatureFlagSettings = configuration.FeatureFlagSettings{
			EnableBatchAllocationProcessing: true,
			MaxIPReservationBatchSize:       100,
			UsePrepopulatedIPAllocations:    true,
		}
		dncCfg.DNCSettings.Managed = true
	}

	mustCreateK8sResources(ctx, t, dncPort, nmagentPort, cfgFunc)

	// arbitrary duration, port forward seems to work better if we wait for some time
	deployWaitTime := time.Second * 15
	t.Logf("waiting %s", deployWaitTime.String())
	time.Sleep(deployWaitTime)

	ncsPerSubnet := 900
	workerCount := 50
	vnetCount := 8

	t.Logf("benchmarking %d ncs per subnet, on %d vnets, with %d concurrent requests", ncsPerSubnet, vnetCount, workerCount)
	t.Logf("cosmos db account name: %s, partition %s", cosmosAccountName, partition)

	vnets := generateOpenAITopology(vnetCount)

	pf, err := k8s.NewPortForwarder(restConfig)
	require.NoError(t, err)

	nmAgentPFOpts := podPortForwardOpts{namespace: loadTestNamespace, labelSelector: "app=dnc", localPort: nmagentPort, destinationPort: nmagentPort}
	err = hydrateNMAgent(ctx, t, pf, nmAgentPFOpts, vnets)
	require.NoError(t, err)

	dncPFOpts := podPortForwardOpts{namespace: loadTestNamespace, labelSelector: "app=dnc", localPort: dncPort, destinationPort: dncPort}
	dncPortForward, err := getPortForwardHandle(ctx, t, pf, dncPFOpts)
	require.NoError(t, err)

	defer dncPortForward.Stop()

	t.Logf("successfully port forwarded to dnc: %s", dncPortForward.Url())

	dncClient := client.NewDncClient(dncPortForward.Url())
	b := benchmark.DNC{Client: dncClient}

	vnetIDs := extractVnetIDs(vnets)
	vnetCreateTimes, err := b.CreateVnets(vnetIDs)
	require.NoError(t, err)
	t.Logf("%d vnets posted. avg time: %v", len(vnets), benchmark.Average(vnetCreateTimes))

	vnetProcessWait := time.Second * 10
	t.Logf("waiting %s for some background operations", vnetProcessWait.String())
	time.Sleep(vnetProcessWait)

	orchestratorID := uuid.New().String()

	t.Logf("creating node set (orchestrator ID) %s", orchestratorID)
	apiErr := dncClient.CreateNodeSet(orchestratorID)
	require.Nil(t, apiErr)

	t.Log("posting subnets")
	vnetToSubnets := extractVnetToSubnets(vnets)
	subnetJoinTimes, err := b.JoinSubnets(ctx, vnetToSubnets)
	require.NoError(t, err)
	t.Logf("%d subnets joined. avg time: %v", len(subnetJoinTimes), benchmark.Average(subnetJoinTimes))

	nodeRequests, ncRequests := generateOpenAINodesAndNCs(vnets, orchestratorID, ncsPerSubnet, 32)
	t.Logf("creating %d nodes on node set (orchestrator ID) %s", len(nodeRequests), orchestratorID)
	nodeCreationTimes, err := b.CreateNodes(ctx, workerCount, nodeRequests)
	require.NoError(t, err)
	t.Logf("created %d nodes", len(nodeRequests))

	t.Logf("submitting %d ncs for processing", len(ncRequests))
	ncCreationTimes, err := b.CreateNCs(ctx, workerCount, ncRequests)
	require.NoError(t, err)
	t.Logf("submitted %d ncs for processing", len(ncRequests))

	t.Logf("verifying %d ncs", len(ncRequests))
	collectCtx, cancel := context.WithTimeout(ctx, time.Minute*20)
	defer cancel()
	allStats, err := b.GetAllNCResponses(collectCtx, workerCount, ncRequests)
	require.NoError(t, err)

	var (
		totalProcessingTimes   []time.Duration // from nc creation to program complete
		timesToReserveComplete []time.Duration // from nc creation to reserve complete
		allocationTimes        []time.Duration // from prepare complete to reserve complete (time spent in stage)
		programmedNCs          []benchmark.NCResponse
	)

	for _, stats := range allStats {
		if stats.TotalProcessingTime != 0 {
			totalProcessingTimes = append(totalProcessingTimes, stats.TotalProcessingTime)
			programmedNCs = append(programmedNCs, stats)
		}

		reserve, ok := stats.StateTransitions[datamodel.Reserve.String()]
		if ok {
			timesToReserveComplete = append(timesToReserveComplete, reserve.CompletedOn.Sub(stats.NCResponse.CreatedDateTime))
			allocationTimes = append(allocationTimes, reserve.Duration)
		}
	}

	logStats(t, nodeCreationTimes, "create node", defaultPercentiles)
	logStats(t, ncCreationTimes, "create (not process) NC", defaultPercentiles)
	logStats(t, totalProcessingTimes, "reach NC Program Complete state", defaultPercentiles)
	logStats(t, timesToReserveComplete, "allocate all IPs on each NC (from NC creation)", defaultPercentiles)
	logStats(t, allocationTimes, "allocate all IPs on each NC (reservation time only)", defaultPercentiles)

	createNodeTestCases := assertStats(t, "create node", nodeCreationTimes, []percentileAssertion{
		{percentile: 0.5, below: 5 * time.Second},
		{percentile: 0.9, below: 5 * time.Second},
		{percentile: 0.95, below: 5 * time.Second},
		{percentile: 0.99, below: 5 * time.Second},
		{percentile: 0.999, below: 5 * time.Second},
	})

	openAISuite.AddTestCases(createNodeTestCases)

	reachProgramTestCases := assertStats(t, "reach NC Program Complete state", totalProcessingTimes, []percentileAssertion{
		{percentile: 0.5, below: 1*time.Minute + 30*time.Second},
		{percentile: 0.9, below: 2*time.Minute + 30*time.Second},
		{percentile: 0.95, below: 3 * time.Minute},
		{percentile: 0.99, below: 4 * time.Minute},
		{percentile: 0.999, below: 5 * time.Minute},
	})

	openAISuite.AddTestCases(reachProgramTestCases)

	allocateIPTestCases := assertStats(t, "allocate secondary IPs on NC", allocationTimes, []percentileAssertion{
		{percentile: 0.5, below: 20 * time.Second},
		{percentile: 0.9, below: 30 * time.Second},
		{percentile: 0.95, below: 40 * time.Second},
		{percentile: 0.99, below: 50 * time.Second},
		{percentile: 0.999, below: 1 * time.Minute},
	})

	openAISuite.AddTestCases(allocateIPTestCases)

	sort.Slice(programmedNCs, func(i, j int) bool {
		return programmedNCs[i].TotalProcessingTime < programmedNCs[j].TotalProcessingTime
	})
	tailNC := programmedNCs[len(programmedNCs)-1]

	bs, _ := json.MarshalIndent(tailNC.NCResponse, "", "  ")
	t.Logf("tail program time: nc id %s, total processing time %v", tailNC.NCResponse.NcID, tailNC.TotalProcessingTime.String())
	for k, v := range tailNC.StateTransitions {
		t.Logf("%v: took %v, completed on %v", k, v.Duration.String(), v.CompletedOn.String())
	}
	t.Log(string(bs))

	defer func() {
		openAISuite.Time = junit.FmtDuration(time.Since(start))
		suites.Suites = append(suites.Suites, openAISuite)
	}()

	t.Logf("force deleting node set (orchestrator ID) %s", orchestratorID)
	nodeSetDeletionStart := time.Now()
	apiErr = dncClient.DeleteNodeSet(orchestratorID, true)
	require.Nil(t, apiErr)

	t.Log("polling for remaining nodes")

	cleanupRetrier := retry.Retrier{Attempts: 100, Delay: time.Second * 5}
	err = cleanupRetrier.Do(ctx, func() error {
		resp, err := dncClient.GetNodesExt(client.NodesQueryParams{NodeSet: orchestratorID})
		if err != nil {
			return err
		}

		if len(resp.Nodes) > 0 {
			t.Logf("%d nodes remaining in node set", len(resp.Nodes))
			return errors.New("nodes remaining, retry")
		}

		return nil
	})
	require.NoError(t, err)
	t.Logf("all nodes on node set (orchestrator ID) %s removed. took %v", orchestratorID, time.Since(nodeSetDeletionStart))
}

type percentileAssertion struct {
	percentile float64
	below      time.Duration
}

func mustCreateK8sResources(ctx context.Context, t *testing.T, dncPort int, nmagentPort int, configFn func(*configuration.Configuration)) {
	dncDeployment, err := k8s.ParseDeployment(baseDNCDeploymentYAMLFilePath)
	require.NoError(t, err)

	dncCfg, err := getBaseDNCConfig(baseDNCConfigJSONFilePath)
	require.NoError(t, err)

	configFn(&dncCfg)

	cfgMap, err := dncConfigMap(dncCfg)
	require.NoError(t, err)

	t.Logf("creating namespace %s", loadTestNamespace)
	err = k8sShim.CreateNamespace(ctx, loadTestNamespace)
	require.NoError(t, err)

	t.Logf("creating configmap %s", cfgMap.Name)
	err = k8sShim.CreateOrReplaceConfigmap(ctx, loadTestNamespace, cfgMap)
	require.NoError(t, err)

	dncDeployment.Spec.Template.Spec.Containers = []v1.Container{
		dncContainerSpec(dncImage, dncPort),
		mockNmagentContainerSpec(mockNMAgentImage, nmagentPort),
	}

	t.Logf("creating deployment %s", dncDeployment.Name)
	err = k8sShim.CreateOrReplaceDeployment(ctx, loadTestNamespace, dncDeployment)
	require.NoError(t, err)

	t.Log("all resources created")
}

func logStats(t *testing.T, samples []time.Duration, action string, ths []float64) {
	for _, th := range ths {
		pTh := benchmark.Percentile(samples, th)
		t.Logf("%vth percentile time to %s (%d samples): %v", th*100, action, len(samples), pTh)
		pAvgTh := benchmark.PercentileAvg(samples, th)
		t.Logf("%vth percentile average time to %s (%d samples): %v", th*100, action, len(samples), pAvgTh)
	}
}

func assertStats(t *testing.T, action string, samples []time.Duration, pas []percentileAssertion) []junit.TestCase {
	var tt []junit.TestCase

	for _, pa := range pas {
		name := fmt.Sprintf("%s: %vth percentile (%d samples) below %v", action, pa.percentile, len(samples), pa.below)
		t.Run(name, func(t *testing.T) {
			pTh := benchmark.Percentile(samples, pa.percentile)
			pass := assert.LessOrEqual(t, pTh, pa.below)

			tc := junit.TestCase{Classname: action, Name: name, Time: junit.FmtDuration(pTh)}

			if !pass {
				failure := junit.Failure{
					Message:  "Failed",
					Contents: fmt.Sprintf("%s: %vth percentile %v below %v", action, pa.percentile, pTh, pa.below),
				}
				tc.Failure = &failure
			}

			tt = append(tt, tc)
		})
	}

	return tt
}

func extractVnetIDs(vnets map[string]nmagentcontracts.VirtualNetworkResponse) []string {
	vnetIDs := make([]string, 0, len(vnets))
	for vnetID := range vnets {
		vnetIDs = append(vnetIDs, vnetID)
	}
	return vnetIDs
}

func extractVnetToSubnets(vnets map[string]nmagentcontracts.VirtualNetworkResponse) map[string][]string {
	vnetToSubnets := make(map[string][]string)
	for vnetID, vnet := range vnets {
		for _, subnet := range vnet.Subnets {
			vnetToSubnets[vnetID] = append(vnetToSubnets[vnetID], subnet.SubnetName)
		}
	}
	return vnetToSubnets
}

func hydrateNMAgent(ctx context.Context, t *testing.T, pf *k8s.PortForwarder, pfOpts podPortForwardOpts, vnets map[string]nmagentcontracts.VirtualNetworkResponse) error {
	pfHandle, err := getPortForwardHandle(ctx, t, pf, pfOpts)
	if err != nil {
		return err
	}
	defer pfHandle.Stop()

	vnetLoadFunc := func() error {
		if err := postNMAgentVnets(pfHandle.Url(), vnets); err != nil {
			t.Logf("could not post payload to mock nmagent: %v. retrying...", err)
			return err
		}
		return nil
	}

	if err := defaultRetrier.Do(ctx, vnetLoadFunc); err != nil {
		return err
	}

	t.Logf("successfully posted to mock nmagent: %+v", vnets)
	return nil
}

func postNMAgentVnets(host string, vnets map[string]nmagentcontracts.VirtualNetworkResponse) error {
	endpoint := host + "/test/joinedVirtualNetworks/"

	bs, err := json.Marshal(vnets)
	if err != nil {
		return err
	}
	res, err := http.Post(endpoint, "application/json", bytes.NewBuffer(bs))
	if err != nil {
		return err
	}
	defer res.Body.Close()

	responseBodyBytes, _ := ioutil.ReadAll(res.Body)

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("could not post vnets to %s: got http error code %d: %v", endpoint, res.StatusCode, string(responseBodyBytes))
	}

	return nil
}

func generateOpenAITopology(vnetCount int) map[string]nmagentcontracts.VirtualNetworkResponse {
	allVnets := make(map[string]nmagentcontracts.VirtualNetworkResponse)

	// todo: configurable subnet mask
	for i := 0; i < vnetCount; i++ {
		vnetID, vnet := generateVnet()
		vnet.Subnets = append(vnet.Subnets, nmagentcontracts.Subnet{SubnetName: "subnet1", AddressPrefix: "10.1.0.0/16", DefaultGateway: "10.1.0.1"})
		allVnets[vnetID] = vnet
	}

	return allVnets
}

func generateOpenAINodesAndNCs(vnets map[string]nmagentcontracts.VirtualNetworkResponse, nodeSet string, ncsPerSubnet int, secondaryIPsPerNC int) ([]benchmark.NodeRequest, []benchmark.NCRequest) {
	var (
		nodeRequests []benchmark.NodeRequest
		ncRequests   []benchmark.NCRequest
	)

	infraVnet := uuid.New().String()
	for vnetID, vnet := range vnets {
		for _, subnet := range vnet.Subnets {
			for i := 0; i < ncsPerSubnet; i++ {
				nodeId, nodeReq := generateNode(infraVnet, nodeSet)
				ncID, ncReq := generateNC(nodeId, subnet.SubnetName, secondaryIPsPerNC)

				nodeRequests = append(nodeRequests, benchmark.NodeRequest{NodeID: nodeId, NodeReq: nodeReq})
				ncRequests = append(ncRequests, benchmark.NCRequest{NCID: ncID, NCReq: ncReq, VnetID: vnetID})
			}
		}
	}

	return nodeRequests, ncRequests
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

func generateNode(infraVnet, nodeSet string) (string, api.NodeInfoRequest) {
	nodeID := uuid.New().String()
	node := api.NodeInfoRequest{
		IPAddresses:           []string{"127.0.0.1"},
		InfrastructureNetwork: infraVnet,
		OrchestratorType:      "Kubernetes",
		NodeSet:               nodeSet,
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

func generateRunPartition() string {
	return "loadtest" + time.Now().Format("2006-01-02-150405")
}
