// -----------------------------------------------------------------------------
// <copyright file="ExtensionMethods.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.Gateway
{
    using System;
    using System.Collections.Generic;
    using System.Linq;
    using System.Security.Claims;

    using Microsoft.Extensions.DependencyInjection;

    public static class ExtensionMethods
    {
        /// <summary>
        /// Inject IServicePorts to Service collection as dependency
        /// </summary>
        /// <param name="services"></param>
        public static void AddServicePorts(this IServiceCollection services)
        {
            // Add service ports as service dependency. This is just a dictionary for reads, so singleton instance is fine.
            ServicePorts ports = ServicePorts.Load();
            services.AddSingleton(typeof(IServicePorts), ports);

            ServicePorts servicePorts = new ServicePorts();
            foreach (KeyValuePair<string, ServicePort> kvp in ports.GetServicePortsMapping())
            {
                // Add http client as service dependency per service gateway routes to.
                // All services injected in this method become available in controller
                services.AddHttpClient(kvp.Key, c =>
                {
                    // Assume this is an "external" service which requires an API KEY
                    c.BaseAddress = new UriBuilder(
                        kvp.Value.Scheme,
                        kvp.Value.GetHostName(),
                        kvp.Value.Port).Uri;
                });
            }
        }

        /// <summary>
        /// Inject IHttpRequestWrapper to Service collection as dependency
        /// </summary>
        /// <param name="services"></param>
        public static void AddHttpRequestWrapper(this IServiceCollection services)
        {
            services.AddTransient<IHttpRequestWrapper, HttpRequestWrapper>();
        }

        /// <summary>
        /// Validate scp (scope) claim
        /// </summary>
        /// <param name="claimsPrincipal"></param>
        /// <param name="validValue"></param>
        /// <returns></returns>
        public static bool HasScpClaim(this ClaimsPrincipal claimsPrincipal, string validValue)
        {
            Claim scpClaim = claimsPrincipal.Claims.FirstOrDefault(c => c.Type == "scp");
            if (scpClaim == null || string.IsNullOrWhiteSpace(scpClaim.Value))
            {
                return false;
            }

            string[] claimArr = scpClaim.Value.Split(new string[] { " ", "," }, StringSplitOptions.RemoveEmptyEntries);
            if (claimArr == null || claimArr.Length == 0)
            {
                return false;
            }

            if (!claimArr.Any(s => s.Equals(validValue, StringComparison.OrdinalIgnoreCase)))
            {
                return false;
            }

            return true;
        }

        /// <summary>
        /// Validate roles claim
        /// </summary>
        /// <param name="claimsPrincipal"></param>
        /// <param name="validValue"></param>
        /// <returns></returns>
        public static bool HasRoleClaim(this ClaimsPrincipal claimsPrincipal, string validValue)
        {
            return claimsPrincipal.HasClaim("roles", validValue);
        }

        /// <summary>
        /// Validate appid claim
        /// </summary>
        /// <param name="claimsPrincipal"></param>
        /// <param name="validValue"></param>
        /// <returns></returns>
        public static bool HasAppIdClaim(this ClaimsPrincipal claimsPrincipal, string validValue)
        {
            string[] appids = validValue.Split(",");

            foreach (string id in appids)
            {
                if (claimsPrincipal.HasClaim("appid", id))
                {
                    return true;
                }
            }

            return false;
        }
    }
}
