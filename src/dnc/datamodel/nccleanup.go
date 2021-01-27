package datamodel

import (
	"dnc/api"
	"net/http"
)

// GetNetworkContainerCleanupStatus returns nc cleanup status. An error is returned only when cleanup status cannot be determined, such as DNC server error
func GetNetworkContainerCleanupStatus(networkID, ncID string) (api.NetworkContainerCleanupResponse, *api.JsonErr) {
	cleanupResp := api.NetworkContainerCleanupResponse{Completed: false}
	// First, check the network container table to see if the NC exists
	_, err := GetNetworkContainerConfiguration(networkID, ncID)
	if err == nil {
		// no error means the nc exists, so cleanup isn't done
		return cleanupResp, nil
	}

	if err.HTTPStatus != http.StatusNotFound {
		// expecting a 404, anything else should be considered an error and passed back to caller
		return cleanupResp, err
	}

	// NC doesn't exist. Now check pending cleanup jobs table to ensure there are no pending jobs for this NC
	// TODO: add this implementation
	// For now, assume there are no pending jobs and return true
	cleanupResp.Completed = true
	return cleanupResp, nil
}
