// -----------------------------------------------------------------------------
// <copyright file="DatastoreExtensions.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using Microsoft.Azure.ContainerNetworking.CoreLib;

    public static class DatastoreExtensions
    {
        public static Orchestrator ToOrchestrator(this OrchestratorResourceEntity entity)
        {
            string subId = entity.PartitionKey;
            string[] rowParts = entity.RowKey.Split(new string[] { "_" }, StringSplitOptions.None);

            return new Orchestrator(
                subId,
                rowParts[0],
                rowParts[1],
                entity.Location,
                new OrchestratorProperties
                {
                    State = (ProvisioningState)Enum.Parse(typeof(ProvisioningState), entity.State, true),
                    ResourceGuid = entity.ResouceGuid,
                    ControllerDetails = new ControllerDetails
                    {
                        Id = entity.ControllerId
                    },
                    orchestratorInstanceDetails = new OrchestratorInstanceDetails
                    {
                        Type = (OrchestratorType)Enum.Parse(typeof(OrchestratorType), entity.OrchestratorType, true),
                        Properties = JsonUtility.Deserialize<OrchestratorInstanceProperties>(entity.Properties)
                    }
                });
        }
    }
}
