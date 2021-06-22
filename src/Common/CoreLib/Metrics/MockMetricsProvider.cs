// -----------------------------------------------------------------------------
// <copyright file="MdmStatsDProvider.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using System;
    using System.Collections.Generic;
    using System.Linq;
    using JustEat.StatsD;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Configuration;

    public class MockMetricsProvider : IMetricsProvider
    {
        public void EmitMetric(string metricName, double value)
        {
            Log($"Incrementing counter for {metricName} by {value}");
        }

        public void IncrementCounter(string metricName, IEnumerable<DimensionValue> dimensionValues)
        {
            Log($"Incrementing counter for {metricName}");
        }

        public void EmitMetric(string metricName, double value, DimensionValue dimensionValue)
        {
            Log($"Emitting Metric for {metricName} with value {value}");
        }

        public void EmitMetric(string metricName, double value, IEnumerable<DimensionValue> dimensionValues)
        {
            Log($"Emitting Metric for {metricName} with value {value}");
        }

        private void Log(string message)
        {
            Console.WriteLine(string.Format("[DNC RP] {0}", message));
        }
    }
}
