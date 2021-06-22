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

    public class MdmStatsDProvider : IMetricsProvider
    {
        private readonly MdmMetricsConfig mdmConfig;
        private readonly IStatsDPublisher statsd;
        private readonly IEnumerable<DimensionValue> defaultDimensions;

        public MdmStatsDProvider(MdmMetricsConfig config, IEnumerable<DimensionValue> defaultDimensions)
        {
            this.mdmConfig = config;
            this.defaultDimensions = defaultDimensions;

            var statsDConfig = new StatsDConfiguration
            {
                Host = mdmConfig.StatsDHost,
                Port = mdmConfig.StatsDPort,
                SocketProtocol = (SocketProtocol)Enum.Parse(typeof(SocketProtocol), mdmConfig.StatsDSocketProtocol),
                OnError = LogError
            };

            this.statsd = new StatsDPublisher(statsDConfig);
        }

        public void EmitMetric(string metricName, double value)
        {
            IEnumerable<DimensionValue> dimensionValues = null;
            this.EmitMetric(metricName, value, dimensionValues);
        }

        public void IncrementCounter(string metricName, IEnumerable<DimensionValue> dimensionValues = null)
        {
            string bucket = this.CreateStatsDBucket(metricName, dimensionValues);
            this.statsd.Increment(bucket);
        }

        public void EmitMetric(string metricName, double value, DimensionValue dimensionValue)
        {
            this.EmitMetric(metricName, value, new List<DimensionValue> { dimensionValue });
        }

        public void EmitMetric(string metricName, double value, IEnumerable<DimensionValue> dimensionValues)
        {
            string bucket = this.CreateStatsDBucket(metricName, dimensionValues);
            this.statsd.Gauge(value, bucket);
            // Can be uncommented for debugging
            //this.Log($"EmitMetric: metricname={metricName}, value={value}, bucket='{bucket}'");
        }

        public static MdmStatsDProvider CreateMdmStatsDProvider(IConfiguration configuration)
        {
            MdmMetricsConfig mdmMetricsConfig = configuration.GetSection("MdmMetrics").Get<MdmMetricsConfig>();

            List<DimensionValue> list = new List<DimensionValue>();
            list.Add(new DimensionValue { Name = "Region", Value = configuration["AzureRegion"] });
            list.Add(new DimensionValue { Name = "PodName", Value = Environment.GetEnvironmentVariable("POD_NAME") });

            return new MdmStatsDProvider(mdmMetricsConfig, list);
        }

        private string CreateStatsDBucket(string metricName, IEnumerable<DimensionValue> dimensionValues)
        {
            StatsDBucket bucket = new StatsDBucket
            {
                Namespace = this.mdmConfig.Namespace,
                Metric = metricName,
                Dims = new Dictionary<string, string>()
            };

            List<DimensionValue> dimList = new List<DimensionValue>();
            if (this.defaultDimensions != null && this.defaultDimensions.Any())
            {
                dimList.AddRange(this.defaultDimensions);
            }

            if (dimensionValues != null && dimensionValues.Any())
            {
                dimList.AddRange(dimensionValues);
            }

            if (dimList.Any())
            {
                foreach (DimensionValue dimVal in dimList)
                {
                    bucket.Dims.Add(dimVal.Name, dimVal.Value);
                }
            }

            return JsonUtility.Serialize<StatsDBucket>(bucket);
        }

        private void Log(string message)
        {
            Console.WriteLine(string.Format("[DNC RP] {0}", message));
        }

        private bool LogError(Exception exception)
        {
            Console.WriteLine(string.Format("[Error] {0}", exception.ToString()));
            return true;
        }
    }
}
