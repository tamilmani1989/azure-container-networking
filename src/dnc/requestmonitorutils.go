package main

import (
	"dnc/api"
	"dnc/datamodel"
)

// transitionNCStateAndLog persists the next state and status for the NC, as well as logging completion from the current stage.
func transitionNCStateAndLog(nc *datamodel.NetworkContainerEntity, fromState, nextState datamodel.NetworkContainerState, status datamodel.Status) *api.JsonErr {
	nc.Config.LogStateCompletion(fromState)
	return transitionNCState(nc, nextState, status)
}

// transitionNCStateAndLog persists the next state and status for the NC.
func transitionNCState(nc *datamodel.NetworkContainerEntity, nextState datamodel.NetworkContainerState, status datamodel.Status) *api.JsonErr {
	nc.Config.State = nextState
	nc.Config.Status = status
	nc.Config.FailureDetail = api.JsonErr{}

	return datamodel.NcRepo.PersistNetworkContainerData(nc)
}

// markNCAsFailed persists the NC as failed.
// This should be used when an NC processing failure should not be retried.
func markNCAsFailed(nc *datamodel.NetworkContainerEntity, err api.JsonErr) (bool, *api.JsonErr) {
	nc.Config.Status = datamodel.Failed
	nc.Config.FailureDetail = err
	if err := datamodel.NcRepo.PersistNetworkContainerData(nc); err != nil {
		return false, err // don't remove from queue until failure can be persisted
	}
	return true, nil
}
