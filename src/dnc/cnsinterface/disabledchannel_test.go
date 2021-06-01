package cnsinterface_test

import (
	"dnc/api"
	"dnc/cnsinterface"
	"net/http"
	"testing"
)

type nodeInfoFetcher struct {
	nodeInfos map[string]*api.NodeInfoResponse
}

func (n *nodeInfoFetcher) GetNodeInfo(name string) (*api.NodeInfoResponse, *api.JsonErr) {
	if nodeInfo, ok := n.nodeInfos[name]; ok {
		return nodeInfo, nil
	}

	return nil, &api.JsonErr{
		Code:       api.NodeNotFound,
		Text:       "node not found",
		HTTPStatus: http.StatusNotFound,
	}
}

func TestGetNCLimitForNode(t *testing.T) {
	node1Cores := 4
	cnsInterface := cnsinterface.DisabledChannel{
		NodeInfoFetcher: &nodeInfoFetcher{
			map[string]*api.NodeInfoResponse{
				"node1": {
					NodeInfo: api.NodeInfoRequest{
						NumCores: node1Cores,
					},
				},
				"node2": {
					NodeInfo: api.NodeInfoRequest{},
				},
			},
		},
		NICsPerVirtualCore: 3,
		DefaultNCLimit:     1,
	}

	// node1 should return capacity from nodeInfo
	node1Limit, jsonErr := cnsInterface.GetNCLimitForNode("node1")
	if jsonErr != nil {
		t.Fatalf("received error when fetching node info for node1: %+v", jsonErr)
	}

	if node1Limit != cnsInterface.NICsPerVirtualCore*node1Cores {
		t.Fatalf("expected %d, got %d for GetNCLimitForNode(node1)", cnsInterface.NICsPerVirtualCore*node1Cores, node1Limit)
	}

	// node2 should return default capacity of 1
	node2Limit, jsonErr := cnsInterface.GetNCLimitForNode("node2")
	if jsonErr != nil {
		t.Fatalf("received error when fetching node info for node2: %+v", jsonErr)
	}

	if node2Limit != 1 {
		t.Fatalf("expected %d, got %d for GetNCLimitForNode(node2)", 1, node2Limit)
	}
}
