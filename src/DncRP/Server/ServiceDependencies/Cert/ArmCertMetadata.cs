// -----------------------------------------------------------------------------
// <copyright file="ClientCertificate.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;

    /// <summary>
    /// Certificate data that is returned by ARM metadata endpoint
    /// https://armwiki.azurewebsites.net/authorization/AuthenticateBetweenARMandRP.html
    /// </summary>
    public class ClientCertificate
    {
        public string thumbprint
        {
            get; set;
        }

        public DateTime notBefore
        {
            get; set;
        }

        public DateTime notAfter
        {
            get; set;
        }

        public string certificate
        {
            get; set;
        }
    }

    public class ArmCertMetadata
    {
        public ClientCertificate[] clientCertificates
        {
            get; set;
        }
    }
}
