// -----------------------------------------------------------------------------
// <copyright file="Startup.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.MacInventoryManager
{
    using System;
    using System.Linq;
    using System.Net.Http;
    using System.Net.Mime;
    using Microsoft.AspNetCore.Builder;
    using Microsoft.AspNetCore.Diagnostics.HealthChecks;
    using Microsoft.AspNetCore.Hosting;
    using Microsoft.AspNetCore.Http;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Azure.ContainerNetworking.MacInventoryManager.DataAccess;
    using Microsoft.Azure.ContainerNetworking.MacManager.Common;
    using Microsoft.Extensions.Configuration;
    using Microsoft.Extensions.DependencyInjection;
    using Microsoft.Extensions.Hosting;
    using Newtonsoft.Json;

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

        // This method gets called by the runtime. Use this method to add services to the container.
        public void ConfigureServices(IServiceCollection services)
        {
            // The following line enables Application Insights telemetry collection.
            CustomLog.SetPrefix("[MacInventoryManager]");
            services.AddApplicationInsightsTelemetry(Configuration);

            // Add custom properties to telemetry (log)
            HttpUtility httpUtility = new HttpUtility(new HttpClient());
            ImdsRetriever imdsRetriever = new ImdsRetriever(httpUtility);

            services.AddSingleton<IHttpUtility, HttpUtility>(provider => httpUtility);
            services.AddSingleton<IImdsRetriever, ImdsRetriever>(provider => imdsRetriever);
            services.AddSingleton<IKeyVaultUtility, KeyVaultUtility>();

            // set up IOptions from configuration
            services.Configure<DatastoreSettings>(Configuration.GetSection("DataStoreSettings"));
            services.Configure<AzureSettings>(Configuration.GetSection("AzureSettings"));
            ClientIdentity[]  identities = Configuration.GetSection("ClientIdentities").Get<ClientIdentity[]>();
            services.Configure<ClientIdentityCollection>(x => x.Identities = identities);

            services.AddSingleton<IMacInventoryManagerService, MacInventoryManagerService>();
            services.AddSingleton<CosmosClientProvider>();

            services.AddSingleton<MacInventoryInitializationState>();
            services.AddHostedService<MacInventoryInitializationService>();
            services.AddSingleton<IMacAddressRangeComponent, MacAddressRangeComponent>();
            services.AddMacManagerCommonServices();

            services.AddHttpContextAccessor();
            services.AddHealthChecks().AddCheck<HealthCheck>("macinventorymanager_health_check");

            services.AddControllers();
            services.AddMdmServices(Configuration);

            Console.WriteLine("Startup ConfigureServices done");
        }

        // This method gets called by the runtime. Use this method to configure the HTTP request pipeline.
        public void Configure(IApplicationBuilder app, IWebHostEnvironment env)
        {
            if (env.IsDevelopment())
            {
                app.UseDeveloperExceptionPage();
            }

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
