// -----------------------------------------------------------------------------
// <copyright file="ConnectionStringProvider.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Collections.Generic;
    using System.Threading;
    using System.Threading.Tasks;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Options;

    /// <summary>
    /// Note: This provider class will change once we settle on an approach to rotate key.
    /// TODO: Add unit tests. Righ now, CosmosDbConnection tests give coverage to this class
    /// TODO: but those are service tests (e2e test) and not unit tests.
    /// </summary>
    public class ConnectionStringProvider : IConnectionStringProvider
    {
        private DatastoreSettings datastoreSettings;
        private IEnumerable<string> connectionStringsList;
        private readonly SemaphoreSlim semaphoreSlim;
        private IHttpUtility httpUtility;

        public ConnectionStringProvider(
            IHttpUtility httpUtility,
            IOptionsMonitor<DatastoreSettings> datastoreSettingsOptions)
        {
            this.httpUtility = httpUtility;
            this.datastoreSettings = datastoreSettingsOptions.CurrentValue;
            this.semaphoreSlim = new SemaphoreSlim(1, 1);
            this.connectionStringsList = null;
        }

        public async Task<IEnumerable<string>> GetConnectionStringsForTableAPI()
        {
            if (this.connectionStringsList == null)
            {
                await semaphoreSlim.WaitAsync();

                try
                {
                    if (this.connectionStringsList == null)
                    {
                        // Get azure AD token
                        IAzureAD azureAD = new AzureAD();
                        string token = await azureAD.GetAccessToken(
                            CurrentContext.ManagedIdentity,
                            AzureInfo.GetArmEndpoint(CurrentContext.AzureEnvironment),
                            CurrentContext.AzureEnvironment);

                        // Get data store (cosmos db) keys
                        ICosmosDb cosmosDb = new CosmosDb(this.httpUtility);
                        CosmosDbRWKeys keys = await cosmosDb.GetReadWriteKeys(
                            this.datastoreSettings.SubscriptionId,
                            this.datastoreSettings.ResourceGroup,
                            this.datastoreSettings.DBAccountName,
                            token,
                            CurrentContext.AzureEnvironment);

                        string connStrFormat = CosmosDbUriFormats.TableAPIConnectionStringFormat;
                        List<string> list = new List<string>();
                        list.Add(string.Format(
                            connStrFormat,
                            this.datastoreSettings.DBAccountName,
                            keys.PrimaryMasterKey,
                            this.datastoreSettings.DBAccountName));

                        list.Add(string.Format(
                            connStrFormat,
                            this.datastoreSettings.DBAccountName,
                            keys.SecondaryMasterKey,
                            this.datastoreSettings.DBAccountName));

                        this.connectionStringsList = list;
                    }
                }
                finally
                {
                    semaphoreSlim.Release();
                }
            }

            return this.connectionStringsList;
        }
    }
}
