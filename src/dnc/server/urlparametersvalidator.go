// Copyright 2019 Microsoft. All rights reserved.

package server

import (
	"dnc/api"
	"dnc/utils"
)

// Utility functions to validate URL parameters for different DNC requests.

// ValidateNetworkRequest - Validates network request URL parameters
func ValidateNetworkRequest(networkID string) *api.JsonErr {
	if !utils.IsValidUUID(networkID) {
		return api.BadRequestf(api.InvalidNetworkID, api.NetworkIDRequiresUUIDFormat)
	}

	return nil
}

// ValidateNetworkContainerRequest - Validates network container request URL parameters.
func ValidateNetworkContainerRequest(networkID string, networkContainerID string) *api.JsonErr {
	if validationErr := ValidateNetworkRequest(networkID); validationErr != nil {
		return validationErr
	}

	if !utils.IsValidUUID(networkContainerID) {
		return api.BadRequestf(api.InvalidNetworkContainerID, api.NetworkContainerIDRequiresUUIDFormat)
	}

	return nil
}

// ValidateReservationRequest - Validates reservation request URL parameters.
func ValidateReservationRequest(networkID string, reservationID string) *api.JsonErr {
	if validationErr := ValidateNetworkRequest(networkID); validationErr != nil {
		return validationErr
	}

	if !utils.IsValidUUID(reservationID) {
		return api.BadRequestf(api.InvalidReservationID, api.ReservationIDRequiresUUIDFormat)
	}

	return nil
}

// ValidateReservationSetRequest - Validates reservation set request URL parameters.
func ValidateReservationSetRequest(networkID string, reservationSetID string) *api.JsonErr {
	if validationErr := ValidateNetworkRequest(networkID); validationErr != nil {
		return validationErr
	}

	if !utils.IsValidUUID(reservationSetID) {
		return api.BadRequestf(api.InvalidReservationSetID, api.ReservationSetIDRequiresUUIDFormat)
	}

	return nil
}
