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
    using Microsoft.AspNetCore.WebUtilities;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Configuration;
    using Microsoft.Extensions.DependencyInjection;
    using Microsoft.Extensions.Hosting;
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
            // The following line enables Application Insights telemetry collection.
            CustomLog.SetPrefix("[DNC RP]");
            services.AddApplicationInsightsTelemetry(Configuration);

            // Add custom properties to telemetry (log)
            HttpUtility httpUtility = new HttpUtility(new HttpClient());
            ImdsRetriever imdsRetriever = new ImdsRetriever(httpUtility);
            KeyVaultUtility keyVaultUtility = new KeyVaultUtility();

            services.AddSingleton<IHttpUtility, HttpUtility>(provider => httpUtility);
            services.AddSingleton<IImdsRetriever, ImdsRetriever>(provider => imdsRetriever);
            services.AddSingleton<IKeyVaultUtility, KeyVaultUtility>(provider => keyVaultUtility);

            // initialize context first
            ContextInitializer.InitializeContext(Configuration, imdsRetriever);

            // set up IOptions from configuration
            services.Configure<DatastoreSettings>(Configuration.GetSection("DataStoreSettings"));
            services.Configure<ArmCertificatesSettings>(Configuration.GetSection("ArmCertificatesSettings"));
            services.Configure<CertificateValidatorSettings>(cvs =>
            {
                cvs.Thumbprint = Configuration["Thumbprint"];
            });

            services.AddSingleton<IConnectionStringProvider, ConnectionStringProvider>();
            services.AddSingleton<ICertificateRetriever, ArmCertificateRetriever>();
            services.AddSingleton<ICertificateValidator, CertificateValidator>();
            services.AddSingleton<ITableDataStore, CosmosDbDataStore>();
            services.AddSingleton<ITableOperations, TableOperations>();
            // service to initialize cosmosdb and create table at startup
            services.AddHostedService<TableStoreInitHostedService>();

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
            });
        }
    }
}
