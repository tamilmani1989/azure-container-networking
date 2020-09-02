// -----------------------------------------------------------------------------
// <copyright file="CosmosDbKeys.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using System;
    using System.Net.Http;
    using System.Net.Http.Headers;
    using System.Threading.Tasks;

    public class CosmosDb : ICosmosDb
    {
        private IHttpUtility httpUtility;

        /// <summary>
        /// Create instance of CosmosDb class
        /// </summary>
        /// <param name="httpUtility">an instance of HttpUtility class</param>
        public CosmosDb(IHttpUtility httpUtility)
        {
            this.httpUtility = httpUtility;
        }

        /// <summary>
        /// Get read write keys
        /// </summary>
        /// <param name="subscriptionId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="cosmosDbAccount"></param>
        /// <param name="azureADToken"></param>
        /// <param name="azureEnvironment"></param>
        /// <returns></returns>
        public async Task<CosmosDbRWKeys> GetReadWriteKeys(
            string subscriptionId, string resourceGroup, string cosmosDbAccount, string azureADToken, string azureEnvironment)
        {
            CosmosDbKeys keys = await this.GetKeys(subscriptionId, resourceGroup, cosmosDbAccount, azureADToken, azureEnvironment);
            return new CosmosDbRWKeys { PrimaryMasterKey = keys.PrimaryMasterKey, SecondaryMasterKey = keys.SecondaryMasterKey };
        }

        /// <summary>
        /// Get readonly keys
        /// </summary>
        /// <param name="subscriptionId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="cosmosDbAccount"></param>
        /// <param name="azureADToken"></param>
        /// <param name="azureEnvironment"></param>
        /// <returns></returns>
        public async Task<CosmosDbReadOnlyKeys> GetReadOnlyKeys(
            string subscriptionId, string resourceGroup, string cosmosDbAccount, string azureADToken, string azureEnvironment)
        {
            CosmosDbKeys keys = await this.GetKeys(subscriptionId, resourceGroup, cosmosDbAccount, azureADToken, azureEnvironment);
            return new CosmosDbReadOnlyKeys { PrimaryReadonlyMasterKey = keys.PrimaryMasterKey, SecondaryReadonlyMasterKey = keys.SecondaryMasterKey };
        }

        /// <summary>
        /// Get all keys from invoking ARM URI
        /// </summary>
        /// <param name="subscriptionId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="cosmosDbAccount"></param>
        /// <param name="azureADToken"></param>
        /// <param name="azureEnvironment"></param>
        /// <returns></returns>
        private async Task<CosmosDbKeys> GetKeys(
            string subscriptionId, string resourceGroup, string cosmosDbAccount, string azureADToken, string azureEnvironment)
        {
            // https://management.azure.com/subscriptions/9b8218f9-902a-4d20-a65c-e98acec5362f/resourceGroups/testCosmosdb/providers/Microsoft.DocumentDb/databaseAccounts/acntestcosmosdb/listKeys/?api-version=2016-03-31
            string urlFormat = CosmosDbUriFormats.ListKeysUriFormat;
            string url = string.Format(
                urlFormat,
                AzureInfo.GetArmEndpoint(azureEnvironment),
                subscriptionId,
                resourceGroup,
                cosmosDbAccount);

            HttpRequestMessage requestMessage = new HttpRequestMessage
            {
                Method = HttpMethod.Post,
                RequestUri = new Uri(url),
            };

            requestMessage.Headers.Authorization = new AuthenticationHeaderValue("Bearer", azureADToken);

            CosmosDbKeys keys = await this.httpUtility.SendRequestAndGetJsonAsync<CosmosDbKeys>(requestMessage);
            return keys;
        }
    }
}