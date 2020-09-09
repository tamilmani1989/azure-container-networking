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
            ILogger logger)
            : base(httpUtility, logger)
        {
            this.logger = logger;
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
        /// Creates or updates the service association link.
        /// </summary>
        /// <param name="region">The location. TODO, get it from currentContext</param>
        /// <param name="subscriptionId">Subscription id to get TokenAccess</param>
        /// <param name="parentResourceGuid">The parent resource GUID.</param>
        /// <param name="subnetResourceId">The resource Id of the subnet.</param>
        /// <param name="primaryContextRequestId">The primary context request Id.</param>
        /// <param name="secondaryContextRequestId">The secondary context request Id.</param>
        public async Task<ServiceAssociationLink> CreateOrUpdateServiceAssociationLink(
            string region,
            string subscriptionId,
            Guid parentResourceGuid,
            string subnetResourceId,
            string primaryContextRequestId,
            string secondaryContextRequestId)
        {
            var sal = new ServiceAssociationLink
            {
                Properties = new ServiceAssociationLinkProperties
                {
                    LinkedResourceType = LinkedResourceType
                }
            };

            sal.Properties.AllowDelete = true;
            sal.Properties.Locations = new string[] { region };

            sal.Properties.ParentResourceGuid = parentResourceGuid.ToString();

            if (!string.IsNullOrEmpty(primaryContextRequestId) || !string.IsNullOrEmpty(secondaryContextRequestId))
            {
                sal.Properties.Details = new ServiceAssociationLinkDetail[]
                {
                new ServiceAssociationLinkDetail
                {
                    Name = SalDetailName,
                    Properties = new ServiceAssociationLinkDetailProperties()
                }
                };

                if (!string.IsNullOrEmpty(primaryContextRequestId))
                {
                    sal.Properties.Details[0].Properties.PrimaryContextRequestId = primaryContextRequestId;
                }

                if (!string.IsNullOrEmpty(secondaryContextRequestId))
                {
                    sal.Properties.Details[0].Properties.SecondaryContextRequestId = secondaryContextRequestId;
                }
            }

            return await PutResource<ServiceAssociationLink>(
                        subscriptionId: subscriptionId,
                        resourceId: this.GetSALResourceId(subnetResourceId),
                        apiVersion: ApiVersion,
                        resource: sal)
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
        public Task<ResourceOwnershipResponse> SetResourceOwnershipForCustomerVnet(
            string region,
            string subscriptionId,
            string resourceGroupName,
            string virtualNetworkName,
            VirtualNetworkOwnershipAction action)
        {
            var resourceId = $"/subscriptions/{subscriptionId}/providers/{ResourceProviderNamespace}/locations/{region}";

            var request = new ResourceOwnershipRequest
            {
                VirtualNetworks = new VirtualNetworkResourceOwnershipRequest[]
                {
                    new VirtualNetworkResourceOwnershipRequest
                    {
                        Id = $"/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{ResourceProviderNamespace}/{VNetResourceType}/{virtualNetworkName}",
                        OwnershipAction = action,
                        Properties = new VirtualNetworkOwnershipRequestProperties
                        {
                            EnableServiceEndpoints = true
                        }
                    }
                }
            };

            return PostResourceActionAsync<ResourceOwnershipRequest, ResourceOwnershipResponse>(
                subscriptionId: subscriptionId,
                resourceId: resourceId,
                action: "setResourceOwnership",
                apiVersion: ApiVersion,
                request: request);
        }
    }
}
