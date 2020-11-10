// -----------------------------------------------------------------------------
// <copyright file="MetricAccountInfo.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    public class MdmMetricsConfig
    {
        public string Namespace
        {
            get; set;
        }

        public string AccountName
        {
            get; set;
        }

        public string StatsDHost
        {
            get; set;
        }

        public int StatsDPort
        {
            get; set;
        }

        public string StatsDSocketProtocol
        {
            get; set;
        }
    }
}