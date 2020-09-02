// -----------------------------------------------------------------------------
// <copyright file="CosmosRWKeys.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    /// <summary>
    /// CosmosDb keys
    /// </summary>
    internal class CosmosDbKeys
    {
        public string PrimaryMasterKey
        {
            get; set;
        }
        public string SecondaryMasterKey
        {
            get; set;
        }

        public string PrimaryReadonlyMasterKey
        {
            get; set;
        }
        public string SecondaryReadonlyMasterKey
        {
            get; set;
        }
    }

    /// <summary>
    /// CosmosDb keys that have read/write permissions
    /// </summary>
    public class CosmosDbRWKeys
    {
        public string PrimaryMasterKey
        {
            get; set;
        }
        public string SecondaryMasterKey
        {
            get; set;
        }
    }

    /// <summary>
    /// CosmosDb keys that have read only permissions
    /// </summary>
    public class CosmosDbReadOnlyKeys
    {
        public string PrimaryReadonlyMasterKey
        {
            get; set;
        }
        public string SecondaryReadonlyMasterKey
        {
            get; set;
        }
    }
}
