// --------------------------------------------------------------------------------------------------------------------
// <copyright file="Site.Master.cs" company="Microsoft Corporation">
// Copyright (c) company. All rights reserved.
// </copyright>
// <author>Nihara</author>
// <summary>Sample class</summary>
// --------------------------------------------------------------------------------------------------------------------
namespace WebRole1
{
    using System;
    using System.Web;
    using System.Web.Security;
    using System.Web.UI;

    /// <summary>
    /// Site Master class
    /// </summary>
    public partial class SiteMaster : MasterPage
    {
        /// <summary>
        /// Token Key
        /// </summary>
        private const string AntiXsrfTokenKey = "__AntiXsrfToken";

        /// <summary>
        /// User Name Key
        /// </summary>
        private const string AntiXsrfUserNameKey = "__AntiXsrfUserName";

        /// <summary>
        /// Token Value
        /// </summary>
        private string antiXsrfTokenValue;

        /// <summary>
        /// Page initialization handler
        /// </summary>
        /// <param name="sender">Sender object</param>
        /// <param name="e">event args</param>
        protected void Page_Init(object sender, EventArgs e)
        {
            // The code below helps to protect against XSRF attacks
            var requestCookie = Request.Cookies[AntiXsrfTokenKey];
            Guid requestCookieGuidValue;
            if (requestCookie != null && Guid.TryParse(requestCookie.Value, out requestCookieGuidValue))
            {
                // Use the Anti-XSRF token from the cookie
                this.antiXsrfTokenValue = requestCookie.Value;
                Page.ViewStateUserKey = this.antiXsrfTokenValue;
            }
            else
            {
                // Generate a new Anti-XSRF token and save to the cookie
                this.antiXsrfTokenValue = Guid.NewGuid().ToString("N");
                Page.ViewStateUserKey = this.antiXsrfTokenValue;

                var responseCookie = new HttpCookie(AntiXsrfTokenKey)
                {
                    HttpOnly = true,
                    Value = this.antiXsrfTokenValue
                };

                if (FormsAuthentication.RequireSSL && Request.IsSecureConnection)
                {
                    responseCookie.Secure = true;
                }

                Response.Cookies.Set(responseCookie);
            }

            Page.PreLoad += this.Master_Page_PreLoad;
        }

        /// <summary>
        /// Page pre-load handler
        /// </summary>
        /// <param name="sender">sender object</param>
        /// <param name="e">event args</param>
        protected void Master_Page_PreLoad(object sender, EventArgs e)
        {
            if (!this.IsPostBack)
            {
                // Set Anti-XSRF token
                this.ViewState[AntiXsrfTokenKey] = Page.ViewStateUserKey;
                this.ViewState[AntiXsrfUserNameKey] = Context.User.Identity.Name ?? string.Empty;
            }
            else
            {
                // Validate the Anti-XSRF token
                if ((string)ViewState[AntiXsrfTokenKey] != this.antiXsrfTokenValue
                    || (string)ViewState[AntiXsrfUserNameKey] != (Context.User.Identity.Name ?? string.Empty))
                {
                    throw new InvalidOperationException("Validation of Anti-XSRF token failed.");
                }
            }
        }

        /// <summary>
        /// Page load handler
        /// </summary>
        /// <param name="sender">sender object</param>
        /// <param name="e">event args</param>
        protected void Page_Load(object sender, EventArgs e)
        {
            // Page load logic
        }
    }
}