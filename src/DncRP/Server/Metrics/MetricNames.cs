// -----------------------------------------------------------------------------
// <copyright file="StatsDBucket.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    public class MetricNames
    {
        public const string DncRPUP = "DncRPUP";

        public const string GetOrchestrator = "GetOrchestrator";
        public const string PutOrchestrator = "PutOrchestrator";
        public const string DeleteOrchestrator = "DeleteOrchestrator";

        public const string GetController = "GetController";
        public const string PutController = "PutController";
        public const string PatchController = "PatchController";
        public const string DeleteController = "DeleteController";

        public const string GetDelegateSubnet = "GetDelegateSubnet";
        public const string PutDelegateSubnet = "PutDelegateSubnet";
        public const string DeleteDelegateSubnet = "DeleteDelegateSubnet";

        public const string PutSubscriptionRegistration = "PutSubscriptionRegistration";
    }

    public struct DimensionNames
    {
        public const string Subscription = "Subscription";
        public const string ResourceGroup = "ResourceGroup";
        public const string HttpStatusCode = "HttpStatusCode";
        public const string InternalErrorCode = "InternalErrorCode";
        public const string Result = "Result";
        public const string ResourceGuid = "ResourceGuid";
    }

    public struct MetricResult
    {
        public const string Success = "Success";
        public const string Failure = "Failure";
    }
}
