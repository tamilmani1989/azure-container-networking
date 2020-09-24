// -----------------------------------------------------------------------------
// <copyright file="ArmResourceIdParser.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Collections.Generic;

    /// <summary>
    /// This class parse the ARM Resource URIs.
    /// </summary>
    public static class ArmResourceIdParser
    {
        /// <summary>
        /// Subscription key name.
        /// </summary>
        private const string SubscriptionsKey = "subscriptions";

        /// <summary>
        /// Resource group key name
        /// </summary>
        private const string ResourceGroupsKey = "resourceGroups";

        /// <summary>
        /// The providers key name
        /// </summary>
        private const string ProvidersKey = "providers";

        /// <summary>
        /// VirtualNetworks resource, key is parsed in lower case.
        /// </summary>
        public const string VirtualNetworkKey = "virtualnetworks";

        /// <summary>
        /// subnet resource, key is parsed in lower case.
        /// </summary>
        public const string SubnetKey = "subnets";

        /// <summary>
        /// Parse ARM resource Id
        /// Expected format is /subscriptions/{subscription}/resourceGroups/{rg}/providers/{resourceProviderGroupName}/{resourceType}/{resourcename}/{childresourceType}/{childresourcename}...
        /// </summary>
        /// <param name="resourceId">The resource Id</param>
        /// <param name="subscriptionId">The subscription abstracted from resource Id</param>
        /// <param name="resourceGroupName">The resource group name absctracted from resource Id</param>
        /// <param name="resources">The collection of key-value pair, each one contains resource type and resource name</param>
        /// <returns>True if parse succeeded. otherwise return false</returns>
        public static bool TryParseResourceId(
            string resourceId,
            string resourceProviderName,
            out string subscriptionId,
            out string resourceGroupName,
            out Dictionary<string, string> resources)
        {
            subscriptionId = string.Empty;
            resourceGroupName = string.Empty;
            resources = null;

            if (string.IsNullOrEmpty(resourceId))
            {
                return false;
            }

            if (!resourceId.StartsWith("/", StringComparison.OrdinalIgnoreCase))
            {
                return false;
            }

            var items = resourceId.Substring(1).Split('/');

            if (items.Length % 2 != 0 || items.Length < 2)
            {
                return false;
            }

            if (items.Length >= 2)
            {
                if (items[0].Equals(SubscriptionsKey, StringComparison.OrdinalIgnoreCase))
                {
                    subscriptionId = items[1];
                }
                else
                {
                    return false;
                }
            }

            if (items.Length >= 4)
            {
                if (items[2].Equals(ResourceGroupsKey, StringComparison.OrdinalIgnoreCase))
                {
                    resourceGroupName = items[3];
                }
                else
                {
                    return false;
                }
            }

            if (items.Length >= 6)
            {
                if (!items[4].Equals(ProvidersKey, StringComparison.OrdinalIgnoreCase) ||
                    (!items[5].Equals(resourceProviderName, StringComparison.OrdinalIgnoreCase)))
                {
                    return false;
                }
            }

            if (items.Length >= 8)
            {
                resources = new Dictionary<string, string>();
                int i = 6;
                while (i < items.Length)
                {
                    string resourceType = items[i++];
                    string resourceName = items[i++];

                    if (string.IsNullOrWhiteSpace(resourceType))
                    {
                        return false;
                    }

                    resources.Add(resourceType.ToLower(), resourceName);
                }
            }

            return true;
        }
    }
}
