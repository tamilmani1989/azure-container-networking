// --------------------------------------------------------------------------------------------------------------------
// <copyright file="IDncClient.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

using System.Net.Http;
using System;
using System.Collections.Generic;
using System.Linq;
using System.Net;
using System.Threading.Tasks;

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    public interface IDncClient
    {
       Task PutSubnetAsync(string networkId, string subnetName, string authToken);
       Task DeleteSubnetAsync(string networkId, string subnetName);
       Task<HttpResponseMessage> GetSubnetAsync(string networkId, string subnetName);
    }
}
