// -----------------------------------------------------------------------------
// <copyright file="TableOperations.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Microsoft.Extensions.Logging;

    /// <summary>
    /// TableOperations, this is used to initialize all DataModel interfaces
    /// </summary>
    public class TableOperations : ITableOperations
    {
        private ISubnetTokenMapOperations subnetTokenMapOperation;
        private IControllerConfiguration controllerConfiguration;
        private IDelegatedSubnetConfigurationOperations delegatedSubnetConfiguration;
        private IOrchestratorOperations orchestratorOperations;
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
            subnetTokenMapOperation = new SubnetTokenMapOperations(dataStore, logger);
            controllerConfiguration = new ControllerConfiguration(dataStore, logger);
            delegatedSubnetConfiguration = new DelegatedSubnetConfigurationOperations(dataStore, logger);
            orchestratorOperations = new OrchestratorOperations(dataStore, logger);
        }

        /// <summary>
        /// Return SubnetTokenMapOperations instance
        /// </summary>
        public ISubnetTokenMapOperations SubnetTokenMapOperations
        {
            get => subnetTokenMapOperation;
        }

        /// <summary>
        /// Return DelegatedSubnetConfigurationOperations instance
        /// </summary>
        public IDelegatedSubnetConfigurationOperations DelegatedSubnetConfigurationOperations
        {
            get => delegatedSubnetConfiguration;
        }

        public IControllerConfiguration ControllerConfiguration
        {
            get => controllerConfiguration;
        }

        public IOrchestratorOperations OrchestratorOperations
        {
            get => orchestratorOperations;
        }
    }
}
