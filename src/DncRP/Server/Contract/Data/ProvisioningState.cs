// --------------------------------------------------------------------------------------------------------------------
// <copyright file="ProvisioningState.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    /// <summary>
    /// Resource provisioning states
    /// </summary>
    public enum ProvisioningState
    {
        Unknown = 0,
        Succeeded,
        Provisioning,
        Deleting,
        Failed
    }
}
