// -----------------------------------------------------------------------------
// <copyright file="ListOperationsUnitTests.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP.Unittests
{
    using Microsoft.AspNetCore.Mvc;
    using Microsoft.Extensions.Logging;
    using Moq;
    using Newtonsoft.Json.Linq;
    using Xunit;

    public class ListOperationsUnitTests
    {
        [Fact]
        public void Test_OperationsList_Json()
        {
            Mock<ILogger<ListOperationsController>> logger = new Mock<ILogger<ListOperationsController>>();
            ListOperationsController listOperations = new ListOperationsController(null, logger.Object);
            ContentResult result =  listOperations.GetOperationsList();
            Assert.NotNull(JObject.Parse(result.Content));
        }
    }
}
