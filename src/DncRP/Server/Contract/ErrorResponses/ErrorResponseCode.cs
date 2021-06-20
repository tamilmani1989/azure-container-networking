//-----------------------------------------------------------
// <copyright file="ErrorResponseCode.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
//-----------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    /// <summary>
    /// The error response code.
    /// </summary>
    public enum ErrorResponseCode
    {
        /// <summary>
        /// Not Set
        /// </summary>
        None,

        /// <summary>
        /// VirtualNetworkResource Not found
        /// </summary>
        VirtualNetworkNotFound,

        /// <summary>
        /// Invalidate VNet Location
        /// </summary>
        InvalidVirtualNetworkLocation,

        /// <summary>
        /// Virtual Network not ready to allow delegate Subnet
        /// Provisioning State is not Succeeded.
        /// </summary>
        VirtualNetworkNotReady,

        /// <summary>
        /// DelegatedSubnet Resource Not found
        /// </summary>
        DelegatedSubnetResourceNotFound,

        /// <summary>
        /// SubnetAlreadyDelegatedToDifferentResource
        /// </summary>
        SubnetAlreadyDelegatedToDifferentResource,

        /// <summary>
        /// Cant update AssociatedSubnetId
        /// </summary>
        AssociatedSubnetIdChanged,

        /// <summary>
        /// Cant update associated ControllerId
        /// </summary>
        ControllerIdChanged,

        /// <summary>
        /// Invalid Associated SubnetId
        /// </summary>
        InvalidSubnetId,

        /// <summary>
        /// Delete SubnetTokenMap failed
        /// </summary>
        SubnetTokenMapDeleteFailed,

        /// <summary>
        /// Failed to create SAL token
        /// </summary>
        ServiceAssociationLinkFailed,

        /// <summary>
        /// Controller ARM ID is not in correct format
        /// </summary>
        ControllerIdInvalidFormat,

        /// <summary>
        /// TenantIdUpdateNotSupported
        /// </summary>
        TenantIdUpdateNotSupported,

        /// <summary>
        /// SubscriptionRegistrationFails
        /// </summary>
        SubscriptionRegistrationFails,

        /// <summary>
        /// SubscriptionNotRegistered
        /// </summary>
        SubscriptionNotRegistered,

        /// <summary>
        /// SubscriptionResourceDeleteFailed
        /// </summary>
        SubscriptionResourceDeleteFailed,

        /// <summary>
        /// ControllerIdNotFound
        /// </summary>
        ControllerIdNotFound,

        /// <summary>
        /// JoinSubnetFailed
        /// </summary>
        JoinSubnetFailed,

        /// <summary>
        /// DeleteSubnetFailed
        /// </summary>
        DeleteSubnetFailed,

        /// <summary>
        /// SubnetNotFound
        /// </summary>
        SubnetNotFound,


        /// <summary>
        /// DependentResourcesExists
        /// </summary>
        DependentResourceExists,

        /// <summary>
        /// InvalidOrchestratorRequest
        /// </summary>
        InvalidOrchestratorRequest
    }
}