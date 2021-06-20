// -----------------------------------------------------------------------------
// <copyright file="IControllerConfiguration.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Collections.Generic;
    using System.Threading.Tasks;

    public interface IControllerConfiguration
    {
        /// <summary>
        /// Get DNC controller
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="controllerName"></param>
        /// <returns></returns>
        Task<Controller> GetDncControllerAsync(string subId, string resourceGroup, string controllerName);

        /// <summary>
        /// Get DNC controller using ARM controller ID as input
        /// </summary>
        /// <param name="armControllerId"></param>
        /// <returns></returns>
        Task<Controller> GetDncControllerAsync(string armControllerId);

        /// <summary>
        /// Create a new DNC controller or replace an existing one
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="controllerName"></param>
        /// <returns></returns>
        Task<Controller> InsertOrReplaceDncControllerAsync(
            string subId, string resourceGroup, string controllerName, ControllerRequest controllerRequest);

        /// <summary>
        /// Modify an existing controller
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="controllerName"></param>
        /// <returns></returns>
        Task<Controller> MergeDncControllerAsync(
            string subId, string resourceGroup, string controllerName, ControllerRequest controllerRequest);

        /// <summary>
        /// Delete DNC controller
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="controllerName"></param>
        /// <returns></returns>
        Task<Controller> DeleteDncControllerAsync(string subId, string resourceGroup, string controllerName);

        /// <summary>
        /// Get Delegated subnet resources for a given controller Id.
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="controllerName"></param>
        /// <returns></returns>
        Task<List<DelegateSubnet>> GetDelegatedSubnetResourcesByControllerId(string armControllerId);

        /// <summary>
        /// Get Orchestrator resources for a given controller Id.        
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="controllerName"></param>
        /// <returns></returns>
        Task<List<Orchestrator>> GetOrchestratorResourcesByControllerId(string armControllerId);
    }
}
