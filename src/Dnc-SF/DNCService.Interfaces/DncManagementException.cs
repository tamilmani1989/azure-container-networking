//-----------------------------------------------------------------------
// <copyright file="DncManagementException.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// 
//-----------------------------------------------------------------------

namespace DNCService.API
{
    using System;
    using System.Runtime.Serialization;

    /// <summary>
    /// Base class definition for exceptions thrown by Dnc service.
    /// </summary>
    [DataContract]
    [Serializable]
    [KnownType(typeof(DncNodeManagementException))]
    [KnownType(typeof(DncNetworkManagementException))]
    [KnownType(typeof(DncNetworkContainerException))]
    public class DncManagementException : Exception
    {
        /// <summary>
        /// Error code that identifies the type of error.
        /// </summary>
        [DataMember]
        public int ErrorCode { get; private set; }

        /// <summary>
        /// Status code.
        /// </summary>
        [DataMember]
        public int HTTPStatus { get; set; }

        public DncManagementException(ErrorInfo errorInfo) : base (errorInfo.Message, null)
        {
            this.ErrorCode = (int)errorInfo.ErrorCode;
            this.HTTPStatus = errorInfo.HTTPStatus;
        }

        public override string ToString()
        {
            return $"ErrorCode:{this.ErrorCode} Message:{this.Message}";
        }

        public ErrorInfo ErrorInfo
        {
            get

            {
                return new ErrorInfo((ErrorCode)this.ErrorCode, this.Message, this.HTTPStatus);
            }
        }

        protected DncManagementException(SerializationInfo info, StreamingContext context)
           : base(info, context)
        {
            this.ErrorCode = info.GetInt32(nameof(this.ErrorCode));
            this.HTTPStatus = info.GetInt32(nameof(this.HTTPStatus));
        }

        public override void GetObjectData(SerializationInfo info, StreamingContext context)
        {
            if (info == null)
            {
                throw new ArgumentNullException("info");
            }

            info.AddValue(nameof(this.ErrorCode), this.ErrorCode);
            info.AddValue(nameof(this.HTTPStatus), this.HTTPStatus);

            // MUST call through to the base class to let it save its own state
            base.GetObjectData(info, context);
        }
    }
}