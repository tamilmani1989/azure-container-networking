package main

import (
	"dnc/api"
	"dnc/datamodel"
)

// transitionNCStateAndLog persists the next state and status for the NC, as well as logging completion from the current stage.
func transitionNCStateAndLog(nc *datamodel.NCConfiguration, fromState, nextState datamodel.NetworkContainerState, status datamodel.Status) *api.JsonErr {
	nc.LogStateCompletion(fromState)
	return transitionNCState(nc, nextState, status)
}

// transitionNCStateAndLog persists the next state and status for the NC.
func transitionNCState(nc *datamodel.NCConfiguration, nextState datamodel.NetworkContainerState, status datamodel.Status) *api.JsonErr {
	nc.State = nextState
	nc.Status = status
	nc.FailureDetail = api.JsonErr{}

	return datamodel.PersistNCConfiguration(nc, datamodel.DontAddToPendingQ)
}

// markNCAsFailed persists the NC as failed.
// This should be used when an NC processing failure should not be retried.
func markNCAsFailed(ncConfig *datamodel.NCConfiguration, err api.JsonErr) (bool, *api.JsonErr) {
	ncConfig.Status = datamodel.Failed
	ncConfig.FailureDetail = err
	if err := datamodel.PersistNCConfiguration(ncConfig, datamodel.DontAddToPendingQ); err != nil {
		return false, err // don't remove from queue until failure can be persisted
	}
	return true, nil
}
