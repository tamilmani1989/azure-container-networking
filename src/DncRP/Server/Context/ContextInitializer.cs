// -----------------------------------------------------------------------------
// <copyright file="ContextInitializer.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.IO;
    using System.Linq;
    using System.Threading.Tasks;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Configuration;

    /// <summary>
    /// Current execution context
    /// </summary>
    public static class ContextInitializer
    {
        /// <summary>
        /// Initialize context
        /// </summary>
        /// <param name="configuration"></param>
        /// <param name="imdsRetriever"></param>
        public static void InitializeContext(IConfiguration configuration, IImdsRetriever imdsRetriever)
        {
            string azEnv = configuration["AzureEnvironment"];
            string region = configuration["AzureRegion"];

            if (string.IsNullOrWhiteSpace(azEnv) || string.IsNullOrWhiteSpace(region))
            {
                // https://github.com/davidfowl/AspNetCoreDiagnosticScenarios/blob/master/AsyncGuidance.md#avoid-using-taskrun-for-long-running-work-that-blocks-the-thread
                // Blocking is not ideal here but this approach avoids deadblock and unwraps any exception thrown inside delegate in Run method
                Task<InstanceResponse> task = Task.Run(() => imdsRetriever.GetInstanceMetadata(
                                  configuration["Imds:InstanceEndpoint"]));
                InstanceResponse response = task.GetAwaiter().GetResult();

                azEnv = response.Compute.AzureEnvironment;
                region = response.Compute.AzureRegion;
            }

            CurrentContext.SetContext(azEnv, region);

            ClientIdentity[] clientIdentities = configuration.GetSection("ClientIdentities").Get<ClientIdentity[]>();

            if (clientIdentities != null)
            {
                ClientIdentity managedIdetity = clientIdentities.FirstOrDefault(x => x.Tag == ClientIdentityTags.ManagedIdentity);
                ClientIdentity azureADArmTokenIdentity = clientIdentities.FirstOrDefault(x => x.Tag == ClientIdentityTags.AzureADArmToken);

                CurrentContext.SetManagedIdentity(managedIdetity);
                CurrentContext.SetArmAccessIdentity(azureADArmTokenIdentity);
            }

            CurrentContext.SetKeyVaultUrl(configuration["KeyVault:Url"]);
        }
    }
}
