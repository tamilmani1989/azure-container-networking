// -----------------------------------------------------------------------------
// <copyright file="CurrentContext.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
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

        public static void SetContext(string azureEnv, string azureRegion)
        {
            AzureEnvironment = azureEnv;
            AzureRegion = azureRegion;
        }
    }
}
