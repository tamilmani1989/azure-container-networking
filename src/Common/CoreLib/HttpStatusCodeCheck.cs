// -----------------------------------------------------------------------------
// <copyright file="HttpStatusCodeCheck.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using System.Net;
    using System.Runtime.CompilerServices;

    public static class HttpStatusCodeCheck
    {
        public static bool IsSuccess(HttpStatusCode statusCode)
        {
            return IsSuccess((int)statusCode);
        }

        public static bool IsSuccess(int statusCode)
        {
            return ((statusCode >= 200) && (statusCode <= 299));
        }

        public static bool IsResourceNotFound(HttpStatusCode statusCode)
        {
            return statusCode == HttpStatusCode.NotFound;
        }

        public static bool IsResourceNotFound(int statusCode)
        {
            return statusCode == (int)HttpStatusCode.NotFound;
        }

        public static bool IsInternalServerError(HttpStatusCode statusCode)
        {
            return statusCode == HttpStatusCode.InternalServerError;
        }

        public static bool IsInternalServerError(int statusCode)
        {
            return statusCode == (int)HttpStatusCode.InternalServerError;
        }
    }
}
