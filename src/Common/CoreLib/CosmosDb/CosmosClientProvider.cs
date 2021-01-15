// -----------------------------------------------------------------------------
// <copyright file="CosmosClientProvider.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using System;
    using System.Globalization;
    using System.Linq;
    using System.Threading;
    using System.Threading.Tasks;
    using Microsoft.Azure.Cosmos;
    using Microsoft.Extensions.Options;

    public sealed class CosmosClientProvider
    {
        private readonly DatastoreSettings datastoreSettings;
        private readonly AzureSettings azureSettings;
        private readonly SemaphoreSlim semaphoreSlim = new SemaphoreSlim(1);
        private readonly IHttpUtility httpUtility;
        private readonly ClientIdentityCollection clientIdentityCollection;
        private CosmosClient client;

        public CosmosClientProvider(
            IHttpUtility httpUtility,
            IOptions<DatastoreSettings> datastoreSettingsOptions,
            IOptions<AzureSettings> azureSettingsOptions,
            IOptions<ClientIdentityCollection> clientIdentityCollectionOptions)
        {
            this.httpUtility = httpUtility;
            this.datastoreSettings = datastoreSettingsOptions.Value;
            this.azureSettings = azureSettingsOptions.Value;
            this.clientIdentityCollection = clientIdentityCollectionOptions.Value;
        }

        public CosmosClient CosmosClient
        {
            get
            {
                if (this.client == null)
                {
                    throw new InvalidOperationException("Client is null, ensure InitializeAsync() is called on startup.");
                }

                return this.client;
            }
        }

        public async Task InitializeAsync(CancellationToken cancellationToken)
        {
            if (this.client == null)
            {
                await semaphoreSlim.WaitAsync();

                try
                {
                    if (this.client == null)
                    {
                        // Get azure AD token
                        ClientIdentity identity = this.clientIdentityCollection.Identities.FirstOrDefault(x => x.Tag == ClientIdentityTags.ManagedIdentity);

                        if (identity == null)
                        {
                            throw new InvalidOperationException("Could not find managed identity, ensure it's specified in the config file");
                        }

                        if (string.IsNullOrWhiteSpace(this.datastoreSettings.SubscriptionId))
                        {
                            throw new ArgumentNullException(nameof(this.datastoreSettings.SubscriptionId));
                        }

                        if (string.IsNullOrWhiteSpace(this.datastoreSettings.ResourceGroup))
                        {
                            throw new ArgumentNullException(nameof(this.datastoreSettings.ResourceGroup));
                        }

                        if (string.IsNullOrWhiteSpace(this.datastoreSettings.DBAccountName))
                        {
                            throw new ArgumentNullException(nameof(this.datastoreSettings.DBAccountName));
                        }

                        IAzureAD azureAD = new AzureAD();
                        string token = await azureAD.GetAccessToken(
                            identity,
                            AzureInfo.GetArmEndpoint(azureSettings.AzureEnvironment),
                            azureSettings.AzureEnvironment);

                        // Get data store (cosmos db) keys
                        ICosmosDb cosmosDb = new CosmosDb(this.httpUtility);
                        CosmosDbRWKeys keys = await cosmosDb.GetReadWriteKeys(
                            this.datastoreSettings.SubscriptionId,
                            this.datastoreSettings.ResourceGroup,
                            this.datastoreSettings.DBAccountName,
                            token,
                            azureSettings.AzureEnvironment);

                        this.client = new CosmosClient(
                            String.Format(CultureInfo.InvariantCulture, AzureInfo.GetCosmosDbEndpoint(azureSettings.AzureEnvironment), this.datastoreSettings.DBAccountName),
                            keys.PrimaryMasterKey,
                            new CosmosClientOptions()
                            {
                                AllowBulkExecution = true
                            });
                    }
                }
                finally
                {
                    semaphoreSlim.Release();
                }
            }
        }
    }
}
