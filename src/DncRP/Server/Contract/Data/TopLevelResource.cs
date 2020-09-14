// --------------------------------------------------------------------------------------------------------------------
// <copyright file="TopLevelResource.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Text.Json.Serialization;

    public class TopLevelResource
    {
        /// <summary>
        /// Gets the id for the resource.
        /// </summary>
        [JsonPropertyName("id")]
        public string Id
        {
            get
            {
                if (string.IsNullOrEmpty(this.SubscriptionId) ||
                    string.IsNullOrEmpty(this.Type) ||
                    string.IsNullOrEmpty(this.Name))
                {
                    return null;
                }

                return GetFullyQualifiedResourceId();
            }
        }

        /// <summary>
        /// Gets or sets the location of the resource.
        /// </summary>
        [JsonPropertyName("location")]
        public string Location { get; set; }

        /// <summary>
        /// Gets or sets the resource name.
        /// </summary>
        [JsonPropertyName("name")]
        public string Name { get; set; }

        /// <summary>
        /// Gets or sets the type of the resource.
        /// </summary>
        [JsonPropertyName("location")]
        public string Type { get; protected set; }

        /// <summary>
        /// Gets or sets the type of the resource.
        /// </summary>
        [JsonPropertyName("state")]
        public string State { get; protected set; }


        [JsonPropertyName("properties")]
        public TopLevelResourceProperties Properties { get; protected set; }


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
        /// Gets or sets if this Resource is an Extension resource.
        /// </summary>
        [JsonIgnore]
        protected bool isExtensionResource { get; set; }

        /// <summary>
        /// ParentResource Id, only set if isExtensionResource is set
        /// </summary>
        [JsonIgnore]
        protected string parentResourceId { get; set; }

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
                return $"/subscriptions/{SubscriptionId}/resourceGroups/{ResourceGroup}/providers/{parentResourceId}/providers/{Type}/{Name}";
            }

            return $"/subscriptions/{SubscriptionId}/resourceGroups/{ResourceGroup}/providers/{Type}/{Name}";
        }
    }
}
