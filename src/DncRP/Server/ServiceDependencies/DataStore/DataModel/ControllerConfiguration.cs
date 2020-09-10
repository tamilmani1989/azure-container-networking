// -----------------------------------------------------------------------------
// <copyright file="ControllerConfiguration.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Threading.Tasks;

    internal class ControllerConfiguration : IControllerConfiguration
    {
        private ITableDataStore dataStore;
        private const string tableName = "ControllerConfiguration";

        /// <summary>
        /// Create an instance of TableOperations
        /// TODO: Pass the DataStore Settings
        /// </summary>
        /// <param name="storageConnectionString">Connection string, configuration from appsettings.Json</param>
        public ControllerConfiguration(
            ITableDataStore tableDataStore)
        {
            this.dataStore = tableDataStore;
        }

        /// <summary>
        /// Get DNC controller
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="controllerName"></param>
        /// <returns></returns>
        public async Task<ControllerResponse> GetDncController(string subId, string resourceGroup, string controllerName)
        {
            throw new NotImplementedException();
        }

        /// <summary>
        /// Create a new DNC controller or replace an existing one
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="controllerName"></param>
        /// <returns></returns>
        public async Task<ControllerResponse> InsertOrReplaceDncController(
            string subId, string resourceGroup, string controllerName, ControllerRequest controllerRequest)
        {
            throw new NotImplementedException();
        }

        /// <summary>
        /// Modify an existing controller
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="controllerName"></param>
        /// <returns></returns>
        public async Task<ControllerResponse> MergeDncController(
            string subId, string resourceGroup, string controllerName, ControllerRequest controllerRequest)
        {
            throw new NotImplementedException();
        }

        /// <summary>
        /// Delete DNC controller
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="controllerName"></param>
        /// <returns></returns>
        public async Task<ControllerResponse> DeleteDncController(string subId, string resourceGroup, string controllerName)
        {
            throw new NotImplementedException();
        }
    }
}
