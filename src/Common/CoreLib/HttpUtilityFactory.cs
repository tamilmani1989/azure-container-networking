// -----------------------------------------------------------------------------
// <copyright file="HttpUtilityFactory.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------


namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
	using System.Collections.Generic;
    using System;

    public class HttpUtilityFactory: IHttpUtilityFactory
    {
        private Dictionary<string, IHttpUtility> httpUtilities;

        public HttpUtilityFactory()
        {
            httpUtilities = new Dictionary<string, IHttpUtility>();
        }

        public void Register(string name, IHttpUtility httpUtility)
        {
            httpUtilities[name] = httpUtility;
        }

        public IHttpUtility Resolve(string name)
        {
            if (!httpUtilities.ContainsKey(name))
            {
                throw new Exception("httputility with this name not registered:" + name);
            }

            return httpUtilities[name];
        }
    }
}
