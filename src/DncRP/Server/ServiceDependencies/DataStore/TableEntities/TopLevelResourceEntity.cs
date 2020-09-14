// -----------------------------------------------------------------------------
// <copyright file="TopLevelResource.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Microsoft.Azure.Cosmos.Table;
    using System;

    /// <summary>
    /// Defines Table properties required by TopLevelResource
    /// </summary>
    public class TopLevelResourceEntity : TableEntity
    {
        public int ProvisioningStatus { get; set; }

        public string ResourceName { get; set; }

        public string ResourceGuid { get; set; }
    }
}
