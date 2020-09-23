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

    [Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroup}/providers/Microsoft.Network/virtualNetworks/{vnetName}/subnets/{subnet}/providers/Microsoft.DelegatedNetwork/delegatedSubnets/{delegatedSubnetName}")]
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
        /// <param name="vnetName">VirtualNetwork name</param>
        /// <param name="subnet">Subnet name</param>
        /// <param name="delegatedId">Delegated subnet resource name</param>
        /// <returns></returns>
        [HttpGet]
        public async Task<ContentResult> GetDelegateSubnet(
            string subscriptionId,
            string resourceGroup,
            string vnetName,
            string subnet,
            string delegatedSubnetName)
        {
            this.logger.LogInformation($"GetDelegateSubnet URI: {HttpContext.Request.GetDisplayUrl()}");

            // TODO Validate arguments
            try
            {
                // Check if SubnetConfiguration is already defined
                SubscriptionOperation.ValidateSubscriptionIsRegistered(subscriptionId);

                // Check if DelegatedSubnetConfiguration exists
                DelegateSubnet delegatedSubnet = await tableOperations.DelegatedSubnetConfigurationOperations.TryGetDelegatedSubnetConfigurationAsync(
                    subscriptionId,
                    resourceGroup,
                    vnetName,
                    subnet,
                    delegatedSubnetName);

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

                return ConvertToExceptionResult(message, statusCode);
            }
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
        [HttpPut]
        public async Task<ContentResult> PutDelegateSubnet(
            string subscriptionId, 
            string resourceGroup, 
            string vnetName, 
            string subnet, 
            string delegatedSubnetName)
        {
            this.logger.LogInformation($"PutDelegateSubnet URI: {HttpContext.Request.GetDisplayUrl()}");

            // TODO Validate arguments
            try
            {
                DelegateSubnet delegatedSubnetDefinition = await ExecuteDelegateSubnetAsync(subscriptionId, resourceGroup, vnetName, subnet, delegatedSubnetName);

                // TODO - For long running operations in ARM need to return OperationStatus or ProvisioningState
                ContentResult result = new ContentResult
                {
                    StatusCode = (int)HttpStatusCode.OK,
                    Content = JsonConvert.SerializeObject(delegatedSubnetDefinition),
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

                return ConvertToExceptionResult(message, statusCode);
            }
        }

        /// <summary>
        /// PatchDelegateSubnet info
        /// Patch API is not supported
        /// </summary>
        /// <param name="subscriptionId">subscription id</param>
        /// <param name="resourceGroup">resource group name</param>
        /// <param name="vnetName">VirtualNetwork name</param>
        /// <param name="subnet">Subnet name</param>
        /// <param name="delegatedId">Delegated subnet resource name</param>
        /// <returns></returns>
        [HttpPatch]
        public Task<ContentResult> PatchDelegateSubnet(
            string subscriptionId,
            string resourceGroup,
            string vnetName,
            string subnet,
            string delegatedId)
        {
            throw new NotSupportedException("Patch-DelegateSubnet is not supported. Try Put-DelegateSubnet");
        }

        /// <summary>
        /// DeleteDelegatedSubnet 
        /// </summary>
        /// <param name="subscriptionId">subscription id</param>
        /// <param name="resourceGroup">resource group name</param>
        /// <param name="vnetName">VirtualNetwork name</param>
        /// <param name="subnet">Subnet name</param>
        /// <param name="delegatedId">Delegated subnet resource name</param>
        /// <returns></returns>
        [HttpDelete]
        public Task<ContentResult> DeleteDelegateSubnet(
            string subscriptionId,
            string resourceGroup,
            string vnetName,
            string subnet,
            string delegatedId)
        {
            // TODO - Support this API later.
            throw new NotSupportedException();
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
        private async Task<DelegateSubnet> ExecuteDelegateSubnetAsync(
            string subscriptionId, 
            string resourceGroup, 
            string vnetName, 
            string subnet, 
            string resourceName)
        {
            // Check if SubnetConfiguration is already defined
            SubscriptionOperation.ValidateSubscriptionIsRegistered(subscriptionId);

            // Check if DelegatedSubnetConfiguration exists
            DelegateSubnet delegateSubnet = await tableOperations.DelegatedSubnetConfigurationOperations.TryGetDelegatedSubnetConfigurationAsync(
                subscriptionId,
                resourceGroup,
                vnetName,
                subnet,
                resourceName,
                throwOnResourceNameComparison: true);

            if (delegateSubnet == null)
            {
                // Get the VirtualNetwork and ensure it is ready to update.
                string VNetResourceId = AzVNetBroker.GetVNetResourceId(subscriptionId, resourceGroup, vnetName);
                NetworkModel.VirtualNetwork virtualNetwork = await GetAzureVNetDataProvider().GetVirtualNetworkAsync(subscriptionId, VNetResourceId);
                ValidateVirtualNetwork(virtualNetwork, VNetResourceId);

                // Insert the Subnet configuration
                string delegatedSubnetGuid = Guid.NewGuid().ToString();
                delegateSubnet = await tableOperations.DelegatedSubnetConfigurationOperations.InsertOrUpdateAsync(
                    subscriptionId,
                    resourceGroup,
                    vnetName,
                    subnet,
                    resourceName,
                    delegatedSubnetGuid,
                    virtualNetwork.ResourceGuid);

                await CreateSubnetTokenAsync(
                    subscriptionId,
                    resourceGroup,
                    vnetName,
                    subnet,
                    delegatedSubnetGuid,
                    virtualNetwork.ResourceGuid);
            }
            else
            {
                // check if SubnetToken exists
                string delegatedSubnetGuid = delegateSubnet.Properties.ResourceGuid;
                bool exists = await tableOperations.SubnetTokenMapOperations.SubnetTokenExistsAsync(delegatedSubnetGuid, subnet);
                if (!exists)
                {
                    // Create if one is not created yet.
                    // TODO: Check if there is already an operation pending for this resource
                    await CreateSubnetTokenAsync(
                        subscriptionId, 
                        resourceGroup,
                        vnetName, 
                        subnet,
                        delegatedSubnetGuid,
                        delegateSubnet.VNetResourceGuid);
                }
            }

            return delegateSubnet;
        }

        private async Task CreateSubnetTokenAsync(
            string subscriptionId, 
            string resourceGroup, 
            string vnetName, 
            string subnet, 
            string delegatedSubnetGuid, 
            string vnetResourceGuid)
        {
            logger.LogInformation($"Creating SubnetToken for:subscriptionId {subscriptionId}, ResourceGroup: {resourceGroup}, " +
                $"vnetName: {vnetName}, subnet: {subnet}, delegatedSubnetGuid: {delegatedSubnetGuid}, vnetResourceGuid: {vnetResourceGuid}");
            string VNetResourceId = AzVNetBroker.GetVNetResourceId(subscriptionId, resourceGroup, vnetName);

            // SetResourceOwnership is already set
            // Avoid this call if Ownership is already set on a VNet
            ResourceOwnershipResponse resourceOwnershipResponse = 
                await GetAzureVNetDataProvider()
                .SetResourceOwnershipForCustomerVnetAsync(
                    subscriptionId,
                    VNetResourceId);

            // Create the SAL and generate the token
            string subnetResourceId = $"{VNetResourceId.TrimEnd('/')}/subnets/{subnet}";
            ServiceAssociationLinkDetail salDetail = await GetAzureVNetDataProvider().GetOrCreateServiceAssociationLinkAsync(
                subscriptionId, vnetResourceGuid.ToString(), subnetResourceId);

            await tableOperations.SubnetTokenMapOperations.InsertOrUpdateAsync(
                    delegatedSubnetGuid,
                    subnet,
                    salDetail.Properties.PrimaryContextId,
                    salDetail.Properties.SecondaryContextId,
                    vnetResourceGuid);
        }


        /// <summary>
        /// Gets the azure VNet data provider.
        /// </summary>
        /// <returns>The azure VNet data provider.</returns>
        private AzVNetBroker GetAzureVNetDataProvider()
        {
            return azVnetBroker;
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
    }
}
