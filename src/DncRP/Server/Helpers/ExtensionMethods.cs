// -----------------------------------------------------------------------------
// <copyright file="DatastoreSettings.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Collections.Generic;

    public static class ExtensionMethods
    {
        public static DatastoreSettings FindByRegion(this DatastoreSettings[] list, string region)
        {
            if (list == null || region == null)
            {
                return null;
            }

            foreach (DatastoreSettings s in list)
            {
                if (s.Region.Equals(region, StringComparison.OrdinalIgnoreCase))
                {
                    return s;
                }
            }

            return null;
        }
    }
}
