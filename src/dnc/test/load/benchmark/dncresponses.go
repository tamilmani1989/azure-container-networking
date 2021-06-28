package benchmark

import (
	"dnc/api"
	"dnc/datamodel"
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
	StateTransitions    map[string]StateStats
	NCResponse          api.NetworkContainerResponse
	TotalProcessingTime time.Duration
}

type StateStats struct {
	CompletedOn time.Time
	Duration    time.Duration
}

const none = "none"

// state transitions order. there is no associate step for mdnc
var precedingStates = map[string]string{
	datamodel.Prepare.String():         none,
	datamodel.Reserve.String():         datamodel.Prepare.String(),
	datamodel.AllocateLocalIP.String(): datamodel.Reserve.String(),
	datamodel.Publish.String():         datamodel.AllocateLocalIP.String(),
	// program gets logged at the same time as publish.
	// see comment in code for explanation.
	datamodel.Program.String(): datamodel.AllocateLocalIP.String(),
}

func ncResponseFromNC(nc api.NetworkContainerResponse) NCResponse {
	stats := NCResponse{NCResponse: nc, StateTransitions: make(map[string]StateStats)}
	if nc.CreatedDateTime.IsZero() {
		return stats
	}

	states := make(map[string]time.Time)

	for _, loggedState := range nc.StateTransitionLog {
		if !loggedState.CompletedOn.IsZero() {
			states[loggedState.State] = loggedState.CompletedOn
		}
	}

	for state, completedOn := range states {
		preceding, ok := precedingStates[state]
		if !ok {
			continue
		}

		if preceding == none {
			stats.StateTransitions[state] = StateStats{CompletedOn: completedOn, Duration: completedOn.Sub(nc.CreatedDateTime)}
			continue
		}

		precedingCompletedOn, ok := states[preceding]
		if !ok {
			continue
		}

		stats.StateTransitions[state] = StateStats{CompletedOn: completedOn, Duration: completedOn.Sub(precedingCompletedOn)}
	}

	ncCompletedOn, ok := states[datamodel.Program.String()]
	if ok {
		stats.TotalProcessingTime = ncCompletedOn.Sub(nc.CreatedDateTime)
	}

	return stats
}
