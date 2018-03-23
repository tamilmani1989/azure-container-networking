package api

// ErrorCode - Error Code reported to the client
type ErrorCode int

const (
	// PrimaryNotAvilable
	PrimaryNotAvailable ErrorCode = iota + 1

	// NetworkNotFound - Network not found
	NetworkNotFound

	// SubnetNotFound - Subnet not found
	SubnetNotFound

	// ReservationNotFound - Reservation not found
	ReservationNotFound

	// JoinDenied - Network join denied.
	JoinDenied

	// NodeNotFound - Node not found
	NodeNotFound

	// AllocationProcessFailedInternalError - Allocation processing failed due to internal error
	AllocationProcessFailedInternalError

	// AllocationFailed - Allocation failed due to unavailiabilty of resource
	AllocationFailed

	// NodeProcessingFailedDueToInternalError - Processing of node failed due to internal error
	NodeProcessingFailedDueToInternalError

	// StatusInternalServerError - Internal server error.
	StatusInternalServerError

	// NetworkStatusInvalid - Network status in invalid.
	NetworkStatusInvalid

	// SubnetConfigurationNotFound - Subnet configuration not found
	SubnetConfigurationNotFound

	// SubnetIsNotDelegated - Subnet is not a delegated subnet.
	SubnetIsNotDelegated

	// SubnetIsNotJoined - Subnet is not joined.
	SubnetIsNotJoined

	// NetworkInUse - Network is in use.
	NetworkInUse

	// ReservationInUse - Reservation is in use.
	ReservationInUse

	// InvalidIPSubnet - Invalid subnet
	InvalidIPSubnet

	// InvalidIPConstraint - Invalid IP constraint
	InvalidIPConstraint

	// IPConstraintIsOutsideSubnetRange - IPConstraint is not in subnet range
	IPConstraintIsOutsideSubnetRange

	// InvalidAuthenticationToken - Access token is invalid.
	InvalidAuthenticationToken

	// InvalidSubnetName - Subnet name is invalid.
	InvalidSubnetName

	// InvalidNodeID -  Node ID specified is invalid.
	InvalidNodeID

	// InvalidReservationID -  Reservation ID specified is invalid.
	InvalidReservationID

	// InvalidNetworkID -  Network ID specified is invalid.
	InvalidNetworkID

	// InvalidNetworkContainerID -  Network container ID specified is invalid.
	InvalidNetworkContainerID

	// NodeNotRegistered - Node is not registered
	NodeNotRegistered

	// UnSupprotedNetworkType - NetworkType is not supported
	UnSupprotedNetworkType

	// InvalidNodeIPAddress -Invalid node IPAddress
	InvalidNodeIPAddress

	// JsonBodyUnMarshalFailed - Failed to unmarshal Json body
	JsonBodyUnMarshalFailed

	// JsonMarshallingFailed - Failed to marshal
	JsonMarshallingFailed

	// JsonUnMarshallingFailed - Failed to Unmarshal
	JsonUnMarshallingFailed

	// NodeInUse - Node is in use
	NodeInUse

	// NetworkContainerNotFound - Network container not found.
	NetworkContainerNotFound

	// APIVersionMissing - Api version is missing in request
	APIVersionMissing

	// APIVersionNotSupported - Api version is not supported
	APIVersionNotSupported

	// ReservationStateInvalid - Reservation state is invalid
	ReservationStateInvalid

	// NodeConstraintMismatch - mismatch in node constraint and NC association
	NodeConstraintMismatch

	// NetworkJoinFailed - Join of network failed.
	NetworkJoinFailed

	// DeocodeResponseFailed - Deocoding of http response failed
	DeocodeResponseFailed

	// ParseResponseCodeFailed - Parsing of response code failed
	ParseResponseCodeFailed

	// PublishingFailed - NMAgent publishing failed
	PublishingFailed

	// UnpublishingFailed - NMAgent unpublishing failed
	UnpublishingFailed

	// ProcessingPending - Processing or request is pending.
	ProcessingPending

	// AssociationFailed - Association of NC to nic failed.
	AssociationFailed

	// DisAssociationFailed - Disassociation failed.
	DisAssociationFailed
)

func (e ErrorCode) String() string {
	switch e {
	case PrimaryNotAvailable:
		return "PrimaryNotAvailable"

	case NetworkNotFound:
		return "NetworkNotFound"

	case SubnetNotFound:
		return "SubnetNotFound"

	case JoinDenied:
		return "JoinDenied"

	case NodeNotFound:
		return "NodeNotFound"

	case AllocationProcessFailedInternalError:
		return "AllocationProcessFailedInternalError"

	case AllocationFailed:
		return "AllocationFailed"

	case NodeProcessingFailedDueToInternalError:
		return "NodeProcessingFailedDueToInternalError"

	case StatusInternalServerError:
		return "StatusInternalServerError"

	case NetworkStatusInvalid:
		return "NetworkStatusInvalid"

	case SubnetConfigurationNotFound:
		return "SubnetConfigurationNotFound"

	case SubnetIsNotDelegated:
		return "SubnetIsNotDelegated"

	case NetworkInUse:
		return "NetworkInUse"

	case ReservationInUse:
		return "ReservationInUse"

	case InvalidIPSubnet:
		return "InvalidIPSubnet"

	case InvalidAuthenticationToken:
		return "InvalidAuthenticationToken"

	case InvalidSubnetName:
		return "InvalidSubnetName"

	case InvalidNodeID:
		return "InvalidNodeID"

	case NodeNotRegistered:
		return "NodeNotRegistered"

	case UnSupprotedNetworkType:
		return "UnSupprotedNetworkType"

	case InvalidNodeIPAddress:
		return "InvalidNodeIPAddress"

	case JsonBodyUnMarshalFailed:
		return "JsonBodyUnMarshalFailed"

	case JsonMarshallingFailed:
		return "JsonMarshallingFailed"

	case JsonUnMarshallingFailed:
		return "JsonUnMarshallingFailed"

	case NodeInUse:
		return "NodeInUse"

	case NetworkContainerNotFound:
		return "NetworkContainerNotFound"

	case APIVersionMissing:
		return "APIVersionMissing"

	case APIVersionNotSupported:
		return "APIVersionNotSupported"

	case InvalidNetworkID:
		return "InvalidNetworkID"

	case InvalidNetworkContainerID:
		return "InvalidNetworkContainerID"

	case InvalidIPConstraint:
		return "InvalidIPConstraint"

	case IPConstraintIsOutsideSubnetRange:
		return "IPConstraintIsOutsideSubnetRange"

	case SubnetIsNotJoined:
		return "SubnetIsNotJoined"

	case ReservationStateInvalid:
		return "ReservationStateInvalid"

	case ReservationNotFound:
		return "ReservationNotFound"

	case NodeConstraintMismatch:
		return "NodeConstraintMismatch"

	case NetworkJoinFailed:
		return "NetworkJoinFailed"

	case DeocodeResponseFailed:
		return "DeocodeResponseFailed"

	case ParseResponseCodeFailed:
		return "ParseResponseCodeFailed"

	case PublishingFailed:
		return "PublishingFailed"

	case ProcessingPending:
		return "ProcessingPending"

	case AssociationFailed:
		return "AssociationFailed"

	case DisAssociationFailed:
		return "DisAssociationFailed"

	case UnpublishingFailed:
		return "UnpublishingFailed"
	}

	return "Unknown Status"
}
