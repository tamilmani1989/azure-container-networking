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
            // Even though ASP.NET implicitly reads appsettings.Development.json based on environment variable,
            // we are setting its path explicitly here. This is on purpose since in prod scenario, we get it
            // from a custom path and we want both dev/prod scenarios to work the same (pass settings file location explicitly)
            const string settingsFilePathEnvVar = "DNCRP_CONFIG_PATH";
            bool isDevelopment = Environment.GetEnvironmentVariable("ASPNETCORE_ENVIRONMENT") == Environments.Development;
            string settingsFileName = isDevelopment ? "appsettings.Development.json" : "appsettings.json";

            string settingsFileDirectory = AppInfo.GetFilePath(settingsFileName, new[] { settingsFilePathEnvVar });
            string settingsFileFullPath = Path.Combine(settingsFileDirectory, settingsFileName);
            Console.WriteLine("Dnc RP settings file path: {0}", settingsFileFullPath);

            var config = new ConfigurationBuilder()
                .SetBasePath(settingsFileDirectory)
                .AddJsonFile(settingsFileFullPath, optional: false)
                .AddCommandLine(args)
                .Build();

            CreateHostBuilder(args, config, settingsFileFullPath).Build().Run();
        }

        /// <summary>
        /// Create a host build that will host the service
        /// </summary>
        /// <param name="args"></param>
        /// <param name="config"></param>
        /// <returns></returns>
        public static IHostBuilder CreateHostBuilder(string[] args, IConfiguration config, string configPath) =>
            Host.CreateDefaultBuilder(args)
            .ConfigureWebHostDefaults(webBuilder =>
            {
                webBuilder
                .ConfigureAppConfiguration(builder => builder.AddJsonFile(configPath))
                .UseKestrel(options =>
                {
                    // Using AllowCertificate instead of RequireCertificate since with AllowCertificate, cert validation
                    // errors show up in asp.net logs and help with debugging. With Require, it shows up as SSL error
                    // and can be bit harder to debug. Also, only tls1.2 and tls1.3 due to compliance
                    options.ConfigureHttpsDefaults(o =>
                    {
                        o.ClientCertificateMode = ClientCertificateMode.AllowCertificate;
                        o.SslProtocols = System.Security.Authentication.SslProtocols.Tls13 | System.Security.Authentication.SslProtocols.Tls12;
                    });

                    options.ListenAnyIP(Convert.ToInt32(config["Port"]), listenOptions =>
                    {
                        listenOptions.UseHttps(Certificates.GetKeyVaultCertificate(config["KeyVault:Url"], config["KeyVault:SslCertName"]));
                    });
                })
                .UseStartup<Startup>();
            });
    }
}
