// --------------------------------------------------------------------------------------------------------------------
// <copyright file="DelegateRequestParameters.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;
    using System;
    using System.Collections;
    using System.IO;

    public class SubscriptionProperties
    {
        [JsonProperty("tenantId")]
        public string TenantId { get; set; }

        [JsonProperty("locationPlacementId")]
        public string LocationPlacementId { get; set; }

        [JsonProperty("quotaId")]
        public string QuotaId { get; set; }
    }

    /// <summary>
    /// Subscription Registration
    /// https://github.com/Azure/azure-resource-manager-rpc/blob/master/v1.0/subscription-lifecycle-api-reference.md
    /// </summary>
    public class SubscriptionRegistrationRequest
    {
        [JsonProperty("state")]
        public string State { get; set; }

        [JsonProperty("registrationDate")]
        public DateTime RegistrationDate { get; set; }

        public SubscriptionProperties Properties
        {
            get; set;
        }

        /// <summary>
        /// Convert state to enum
        /// </summary>
        /// <param name="state"></param>
        /// <returns></returns>
        public static SubscriptionRegistrationState ParseRegistrationState(string state)
        {
            return (SubscriptionRegistrationState)Enum.Parse(typeof(SubscriptionRegistrationState), state, true);
        }
    }
}
