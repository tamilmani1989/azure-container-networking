//-----------------------------------------------------------------------
// <copyright file="SetOrchestratorTypeRequest.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// 
//-----------------------------------------------------------------------

using Newtonsoft.Json;

namespace Cns.Data
{
    /// <summary>
    /// Defines the orchestrator type request.
    /// </summary>
    public class SetOrchestratorTypeRequest
    {
        /// <summary>
        /// Type of orchestrator.
        /// </summary>
        [JsonProperty("OrchestratorType")]
        public string OrchestratorType { get; private set; }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="orchestratorType">Supply the orchestrator type.</param>
        public SetOrchestratorTypeRequest(string orchestratorType)
        {
            this.OrchestratorType = orchestratorType;
        }

        /// <summary>
        /// ToString implementation
        /// </summary>
        /// <returns>String representation of the object.</returns>
        public override string ToString()
        {
            return $"OrchestratorType:{OrchestratorType}";
        }
    }
}
