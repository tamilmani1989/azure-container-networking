// ------------------------------------------------------------------------------------------
// <copyright file="HealthCheck.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// ------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Diagnostics;
    using System.Threading;
    using System.Threading.Tasks;
    using Microsoft.Extensions.Diagnostics.HealthChecks;

    public class HealthResult
    {
        public string Uptime;
    }

    public class HealthCheck : IHealthCheck
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
