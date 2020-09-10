// --------------------------------------------------------------------------------------------------------------------
// <copyright file="DelegateSubnetController.cs" company="Microsoft">
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

    public class DelegateSubnetController : DncRpControllerBase<DelegateSubnetController>
    {
        public DelegateSubnetController (
            ILogger<DelegateSubnetController> logger,
            ITableOperations tableOperations)
            : base (logger, tableOperations)
        {
            logger.LogInformation("DelegateSubnetController initialized");
        }

        /// <summary>
        /// PutDelegateSubnet. 
        /// This is an Extension resource call Microsoft.Network/VirtualNetwork resource
        /// 
        /// TODO: Add DeleteDelegatedSubnet call
        /// </summary>
        /// <param name="subscriptionId">subscription id</param>
        /// <param name="resourceGroup">resource group name</param>
        /// <param name="vnetName">VirtualNetwork name</param>
        /// <param name="subnet">Subnet name</param>
        /// <param name="delegatedId">Delegated subnet resource name</param>
        /// <returns></returns>
        [Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroup}/providers/Microsoft.Network/virtualNetworks/{vnetName}/subnets/{subnet}/providers/Microsoft.DelegatedNetwork/delegatedSubnets/{delegatedId}")]
        [HttpPut]
        public async Task<ContentResult> PutDelegateSubnet(
            string subscriptionId, 
            string resourceGroup, 
            string vnetName, 
            string subnet, 
            string delegatedId)
        {
            this.logger.LogInformation($"PutDelegateSubnet URI: {HttpContext.Request.GetDisplayUrl()}");

            // TODO Validate arguments

            DelegateSubnetDefinition delegatedSubnetDefinition = await ExecuteDelegateSubnetAsync(subscriptionId, resourceGroup, vnetName, subnet, delegatedId);

            // TODO - For long running operations in ARM need to return OperationStatus or ProvisioningState
            ContentResult result = new ContentResult { 
                StatusCode = (int)HttpStatusCode.OK, 
                Content = JsonConvert.SerializeObject(delegatedSubnetDefinition), 
                ContentType = "application/json"
            };

            return result;
        }

        /// <summary>
        /// Asynchronously Delegate a subnet and generate subnet token if not 
        /// generated.
        /// </summary>
        /// <param name="subscriptionId">subscription id</param>
        /// <param name="resourceGroup">resource group name</param>
        /// <param name="vnetName">Virtual Network name</param>
        /// <param name="subnet">Subnet</param>
        /// <param name="delegatedId">delegated Subnet Id</param>
        /// <returns></returns>
        private async Task<DelegateSubnetDefinition> ExecuteDelegateSubnetAsync(
            string subscriptionId, 
            string resourceGroup, 
            string vnetName, 
            string subnet, 
            string delegatedId)
        {
            // Check if SubnetConfiguration is already defined
            await SubscriptionOperation.ValidateSubscriptionIsRegistered(subscriptionId);

            // Todo get the subnet token configuration

            DelegateSubnetDefinition subDef = new DelegateSubnetDefinition(subscriptionId, resourceGroup, vnetName, subnet, delegatedId, Guid.NewGuid());
            return subDef;
        }
    }
}
