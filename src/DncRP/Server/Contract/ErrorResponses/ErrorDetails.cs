// -----------------------------------------------------------------------------
// <copyright file="ErrorDetails.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Collections.Generic;
    using Newtonsoft.Json;
    using Microsoft.Azure.ContainerNetworking.CoreLib;

    public class ErrorDefinition
    {
        /// <summary>
        /// Gets service specific error code which serves as the substatus for
        /// the HTTP error code.
        /// </summary>
        [JsonProperty("code")]
        public string Code
        {
            get; set;
        }

        /// <summary>
        /// Gets description of the error.
        /// </summary>
        [JsonProperty("message")]
        public string Message
        {
            get; set;
        }

        /// <summary>
        /// Gets internal error details.
        /// </summary>
        [JsonProperty("details")]
        public IList<ErrorDefinition> Details
        {
            get; set;
        }
    }

    public class ErrorResponse
    {
        /// <summary>
        /// Gets or sets error description
        /// </summary>
        [JsonProperty("error")]
        public ErrorDefinition Error
        {
            get; set;
        }
    }

    public class ErrorDetails
    {
        /// <summary>
        /// Gets or sets error description
        /// </summary>
        [JsonProperty("ErrorResponse")]
        public ErrorResponse ErrorResponse
        {
            get; set;
        }

        /// <summary>
        /// Creates a ARM compatible Json schema with error message and code
        /// </summary>
        /// <param name="topLevelResourceError">topLevelResourceError</param>
        /// <returns></returns>
        public static string CreateErrorAsJson(TopLevelResourceError topLevelResourceError)
        {
            ErrorDetails details = new ErrorDetails
            {
                ErrorResponse = new ErrorResponse
                {
                    Error = new ErrorDefinition
                    {
                        Message = topLevelResourceError.Error + $"InternalErrorResponse: {topLevelResourceError.ErrorResponseCode}",
                        Code = topLevelResourceError.StatusCode.ToString()
                    }
                }
            };

            return JsonUtility.Serialize(details);
        }

        /// <summary>
        /// Creates a ARM compatible Json schema with error message and code
        /// </summary>
        /// <param name="errorMessage"></param>
        /// <param name="code"></param>
        /// <returns></returns>
        public static string CreateErrorAsJson(string errorMessage, int errorCode)
        {
            ErrorDetails details = new ErrorDetails
            {
                ErrorResponse = new ErrorResponse
                {
                    Error = new ErrorDefinition
                    {
                        Message = errorMessage,
                        Code = errorCode.ToString()
                    }
                }
            };

            return JsonUtility.Serialize(details);
        }
    }
}
