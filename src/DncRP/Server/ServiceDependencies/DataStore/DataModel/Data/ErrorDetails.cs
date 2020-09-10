// -----------------------------------------------------------------------------
// <copyright file="ErrorDetails.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System.Collections.Generic;
    using System.Text.Json;
    using System.Text.Json.Serialization;

    public class ErrorDefinition
    {
        /// <summary>
        /// Gets service specific error code which serves as the substatus for
        /// the HTTP error code.
        /// </summary>
        [JsonPropertyName("code")]
        public string Code
        {
            get; set;
        }

        /// <summary>
        /// Gets description of the error.
        /// </summary>
        [JsonPropertyName("message")]
        public string Message
        {
            get; set;
        }

        /// <summary>
        /// Gets internal error details.
        /// </summary>
        [JsonPropertyName("details")]
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
        [JsonPropertyName("error")]
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
        [JsonPropertyName("ErrorResponse")]
        public ErrorResponse ErrorResponse
        {
            get; set;
        }

        /// <summary>
        /// Creates a ARM compatible Json schema with error message and code
        /// </summary>
        /// <param name="errorMessage"></param>
        /// <param name="code"></param>
        /// <returns></returns>
        public static string CreateErrorAsJson(string errorMessage, int code)
        {
            ErrorDetails details = new ErrorDetails
            {
                ErrorResponse = new ErrorResponse
                {
                    Error = new ErrorDefinition
                    {
                        Message = errorMessage,
                        Code = code.ToString()
                    }
                }
            };

            return JsonSerializer.Serialize(details);
        }
    }
}
