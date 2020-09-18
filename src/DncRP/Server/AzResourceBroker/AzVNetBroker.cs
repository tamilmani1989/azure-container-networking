// -----------------------------------------------------------------------------
// <copyright file="AzVNetBroker.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Collections.Generic;
    using System.Threading.Tasks;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Configuration;
    using Microsoft.Extensions.Logging;
    using NetworkModel = Microsoft.Azure.Management.Network.Models;

    /// <summary>
    /// The Azure VNet Broker to get or update VNet resource.
    /// </summary>
    public class AzVNetBroker : AzResourceBroker
    {
        /// <summary>
        /// Network provider namespace.
        /// </summary>
        private const string ResourceProviderNamespace = "Microsoft.Network";

        /// <summary>
        /// Virtual network resource type.
        /// </summary>
        private const string VNetResourceType = "virtualNetworks";

        /// <summary>
        /// The API version for azure VNet operations.
        /// </summary>
        private const string ApiVersion = "2018-04-01";

        /// <summary>
        /// The service association link name.
        /// </summary>
        private const string SalName = "ManagedDNC";

        /// <summary>
        /// The service association link detail name.
        /// </summary>
        private const string SalDetailName = "ManagedDNCDetail";

        /// <summary>
        /// Managed DNC linked resource type constant.
        /// </summary>
        private const string LinkedResourceType = "Microsoft.DelegatedNetwork/controller";

        /// <summary>
        /// Initializes a new instance AzureVNetBroker class.
        /// </summary>
        /// <param name="httpUtility">HTTP Utility</param>
        /// <param name="logger">Logger</param>
        /// <param name="azEnvName">azEnvName (Temporary until we have Vipin's changes</param>
        public AzVNetBroker(
            IHttpUtility httpUtility, 
            ILogger logger,
            IAzureAD azureAd)
            : base(httpUtility, logger, azureAd)
        {
            this.logger = logger;
        }

        public static string GetVNetResourceId(string subscriptionId,
            string resourceGroup,
            string vnetName)
        {
            string resourceId = $"/subscriptions/{subscriptionId}/resourceGroups/{resourceGroup}/providers/{ResourceProviderNamespace}/virtualNetworks/{vnetName}";
            return resourceId;
        }

        /// <summary>
        /// Gets the SAL resourceId for the subnet.
        /// </summary>
        /// <param name="subnetResourceId">The resourceId of the subnet.</param>
        private string GetSALResourceId(string subnetResourceId)
        {
            return $"/{subnetResourceId.TrimEnd('/')}/ServiceAssociationLinks/{SalName}";
        }

        /// <summary>
        /// Get the virtual network.
        /// </summary>
        /// <param name="subscriptionId"></param>
        /// <param name="resourceGroupName"></param>
        /// <param name="vnetName"></param>
        /// <returns></returns>
        public async Task<NetworkModel.VirtualNetwork> GetVirtualNetworkAsync(
            string subscriptionId,
            string resourceId)
        {
            NetworkModel.VirtualNetwork virtualNetwork = await this.GetResourceAsync<NetworkModel.VirtualNetwork>(
                subscriptionId: subscriptionId,
                resourceId: resourceId,
                apiVersion: ApiVersion);

            logger.LogInformation($"GetVirtualNetworkAsync: Subscription: {subscriptionId}, resourceId: {resourceId}, " +
                $"VnetGuid: {virtualNetwork.ResourceGuid}, Response: {virtualNetwork}");

            return virtualNetwork;
        }

        /// <summary>
        /// Creates or updates the service association link.
        /// </summary>
        /// <param name="region">The location. TODO, get it from currentContext</param>
        /// <param name="subscriptionId">Subscription id to get TokenAccess</param>
        /// <param name="parentResourceGuid">The parent resource GUID.</param>
        /// <param name="subnetResourceId">The resource Id of the subnet.</param>
        /// <param name="primaryContextRequestId">The primary context request Id.</param>
        /// <param name="secondaryContextRequestId">The secondary context request Id.</param>
        public async Task<ServiceAssociationLink> CreateOrUpdateServiceAssociationLinkAsync(
            string subscriptionId,
            string parentResourceGuid,
            string subnetResourceId,
            Guid primaryContextRequestId,
            Guid secondaryContextRequestId)
        {
            if (primaryContextRequestId == Guid.Empty)
            {
                throw new ArgumentNullException("primaryContextRequestId");
            }

            var sal = new ServiceAssociationLink
            {
                Properties = new ServiceAssociationLinkProperties
                {
                    LinkedResourceType = LinkedResourceType
                }
            };

            //sal.Properties.AllowDelete = true;
            sal.Properties.Locations = new string[] { CurrentContext.AzureRegion };
            sal.Properties.ParentResourceGuid = parentResourceGuid.ToString();

            sal.Properties.Details = new ServiceAssociationLinkDetail[]
            {
                new ServiceAssociationLinkDetail
                {
                    Name = SalDetailName,
                    Properties = new ServiceAssociationLinkDetailProperties()
                }
            };

            sal.Properties.Details[0].Properties.PrimaryContextRequestId = primaryContextRequestId.ToString();

            if (secondaryContextRequestId != Guid.Empty)
            {
                sal.Properties.Details[0].Properties.SecondaryContextRequestId = secondaryContextRequestId.ToString();
            }

            return await PutResourceAsync<ServiceAssociationLink>(
                        subscriptionId: subscriptionId,
                        resourceId: this.GetSALResourceId(subnetResourceId),
                        apiVersion: ApiVersion,
                        requestBody: sal)
                    .ConfigureAwait(continueOnCapturedContext: false);
        }

        /// <summary>
        /// Sets resource ownership.
        /// </summary>
        /// <param name="region">region, get it from CurrentContext.</param>
        /// <param name="subscriptionId">The subscription Id.</param>
        /// <param name="resourceGroupName">The resource group name.</param>
        /// <param name="virtualNetworkName">The virtual network name.</param>
        /// <param name="action">The ownership action.</param>
        /// <param name="enableServiceEndpoints">The flag indicating whether enable service endpoints.</param>
        /// <returns>The resource ownership response.</returns>
        public async Task<ResourceOwnershipResponse> SetResourceOwnershipForCustomerVnetAsync(
            string subscriptionId,
            string vnetResourceId)
        {
            var resourceId = $"/subscriptions/{subscriptionId}/providers/{ResourceProviderNamespace}/locations/{CurrentContext.AzureRegion}";

            var request = new ResourceOwnershipRequest
            {
                VirtualNetworks = new VirtualNetworkResourceOwnershipRequest[]
                {
                    new VirtualNetworkResourceOwnershipRequest
                    {
                        Id = vnetResourceId,
                        OwnershipAction = VirtualNetworkOwnershipAction.Set,
                        Properties = new VirtualNetworkOwnershipRequestProperties
                        {
                            EnableServiceEndpoints = true
                        }
                    }
                }
            };

            return await PostResourceActionAsync<ResourceOwnershipRequest, ResourceOwnershipResponse>(
                subscriptionId: subscriptionId,
                resourceId: resourceId,
                action: "setResourceOwnership",
                apiVersion: ApiVersion,
                request: request);
        }
    }
}
