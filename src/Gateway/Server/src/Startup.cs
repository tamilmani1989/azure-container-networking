// --------------------------------------------------------------------------------------------------------------------
// <copyright file="Startup.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.Gateway
{
    using System;
    using System.Collections.Generic;
    using System.IdentityModel.Tokens.Jwt;
    using System.Linq;
    using System.Net.Mime;
    using System.Text.Json;
    using Microsoft.ApplicationInsights.Channel;
    using Microsoft.ApplicationInsights.Extensibility;
    using Microsoft.ApplicationInsights.WindowsServer.TelemetryChannel;
    using Microsoft.AspNetCore.Authentication;
    using Microsoft.AspNetCore.Authentication.AzureAD.UI;
    using Microsoft.AspNetCore.Authentication.JwtBearer;
    using Microsoft.AspNetCore.Authentication.OpenIdConnect;
    using Microsoft.AspNetCore.Authorization;
    using Microsoft.AspNetCore.Builder;
    using Microsoft.AspNetCore.Diagnostics.HealthChecks;
    using Microsoft.AspNetCore.Hosting;
    using Microsoft.AspNetCore.Http;
    using Microsoft.AspNetCore.Mvc.Authorization;
    using Microsoft.AspNetCore.Mvc.Formatters;
    using Microsoft.Extensions.Configuration;
    using Microsoft.Extensions.DependencyInjection;
    using Microsoft.Extensions.Diagnostics.HealthChecks;
    using Microsoft.Extensions.Hosting;
    using Microsoft.Extensions.Logging;
    using Microsoft.IdentityModel.Logging;
    using Newtonsoft.Json;

    /// <summary>
    /// This class configures the entire run time behavior of gateway service.
    /// Gateway dependencies injection, logging, authentication/authorization policies etc.
    /// are configured here.
    /// </summary>
    public class Startup
    {
        public Startup(IConfiguration configuration)
        {
            Configuration = configuration;
        }

        public IConfiguration Configuration
        {
            get;
        }

        /// <summary>
        /// This method gets called by the runtime. Use this method to add services to the container.
        /// </summary>
        /// <param name="services"></param>
        public void ConfigureServices(IServiceCollection services)
        {
            // Add custom properties to telemetry (log)
            services.AddSingleton<ITelemetryInitializer, CustomTelemetryProperties>();

            // The following line enables Application Insights telemetry collection.
            services.AddApplicationInsightsTelemetry();

            services.AddServicePorts();

            services.AddHttpRequestWrapper();

            services.AddHttpContextAccessor();

            services.AddHealthChecks().AddCheck<HealthCheck>("gateway_health_check");

            // Gateway supports auth through two mechanisms. 
            // 1. Azure AD user based auth - using OAuth 2.0 (auth code flow). Used when users login to gateway.
            //    https://docs.microsoft.com/en-us/azure/active-directory/azuread-dev/v1-protocols-oauth-code
            // 2. JWT token validation - using OAuth 2.0 (client credential grant). Used when client applications (instead of users) access gateway.
            //    https://docs.microsoft.com/en-us/azure/active-directory/azuread-dev/v1-oauth2-client-creds-grant-flow
            // Configure the following.
            // a. Add authentication before adding controller
            // b. Add JWT and Azure AD (that uses OpenIdConnect) and cookies.
            // c. Use a smart policy scheme to choose the correct authentication scheme at runtime
            services
                .AddAuthentication(sharedOptions =>
                {
                    sharedOptions.DefaultScheme = "smart";
                    sharedOptions.DefaultChallengeScheme = "smart";
                })
                .AddPolicyScheme("smart", "Authorization Bearer or OIDC", options =>
                {
                    options.ForwardDefaultSelector = context =>
                    {
                        var authHeader = context.Request.Headers["Authorization"].FirstOrDefault();
                        if (authHeader?.StartsWith("Bearer ", StringComparison.OrdinalIgnoreCase) == true)
                        {
                            return AzureADDefaults.JwtBearerAuthenticationScheme;
                        }
                        return AzureADDefaults.AuthenticationScheme;
                    };
                })
                .AddAzureADBearer(options => Configuration.Bind("AzureAd", options))
                .AddAzureAD(options => Configuration.Bind("AzureAd", options));


            services.Configure<CookiePolicyOptions>(options =>
            {
                // This lambda determines whether user consent for non-essential cookies is needed for a given request.
                options.CheckConsentNeeded = context => true;
                options.MinimumSameSitePolicy = SameSiteMode.None;
            });

            services.Configure<OpenIdConnectOptions>(AzureADDefaults.OpenIdScheme, options =>
            {
                // Microsoft identity platform
                options.Authority = options.Authority + "/v2.0/";

                // accept several tenants (here simplified)
                options.TokenValidationParameters.ValidateIssuer = false;
            });

            services.Configure<JwtBearerOptions>(AzureADDefaults.JwtBearerAuthenticationScheme, options =>
            {
                // This is a Microsoft identity platform web API.
                options.Authority += "/v2.0";

                //The web API accepts as audiences both the Client ID(options.Audience) and api://{ClientID}.
                options.TokenValidationParameters.ValidAudiences = new[]
                {
                                options.Audience,
                                $"api://{options.Audience}"
                };

                options.TokenValidationParameters.ValidateLifetime = true;
                options.TokenValidationParameters.ValidateIssuer = false;
                options.Events = JwtBearerMiddlewareDiagnostics.Subscribe(options.Events);
            });

            // Set this to false, so claim type doesn't get mapped to long names.
            // For eg. Claim name scp gets mapped to http://schemas.microsoft.com/identity/claims/scope
            // if we don't set this mapping to false. We look up by short names in claim validation.
            JwtSecurityTokenHandler.DefaultMapInboundClaims = false;
            services.AddControllers(options =>
            {
                // Authorization builder needs to be added because we have multiple auth schemes (openid connect and jwt tokens)
                var policy = new AuthorizationPolicyBuilder()
                                .RequireAuthenticatedUser()
                                .RequireAssertion(context =>
                                {
                                    return context.User.HasRoleClaim(Configuration["RoleClaim"]) || context.User.HasScpClaim(Configuration["ScopeClaim"]);
                                })
                                .Build();

                options.Filters.Add(new AuthorizeFilter(policy));
            });
        }

        /// <summary>
        /// This method gets called by the runtime. Use this method to configure the HTTP request pipeline.
        /// </summary>
        /// <param name="app"></param>
        /// <param name="env"></param>
        /// <param name="logger"></param>
        public void Configure(IApplicationBuilder app, IWebHostEnvironment env, ILogger<Startup> logger)
        {
            if (env.IsDevelopment())
            {
                IdentityModelEventSource.ShowPII = true;
                app.UseDeveloperExceptionPage();
            }

            JwtBearerMiddlewareDiagnostics.JwtEventLog = logger;

            app.UseHttpsRedirection();

            app.UseRouting();

            app.UseAuthentication();
            app.UseAuthorization();

            app.UseEndpoints(endpoints =>
            {
                endpoints.MapControllers();

                endpoints.MapHealthChecks("/health", new HealthCheckOptions
                {
                    ResponseWriter = async (context, report) =>
                    {
                        var result = JsonConvert.SerializeObject(
                                         new HealthResult
                                         {
                                             Uptime = report.Entries.FirstOrDefault().Value.Description
                                         },
                                         Formatting.None,
                                         new JsonSerializerSettings
                                         {
                                             NullValueHandling = NullValueHandling.Ignore
                                         }); ; ;

                        context.Response.ContentType = MediaTypeNames.Application.Json;
                        await context.Response.WriteAsync(result);
                    }
                });
            });
        }
    }
}