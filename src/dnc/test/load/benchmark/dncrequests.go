package benchmark

import "dnc/api"

type NodeRequest struct {
	NodeID  string
	NodeReq api.NodeInfoRequest
}

type NCRequest struct {
	NCID   string
	VnetID string
	NCReq  api.NetworkContainerRequestV2
}

func streamNodeRequests(reqs []NodeRequest) chan NodeRequest {
	ch := make(chan NodeRequest)
	go func() {
		for _, item := range reqs {
			ch <- item
		}
		close(ch)
	}()
	return ch
}

func streamNcRequests(reqs []NCRequest) chan NCRequest {
	ch := make(chan NCRequest)
	go func() {
		for _, item := range reqs {
			ch <- item
		}
		close(ch)
	}()
	return ch
}
