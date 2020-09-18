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
    }
}