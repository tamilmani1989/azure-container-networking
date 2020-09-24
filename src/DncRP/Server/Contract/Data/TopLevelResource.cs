// --------------------------------------------------------------------------------------------------------------------
// <copyright file="TopLevelResource.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;

    public abstract class TopLevelResource
    {
        private string resId;
        private string azureEnv;

        /// <summary>
        /// Gets the id for the resource.
        /// </summary>
        [JsonProperty(Required = Required.Always, PropertyName = "id")]
        public string Id
        {
            get
            {
                if (!string.IsNullOrWhiteSpace(resId))
                {
                    return resId;
                }

                if (string.IsNullOrEmpty(this.SubscriptionId) ||
                    string.IsNullOrEmpty(this.Type) ||
                    string.IsNullOrEmpty(this.Name))
                {
                    return null;
                }

                resId = GetFullyQualifiedResourceId();
                return resId;
            }

            set
            {
                resId = value;
            }
        }

        /// <summary>
        /// Gets or sets the location of the resource.
        /// </summary>
        [JsonProperty(Required = Required.Default, PropertyName = "location")]
        public string Location
        {
            get
            {
                if (string.IsNullOrWhiteSpace(this.azureEnv))
                {
                    this.azureEnv = CurrentContext.AzureEnvironment;
                }

                return this.azureEnv;
            }
            set
            {
                this.azureEnv = value;
            }
        }

        /// <summary>
        /// Gets or sets the resource name.
        /// </summary>
        [JsonProperty(Required = Required.Default, PropertyName = "name")]
        public string Name { get; set; }

        /// <summary>
        /// Gets or sets the type of the resource.
        /// </summary>
        [JsonProperty(Required = Required.Default, PropertyName = "type")]
        public string Type { get; set; }

        /// <summary>
        /// Properties of a TopLevelResource
        /// </summary>
        [JsonProperty(PropertyName = "properties")]
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
        /// Gets or sets the name of the resource group.
        /// </summary>
        [JsonIgnore]
        public TopLevelResourceError Error { get; set; }

        /// <summary>
        /// Gets the fully qualified resource identifier.
        /// </summary>
        /// <param name="subscriptionId">The subscription identifier.</param>
        /// <param name="resourceGroup">The resource group.</param>
        /// <param name="resourceType">The type.</param>
        /// <param name="resourceName">Name of the fully qualified.</param>
        protected string GetFullyQualifiedResourceId()
        { 
            return $"/subscriptions/{SubscriptionId}/resourceGroups/{ResourceGroup}/providers/{Type}/{Name}";
        }
    }
}
