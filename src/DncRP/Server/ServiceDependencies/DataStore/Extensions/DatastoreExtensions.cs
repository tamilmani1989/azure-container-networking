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

            OrchestratorProperties props = JsonUtility.Deserialize<OrchestratorProperties>(entity.Properties);
            props.ResourceGuid = entity.ResouceGuid;
            props.State = TopLevelResourceProperties.ParseProvisioningState(entity.State);

            MSIIdentityMetadata metadata = JsonUtility.Deserialize<MSIIdentityMetadata>(entity.IdentityMetadata);
            MSIIdentityResponse identityResponse = new MSIIdentityResponse
            {
                PrincipalId = metadata.PrincipalId,
                TenantId = metadata.HomeTenantId,
                Type = metadata.Type
            };

            return new Orchestrator(
                subId,
                rowParts[0],
                rowParts[1],
                entity.Location,
                (OrchestratorKind)Enum.Parse(typeof(OrchestratorKind), entity.OrchestratorType, true),
                props,
                identityResponse);
        }

        public static OrchestratorPropertiesInternal ToPropertiesInternal(this OrchestratorProperties properties)
        {
            return new OrchestratorPropertiesInternal
            {
                ApiServerEndpoint = properties.ApiServerEndpoint,
                AppId = properties.AppId,
                ClusterRootCA = properties.ClusterRootCA,
                TenantId = properties.TenantId
            };
        }
    }
}
