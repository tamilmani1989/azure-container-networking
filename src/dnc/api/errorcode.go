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

	// AllocationFailed - Allocation failed due to unavailability of the resource
	AllocationFailed

	// NodeProcessingFailedDueToInternalError - Processing of node failed due to internal error
	NodeProcessingFailedDueToInternalError

	// StatusInternalServerError - Internal server error.
	StatusInternalServerError

	// NetworkStatusInvalid - Network status in invalid.
	NetworkStatusInvalid

	// NetworkStateInvalid - Network state is in invalid state.
	NetworkStateInvalid

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

	// InvalidOrchestratorType - Node orchestratory type specified is invalid.
	InvalidOrchestratorType

	// InvalidContainerType - Network container type specified is invalid.
	InvalidContainerType

	// NodeNotRegistered - Node is not registered
	NodeNotRegistered

	// UnSupprotedNetworkType - NetworkType is not supported
	UnSupprotedNetworkType

	// OutOfVlanIDs - VlanID is not supported/exceeded range
	OutOfVlanIDs

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

	// DecodeResponseFailed - Deocoding of http response failed
	DecodeResponseFailed

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

	// JoinFailed - Network join failed.
	JoinFailed

	// JoinPending - Network join is pending.
	JoinPending

	// InvalidNetworkContainerRequest - Network container request is invalid
	InvalidNetworkContainerRequest

	// NetworkConfigurationRetrievalFailed - Retrieval of network configuration retrieval failed.
	NetworkConfigurationRetrievalFailed

	// GetTagDefinitionFailed - Retrieval of tag definitionfailed.
	GetTagDefinitionFailed

	// AgentTagDefinitionRetrievalFailed - Tag definition retrieval from agent failed.
	AgentTagDefinitionRetrievalFailed

	// RepositoryTagRetrievalFailed - Retrieval of tags definition from repository failed.
	RepositoryTagRetrievalFailed

	// TagResolutionPending - Resoultion of tag refernce is pending with Agent
	TagResolutionPending

	// InvalidNetworkStateForDelete - Delete cannot be performed on the current state
	InvalidNetworkStateForDelete

	// CnsNCVersionRetrievalFailed - Retrieval of NC version from CNS Failed
	CnsNCVersionRetrievalFailed

	// InvalidAzureStorageInfo - There is not enough information to initialize azure storage
	InvalidAzureStorageInfo

	// StorageNotInitialized - Storage has not been initialized
	StorageNotInitialized

	// NodeContainerCapacityExhausted - The node has reached it's capacity for deployed network containers.
	NodeContainerCapacityExhausted

	// InvalidBlobName - BlobName cannot be empty.
	InvalidBlobName

	// InvalidContainerName - ContainerName cannot be empty.
	InvalidContainerName

	// UnSupportedAddressFamily - Address family is not supported.
	UnSupportedAddressFamily

	// RequestNotsupported - Request is not supported.
	RequestNotsupported

	// InvalidNodeRequest - Invalid node request.
	InvalidNodeRequest

	// InvalidSubnetRequest - Invalid subnet request.
	InvalidSubnetRequest

	// InvalidNetworkProfileId - Invalid network profile id
	InvalidNetworkProfileId

	// InvalidNetworkRequest - Invalid Network request
	InvalidNetworkRequest

	// NumberOfCPUCoresRetrievalFailed - Failed to retrieve CPU Cores
	NumberOfCPUCoresRetrievalFailed

	// PropertyValueTooLargeStorageError - Storage operation failed due to large property value
	PropertyValueTooLargeStorageError

	// FailedToUpdateSubnetWithReservationInfo - Failed to update subnet configuration in storage with reservation info.
	FailedToUpdateSubnetWithReservationInfo

	// StatusBadRequest - Bad request.
	StatusBadRequest

	//Invalid endpoint policy
	InvalidEndpointPolicy

	// UnregisteredNode - Cannot sync with an unregistered node
	UnregisteredNode

	// InvalidNCLimit - Invalid NC limit (failed to parse string)
	InvalidNCLimit

	// InvalidNodeAZID - Invalid AZID field for a Node
	InvalidNodeAZID

	// InvalidNodeType - Invalid node type
	InvalidNodeType

	// InvalidContainerPlacement - - Invalid node location type for the container Type
	InvalidContainerPlacement

	// InvalidNodeRegistrationRequest - Node Registration request is invalid
	InvalidNodeRegistrationRequest

	// OutOfGreKeys - GRE Key is not supported/exceeded range
	OutOfGreKeys

	// UpdateIPAllocationFailed - Failed to update IP Allocation entity
	UpdateIPAllocationFailed

	// PrepareCleanup Failed - Error occurred during PrepareCleanup state
	PrepareCleanupFailed

	// Property not set - Storage operation failed as expected property is not set
	PropertyNotSetInDb

	// Property not set or empty
	InvalidOrEmptyToBeDeletedOwner

	// NcPendingPublishEntryNotFound is return when the a row in the NcPendingPublishTable is not found
	NcPendingPublishEntryNotFound

	// InvalidPubSubOperationType - - Invalid pub sub operation type
	InvalidPubSubOperationType

	// InvalidOrchestratorContext - Invalid orchestrator context
	InvalidOrchestratorContext

	// SubnetInUse - Subnet in use with allocations
	SubnetInUse

	// ReservationTypeNotMatch means reservation request type doesn't match
	// e.g. reservation has been used for LB front end IP, then no NC can create on it.
	ReservationTypeNotMatch
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

	case StatusBadRequest:
		return "StatusBadRequest"

	case NetworkStatusInvalid:
		return "NetworkStatusInvalid"

	case NetworkStateInvalid:
		return "NetworkStateInvalid"

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

	case InvalidReservationID:
		return "InvalidReservationID"

	case NodeNotRegistered:
		return "NodeNotRegistered"

	case UnSupprotedNetworkType:
		return "UnSupprotedNetworkType"

	case OutOfVlanIDs:
		return "OutOfVlanIDs"

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

	case InvalidOrchestratorType:
		return "InvalidOrchestratorType"

	case InvalidContainerType:
		return "InvalidContainerType"

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

	case DecodeResponseFailed:
		return "DecodeResponseFailed"

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

	case JoinFailed:
		return "JoinFailed"

	case JoinPending:
		return "JoinPending"

	case NetworkConfigurationRetrievalFailed:
		return "NetworkConfigurationRetrievalFailed"

	case GetTagDefinitionFailed:
		return "GetTagDefinitionFailed"

	case AgentTagDefinitionRetrievalFailed:
		return "AgentTagDefinitionRetrievalFailed"

	case RepositoryTagRetrievalFailed:
		return "RepositoryTagRetrievalFailed"

	case TagResolutionPending:
		return "TagResolutionPending"

	case InvalidNetworkContainerRequest:
		return "InvalidNetworkContainerRequest"

	case InvalidNetworkStateForDelete:
		return "InvalidNetworkStateForDelete"

	case CnsNCVersionRetrievalFailed:
		return "CnsNCVersionRetrievalFailed"

	case InvalidAzureStorageInfo:
		return "InvalidAzureStorageInfo"

	case StorageNotInitialized:
		return "StorageNotInitialized"

	case NodeContainerCapacityExhausted:
		return "NodeContainerCapacityExhausted"

	case InvalidBlobName:
		return "InvalidBlobName"

	case InvalidContainerName:
		return "InvalidContainerName"

	case UnSupportedAddressFamily:
		return "UnSupportedAddressFamily"

	case RequestNotsupported:
		return "RequestNotsupported"

	case InvalidNodeRequest:
		return "InvalidNodeRequest"

	case InvalidSubnetRequest:
		return "InvalidSubnetRequest"

	case InvalidNetworkProfileId:
		return "InvalidNetworkProfileId"

	case InvalidNetworkRequest:
		return "InvalidNetworkRequest"

	case NumberOfCPUCoresRetrievalFailed:
		return "NumberOfCPUCoresRetrievalFailed"

	case PropertyValueTooLargeStorageError:
		return "PropertyValueTooLargeStorageError"

	case FailedToUpdateSubnetWithReservationInfo:
		return "FailedToUpdateSubnetWithReservationInfo"

	case InvalidEndpointPolicy:
		return "InvalidEndpointPolicy"

	case UnregisteredNode:
		return "UnregisteredNode"

	case InvalidNCLimit:
		return "InvalidNCLimit"

	case InvalidNodeAZID:
		return "InvalidNodeAZID"

	case InvalidNodeType:
		return "InvalidNodeType"

	case InvalidContainerPlacement:
		return "InvalidContainerPlacement"

	case InvalidNodeRegistrationRequest:
		return "InvalidNodeRegistrationRequest"

	case OutOfGreKeys:
		return "OutOfGreKeys"

	case NcPendingPublishEntryNotFound:
		return "NcPendingPublishEntryNotFound"

	case InvalidPubSubOperationType:
		return "InvalidPubSubOperationType"

	case InvalidOrchestratorContext:
		return "InvalidOrchestratorContext"

	case SubnetInUse:
		return "SubnetInUse"

	case ReservationTypeNotMatch:
		return "ReservationTypeNotMatch"
	}

	return "Unknown Status"
}
