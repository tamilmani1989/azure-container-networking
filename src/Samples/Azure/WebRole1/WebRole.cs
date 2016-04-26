// --------------------------------------------------------------------------------------------------------------------
// <copyright file="WebRole.cs" company="Microsoft Corporation">
// Copyright (c) company. All rights reserved.
// </copyright>
// <author>Nihara</author>
// <summary>Sample Web role class implementation for build verification coverage</summary>
// --------------------------------------------------------------------------------------------------------------------
namespace WebRole1
{
    using Microsoft.WindowsAzure.ServiceRuntime;

    /// <summary>
    /// Web role class
    /// </summary>
    public class WebRole : RoleEntryPoint
    {
        /// <summary>
        /// OnStart method
        /// </summary>
        /// <returns>Flag indicating on start success.</returns>
        public override bool OnStart()
        {
            // For information on handling configuration changes
            // see the MSDN topic at http://go.microsoft.com/fwlink/?LinkId=166357.
            return base.OnStart();
        }
    }
}
