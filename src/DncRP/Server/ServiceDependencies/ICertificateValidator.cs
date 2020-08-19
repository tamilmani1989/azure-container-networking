// -----------------------------------------------------------------------------
// <copyright file="ICertificateValidator.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Security.Cryptography.X509Certificates;
    using System.Threading.Tasks;

    public interface ICertificateValidator
    {
        Task<bool> IsValid(X509Certificate2 certificate);
    }
}
