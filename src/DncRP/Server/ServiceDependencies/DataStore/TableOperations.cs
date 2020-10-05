// -----------------------------------------------------------------------------
// <copyright file="TableOperations.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using Microsoft.Azure.ContainerNetworking.CoreLib;
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
        private ISubscriptionOperation subscriptionOperation;
        private ITableDataStore dataStore;
        ILogger<TableOperations> logger;
        IHttpUtility httpUtility;
        IKeyVaultUtility keyVaultUtility;

        /// <summary>
        /// Create an instance of TableOperations
        /// TODO: Pass the DataStore Settings
        /// </summary>
        /// <param name="storageConnectionString">Connection string, configuration from appsettings.Json</param>
        public TableOperations(
            ITableDataStore tableDataStore,
            ILogger<TableOperations> logger,
            IHttpUtility httpUtility,
            IKeyVaultUtility keyVaultUtility)
        {
            this.dataStore = tableDataStore;
            this.logger = logger;
            this.httpUtility = httpUtility;
            this.keyVaultUtility = keyVaultUtility;

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
            orchestratorOperations = new OrchestratorOperations(
                dataStore, logger, controllerConfiguration, this.httpUtility, this.keyVaultUtility);
            subscriptionOperation = new SubscriptionOperation(dataStore, logger);
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

        public ISubscriptionOperation SubscriptionOperation
        {
            get => subscriptionOperation;
        }
    }
}
