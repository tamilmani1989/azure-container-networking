//-----------------------------------------------------------------------
// <copyright file="Program.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// 
//-----------------------------------------------------------------------

using System;
using System.Diagnostics.Tracing;
using System.Threading;
using DNCService.Eventing;
using Microsoft.ServiceFabric.Services.Runtime;

namespace DNCService
{
    internal static class Program
    {
        /// <summary>
        /// This is the entry point of the service host process.
        /// </summary>
        private static void Main()
        {
            try
            {
                // The ServiceManifest.XML file defines one or more service type names.
                // Registering a service maps a service type name to a .NET type.
                // When Service Fabric creates an instance of this service type,
                // an instance of the class is created in this host process.

                ServiceRuntime.RegisterServiceAsync("DNCServiceType",
                    context => new DNCService(context)).GetAwaiter().GetResult();
                //Creating a new event listener to redirect the traces to a file
                ServiceEventListener listener = new ServiceEventListener("DNCServiceService");
                listener.EnableEvents(ServiceEventSource.Current, EventLevel.LogAlways, EventKeywords.All);
                // Prevents this host process from terminating so services keep running.
                Thread.Sleep(Timeout.Infinite);
            }
            catch (Exception)
            {
                throw;
            }
        }
    }
}
