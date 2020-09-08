// -----------------------------------------------------------------------------
// <copyright file="ImdsRetrieverTests.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP.Unittests
{
    using System.IO;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Configuration;
    using Xunit;

    public class ImdsRetrieverTests
    {
        [Fact]
        public async void TestMetadataHeaderInImdsInstanceRequest()
        {
            TestHttpUtility httpUtility = new TestHttpUtility();
            httpUtility.SetValidateMetadataHeaderFlag(true);

            IImdsRetriever retriever = new ImdsRetriever(httpUtility);
            InstanceResponse response = await retriever.GetInstanceMetadata("https://testuri");
            Assert.True(response != null);
        }
    }
}
