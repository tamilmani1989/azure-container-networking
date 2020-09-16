// --------------------------------------------------------------------------------------------------------------------
// <copyright file="DelegateSubnetDefinition.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    public class DelegateSubnet : TopLevelResource
    {
        /// <summary>
        /// Gets the container group type.
        /// </summary>
        public const string DelegatedResourceType = "Microsoft.DelegatedNetwork/delegatedSubnets";

        /// <summary>
        /// Initializes a new instance of the <see cref="DelegateSubnet"/> class.
        /// </summary>
        public DelegateSubnet(string subscriptionId, string resourceGroup, string vnetName, string subnetName, string resourceName, string delegatedSubGuid)
        {
            this.SubscriptionId = subscriptionId;
            this.ResourceGroup = resourceGroup;
            this.Name = resourceName;
            this.Properties = new DelegateSubnetProperties()
            {
                DelegatedSubnetId = delegatedSubGuid,
                // using the same value as DelegatedSubnetId. ResourceGuid is not exposed on Portal
                ResourceGuid = delegatedSubGuid,

                // Todo update the state
                State = ProvisioningState.Unknown
            };

            this.isExtensionResource = true;
            this.parentResourceId = $"Microsoft.Network/virtualNetworks/{vnetName}/subnets/{subnetName}";
            this.Type = DelegatedResourceType;
        }
    }
}
