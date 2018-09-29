//--------------------------------------------------------------------------
// <copyright file="ErrorInfo.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
// Definititon for error info class.
// </summary>
//--------------------------------------------------------------------------

using System;
using System.Net;
using System.Runtime.Serialization;

namespace DNCService.API
{
    /// <summary>
    /// Class thay defines the error info that will be 
    /// returned by Dnc service as part of the exception.
    /// </summary>
    [DataContract]
    public class ErrorInfo : ICloneable
    {
        /// <summary>
        /// Error code that identifies the type of error.
        /// </summary>
        [DataMember]
        public ErrorCode ErrorCode { get; private set; }

        /// <summary>
        /// Message that describes the error.
        /// </summary>
        [DataMember]
        public string Message { get; private set; }

        /// <summary>
        /// Status code.
        /// </summary>
        [DataMember]
        public int HTTPStatus { get; set; }

        /// <summary>
        /// Type of error
        /// </summary>
        [DataMember]
        public ErrorType ErrorType { get; set; }


        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="errorCode">Supply the error code.</param>
        /// <param name="message">Supply the message.</param>
        /// <param name="errorType">Supply the error type.</param>
        public ErrorInfo(
            ErrorCode errorCode, 
            string message,
            ErrorType errorType): this(errorCode, message, (int) HttpStatusCode.BadRequest, errorType)
        {
        }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="errorCode">Supply the error code.</param>
        /// <param name="message">Supply the message.</param>
        /// <param name="httpStatus">Http status.</param>
        /// <param name="errorType">Type of error.</param>
        public ErrorInfo(ErrorCode errorCode, string message, int httpStatus, ErrorType errorType)
        {
            this.ErrorCode = errorCode;
            this.Message = message;
            this.HTTPStatus = httpStatus;
            this.ErrorType = errorType;
        }

        /// <summary>
        /// Copy constructor.
        /// </summary>
        /// <param name="errorInfo">Supply the error info object.</param>
        public ErrorInfo(ErrorInfo errorInfo) : this(
            errorInfo.ErrorCode, 
            errorInfo.Message,
            errorInfo.HTTPStatus,
            errorInfo.ErrorType)
        {
        }

        /// <summary>
        /// ToString implementation.
        /// </summary>
        /// <returns>String representation of the error info object.</returns>
        public override string ToString()
        {
            return $"ErrorCode:{Enum.GetName(typeof(ErrorCode), this.ErrorCode)}," +
                $"Message:{this.Message}, HttpStatus:{this.HTTPStatus}," +
                $"ErrorType:{Enum.GetName(typeof(ErrorType), this.ErrorType)}";
        }

        /// <summary>
        /// Clone implementation.
        /// </summary>
        /// <returns>Cloned object.</returns>
        public object Clone()
        {
            return new ErrorInfo(this.ErrorCode, this.Message, this.ErrorType);
        }
    }
}
