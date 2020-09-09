// -----------------------------------------------------------------------------
// <copyright file="CurrentContext.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Microsoft.Azure.ContainerNetworking.CoreLib;

    /// <summary>
    /// Current execution context
    /// </summary>
    public static class CurrentContext
    {
        public static string AzureEnvironment
        {
            get; private set;
        }

        public static string AzureRegion
        {
            get; private set;
        }

        public static ClientIdentity ManagedIdentity
        {
            get; private set;
        }

        public static ClientIdentity ArmAccessIdentity
        {
            get; private set;
        }

        public static void SetContext(string azureEnv, string azureRegion)
        {
            AzureEnvironment = azureEnv;
            AzureRegion = azureRegion;
        }

        public static void SetManagedIdentity(ClientIdentity managedIdentity)
        {
            ManagedIdentity = managedIdentity;
        }

        public static void SetArmAccessIdentity(ClientIdentity armAccessIdentity)
        {
            ArmAccessIdentity = armAccessIdentity;
        }
    }
}
