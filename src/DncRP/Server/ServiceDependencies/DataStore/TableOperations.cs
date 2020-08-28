// -----------------------------------------------------------------------------
// <copyright file="TableOperations.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

using Microsoft.Extensions.Logging;

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    /// <summary>
    /// TableOperations, this is used to initialize all DataModel interfaces
    /// </summary>
    public class TableOperations : ITableOperations
    {
        private ISubnetTokenConfiguration subnetTokenConfiguration;
        private ITableDataStore dataStore;
        ILogger<TableOperations> logger;

        /// <summary>
        /// Create an instance of TableOperations
        /// TODO: Pass the DataStore Settings
        /// </summary>
        /// <param name="storageConnectionString">Connection string, configuration from appsettings.Json</param>
        public TableOperations(
            ITableDataStore tableDataStore,
            ILogger<TableOperations> logger)
        {
            this.dataStore = tableDataStore;
            this.logger = logger;

            Init();
        }

        /// <summary>
        /// Initialize the DataModels
        /// </summary>
        /// <returns></returns>
        private void Init()
        {
            subnetTokenConfiguration = new SubnetTokenConfiguration(dataStore);
        }

        public ISubnetTokenConfiguration SubnetTokenConfiguration
        {
            get => subnetTokenConfiguration;
        }
    }
}
