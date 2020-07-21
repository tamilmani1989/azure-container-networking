// -----------------------------------------------------------------------------
// <copyright file="CustomTelemetryProperties.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.Gateway
{
    using Microsoft.ApplicationInsights.Channel;
    using Microsoft.ApplicationInsights.DataContracts;
    using Microsoft.ApplicationInsights.Extensibility;

    /// <summary>
    /// Custom telemety class. It injects custom properties into log messages.
    /// For example - it will inject a property named Context whose value would be gateway,
    /// so we know log came from gateway service.
    /// </summary>
    public class CustomTelemetryProperties : ITelemetryInitializer
    {
        public void Initialize(ITelemetry telemetry)
         {
            //cast ITelemetry to ISupportProperties
            ISupportProperties propTelemetry = (ISupportProperties)telemetry;

            propTelemetry.Properties["Context"] = "Gateway";
        }
    }
}


