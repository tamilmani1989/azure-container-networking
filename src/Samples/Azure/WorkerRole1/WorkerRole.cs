// --------------------------------------------------------------------------------------------------------------------
// <copyright file="WorkerRole.cs" company="Microsoft Corporation">
// Copyright (c) company. All rights reserved.
// </copyright>
// <author>Nihara</author>
// <summary>Sample Worker role class implementation for build verification coverage</summary>
// --------------------------------------------------------------------------------------------------------------------
namespace WorkerRole1
{
    using System.Diagnostics;
    using System.Net;
    using System.Threading;
    using Microsoft.WindowsAzure.ServiceRuntime;

    /// <summary>
    /// Worker Role class
    /// </summary>
    public class WorkerRole : RoleEntryPoint
    {
        /// <summary>
        /// Implemented Run method.
        /// </summary>
        public override void Run()
        {
            // This is a sample worker implementation. Replace with your logic.
            Trace.WriteLine("WorkerRole1 entry point called", "Information");

            while (true)
            {
                Thread.Sleep(10000);
                Trace.WriteLine("Working", "Information");
            }
        }

        /// <summary>
        /// Override OnStart method.
        /// </summary>
        /// <returns>Flag indicating on start success</returns>
        public override bool OnStart()
        {
            // Set the maximum number of concurrent connections
            ServicePointManager.DefaultConnectionLimit = 12;

            // For information on handling configuration changes
            // see the MSDN topic at http://go.microsoft.com/fwlink/?LinkId=166357.
            return base.OnStart();
        }
    }
}
