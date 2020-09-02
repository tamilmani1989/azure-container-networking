// -----------------------------------------------------------------------------
// <copyright file="CosmosDbUriFormats.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    /// <summary>
    /// CosmosDb Url formats
    /// </summary>
    public static class CosmosDbUriFormats
    {
        public const string ListKeysUriFormat = "{0}/subscriptions/{1}/resourceGroups/{2}/providers/Microsoft.DocumentDb/databaseAccounts/{3}/listKeys/?api-version=2016-03-31";
    }
}
