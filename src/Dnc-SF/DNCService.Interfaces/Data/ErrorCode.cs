//--------------------------------------------------------------------------
// <copyright file="AssociationInfo.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
// Error code enums. 
// </summary>
//--------------------------------------------------------------------------

using System.Runtime.Serialization;

namespace DNCService.API
{
    /// <summary>
    /// Error code enumerations.
    /// </summary>
    [DataContract]
    public enum ErrorCode
    {
        [EnumMember]
        PrimaryNotAvailable = 1,

        [EnumMember]
        NetworkNotFound = 2,

        [EnumMember]
        SubnetNotFound = 3,

        [EnumMember]
        ReservationNotFound = 4,

        [EnumMember]
        JoinDenied = 5,

        [EnumMember]
        NodeNotFound = 6,

        [EnumMember]
        AllocationProcessFailedInternalError = 7,

        [EnumMember]
        AllocationFailed = 8,

        [EnumMember]
        NodeProcessingFailedDueToInternalError = 9,

        [EnumMember]
        StatusInternalServerError = 10,

        [EnumMember]
        NetworkStatusInvalid = 11,

        [EnumMember]
        NetworkStateInvalid = 12,

        [EnumMember]
        SubnetConfigurationNotFound = 13,

        [EnumMember]
        SubnetIsNotDelegated = 14,

        [EnumMember]
        SubnetIsNotJoined = 15,

        [EnumMember]
        NetworkInUse = 16,

        [EnumMember]
        ReservationInUse = 17,

        [EnumMember]
        InvalidIPSubnet = 18,

        [EnumMember]
        InvalidIPConstraint = 19,

        [EnumMember]
        IPConstraintIsOutsideSubnetRange = 20,

        [EnumMember]
        InvalidAuthenticationToken = 21,

        [EnumMember]
        InvalidSubnetName = 22,

        [EnumMember]
        InvalidNodeID = 23,

        [EnumMember]
        InvalidReservationID = 24,

        [EnumMember]
        InvalidNetworkID = 25,

        [EnumMember]
        InvalidNetworkContainerID = 26,

        [EnumMember]
        InvalidOrchestratorType = 27,

        [EnumMember]
        InvalidContainerType = 28,

        [EnumMember]
        NodeNotRegistered = 29,

        [EnumMember]
        UnSupportedNetworkType = 30,

        [EnumMember]
        OutOfVlanIDs = 31,

        [EnumMember]
        InvalidNodeIPAddress = 32,

        [EnumMember]
        JsonBodyUnMarshalFailed = 33,

        [EnumMember]
        JsonMarshallingFailed = 34,

        [EnumMember]
        JsonUnMarshallingFailed = 35,

        [EnumMember]
        NodeInUse = 36,

        [EnumMember]
        NetworkContainerNotFound = 37,

        [EnumMember]
        APIVersionMissing = 38,

        [EnumMember]
        APIVersionNotSupported = 39,

        [EnumMember]
        ReservationStateInvalid = 40,

        [EnumMember]
        NodeConstraintMismatch = 41,

        [EnumMember]
        NetworkJoinFailed = 42,

        [EnumMember]
        DecodeResponseFailed = 43,

        [EnumMember]
        ParseResponseCodeFailed = 44,

        [EnumMember]
        PublishingFailed = 45,

        [EnumMember]
        UnpublishingFailed = 46,

        [EnumMember]
        ProcessingPending = 47,

        [EnumMember]
        AssociationFailed = 48,

        [EnumMember]
        DisAssociationFailed = 49,

        [EnumMember]
        JoinFailed = 50,

        [EnumMember]
        JoinPending = 51,

        [EnumMember]
        InvalidNetworkContainerRequest = 52,

        [EnumMember]
        NetworkConfigurationRetrievalFailed = 53,

        [EnumMember]
        GetTagDefinitionFailed = 54,

        [EnumMember]
        AgentTagDefinitionRetrievalFailed = 55,

        [EnumMember]
        RepositoryTagRetrievalFailed = 56,

        [EnumMember]
        TagResolutionPending = 57,

        [EnumMember]
        InvalidNetworkStateForDelete = 58,

        [EnumMember]
        CnsNCVersionRetrievalFailed = 59,

        [EnumMember]
        InvalidAzureStorageInfo = 60,

        [EnumMember]
        StorageNotInitialized = 61,

        [EnumMember]
        NodeContainerCapacityExhausted = 62,

        [EnumMember]
        UnSupportedAddressFamily = 63,

        [EnumMember]
        RequestNotsupported = 64,

        [EnumMember]
        InvalidNodeRequest = 65,

        [EnumMember]
        InvalidSubnetRequest = 66,

        [EnumMember]
        InvalidNetworkProfileId = 67,

        [EnumMember]
        InvalidNetworkRequest = 68,

        // Note: This Enum struct must be identical to dnc/api/errorcode.go
        // Update both when adding new enum members.
    }

    /// <summary>
    /// Error types
    /// </summary>
    [DataContract]
    public enum ErrorType
    {
        [EnumMember]
        Retriable = 0,

        [EnumMember]
        NonRetriable = 1,
    }
}
