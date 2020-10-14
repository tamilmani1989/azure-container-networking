// -----------------------------------------------------------------------------
// <copyright file="AzVNetBroker.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.IO;
    using System.Net;
    using System.Net.Http;
    using System.Threading.Tasks;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
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

        public static string GetVNetResourceId(
            string subscriptionId,
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
        /// Gets the SAL detail resourceId for the subnet.
        /// </summary>
        /// <param name="subnetResourceId">The resourceId of the subnet.</param>
        private string GetSALDetailResourceId(string subnetResourceId)
        {
            return $"{subnetResourceId.TrimEnd('/')}/ServiceAssociationLinks/{SalName}/Details/{SalDetailName}";
        }

        /// <summary>
        /// Get the virtual network.
        /// </summary>
        /// <param name="tenantId">Customer TenantId</param>
        /// <param name="resourceId">VNet resource Id</param>
        /// <returns></returns>
        public async Task<NetworkModel.VirtualNetwork> GetVirtualNetworkAsync(
            string tenantId,
            string resourceId)
        {
            NetworkModel.VirtualNetwork virtualNetwork = await this.GetResourceAsync<NetworkModel.VirtualNetwork>(
                tenantId: tenantId,
                resourceId: resourceId,
                apiVersion: ApiVersion);

            logger.CLogInformation($"GetVirtualNetworkAsync: tenantId: {tenantId}, resourceId: {resourceId}, " +
                $"VnetGuid: {virtualNetwork.ResourceGuid}, Response: {virtualNetwork}");

            return virtualNetwork;
        }


        /// <summary>
        /// Gets or Creates the service association link.
        /// </summary>
        /// <param name="tenantId">Customer tenantId</param>
        /// <param name="vnetResourceId">VnetResourceId</param>
        /// <param name="subnetResourceId">Subnet resourceId</param>
        /// <returns></returns>
        public async Task<ServiceAssociationLinkDetail> GetOrCreateServiceAssociationLinkAsync(
            string tenantId,
            string vnetResourceId,
            string subnetResourceId)
        {
            ServiceAssociationLink salRequest = null;
            try
            {
                ServiceAssociationLinkDetail detail = await TryGetServiceAssociationLinkDetailAsync(tenantId, subnetResourceId);
                if (detail != null && !String.IsNullOrEmpty(detail?.Properties?.PrimaryContextId))
                {
                    return detail;
                }

                // Create the SAL
                salRequest = CreateServiceAssociationLinkRequest(vnetResourceId);

                ServiceAssociationLink salResponse = await PutResourceAsync<ServiceAssociationLink>(
                            tenantId: tenantId,
                            resourceId: this.GetSALResourceId(subnetResourceId),
                            apiVersion: ApiVersion,
                            requestBody: salRequest)
                        .ConfigureAwait(continueOnCapturedContext: false);

                // TODO: Handle response Provisioning State.
                logger.CLogInformation($"SAL Response: {salResponse}");

                detail = await TryGetServiceAssociationLinkDetailAsync(tenantId, subnetResourceId);
                if (detail == null || String.IsNullOrEmpty(detail?.Properties?.PrimaryContextId))
                {
                    throw new Exception($"Failed to Get SAL detail for subnetResourceId:{subnetResourceId}, SALdetail: {detail}");
                }

                return detail;
            }
            catch (Exception ex)
            {
                throw new ErrorResponseMessageException(
                    httpStatus: HttpStatusCode.InternalServerError,
                    errorCode: ErrorResponseCode.ServiceAssociationLinkFailed,
                    errorMessage: $"Failed to create SAL token, salRequest: {salRequest}, " +
                    $"subnetResourceId:{subnetResourceId} internal ex: {ex}.");
            }
        }

        /// <summary>
        /// Sets resource ownership.
        /// </summary>
        /// <param name="tenantId">Customer TenantId</param>
        /// <param name="subscriptionId">Subscription id</param>
        /// <param name="vnetResourceId">Vnet ResourceId</param>
        /// <returns></returns>
        public async Task<ResourceOwnershipResponse> SetResourceOwnershipForCustomerVnetAsync(
            string tenantId,
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
                tenantId: tenantId,
                resourceId: resourceId,
                action: "setResourceOwnership",
                apiVersion: ApiVersion,
                request: request);
        }

        /// <summary>
        /// Deletes the service association link associated to this Subnet.
        /// </summary>
        /// <param name="subscriptionId">Subscription Id of the subnet.</param>
        /// <param name="subnetResourceId">The resourceId of the subnet.</param>
        public async Task<ServiceAssociationLink> DeleteServiceAssociationLinkAsync(
            string tenantId, 
            string subnetResourceId)
        {
            return await this.DeleteResourceAsync<ServiceAssociationLink>(
                tenantId,
                resourceId: this.GetSALResourceId(subnetResourceId),
                apiVersion: ApiVersion);
        }

        /// <summary>
        /// Gets the service association link detail.
        /// </summary>
        /// <param name="subnetResourceId">The resource Id of the subnet</param>
        public async Task<ServiceAssociationLinkDetail> TryGetServiceAssociationLinkDetailAsync(
            string tenantId,
            string subnetResourceId)
        {
            HttpResponseMessage httpResponse = await this.TryGetResourceAsync<ServiceAssociationLinkDetail>(
                tenantId: tenantId,
                resourceId: this.GetSALDetailResourceId(subnetResourceId),
                apiVersion: ApiVersion);

            if (HttpStatusCodeCheck.IsResourceNotFound(httpResponse.StatusCode))
            {
                logger.CLogInformation($"GetServiceAssociationLinkDetail not found for subnetResourceId:{subnetResourceId}");
                return null;
            }

            if (!HttpStatusCodeCheck.IsSuccess(httpResponse.StatusCode))
            {
                throw new Exception($"ResponseMessage: {httpResponse.Content}, status code {httpResponse.StatusCode}, " +
                   $"Reason: {httpResponse.ReasonPhrase}, " +
                   $"Request: {httpResponse.RequestMessage}, ");
            }

            // Parse the response
            if (httpResponse.Content == null ||
                httpResponse.Content.Headers.ContentType.MediaType != "application/json")
            {
                throw new InvalidDataException($"Either Content is null or invalid content type '{httpResponse.Content}' returned. ");
            }

            var contentStream = await httpResponse.Content.ReadAsStringAsync();
            return JsonUtility.Deserialize<ServiceAssociationLinkDetail>(contentStream, DeserializationSettings);
        }

        private ServiceAssociationLink CreateServiceAssociationLinkRequest(
            string vnetResourceId)
        {
            var sal = new ServiceAssociationLink
            {
                Properties = new ServiceAssociationLinkProperties
                {
                    LinkedResourceType = LinkedResourceType
                }
            };

            //sal.Properties.AllowDelete = true;
            sal.Properties.Locations = new string[] { CurrentContext.AzureRegion };
            sal.Properties.ParentResourceGuid = vnetResourceId.ToString();

            sal.Properties.Details = new ServiceAssociationLinkDetail[]
            {
                new ServiceAssociationLinkDetail
                {
                    Name = SalDetailName,
                    Properties = new ServiceAssociationLinkDetailProperties()
                }
            };

            sal.Properties.Details[0].Properties.PrimaryContextRequestId = Guid.NewGuid().ToString();
            sal.Properties.Details[0].Properties.SecondaryContextRequestId = Guid.NewGuid().ToString();

            logger.CLogInformation($"Created ServiceAssociationLink: {sal}");
            return sal;
        }
    }
}
