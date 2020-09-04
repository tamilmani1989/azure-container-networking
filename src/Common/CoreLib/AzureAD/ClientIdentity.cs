// -----------------------------------------------------------------------------
// <copyright file="ClientIdentity.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    /// <summary>
    /// Identity of client to request token from AzureAD
    /// </summary>
    public class ClientIdentity
    {
        public string AppId
        {
            get; set;
        }

        public string TenantId
        {
            get; set;
        }

        public string AppKey
        {
            get; set;
        }
    }
}
