package benchmark

import (
	"dnc/api"
	"sync"
	"time"
)

type ncResponses struct {
	ncs []NCResponse
	m   sync.Mutex
}

func (n *ncResponses) Add(nc NCResponse) {
	n.m.Lock()
	n.ncs = append(n.ncs, nc)
	n.m.Unlock()
}

func (n *ncResponses) GetAll() []NCResponse {
	n.m.Lock()
	c := make([]NCResponse, len(n.ncs))
	copy(c, n.ncs)
	n.m.Unlock()
	return c
}

func (n *ncResponses) Len() int {
	n.m.Lock()
	defer n.m.Unlock()
	return len(n.ncs)
}

type NCResponse struct {
	StateDurations map[string]time.Duration
	NCResponse     api.NetworkContainerResponse
}

func ncResponseFromNC(nc api.NetworkContainerResponse) NCResponse {
	stats := NCResponse{NCResponse: nc, StateDurations: make(map[string]time.Duration)}
	if nc.CreatedDateTime.IsZero() {
		return stats
	}

	for _, loggedState := range nc.StateTransitionLog {
		if !loggedState.CompletedOn.IsZero() {
			stats.StateDurations[loggedState.State] = loggedState.CompletedOn.Sub(nc.CreatedDateTime)
		}
	}

	return stats
}
