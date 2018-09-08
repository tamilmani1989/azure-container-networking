//-----------------------------------------------------------------------
// <copyright file="Startup.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// 
//-----------------------------------------------------------------------
namespace DNCService.Rest
{
    using Microsoft.AspNetCore.Builder;
    using Microsoft.AspNetCore.Hosting;
    using Microsoft.AspNetCore.Routing;
    using Microsoft.Extensions.Configuration;
    using Microsoft.Extensions.DependencyInjection;

    /// <summary>
    /// Defines the StartUp class for the http listener.
    /// </summary>
    public class Startup
    {
        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="env">Supply the hosting environment.</param>
        public Startup(IHostingEnvironment env)
        {
            IConfigurationBuilder builder = new ConfigurationBuilder()
                .SetBasePath(env.ContentRootPath)
                .AddJsonFile("appsettings.json", optional: true, reloadOnChange: true)
                .AddJsonFile($"appsettings.{env.EnvironmentName}.json", optional: true)
                .AddEnvironmentVariables();
            this.Configuration = builder.Build();
        }

        /// <summary>
        /// Configuration root.
        /// </summary>
        public IConfigurationRoot Configuration { get; }

        /// <summary>
        /// Invoked by the runtime. Use this method to add services to the container.
        /// </summary>
        /// <param name="services">Suppl the collection of services.</param>
        public void ConfigureServices(IServiceCollection services)
        {
            services.AddRouting();
        }

        /// <summary>
        /// Configures the app builder using Web API.
        /// </summary>
        public void Configure(IApplicationBuilder app)
        {
            DncWebController dncWebController = new DncWebController();
            var routeBuilder = new RouteBuilder(app);

            // Handlers for Node management
            routeBuilder.MapPost("nodes/{NodeID}", dncWebController.CreateOrUpdateNodeAsync);
            routeBuilder.MapGet("nodes/{NodeID}", dncWebController.GetNodeAsync);
            routeBuilder.MapDelete("nodes/{NodeID}", dncWebController.DeleteNodeAsync);

            // Handlers for Network management
            routeBuilder.MapPost("networks/{NetworkID}", dncWebController.CreateOrUpdateNetworkAsync);
            routeBuilder.MapDelete("networks/{NetworkID}", dncWebController.DeleteNetworkAsync);

            // Handlers for Network subnet management
            routeBuilder.MapPost("networks/{NetworkID}/subnets/{SubnetName}", dncWebController.JoinSubnetAsync);
            routeBuilder.MapGet("networks/{NetworkID}/subnets/{SubnetName}", dncWebController.GetSubnetAsync);

            // Handlers for Network container management
            routeBuilder.MapPost("networks/{NetworkID}/networkcontainer/{NetworkContainerID}", dncWebController.CreateOrUpdateNetworkContainerAsync);
            routeBuilder.MapGet("networks/{NetworkID}/networkcontainer/{NetworkContainerID}", dncWebController.GetNetworkContainerAsync);
            routeBuilder.MapDelete("networks/{NetworkID}/networkcontainer/{NetworkContainerID}", dncWebController.DeleteNetworkContainerAsync);

            IRouter routes = routeBuilder.Build();

            app.UseRouter(routes);
        }
    }
}