// Copyright 2019 Microsoft. All rights reserved.

package api

// Error messages sent to the client as part of failed dnc requests.
const (
	// NetworkIDRequiresUUIDFormat - Network ID parameter has to follow UUID format.
	NetworkIDRequiresUUIDFormat string = "Network ID must be a UUID"

	// NetworkContainerIDRequiresUUIDFormat - Network Container ID parameter has to follow UUID format.
	NetworkContainerIDRequiresUUIDFormat string = "Network Container ID must be a UUID"

	// NetworkIDMissing - Network ID parameter is missing from the URL.
	NetworkIDMissing string = "Network ID is missing"

	// NetworkContainerIDMissing - Network Container ID parameter is missing from the URL.
	NetworkContainerIDMissing string = "Network Container ID is missing"

	// NodeIDMissing - Node ID parameter is missing from the URL.
	NodeIDMissing string = "Node ID is missing"

	// NodeIDNonGUIDValue - Node ID value must be a GUID in certain scenarios (BM Mode).
	NodeIDNonGUIDValue string = "Node ID value must be a GUID."

	// ReservationIDMissing - Reservation ID parameter is missing from the URL.
	ReservationIDMissing string = "Reservation ID is missing"

	// ReservationIDRequiresUUIDFormat - Reservation ID parameter has to follow UUID format.
	ReservationIDRequiresUUIDFormat string = "Reservation ID must be a UUID"

	// SubnetNameMissing - subnet parameter is missing from the URL.
	SubnetNameMissing string = "SubnetName is missing"

	// NCLimitMissing - nc limit parameter is missing from the URL.
	NCLimitMissing string = "NC limit is missing"

	// ReservationSetIDMissing - ReservationSet ID parameter is missing from the URL.
	ReservationSetIDMissing string = "ReservationSet ID is missing"

	// ReservationSetIDRequiresUUIDFormat - ReservationSet ID parameter has to follow UUID format.
	ReservationSetIDRequiresUUIDFormat string = "ReservationSet ID must be a UUID"
)
