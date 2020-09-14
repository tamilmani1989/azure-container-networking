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
        VirtualNetworkNotReady
    }
}