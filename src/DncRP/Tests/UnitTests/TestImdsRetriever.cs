// -----------------------------------------------------------------------------
// <copyright file="TestImdsRetriever.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP.Unittests
{
    using System.Net.Http;
    using System.Threading.Tasks;
    using Microsoft.Azure.ContainerNetworking.CoreLib;

#pragma warning disable CS1998 // Async method lacks 'await' operators and will run synchronously. Test code.
    internal class TestImdsRetriever : IImdsRetriever
    {
        private string azureEnvironment;
        private string azureRegion;

        public TestImdsRetriever(string azureEnv = "", string azureRegion = "")
        {
            this.azureEnvironment = azureEnv;
            this.azureRegion = azureRegion;
        }

        public async Task<InstanceResponse> GetInstanceMetadata(string imdsInstanceUri)
        {
            return new InstanceResponse { Compute = new ComputeResponse { AzureEnvironment = this.azureEnvironment, AzureRegion = this.azureRegion } };
        }
    }
}
