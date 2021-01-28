// -----------------------------------------------------------------------------
// <copyright file="IMetricsProvider.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using System.Collections.Generic;

    public interface IMetricsProvider
    {
        /// <summary>
        /// Emits the standard gauge metric
        /// </summary>
        /// <param name="metricName"></param>
        /// <param name="value"></param>
        void EmitMetric(string metricName, double value);

        /// <summary>
        /// Emits the standard gauge metric along with dimension
        /// </summary>
        /// <param name="metricName"></param>
        /// <param name="value"></param>
        /// <param name="dimensionValue"></param>
        void EmitMetric(string metricName, double value, DimensionValue dimensionValue);

        /// <summary>
        /// Emits the standard gauge metric along with all dimensions specified in the list
        /// </summary>
        /// <param name="metricName"></param>
        /// <param name="value"></param>
        /// <param name="dimensionValues"></param>
        void EmitMetric(string metricName, double value, IEnumerable<DimensionValue> dimensionValues);
    }
}
