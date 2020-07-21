// --------------------------------------------------------------------------------------------------------------------
// <copyright file="IServicePorts.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.Gateway
{
    using System.Collections.Generic;

    public interface IServicePorts
    {
        IDictionary<string, ServicePort> GetServicePortsMapping();
    }
}
