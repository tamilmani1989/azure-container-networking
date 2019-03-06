// Copyright 2019 Microsoft. All rights reserved.

package server

import (
	"dnc/api"
	"dnc/utils"
	"net/http"
)

// Utility functions to validate URL parameters for different DNC requests.

// ValidateNetworkRequest - Validates network request URL parameters
func ValidateNetworkRequest(networkID string) *api.JsonErr {
	if !utils.IsValidUUID(networkID) {
		return &api.JsonErr{Code: api.InvalidNetworkID,
			Text:       api.NetworkIDRequiresUUIDFormat,
			HTTPStatus: http.StatusBadRequest}
	}

	return nil
}

// ValidateNetworkContainerRequest - Validates network container request URL parameters.
func ValidateNetworkContainerRequest(networkID string, networkContainerID string) *api.JsonErr {
	if validationErr := ValidateNetworkRequest(networkID); validationErr != nil {
		return validationErr
	}

	if !utils.IsValidUUID(networkContainerID) {
		return &api.JsonErr{Code: api.InvalidNetworkContainerID,
			Text:       api.NetworkContainerIDRequiresUUIDFormat,
			HTTPStatus: http.StatusBadRequest}
	}

	return nil
}

// ValidateReservationRequest - Validates reservation request URL parameters.
func ValidateReservationRequest(networkID string, reservationID string) *api.JsonErr {
	if validationErr := ValidateNetworkRequest(networkID); validationErr != nil {
		return validationErr
	}

	if !utils.IsValidUUID(reservationID) {
		return &api.JsonErr{Code: api.InvalidReservationID,
			Text:       api.ReservationIDRequiresUUIDFormat,
			HTTPStatus: http.StatusBadRequest}
	}

	return nil
}
