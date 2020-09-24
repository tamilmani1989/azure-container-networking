// --------------------------------------------------------------------------------------------------------------------
// <copyright file="DelegateSubnet.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Newtonsoft.Json;

    public class DelegateSubnet : TopLevelResource
    {
        /// <summary>
        /// Gets the container group type.
        /// </summary>
        public const string DelegatedResourceType = "Microsoft.DelegatedNetwork/delegatedSubnets";

        /// <summary>
        /// VNET Resource Guid.
        /// </summary>
        [JsonIgnore]
        public string VNetResourceGuid { get; set; }

        /// <summary>
        ///  Initializes a new instance of the <see cref="DelegateSubnet"/> class.
        /// </summary>
        /// <param name="subscriptionId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="vnetName"></param>
        /// <param name="subnetName"></param>
        /// <param name="resourceName"></param>
        /// <param name="delegatedSubGuid"></param>
        public DelegateSubnet(
            string subscriptionId, 
            string resourceGroup, 
            string resourceName, 
            string location,
            DelegateSubnetProperties delegateSubnetProperties)
        {
            this.SubscriptionId = subscriptionId;
            this.ResourceGroup = resourceGroup;
            this.Name = resourceName;
            this.Location = location;
            this.Properties = delegateSubnetProperties;

            this.Type = DelegatedResourceType;
        }

        public DelegateSubnet()
        {

        }

        public static DelegateSubnet CreateErrorResponse(string errorMessage, int httpStatusCode, ErrorResponseCode errorResponseCode = ErrorResponseCode.None)
        {
            return new DelegateSubnet
            {
                Error = new TopLevelResourceError
                {
                    Error = errorMessage,
                    StatusCode = httpStatusCode,
                    ErrorResponseCode = errorResponseCode
                }
            };
        }
    }
}
