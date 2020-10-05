// --------------------------------------------------------------------------------------------------------------------
// <copyright file="SubscriptionRegistrationState.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    /// <summary>
    /// Resource provisioning states
    /// </summary>
    public enum SubscriptionRegistrationState
    {
        Registered = 0,
        Unregistered,
        Warned,
        Suspended,
        Deleted
    }
}
