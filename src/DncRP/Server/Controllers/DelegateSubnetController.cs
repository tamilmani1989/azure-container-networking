// --------------------------------------------------------------------------------------------------------------------
// <copyright file="DelegateSubnetController.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Net;
    using System.Threading.Tasks;

    using Microsoft.AspNetCore.Http.Extensions;
    using Microsoft.AspNetCore.Mvc;
    using NetworkModel = Microsoft.Azure.Management.Network.Models;
    using Microsoft.Extensions.Logging;
    using Microsoft.Azure.Management.Network.Models;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Newtonsoft.Json;
    using System.IO;
    using System.Collections.Generic;
    using System.Net.Http;

    [Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroup}/providers/Microsoft.DelegatedNetwork/delegatedSubnets/{resourceName}")]
    public class DelegateSubnetController : DncRpControllerBase<DelegateSubnetController>
    {
        private IHttpUtility httpUtility;
        private AzVNetBroker azVnetBroker;

        public DelegateSubnetController(
            ILogger<DelegateSubnetController> logger,
            ITableOperations tableOperations,
            IHttpUtility httpUtility)
            : base (logger, tableOperations)
        {
            this.httpUtility = httpUtility;
            IAzureAD azureAD = new AzureAD();
            this.azVnetBroker = new AzVNetBroker(httpUtility, logger, azureAD);
            logger.LogInformation("DelegateSubnetController initialized");
        }

        /// <summary>
        /// GetDelegatedSubnet info
        /// </summary>
        /// <param name="subscriptionId">subscription id</param>
        /// <param name="resourceGroup">resource group name</param>
        /// <param name="resourceName">resourceName name</param>
        /// <returns></returns>
        [HttpGet]
        public async Task<ContentResult> GetDelegateSubnet(
            string subscriptionId,
            string resourceGroup,
            string resourceName)
        {
            this.logger.LogInformation($"GetDelegateSubnet URI: {HttpContext.Request.GetDisplayUrl()}");

            // TODO Validate arguments
            try
            {
                // Check if SubnetConfiguration is already defined
                tableOperations.SubscriptionOperation.ValidateSubscriptionRegistrationAsync(subscriptionId);
                // Check if DelegatedSubnetConfiguration exists
                DelegateSubnet delegatedSubnet = await tableOperations.DelegatedSubnetConfigurationOperations.TryGetDelegatedSubnetConfigurationAsync(
                    subscriptionId,
                    resourceGroup,
                    resourceName);

                if (delegatedSubnet == null)
                {
                    throw new ErrorResponseMessageException(
                        httpStatus: HttpStatusCode.NotFound,
                        errorCode: ErrorResponseCode.DelegatedSubnetResourceNotFound,
                        errorMessage: $"ResourceId: {HttpContext.Request.GetDisplayUrl()} not found");
                }

                ContentResult result = new ContentResult
                {
                    StatusCode = (int)HttpStatusCode.OK,
                    Content = JsonUtility.Serialize(delegatedSubnet),
                    ContentType = "application/json"
                };

                return result;
            }
            catch (Exception ex)
            {
                string message = $"Failed to process PutDelegateSubnet call.";
                int statusCode = (int)HttpStatusCode.InternalServerError;

                ErrorResponseMessageException errorException = ex as ErrorResponseMessageException;
                if (errorException != null)
                {
                    statusCode = (int)errorException.HttpStatus;
                    message += errorException.Message + $"InternalErrorCode: {errorException.ErrorCode}";
                }

                message += $"InternalException: {ex}";

                this.logger.LogError($"{message}, StatusCode: {statusCode}");

                return ConvertToExceptionResult(message, statusCode);
            }
        }

        /// <summary>
        /// PutDelegateSubnet.
        /// </summary>
        /// <param name="subscriptionId">subscription id</param>
        /// <param name="resourceGroup">resource group name</param>
        /// <param name="resourceName">resourece name</param>
        /// <returns></returns>
        [HttpPut]
        public async Task<ContentResult> PutDelegateSubnet(
            string subscriptionId, 
            string resourceGroup, 
            string resourceName)
        {
            this.logger.LogInformation($"PutDelegateSubnet URI: {HttpContext.Request.GetDisplayUrl()}");

            DelegateSubnetRequest delegateSubnetRequest = await GetRequestAsync<DelegateSubnetRequest>(HttpContext.Request.Body, "PutDelegateSubnet");
            return await this.ExecuteWithRetryAsync(subscriptionId, () =>
                ExecuteDelegateSubnetAsync(
                       subscriptionId,
                       resourceGroup,
                       resourceName,
                       delegateSubnetRequest),
                HttpMethod.Put,
                "PutDelegateSubnet");
        }

        /// <summary>
        /// PatchDelegateSubnet info
        /// Patch API is not supported
        /// </summary>
        /// <param name="subscriptionId">subscription id</param>
        /// <param name="resourceGroup">resource group name</param>
        /// <param name="resourceName">resourceName name</param>
        /// <returns></returns>
        [HttpPatch]
        public Task<ContentResult> PatchDelegateSubnet(
            string subscriptionId,
            string resourceGroup,
            string resourceName)
        {
            throw new NotSupportedException("Patch-DelegateSubnet is not supported. Try Put-DelegateSubnet");
        }

        /// <summary>
        /// DeleteDelegatedSubnet 
        /// </summary>
        /// <param name="subscriptionId">subscription id</param>
        /// <param name="resourceGroup">resource group name</param>
        /// <param name="resourceName">resourceName name</param>
        /// <returns></returns>
        [HttpDelete]
        public async Task<ContentResult> DeleteDelegateSubnet(
            string subscriptionId,
            string resourceGroup,
            string resourceName)
        {
            this.logger.LogInformation($"DeleteDelegateSubnet URI: {HttpContext.Request.GetDisplayUrl()}");

            // TODO Validate arguments
            try
            {

                tableOperations.SubscriptionOperation.ValidateSubscriptionRegistrationAsync(subscriptionId);
                await ExecuteDeleteDelegateSubnetAsync(subscriptionId, resourceGroup, resourceName);

                ContentResult result = new ContentResult
                {
                    StatusCode = (int)HttpStatusCode.OK,
                    Content = null,
                    ContentType = "application/json"
                };

                return result;
            }
            catch (Exception ex)
            {
                string message = $"Failed to process PutDelegateSubnet call.";
                int statusCode = (int)HttpStatusCode.InternalServerError;

                ErrorResponseMessageException errorException = ex as ErrorResponseMessageException;
                if (errorException != null)
                {
                    statusCode = (int)errorException.HttpStatus;
                    message += errorException.Message + $"InternalErrorCode: {errorException.ErrorCode}";
                }

                message += $"InternalException: {ex}";

                this.logger.LogError($"{message}, StatusCode: {statusCode}");

                return ConvertToExceptionResult(message, statusCode);
            }
        }

        /// <summary>
        /// Asynchronously Delegate a subnet and generate subnet token if not 
        /// generated.
        /// </summary>
        /// <param name="subscriptionId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="resourceName"></param>
        /// <param name="requestBody"></param>
        /// <returns></returns>
        private async Task<DelegateSubnet> ExecuteDelegateSubnetAsync(
            string subscriptionId, 
            string resourceGroup,
            string resourceName,
            DelegateSubnetRequest requestBody)
        {
            string associatedSubnetId = requestBody.Properties.Parameters.Properties.SubnetDetails.Id;
            string subnetSubId;
            string subnetResourceGroupId;
            string vnetName;
            string subnetName;
            ValidateAndGetSubnet(associatedSubnetId, out subnetSubId, out subnetResourceGroupId, out vnetName, out subnetName);

            // Check if DelegatedSubnetConfiguration exists
            DelegateSubnet delegateSubnet = await tableOperations.DelegatedSubnetConfigurationOperations.TryGetDelegatedSubnetConfigurationAsync(
                subscriptionId,
                resourceGroup,
                resourceName);

            if (delegateSubnet == null)
            {
                string tenantId = await tableOperations.SubscriptionOperation.GetTenantIdAsync(subnetSubId);

                // TODO: Validate if there doesn't exist any other DelegateSubnet associated to the same SubnetId
                // Get the VirtualNetwork and ensure it is ready to update.
                string VNetResourceId = AzVNetBroker.GetVNetResourceId(subnetSubId, subnetResourceGroupId, vnetName);
                NetworkModel.VirtualNetwork virtualNetwork = await azVnetBroker.GetVirtualNetworkAsync(tenantId, VNetResourceId);
                ValidateVirtualNetwork(virtualNetwork, VNetResourceId);

                // Insert the Subnet configuration
                string delegatedSubnetGuid = Guid.NewGuid().ToString();
                delegateSubnet = await tableOperations.DelegatedSubnetConfigurationOperations.InsertOrUpdateAsync(
                    subscriptionId,
                    resourceGroup,
                    resourceName,
                    delegatedSubnetGuid,
                    virtualNetwork.ResourceGuid,
                    requestBody);

                await CreateSubnetTokenAsync(
                    tenantId,
                    subnetSubId,
                    resourceGroup,
                    vnetName,
                    subnetName,
                    delegatedSubnetGuid,
                    virtualNetwork.ResourceGuid,
                    associatedSubnetId);
            }
            else
            {
                // check if SubnetToken exists
                string delegatedSubnetGuid = delegateSubnet.Properties.ResourceGuid;
                bool exists = await tableOperations.SubnetTokenMapOperations.SubnetTokenExistsAsync(delegatedSubnetGuid, subnetName);
                if (!exists)
                {
                    string tenantId = await tableOperations.SubscriptionOperation.GetTenantIdAsync(subnetSubId);

                    // Create if one is not created yet.
                    // TODO: Check if there is already an operation pending for this resource
                    await CreateSubnetTokenAsync(
                        tenantId,
                        subnetSubId,
                        resourceGroup,
                        vnetName,
                        subnetName,
                        delegatedSubnetGuid,
                        delegateSubnet.VNetResourceGuid,
                        associatedSubnetId);
                }
            }

            return delegateSubnet;
        }

        private async Task CreateSubnetTokenAsync(
            string tenantId,
            string subnetSubId, 
            string subnetResourceGroupId,
            string vnetName,
            string subnetName,
            string delegatedSubnetGuid, 
            string vnetResourceGuid,
            string subnetResourceId)
        {
            logger.LogInformation($"Creating SubnetToken for:subscriptionId {subnetSubId}, ResourceGroup: {subnetResourceGroupId}, " +
                $"delegatedSubnetGuid: {delegatedSubnetGuid}, vnetResourceGuid: {vnetResourceGuid}, SubnetId: {subnetResourceId}");

            string VNetResourceId = AzVNetBroker.GetVNetResourceId(subnetSubId, subnetResourceGroupId, vnetName);

            // SetResourceOwnership is already set
            // Avoid this call if Ownership is already set on a VNet
            ResourceOwnershipResponse resourceOwnershipResponse = 
                await azVnetBroker
                .SetResourceOwnershipForCustomerVnetAsync(
                    tenantId,
                    subnetSubId,
                    VNetResourceId);

            // Create the SAL and generate the token
            ServiceAssociationLinkDetail salDetail = await azVnetBroker.GetOrCreateServiceAssociationLinkAsync(
                tenantId, vnetResourceGuid.ToString(), subnetResourceId);

            await tableOperations.SubnetTokenMapOperations.InsertOrUpdateAsync(
                    delegatedSubnetGuid,
                    subnetName,
                    salDetail.Properties.PrimaryContextId,
                    salDetail.Properties.SecondaryContextId,
                    vnetResourceGuid);
        }

        private void ValidateAndGetSubnet(string subnetResourceId, 
            out string subnetSubId, 
            out string subnetResourceGroupId, 
            out string vnetName, 
            out string subnetName)
        {
            subnetSubId = String.Empty;
            subnetResourceGroupId = string.Empty;
            vnetName = string.Empty;
            subnetName = string.Empty;
            Dictionary<string, string> subnetResources;

            bool exists = ArmResourceIdParser.TryParseResourceId(subnetResourceId, "Microsoft.Network", out subnetSubId, out subnetResourceGroupId, out subnetResources);

            // Resource keys are parsed in lower case
            if (!exists ||
                !subnetResources.ContainsKey(ArmResourceIdParser.VirtualNetworkKey) || !subnetResources.ContainsKey(ArmResourceIdParser.SubnetKey))
            {
                throw new ErrorResponseMessageException(
                    httpStatus: HttpStatusCode.BadRequest,
                    errorCode: ErrorResponseCode.InvalidSubnetId,
                    errorMessage: $"SubnetId: {subnetResourceId} is invalid.");
            }

            vnetName = subnetResources[ArmResourceIdParser.VirtualNetworkKey];
            subnetName = subnetResources[ArmResourceIdParser.SubnetKey];
        }

        /// <summary>
        /// Validates the virtual network.
        /// </summary>
        /// <param name="subscription">The subscription.</param>
        /// <param name="containerGroup">The container group.</param>
        /// <param name="virtualNetwork">The azure virtual network.</param>
        /// <param name="virtualNetworkResourceId">The virtual network resource Id.</param>
        private static void ValidateVirtualNetwork(
            NetworkModel.VirtualNetwork virtualNetwork,
            string VNetResourceId)
        {
            if (virtualNetwork == null)
            {
                throw new ErrorResponseMessageException(
                    httpStatus: HttpStatusCode.BadRequest,
                    errorCode: ErrorResponseCode.VirtualNetworkNotFound,
                    errorMessage: $"ResourceId: {VNetResourceId} not found");
            }

            if (virtualNetwork.Location != CurrentContext.AzureRegion)
            {
                throw new ErrorResponseMessageException(
                   httpStatus: HttpStatusCode.BadRequest,
                   errorCode: ErrorResponseCode.InvalidVirtualNetworkLocation,
                   errorMessage: $"ResourceId: {VNetResourceId} is created in a different location: {virtualNetwork.Location}, Expected Location: {CurrentContext.AzureRegion}");
            }

            // todo - Handle validation for provisioning state

            //if (virtualNetwork.ProvisioningState != NetworkModel.ProvisioningState.Succeeded)
            //{
            //    throw new ErrorResponseMessageException(
            //        httpStatus: HttpStatusCode.BadRequest,
            //        errorCode: ErrorResponseCode.VirtualNetworkNotReady,
            //        errorMessage: $"ResourceId: {VNetResourceId} is not ready to allow Delegate Subnets. VirtualNetwork Provisioning State= {virtualNetwork.ProvisioningState}");
            //}
        }

        /// <summary>
        /// Asynchronously delete a DelegateSubnet
        /// </summary>
        /// <param name="subscriptionId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="resourceName"></param>
        /// <param name="requestBody"></param>
        /// <returns></returns>
        private async Task ExecuteDeleteDelegateSubnetAsync(
            string subscriptionId,
            string resourceGroup,
            string resourceName)
        {
            // Check if DelegatedSubnetConfiguration exists
            DelegateSubnet delegateSubnet = await tableOperations.DelegatedSubnetConfigurationOperations.TryGetDelegatedSubnetConfigurationAsync(
                subscriptionId,
                resourceGroup,
                resourceName);

            if (delegateSubnet == null)
            {
                logger.LogInformation($"DelegateSubnet, Resource {HttpContext.Request.GetDisplayUrl()} is already deleted");
                return;
            }

            // Remove the SAL in NRP
            DelegateSubnetProperties delegateSubnetProperties = delegateSubnet.Properties as DelegateSubnetProperties;

            string associatedSubnetId = delegateSubnetProperties.SubnetDetails.Id;
            string subnetSubId;
            string subnetResourceGroupId;
            string vnetName;
            string subnetName;
            ValidateAndGetSubnet(associatedSubnetId, out subnetSubId, out subnetResourceGroupId, out vnetName, out subnetName);

            // Delete the SAL
            string tenantId = await tableOperations.SubscriptionOperation.GetTenantIdAsync(subscriptionId);
            await azVnetBroker.DeleteServiceAssociationLinkAsync(tenantId, associatedSubnetId);

            // TODO handle deletion of Resourceownership if no other subnet has SAL created.

            // Delete the TokenMap entry synchronously for now.
            // Later we need to allow cleanup of NC's in this subnet in the background before token cleanup
            await tableOperations.SubnetTokenMapOperations.DeleteAsync(delegateSubnetProperties.ResourceGuid, subnetName);

            // Delete the DelegateSubnetConfiguration
            await tableOperations.DelegatedSubnetConfigurationOperations.DeleteAsync(subscriptionId, resourceGroup, resourceName);
        }
    }
}
