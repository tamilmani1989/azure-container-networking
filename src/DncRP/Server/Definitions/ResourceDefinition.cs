// --------------------------------------------------------------------------------------------------------------------
// <copyright file="ResourceDefinition.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    // System.Text.Json doesnt support Required Property, hence using Newtonsoft.Json
    // https://docs.microsoft.com/en-us/dotnet/standard/serialization/system-text-json-migrate-from-newtonsoft-how-to#required-properties
    using Newtonsoft.Json;

    public class ResourceDefinition
    {
        /// <summary>
        /// Gets the id for the resource.
        /// </summary>
        [JsonProperty(Required = Required.Default)]
        public string Id
        {
            get
            {
                if (string.IsNullOrEmpty(this.SubscriptionId) ||
                    string.IsNullOrEmpty(this.ResourceType) ||
                    string.IsNullOrEmpty(this.ResourceName))
                {
                    return null;
                }

                return GetFullyQualifiedResourceId();
            }
        }

        /// <summary>
        /// Gets or sets the subscription.
        /// </summary>
        [JsonIgnore]
        public string SubscriptionId { get; set; }

        /// <summary>
        /// Gets or sets the name of the resource group.
        /// </summary>
        [JsonIgnore]
        public string ResourceGroup { get; set; }

        /// <summary>
        /// Gets or sets the type of the resource.
        /// </summary>
        [JsonIgnore]
        public string ResourceType { get; protected set; }

        /// <summary>
        /// Gets or sets the resource name.
        /// </summary>
        [JsonIgnore]
        public string ResourceName { get; set; }

        /// <summary>
        /// Gets or sets if this Resource is an Extension resource.
        /// </summary>
        [JsonIgnore]
        protected bool isExtensionResource { get; set; }

        /// <summary>
        /// ParentResource Id, only set if isExtensionResource is set
        /// </summary>
        [JsonIgnore]
        protected string parentResourceId {get; set;}

        /// <summary>
        /// Gets the fully qualified resource identifier.
        /// </summary>
        /// <param name="subscriptionId">The subscription identifier.</param>
        /// <param name="resourceGroup">The resource group.</param>
        /// <param name="resourceType">The type.</param>
        /// <param name="resourceName">Name of the fully qualified.</param>
        protected string GetFullyQualifiedResourceId()
        {
            if (isExtensionResource)
            {
                return $"/subscriptions/{SubscriptionId}/resourceGroups/{ResourceGroup}/providers/{parentResourceId}/providers/{ResourceType}/{ResourceName}";
            }

            return $"/subscriptions/{SubscriptionId}/resourceGroups/{ResourceGroup}/providers/{ResourceType}/{ResourceName}";
        }
    }
}
