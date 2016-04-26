// --------------------------------------------------------------------------------------------------------------------
// <copyright file="global.asax.cs" company="Microsoft Corporation">
// Copyright (c) company. All rights reserved.
// </copyright>
// <author>Nihara</author>
// <summary>Sample class</summary>
// --------------------------------------------------------------------------------------------------------------------
namespace WebRole1
{
    using System;
    using System.Web;

    /// <summary>
    /// Global class
    /// </summary>
    public class Global : HttpApplication
    {
        /// <summary>
        /// Application start handler
        /// </summary>
        /// <param name="sender">sender object</param>
        /// <param name="e">event args</param>
        public void Application_Start(object sender, EventArgs e)
        {
            // Code that runs on application startup
            AuthConfig.RegisterOpenAuth();
        }

        /// <summary>
        /// Application end handler
        /// </summary>
        /// <param name="sender">sender object</param>
        /// <param name="e">event args</param>
        public void Application_End(object sender, EventArgs e)
        {
            // Code that runs on application shutdown
        }

        /// <summary>
        /// Application error handler
        /// </summary>
        /// <param name="sender">sender object</param>
        /// <param name="e">event args</param>
        public void Application_Error(object sender, EventArgs e)
        {
            // Code that runs when an unhandled error occurs
        }
    }
}
