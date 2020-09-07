// --------------------------------------------------------------------------------------------------------------------
// <copyright file="SubscriptionOperation.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Net;
    using System.Net.Http;
    using System.Threading.Tasks;
    using Microsoft.AspNetCore.Authorization;
    using Microsoft.AspNetCore.Http.Extensions;
    using Microsoft.AspNetCore.Mvc;
    using Microsoft.Extensions.Configuration;
    using Microsoft.Extensions.Logging;
    using Newtonsoft.Json;

    public static class SubscriptionOperation
    {
        /// <summary>
        /// TODO - Validate Subscription is registered
        /// </summary>
        /// <param name="subscriptionId"></param>
        /// <returns></returns>
        public static Task ValidateSubscriptionIsRegistered(string subscriptionId)
        {
            throw new NotImplementedException();
        }
    }
}
