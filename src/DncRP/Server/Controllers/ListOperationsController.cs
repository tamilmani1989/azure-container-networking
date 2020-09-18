// --------------------------------------------------------------------------------------------------------------------
// <copyright file="ListOperationsController.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Collections.Generic;
    using System.IO;
    using System.Net;
    using System.Net.Http;
    using System.Threading.Tasks;
    using Microsoft.AspNetCore.Mvc;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Logging;
    using Newtonsoft.Json;

    /// <summary>
    /// Class to list operations
    /// </summary>
    [Route("/providers/Microsoft.DelegatedNetwork/operations")]
    public class ListOperationsController : DncRpControllerBase<ListOperationsController>
    {
        private const string operationsListFile = "operationslist.json";

        public ListOperationsController(
            ITableOperations tableOperations,
            ILogger<ListOperationsController> logger) : base(logger, tableOperations)
        {
            logger.LogInformation("ListOperationsController initialized");
        }

        /// <summary>
        /// Get the DNC controller resource
        /// </summary>
        /// <returns></returns>
        [HttpGet]
        public async Task<ContentResult> GetOperationsList()
        {
            string file = Path.Combine(AppInfo.GetFilePath(operationsListFile), operationsListFile);
            using (StreamReader r = new StreamReader(file))
            {
                string json = r.ReadToEnd();
                return new ContentResult
                {
                    Content = json,
                    ContentType = "application/json",
                    StatusCode = (int)HttpStatusCode.OK
                };
            }
        }
    }
}
