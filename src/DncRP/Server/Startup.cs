// -----------------------------------------------------------------------------
// <copyright file="Startup.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Collections.Generic;
    using System.Net.Http;
    using System.Security.Claims;
    using System.Threading.Tasks;
    using Microsoft.AspNetCore.Authentication.Certificate;
    using Microsoft.AspNetCore.Builder;
    using Microsoft.AspNetCore.Hosting;
    using Microsoft.AspNetCore.Http;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Azure.Cosmos.Table;
    using Microsoft.Extensions.Configuration;
    using Microsoft.Extensions.DependencyInjection;
    using Microsoft.Extensions.Hosting;
    using Microsoft.Extensions.Logging;
    using Microsoft.Extensions.Primitives;

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
            // Add custom properties to telemetry (log)
            IHttpUtility httpUtility = new HttpUtility(new HttpClient());
            services.AddSingleton<IHttpUtility, HttpUtility>(provider =>
            {
                return (HttpUtility)httpUtility;
            });

            IImdsRetriever imdsRetriever = new ImdsRetriever(httpUtility);
            IKeyVaultUtility keyVaultUtility = new KeyVaultUtility();

            // initialize context first
            ContextInitializer.InitializeContext(Configuration, imdsRetriever);

            DatastoreSettings[] datastoreSettings = Configuration.GetSection("DataStoreSettings").Get<DatastoreSettings[]>();
            DatastoreSettings currentDatastoreSettings = datastoreSettings.FindByRegion(CurrentContext.AzureRegion);
            ClientIdentity clientIdentity = Configuration.GetSection("Identity").Get<ClientIdentity>();
            IConnectionStringProvider connectionStringProvider = new ConnectionStringProvider(
                httpUtility, currentDatastoreSettings);

            services.AddSingleton<ICertificateRetriever, ArmCertificateRetriever>(provider =>
            {
                // TODO: Use IOptions instead of directly passing in Configuration
                ILogger<ArmCertificateRetriever> logger = provider.GetRequiredService<ILogger<ArmCertificateRetriever>>();
                return new ArmCertificateRetriever(
                    Configuration,
                    imdsRetriever, httpUtility, logger, Configuration["AzureEnvironment"]);
            });

            services.AddSingleton<ICertificateValidator, CertificateValidator>(provider =>
            {
                // TODO: Use IOptions instead of directly passing in Configuration
                ILogger<CertificateValidator> logger = provider.GetRequiredService<ILogger<CertificateValidator>>();
                ICertificateRetriever retriever = provider.GetRequiredService<ICertificateRetriever>();
                return new CertificateValidator(
                    retriever, logger, Configuration["Thumbprint"]);
            });

            services.AddSingleton<ITableOperations, TableOperations>(provider =>
            {
                ILogger<TableOperations> tableOperationsLogger = provider.GetRequiredService<ILogger<TableOperations>>();
                ILogger<CosmosDbDataStore> cosmosDbLogger = provider.GetRequiredService<ILogger<CosmosDbDataStore>>();

                // Initiazlize the CosmosDbStore
                ITableDataStore cosmosDbStore = new CosmosDbDataStore(connectionStringProvider, cosmosDbLogger);
                cosmosDbStore.InitAsync().GetAwaiter().GetResult();

                return new TableOperations(cosmosDbStore, tableOperationsLogger, httpUtility, keyVaultUtility);
            });

            // The following line enables Application Insights telemetry collection.
            services.AddApplicationInsightsTelemetry();

            services.AddHttpContextAccessor();

            services.AddAuthentication(CertificateAuthenticationDefaults.AuthenticationScheme)
                .AddCertificate(options =>
                {
                    options.AllowedCertificateTypes = CertificateTypes.Chained;
                    options.Events = new CertificateAuthenticationEvents
                    {
                        OnCertificateValidated = async (context) =>
                        {
                            var validationService =
                            context.HttpContext.RequestServices.GetService<ICertificateValidator>();
                            bool isValid = await validationService.IsValid(context.ClientCertificate);
                            if (isValid)
                            {
                                var claims = new[]
                                {
                                    new Claim(
                                        ClaimTypes.NameIdentifier,
                                        context.ClientCertificate.Subject,
                                        ClaimValueTypes.String,
                                        context.Options.ClaimsIssuer),
                                    new Claim(
                                        ClaimTypes.Name,
                                        context.ClientCertificate.Subject,
                                        ClaimValueTypes.String,
                                        context.Options.ClaimsIssuer)
                                };

                                context.Principal = new ClaimsPrincipal(
                                    new ClaimsIdentity(claims, context.Scheme.Name));
                                context.Success();
                                Console.WriteLine("OnCertificateValidated: success");
                            }
                            else
                            {
                                context.Fail("authentication failed");
                                Console.WriteLine("OnCertificateValidated: failed");
                            }
                        },
                        OnAuthenticationFailed = (context) =>
                        {
                            context.Fail("authentication failed");
                            Console.WriteLine("OnAuthenticationFailed: {0}",
                                context.Result != null ? context.Result.Failure.ToString() : "");
                            return Task.CompletedTask;
                        }
                    };
                });

            services.AddControllers();
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
            });
        }
    }
}
