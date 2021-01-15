// -----------------------------------------------------------------------------
// <copyright file="ServiceExtensions.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.MacManager.Common
{
    using Microsoft.Extensions.DependencyInjection;

    public static class ServiceExtensions
    {
        public static IServiceCollection AddMacManagerCommonServices(this IServiceCollection services)
        {
            services.AddSingleton<IMacAddressComponent, MacAddressComponent>();
            return services;
        }
    }
}
