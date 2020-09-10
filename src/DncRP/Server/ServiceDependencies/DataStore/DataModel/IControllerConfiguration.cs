// -----------------------------------------------------------------------------
// <copyright file="IControllerConfiguration.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
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
        Task<ControllerResponse> GetDncController(string subId, string resourceGroup, string controllerName);

        /// <summary>
        /// Create a new DNC controller or replace an existing one
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="controllerName"></param>
        /// <returns></returns>
        Task<ControllerResponse> InsertOrReplaceDncController(
            string subId, string resourceGroup, string controllerName, ControllerRequest controllerRequest);

        /// <summary>
        /// Modify an existing controller
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="controllerName"></param>
        /// <returns></returns>
        Task<ControllerResponse> MergeDncController(
            string subId, string resourceGroup, string controllerName, ControllerRequest controllerRequest);

        /// <summary>
        /// Delete DNC controller
        /// </summary>
        /// <param name="subId"></param>
        /// <param name="resourceGroup"></param>
        /// <param name="controllerName"></param>
        /// <returns></returns>
        Task<ControllerResponse> DeleteDncController(string subId, string resourceGroup, string controllerName);
    }
}
