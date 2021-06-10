// -----------------------------------------------------------------------------
// <copyright file="DatastoreSettings.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Security.Cryptography;
    using System.Text;
    using Microsoft.Azure.ContainerNetworking.CoreLib;

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

        // TODO: workitem: 10093300
        public static int GetDeterministicHashCode(string str)
        {
            unchecked
            {
                int hash1 = (5381 << 16) + 5381;
                int hash2 = hash1;

                for (int i = 0; i < str.Length && str[i] != '\0'; i += 2)
                {
                    hash1 = ((hash1 << 5) + hash1) ^ str[i];
                    if (i == str.Length - 1 || str[i + 1] == '\0')
                        break;
                    hash2 = ((hash2 << 5) + hash2) ^ str[i + 1];
                }

                return hash1 + (hash2 * 1566083941);
            }
        }
    }
}
