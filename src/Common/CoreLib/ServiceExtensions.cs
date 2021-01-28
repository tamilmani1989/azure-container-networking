// -----------------------------------------------------------------------------
// <copyright file="ServiceExtensions.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using Microsoft.Extensions.Configuration;
    using Microsoft.Extensions.DependencyInjection;

    public static class ServiceExtensions
    {
        public static IServiceCollection AddMdmServices(this IServiceCollection services, IConfiguration configuration)
        {
            MdmStatsDProvider metricsProvider = MdmStatsDProvider.CreateMdmStatsDProvider(configuration);
            services.AddSingleton<IMetricsProvider, MdmStatsDProvider>(provider => metricsProvider);
            return services;

        }
    }
}
