// --------------------------------------------------------------------------------------------------------------------
// <copyright file="Program.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.IO;
    using Microsoft.AspNetCore.Hosting;
    using Microsoft.AspNetCore.Server.Kestrel.Https;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Configuration;
    using Microsoft.Extensions.Hosting;

    public class Program
    {
        /// <summary>
        /// Entry point for RP service
        /// </summary>
        /// <param name="args"></param>
        public static void Main(string[] args)
        {
            var config = new ConfigurationBuilder()
                .SetBasePath(Directory.GetCurrentDirectory())
                .AddJsonFile("appsettings.json", optional: false)
                .AddCommandLine(args)
                .Build();

            CreateHostBuilder(args, config).Build().Run();
        }

        /// <summary>
        /// Create a host build that will host the service
        /// </summary>
        /// <param name="args"></param>
        /// <param name="config"></param>
        /// <returns></returns>
        public static IHostBuilder CreateHostBuilder(string[] args, IConfiguration config) =>
            Host.CreateDefaultBuilder(args)
            .ConfigureWebHostDefaults(webBuilder =>
            {
                webBuilder
                .UseKestrel(options =>
                {
                    // Using AllowCertificate instead of RequireCertificate since with AllowCertificate, cert validation
                    // errors show up in asp.net logs and help with debugging. With Require, it shows up as SSL error
                    // and can be bit harder to debug.
                    options.ConfigureHttpsDefaults(o =>
                      o.ClientCertificateMode = ClientCertificateMode.AllowCertificate);

                    options.ListenAnyIP(Convert.ToInt32(config["Port"]), listenOptions =>
                    {
                        listenOptions.UseHttps(Certificates.GetKeyVaultCertificate(config["KeyVault:Url"], config["KeyVault:SslCertName"]));
                    });
                })
                .UseStartup<Startup>();
            });
    }
}
