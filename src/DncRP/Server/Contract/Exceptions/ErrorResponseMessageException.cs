//-----------------------------------------------------------------------
// <copyright file="ErrorResponseMessageException.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//-----------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    using System.Net;
    using System.Runtime.Serialization;

    /// <summary>
    /// API response Message exception
    /// </summary>
    public class ErrorResponseMessageException : Exception
    {
        /// <summary>
        /// Gets or sets the http status code.
        /// </summary>
        public HttpStatusCode HttpStatus { get; set; }

        /// <summary>
        /// Gets or sets the error response code.
        /// </summary>
        public ErrorResponseCode ErrorCode { get; set; }

        /// <summary>
        /// Gets or sets the error message
        /// </summary>
        public string ErrorMessage { get; set; }

        /// <summary>
        /// Initializes a new instance of the <see cref="ErrorResponseMessageException" /> class.
        /// </summary>
        /// <param name="httpStatus">The http status code.</param>
        /// <param name="errorCode">The error response code.</param>
        /// <param name="errorMessage">The error response message.</param>
        /// <param name="innerException">Optional. The inner exception.</param>
        /// <param name="headers">Optional. Extra headers to return in error response.</param>
        /// <param name="details">Optional error details.</param>
        public ErrorResponseMessageException(
            HttpStatusCode httpStatus,
            ErrorResponseCode errorCode,
            string errorMessage,
            Exception innerException = null)
            : base(errorMessage, innerException)
        {
            this.HttpStatus = httpStatus;
            this.ErrorCode = errorCode;
            this.ErrorMessage = errorMessage;
        }
    }
}
