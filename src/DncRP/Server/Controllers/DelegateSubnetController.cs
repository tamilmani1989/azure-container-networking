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
    using System.Collections.Generic;
    using System.Net.Http;

    [Route("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroup}/providers/Microsoft.DelegatedNetwork/delegatedSubnets/{resourceName}")]
    public class DelegateSubnetController : DncRpControllerBase<DelegateSubnetController>
    {
        private IHttpUtility httpUtility;
        private IHttpUtility dncHttpUtility;
        private AzVNetBroker azVnetBroker;

        public DelegateSubnetController(
            ILogger<DelegateSubnetController> logger,
            ITableOperations tableOperations,
            IHttpUtilityFactory httpUtilityFactory
        )
            : base (logger, tableOperations)
        {
            IAzureAD azureAD = new AzureAD();
            this.dncHttpUtility = httpUtilityFactory.Resolve(Constants.DncHttpUtility);
            this.httpUtility = httpUtilityFactory.Resolve(Constants.HttpUtility);
            this.azVnetBroker = new AzVNetBroker(httpUtility, logger, azureAD);
            logger.CLogInformation("DelegateSubnetController initialized");          
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
            this.logger.CLogInformation($"GetDelegateSubnet URI: {HttpContext.Request.GetDisplayUrl()}");

            // TODO Validate arguments
            try
            {
                // Check if SubnetConfiguration is already defined
                await tableOperations.SubscriptionOperation.ValidateSubscriptionRegistrationAsync(subscriptionId);
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
                string message = $"Failed to process GetDelegateSubnet call.";
                int statusCode = (int)HttpStatusCode.InternalServerError;

                ErrorResponseMessageException errorException = ex as ErrorResponseMessageException;
                if (errorException != null)
                {
                    statusCode = (int)errorException.HttpStatus;
                    message += errorException.Message + $"InternalErrorCode: {errorException.ErrorCode}";
                }

                // only return error message in response (instead stack trace) since this is public API
                string externalMessage = $"{message}. {ex.Message}";
                message += $"InternalException: {ex}";

                this.logger.CLogError($"{message}, StatusCode: {statusCode}");

                return ConvertToExceptionResult(externalMessage, statusCode);
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
            this.logger.CLogInformation($"PutDelegateSubnet URI: {HttpContext.Request.GetDisplayUrl()}");

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
            this.logger.CLogInformation($"DeleteDelegateSubnet URI: {HttpContext.Request.GetDisplayUrl()}");

            // TODO Validate arguments
            try
            {
                await tableOperations.SubscriptionOperation.ValidateSubscriptionRegistrationAsync(subscriptionId);
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
                string message = $"Failed to process DeleteDelegateSubnet call.";
                int statusCode = (int)HttpStatusCode.InternalServerError;

                ErrorResponseMessageException errorException = ex as ErrorResponseMessageException;
                if (errorException != null)
                {
                    statusCode = (int)errorException.HttpStatus;
                    message += errorException.Message + $"InternalErrorCode: {errorException.ErrorCode}.";
                }

                // only return error message in response (instead stack trace) since this is public API
                string externalMessage = $"{message} {ex.Message}";
                message += $"InternalException: {ex}";

                this.logger.CLogError($"{message}, StatusCode: {statusCode}");

                return ConvertToExceptionResult(externalMessage, statusCode);
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
            string associatedSubnetId = requestBody.Properties.SubnetDetails.Id;
            string controllerId = requestBody.Properties.ControllerDetails.Id;
            string subnetSubId;
            string subnetResourceGroupId;
            string vnetName;
            string subnetName;

            ValidateAndGetSubnet(associatedSubnetId, out subnetSubId, out subnetResourceGroupId, out vnetName, out subnetName);

            // check if controller exists
            Controller controller = await tableOperations.ControllerConfiguration.GetDncControllerAsync(
                                        controllerId);
            if (controller.Error != null)
            {
                throw new ErrorResponseMessageException(
                       httpStatus: HttpStatusCode.NotFound,
                       errorCode: ErrorResponseCode.ControllerIdNotFound,
                       errorMessage: controller.Error.Error);
            }

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
                    DncSubnetStatus.New,
                    requestBody);

                string authToken = await CreateSubnetTokenAsync(
                    tenantId,
                    vnetName,
                    subnetName,
                    delegatedSubnetGuid,
                    virtualNetwork.ResourceGuid,
                    associatedSubnetId);

                // call dnc gateway for join subnet
                await DncJoinSubnetAsync (
                     subscriptionId,
                     resourceGroup,
                     resourceName,
                     delegatedSubnetGuid,
                     virtualNetwork.ResourceGuid,
                     subnetName,
                     authToken,
                     controller,
                     requestBody);
            }
            else
            {
                // check if SubnetToken exists
                string delegatedSubnetGuid = delegateSubnet.Properties.ResourceGuid;
                string authToken = await tableOperations.SubnetTokenMapOperations.GetPrimarySubnetTokenAsync(delegatedSubnetGuid, subnetName);
                if (authToken == null)
                {
                    logger.CLogInformation("Delegatedsubnet resource exists but token not found. Going to create token");
                    string tenantId = await tableOperations.SubscriptionOperation.GetTenantIdAsync(subnetSubId);

                    // Create if one is not created yet.
                    // TODO: Check if there is already an operation pending for this resource
                    authToken = await CreateSubnetTokenAsync(
                        tenantId,
                        vnetName,
                        subnetName,
                        delegatedSubnetGuid,
                        delegateSubnet.VNetResourceGuid,
                        associatedSubnetId);
                }

                // if customer does multiple put subnet calls when dncsubnetstate in pending, RP calls
                // dnc for each put call. 
                if (delegateSubnet.SubnetJoinStatus != DncSubnetStatus.Completed)
                {
                    logger.CLogInformation("Put call. Token and delegated subnet resource exists. Dnc subnet join status:{0} for subnet:{1}", delegateSubnet.SubnetJoinStatus, subnetName);
                    // call dnc gateway for join subnet
                    await DncJoinSubnetAsync(
                         subscriptionId,
                         resourceGroup,
                         resourceName,
                         delegatedSubnetGuid,
                         delegateSubnet.VNetResourceGuid,
                         subnetName,
                         authToken,
                         controller,
                         requestBody);
                }              
            }

            return delegateSubnet;
        }

        private async Task DncJoinSubnetAsync(
            string subscriptionId,
            string resourceGroup,
            string resourceName,
            string delegatedSubnetGuid,
            string vnetId,
            string subnetName,
            string authToken,
            Controller controller,
            DelegateSubnetRequest dsRequest)
        {
            // get dnc endpoint from controller details
            ControllerProperties controllerProperties = (ControllerProperties)controller.Properties;
            logger.CLogInformation("Going to call join subnet for subnetname {0} vnetid{1} to dnc ep {2}", subnetName,
                 vnetId, controllerProperties.PropertiesList.DncEndpoint);

            // initialize dnc client with gateway endpoint
            DncClient dncClient = new DncClient(this.dncHttpUtility, this.logger, controllerProperties.PropertiesList.DncEndpoint);

            // call dnc join subnet
            await dncClient.PutSubnetAsync(vnetId, subnetName, authToken);
            await tableOperations.DelegatedSubnetConfigurationOperations.InsertOrUpdateAsync(
                                 subscriptionId,
                                 resourceGroup,
                                 resourceName,
                                 delegatedSubnetGuid,
                                 vnetId,
                                 DncSubnetStatus.Pending,
                                 dsRequest);

            // do dnc get subnet to check if join completed
            DncSettings dncSettings = CurrentContext.DncSettings;
            await AsyncRetry.Retry(() => CheckIfSubnetJoined(dncClient, vnetId, subnetName), dncSettings.RetryCount, 
                TimeSpan.FromSeconds(dncSettings.RetryIntervalInSeconds), TimeSpan.FromSeconds(dncSettings.RetryTimeoutInSeconds));
            await tableOperations.DelegatedSubnetConfigurationOperations.InsertOrUpdateAsync(
                      subscriptionId,
                      resourceGroup,
                      resourceName,
                      delegatedSubnetGuid,
                      vnetId,
                      DncSubnetStatus.Completed,
                      dsRequest);
        }

        private async Task CheckIfSubnetJoined(
            IDncClient dncClient,
            string vnetId,
            string subnetName)
        {
            HttpResponseMessage response = await dncClient.GetSubnetAsync(vnetId, subnetName);
            DncResponseStatus dncResponse = await this.dncHttpUtility.ParseHttpResponse<DncResponseStatus>(response);
            if (dncResponse == null || dncResponse.Status != DncSubnetStatus.Completed)
            {
                throw new Exception($"DNC get subnet call returns {dncResponse}");
            }
        }

        private async Task<string> CreateSubnetTokenAsync(
            string tenantId,
            string vnetName,
            string subnetName,
            string delegatedSubnetGuid, 
            string vnetResourceGuid,
            string subnetResourceId)
        {
            string subnetSubId;
            string subnetRg;
            Dictionary<string, string> resources;

            logger.CLogInformation($"Creating SubnetToken for delegatedSubnetGuid: {delegatedSubnetGuid}, vnetResourceGuid: {vnetResourceGuid}, SubnetId: {subnetResourceId}");

            bool exists = ArmResourceIdParser.TryParseResourceId(subnetResourceId, "Microsoft.Network", out subnetSubId, out subnetRg, out resources);
            if (!exists)
            {
                logger.CLogError("Invalid Subnet resourceid {0}", subnetResourceId);
                throw new Exception($"Invalid subnet resourceId {subnetResourceId}");
            }
            logger.CLogInformation("Subnet subid:{0} subnet rg:{1}", subnetSubId, subnetRg);

            string VNetResourceId = AzVNetBroker.GetVNetResourceId(subnetSubId, subnetRg, vnetName);

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
            return salDetail.Properties.PrimaryContextId;
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
                logger.CLogInformation($"DelegateSubnet, Resource {HttpContext.Request.GetDisplayUrl()} is already deleted");
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

            string tenantId = await tableOperations.SubscriptionOperation.GetTenantIdAsync(subscriptionId);
            string VNetResourceId = AzVNetBroker.GetVNetResourceId(subnetSubId, subnetResourceGroupId, vnetName);

            NetworkModel.VirtualNetwork virtualNetwork = await azVnetBroker.GetVirtualNetworkAsync(tenantId, VNetResourceId);
            ValidateVirtualNetwork(virtualNetwork, VNetResourceId);
            
            // call dnc and delete subnet
            await DncDeleteSubnetAsync(
                subscriptionId, 
                resourceGroup, 
                virtualNetwork.ResourceGuid, 
                subnetName, 
                delegateSubnet);

            logger.CLogInformation("Deleting ServiceAssociationLink for {0}", associatedSubnetId);
            // Delete the SAL
            HttpResponseMessage response = await azVnetBroker.DeleteServiceAssociationLinkAsync(tenantId, associatedSubnetId);
            response.EnsureSuccessStatusCode();

            // TODO handle deletion of Resourceownership if no other subnet has SAL created.

            // Delete the TokenMap entry synchronously for now.
            // Later we need to allow cleanup of NC's in this subnet in the background before token cleanup
            await tableOperations.SubnetTokenMapOperations.DeleteAsync(delegateSubnetProperties.ResourceGuid, subnetName);

            // Delete the DelegateSubnetConfiguration
            await tableOperations.DelegatedSubnetConfigurationOperations.DeleteAsync(subscriptionId, resourceGroup, resourceName);
        }

        private async Task DncDeleteSubnetAsync(
            string subscriptionId,
            string resourceGroupName,
            string vnetId,
            string subnetName,
            DelegateSubnet delegateSubnet)
        {
            // check if subnet join in progress or completed
            if (delegateSubnet.SubnetJoinStatus == DncSubnetStatus.New)
            {
                logger.CLogInformation("Subnet not joined. Not calling dnc delete subnet");
                return;
            }

            // Get dnc endpoint from controllerconfiguration
            DelegateSubnetProperties dsProperties = (DelegateSubnetProperties)delegateSubnet.Properties;
            Controller controller = await tableOperations.ControllerConfiguration.GetDncControllerAsync(
                                        dsProperties.ControllerDetails.Id);
            if (controller == null)
            {
                throw new ErrorResponseMessageException(
                       httpStatus: HttpStatusCode.NotFound,
                       errorCode: ErrorResponseCode.ControllerIdNotFound,
                       errorMessage: $"ResourceId: {HttpContext.Request.GetDisplayUrl()} not found");
            }

            ControllerProperties controllerProperties = (ControllerProperties)controller.Properties;
            logger.CLogInformation("Going to call dnc delete subnet for subnetname {0} vnet id {1} to dnc ep {2}", subnetName,
                 vnetId, controllerProperties.PropertiesList.DncEndpoint);
            // Initialize dnc client with gateway endpoint
            DncClient dncClient = new DncClient(this.dncHttpUtility, this.logger,
                                      controllerProperties.PropertiesList.DncEndpoint);
            
            // call dnc delete subnet
            await dncClient.DeleteSubnetAsync(vnetId, subnetName);

            DncSettings dncSettings = CurrentContext.DncSettings;
            // call dnc get subnet to check if subnet is deleted
            await AsyncRetry.Retry(() => CheckIfSubnetDeleted(dncClient, vnetId, subnetName), dncSettings.RetryCount,
                TimeSpan.FromSeconds(dncSettings.RetryIntervalInSeconds), TimeSpan.FromSeconds(dncSettings.RetryTimeoutInSeconds));
        }

        private async Task CheckIfSubnetDeleted(
            IDncClient dncClient,
            string vnetId,
            string subnetName)   
        {
            HttpResponseMessage response = await dncClient.GetSubnetAsync(vnetId, subnetName);
            if (response.StatusCode != HttpStatusCode.NotFound)
            {
                logger.CLogError("DNC Subnet Pending delete. status code:{0}", response.StatusCode);
                throw new Exception($"DNC Subnet Pending delete. status code:{response.StatusCode}");
            }
        }
    }
}
