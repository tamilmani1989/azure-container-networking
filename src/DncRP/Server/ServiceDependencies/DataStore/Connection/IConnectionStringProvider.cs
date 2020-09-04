// -----------------------------------------------------------------------------
// <copyright file="IConnectionStringProvider.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Collections.Generic;
    using System.Threading.Tasks;

    public interface IConnectionStringProvider
    {
        Task<IEnumerable<string>> GetConnectionStringsForTableAPI();
    }
}
