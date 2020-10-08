// --------------------------------------------------------------------------------------------------------------------
// <copyright file="Program.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.Gateway
{
    using System;
    using System.IO;

    using Microsoft.AspNetCore.Hosting;
    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Extensions.Configuration;
    using Microsoft.Extensions.Hosting;

    public class Program
    {
        /// <summary>
        /// Entry point for gateway service
        /// </summary>
        /// <param name="args"></param>
        public static void Main(string[] args)
        {
            const string settingsFileName = "appsettings.json";
            const string settingsFilePathEnvVar = "GATEWAY_CONFIG_PATH";

            var config = new ConfigurationBuilder()
                .SetBasePath(AppInfo.GetFilePath(settingsFileName, new[] { settingsFilePathEnvVar }))
                .AddJsonFile(settingsFileName, optional: false)
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
                         options.ListenAnyIP(Convert.ToInt32(config["Port"]), listenOptions =>
                         {
                             listenOptions.UseHttps(Certificates.GetKeyVaultCertificate(config["KeyVault:Url"], config["KeyVault:SslCertName"]));
                         });
                     })
                    .UseStartup<Startup>();
                });
    }
}
