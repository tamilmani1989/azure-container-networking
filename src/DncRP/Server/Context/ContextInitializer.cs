// -----------------------------------------------------------------------------
// <copyright file="ContextInitializer.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
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

            CurrentContext.SetContext(azEnv.ToLower(), region.ToLower());

            ClientIdentity[] clientIdentities = configuration.GetSection("ClientIdentities").Get<ClientIdentity[]>();

            if (clientIdentities != null)
            {
                ClientIdentity managedIdetity = clientIdentities.FirstOrDefault(x => x.Tag == ClientIdentityTags.ManagedIdentity);
                ClientIdentity azureADArmTokenIdentity = clientIdentities.FirstOrDefault(x => x.Tag == ClientIdentityTags.AzureADArmToken);

                CurrentContext.SetManagedIdentity(managedIdetity);
                CurrentContext.SetArmAccessIdentity(azureADArmTokenIdentity);
            }

            CurrentContext.SetKeyVaultUrl(configuration["KeyVault:Url"]);

            PrintContextAndConfigToConsole(configuration);
        }

        private static void PrintContextAndConfigToConsole(IConfiguration configuration)
        {
            Console.WriteLine("AzureEnvironment: {0}", CurrentContext.AzureEnvironment);
            Console.WriteLine("AzureRegion: {0}", CurrentContext.AzureRegion);

            if (CurrentContext.ManagedIdentity != null)
            {
                Console.WriteLine("ManagedIdentity: {0}", CurrentContext.ManagedIdentity.AppId);
            }

            if (CurrentContext.ArmAccessIdentity != null)
            {
                Console.WriteLine("ArmAccessIdentity AppID: {0}", CurrentContext.ArmAccessIdentity.AppId);
                Console.WriteLine("ArmAccessIdentity KVCertPath: {0}", CurrentContext.ArmAccessIdentity.KeyVaultCertPath);
            }

            Console.WriteLine("Thumbprint: {0}", configuration["Thumbprint"]);
            Console.WriteLine("InstrumentationKey: {0}", configuration["ApplicationInsights:InstrumentationKey"]);

            DatastoreSettings settings = configuration.GetSection("DataStoreSettings").Get<DatastoreSettings>();
            if (settings != null)
            {
                Console.WriteLine("Datastore SubscriptionId: {0}", settings.SubscriptionId);
                Console.WriteLine("Datastore DBAccountName: {0}", settings.DBAccountName);
                Console.WriteLine("Datastore ResourceGroup: {0}", settings.ResourceGroup);
                Console.WriteLine("Datastore Region: {0}", settings.Region);
            }

            MdmMetricsConfig mdmMetricsConfig = configuration.GetSection("MdmMetrics").Get<MdmMetricsConfig>();
            if (mdmMetricsConfig != null)
            {
                Console.WriteLine("MdmMetrics AccountName: {0}", mdmMetricsConfig.AccountName);
                Console.WriteLine("MdmMetrics Namespace: {0}", mdmMetricsConfig.Namespace);
                Console.WriteLine("MdmMetrics StatsDHost: {0}", mdmMetricsConfig.StatsDHost);
                Console.WriteLine("MdmMetrics StatsDPort: {0}", mdmMetricsConfig.StatsDPort);
                Console.WriteLine("MdmMetrics StatsDSocketProtocol: {0}", mdmMetricsConfig.StatsDSocketProtocol);
            }
        }
    }
}
