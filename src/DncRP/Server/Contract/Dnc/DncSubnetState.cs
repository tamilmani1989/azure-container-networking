// --------------------------------------------------------------------------------------------------------------------
// <copyright file="DncSubnetState.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    /// <summary>
    /// Resource provisioning states
    /// </summary>
    public class DncSubnetState
    {
        public const string New = "New";
        public const string Pending = "Pending";
        public const string Completed = "Completed";
        public const string Failed = "Failed";
    }
}
