// -----------------------------------------------------------------------------
// <copyright file="ICosmosDb.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using System.Threading.Tasks;

    public interface ICosmosDb
    {
        /// <summary>
        /// Get read write keys for CosmosDb
        /// </summary>
        /// <param name="subscriptionId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="cosmosDbAccount"></param>
        /// <param name="azureADToken"></param>
        /// <param name="azureEnvironment"></param>
        /// <returns></returns>
        Task<CosmosDbRWKeys> GetReadWriteKeys(
            string subscriptionId,
            string resourceGroup,
            string cosmosDbAccount,
            string azureADToken,
            string azureEnvironment);

        /// <summary>
        ///  Get read only keys for CosmosDb
        /// </summary>
        /// <param name="subscriptionId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="cosmosDbAccount"></param>
        /// <param name="azureADToken"></param>
        /// <param name="azureEnvironment"></param>
        /// <returns></returns>
        Task<CosmosDbReadOnlyKeys> GetReadOnlyKeys(
         string subscriptionId,
         string resourceGroup,
         string cosmosDbAccount,
         string azureADToken,
         string azureEnvironment);
    }
}