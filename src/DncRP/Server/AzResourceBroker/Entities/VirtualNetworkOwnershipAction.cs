// -----------------------------------------------------------------------------
// <copyright file="VirtualNetworkOwnershipAction.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;

    /// <summary>
    /// The virtual network ownership action enum.
    /// </summary>
    public enum VirtualNetworkOwnershipAction
    {
        /// <summary>
        /// The action is not specified.
        /// </summary>
        NotSpecified = -1,

        /// <summary>
        /// The action is set.
        /// </summary>
        Set = 0,

        /// <summary>
        /// The action is remove.
        /// </summary>
        Remove = 1
    }
}
