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
    using System.Linq;
    using System.Net.Http;
    using System.Net.Mime;
    using System.Security.Claims;
    using System.Threading;
    using System.Threading.Tasks;
    using JustEat.StatsD;
    using Microsoft.AspNetCore.Authentication.Certificate;
    using Microsoft.AspNetCore.Builder;
    using Microsoft.AspNetCore.Diagnostics.HealthChecks;
    using Microsoft.AspNetCore.Hosting;
    using Microsoft.AspNetCore.Http;
    using Microsoft.AspNetCore.WebUtilities;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Cloud.InstrumentationFramework.Metrics.Extensions;
    using Microsoft.Extensions.Configuration;
    using Microsoft.Extensions.DependencyInjection;
    using Microsoft.Extensions.Hosting;
    using Microsoft.Extensions.Logging;
    using Microsoft.Extensions.Primitives;
    using Newtonsoft.Json;
    using Microsoft.Extensions.Options;
    using global::Azure.Security.KeyVault.Secrets;

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
            CustomLog.SetPrefix("[DNC RP]");
            services.AddApplicationInsightsTelemetry(Configuration);

            HttpUtility httpUtility = new HttpUtility(new HttpClient());
            ImdsRetriever imdsRetriever = new ImdsRetriever(httpUtility);
            KeyVaultUtility keyVaultUtility = new KeyVaultUtility();
            
            services.AddSingleton<IHttpUtility, HttpUtility>(provider => httpUtility);

            HttpClientHandler httpClientHandler = new HttpClientHandler();
            AmeCertificateValidator.IntializeAmeCertificateValidator(Configuration.GetSection("AmeCertValidationSettings").Get<AmeCertValidationSettings>(), httpUtility);
            httpClientHandler.ServerCertificateCustomValidationCallback = AmeCertificateValidator.ValidateAmeRootCA;
            // Add custom properties to telemetry (log)
            var httpUtilityFactory = new HttpUtilityFactory();
            httpUtilityFactory.Register(Constants.HttpUtility, new HttpUtility(new HttpClient()));
            httpUtilityFactory.Register(Constants.DncHttpUtility, new HttpUtility(new HttpClient(httpClientHandler)));
            services.AddSingleton<IHttpUtilityFactory>(httpUtilityFactory);

            // initialize context first
            ContextInitializer.InitializeContext(Configuration, imdsRetriever);

            // Initialize MDM
            MdmStatsDProvider metricsProvider = MdmStatsDProvider.CreateMdmStatsDProvider(Configuration);
            services.AddSingleton<IMetricsProvider, MdmStatsDProvider>(provider => metricsProvider);

            // set up IOptions from configuration
            services.Configure<DatastoreSettings>(Configuration.GetSection("DataStoreSettings"));
            services.Configure<ArmCertificatesSettings>(Configuration.GetSection("ArmCertificatesSettings"));
            services.Configure<CertificateValidatorSettings>(cvs =>
            {
                cvs.Thumbprint = Configuration["Thumbprint"];
            });
            
            services.AddSingleton<IImdsRetriever, ImdsRetriever>(provider => imdsRetriever);
            services.AddSingleton<IKeyVaultUtility, KeyVaultUtility>(provider => keyVaultUtility);
            services.AddSingleton<ICertificateValidator, CertificateValidator>();
            services.AddSingleton<IConnectionStringProvider, ConnectionStringProvider>();
            services.AddSingleton<ICertificateRetriever, ArmCertificateRetriever>();
            services.AddSingleton<ITableDataStore, CosmosDbDataStore>();
            services.AddSingleton<ITableOperations, TableOperations>();
            // service to initialize cosmosdb and create table at startup
            services.AddHostedService<TableStoreInitHostedService>();

            services.AddHttpContextAccessor();
            services.AddHealthChecks().AddCheck<HealthCheck>("dncrp_health_check");

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
                                Console.WriteLine("OnCertificateValidated failed. {0}",
                                    context.Result != null && context.Result.Failure != null
                                    ? context.Result.Failure.ToString()
                                    : "<emty>");
                            }
                        },
                        OnAuthenticationFailed = (context) =>
                        {
                            context.Fail("authentication failed");
                            Console.WriteLine("OnAuthenticationFailed: Result: {0}",
                                context.Result != null && context.Result.Failure != null ? context.Result.Failure.ToString() : "<emty>");
                            Console.WriteLine("OnAuthenticationFailed: Exception: {0}",
                                context.Exception != null ? context.Exception.ToString() : "<empty>");
                            Console.WriteLine("OnAuthenticationFailed: Body is {0}",
                                context.HttpContext.Request != null ? "non-null" : "null");
                            Console.WriteLine("OnAuthenticationFailed. Request content type: {0}, length: {1}",
                                context.Request.ContentType, context.Request.ContentLength);

                            return Task.CompletedTask;
                        }
                    };
                });

            services.AddControllers();
            StartLoggingServiceUpMetric(metricsProvider);
           
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

            IServiceProvider sp = app.ApplicationServices;
            RenewMSICerts(
                sp.GetRequiredService<ITableOperations>(), 
                sp.GetRequiredService<IHttpUtility>(),
                sp.GetRequiredService<IKeyVaultUtility>()).GetAwaiter().GetResult();
        }

        private async Task RenewMSICerts(
            ITableOperations tableOperations, 
            IHttpUtility httpUtility,
            IKeyVaultUtility keyVaultUtility)
        {
            Console.WriteLine("[DNC RP] check renewing msi certs");
            List<OrchestratorConfigEntity> configEntityList = await tableOperations.OrchestratorOperations.GetAllOrchestratorsConfig();
            DateTime currentTime = DateTime.UtcNow;
            Console.WriteLine("[DNC RP] Current utc time:{0}", currentTime.ToString());
            Console.WriteLine("[DNC RP] config entity count:{0}", configEntityList.Count());
           
            foreach (OrchestratorConfigEntity entity in configEntityList)
            {
                MSIIdentity identity = JsonUtility.Deserialize<MSIIdentity>(entity.IdentityProperties);
                OrchestratorPropertiesInternal props = JsonUtility.Deserialize<OrchestratorPropertiesInternal>(entity.Properties);
                if (props != null && identity != null)
                {
                    if (identity.RenewAfter < currentTime)
                    {
                        Console.WriteLine("[DNC RP] Renew date {0} after past current date {1}. renew cert for orchestrator id:{2}", identity.RenewAfter.ToString(),
                           currentTime.ToString(), entity.RowKey);

                        MSIIdentity newIdentity = await getNewCert(httpUtility, identity, entity.RowKey);
                        if (newIdentity != null)
                        {
                            Console.WriteLine("[DNC RP] Got new renewed identity for orchestrator id: {0}, client id: {1} newmsi:{2}  renew after:{3}", entity.RowKey, identity.ClientId, newIdentity.ClientId, newIdentity.RenewAfter.ToString());

                            int httpStatusCode = await tableOperations.OrchestratorOperations.InsertOrReplaceOrchestratorConfigAsync(entity.PartitionKey, entity.RowKey, props.PrivateEndpointName,
                                entity.OrchestratorType, props, newIdentity);
                            if (!HttpStatusCodeCheck.IsSuccess(httpStatusCode))
                            {
                                Console.WriteLine("[DNC RP] Insert to orchestratorconfig failed for orchestrator id {0} with http code:{1}", entity.RowKey, httpStatusCode);
                            }
                        }
                    }
                    else
                    {
                        Console.WriteLine("[DNC RP] Skipping MSI renewal as cert renewal date {0} not past yet for orchestratorid:{1}", identity.RenewAfter.ToString(), entity.RowKey);
                    }
                }
            }
        }

        private async Task<MSIIdentity> getNewCert(IHttpUtility httpUtility, MSIIdentity identity, string orchestratorId)
        {
            MSIIdentityMetadata identityMetadata = new MSIIdentityMetadata();
            identityMetadata.Url = identity.ClientSecretUrl;
            try
            {
                return await ExtensionMethods.GetMSIIdentity(httpUtility, identityMetadata);
            }
            catch (Exception ex)
            {
                Console.WriteLine("[DNC RP] Exception caught while getting newcert for orchestratorId {0}: {1}", orchestratorId, ex.ToString());
                return null;
            }
            
        }

        private void StartLoggingServiceUpMetric(MdmStatsDProvider provider)
        {
            Thread thread = new Thread(ServiceUpMetricThread);
            thread.Start(provider);
        }

        private void ServiceUpMetricThread(object data)
        {
            IMetricsProvider provider = (IMetricsProvider)data;
            while (true)
            {
                provider.EmitMetric(MetricNames.DncRPUP, 1);
                Thread.Sleep(TimeSpan.FromSeconds(60));
            }
        }
    }
}
