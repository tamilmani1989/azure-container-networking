// -----------------------------------------------------------------------------
// <copyright file="MacAllocationRequest.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.MacManager.Common
{
    using System;
    using System.Diagnostics;
    using System.Threading;
    using System.Threading.Tasks;

    using Microsoft.Extensions.Diagnostics.HealthChecks;

    public sealed class HealthResult
    {
        public string Uptime { get; set; }
    }

    public sealed class HealthCheck : IHealthCheck
    {
        public Task<HealthCheckResult> CheckHealthAsync(
            HealthCheckContext context,
            CancellationToken cancellationToken = default(CancellationToken))
        {
            Process curProcess = Process.GetCurrentProcess();
            TimeSpan uptime = DateTime.UtcNow - curProcess.StartTime.ToUniversalTime();
            string uptimeFormatted = uptime.ToString("c");

            return Task.FromResult(HealthCheckResult.Healthy(uptimeFormatted));
        }
    }
}
