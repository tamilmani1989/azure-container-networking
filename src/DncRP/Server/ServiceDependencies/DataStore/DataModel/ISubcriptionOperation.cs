// -----------------------------------------------------------------------------
// <copyright file="ISubscriptionOperation.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// ---------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Threading.Tasks;

    /// <summary>
    /// Defines SubnetTokenConfiguration interface for Data operations
    /// </summary>
    public interface ISubscriptionOperation
    {
        /// <summary>
        /// GetTenantInfo
        /// </summary>
        /// <param name="subscriptionId"></param>
        /// <returns></returns>
        Task<string> GetTenantIdAsync(string subscriptionId);

        /// <summary>
        /// IsSubscriptionRegistered
        /// </summary>
        /// <param name="subscriptionId">subscriptionId</param>
        /// <returns></returns>
        Task<bool> IsSubscriptionRegisteredAsync(
            string subscriptionId);

        /// <summary>
        /// Validate Subscription is registered
        /// </summary>
        /// <param name="subscriptionId"></param>
        /// <returns></returns>
        Task ValidateSubscriptionRegistrationAsync(string subscriptionId);

        /// <summary>
        /// Insert of update into SubscriptionResource table.
        /// </summary>
        /// <param name="subscriptionId">subscriptionId</param>
        /// <param name="tenantId">tenantId</param>
        /// <returns></returns>
        Task RegisterSubAsync(
            string subscriptionId,
            string tenantId,
            string requestBody);

        /// <summary>
        /// Insert of update into SubscriptionResource table.
        /// </summary>
        /// <param name="subscriptionId">subscriptionId</param>
        /// <param name="tenantId">tenantId</param>
        /// <param name="state">state</param>
        /// <returns></returns>
        Task UnregisterSubAsync(
            string subscriptionId);
    }
}
