//--------------------------------------------------------------------------
// <copyright file="RequestStatus.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
// Definititon for RequestStatus class.
// </summary>
//--------------------------------------------------------------------------

using System.Runtime.Serialization;

namespace DNCService.API
{
    /// <summary>
    /// Defines the request class.
    /// </summary>
    [DataContract]
    public class RequestStatus
    {
        /// <summary>
        /// State of the request.
        /// </summary>
        [DataMember]
        public string State { get; private set; }

        /// <summary>
        /// Status of the request.
        /// </summary>
        [DataMember]
        public string Status { get; private set; }

        /// <summary>
        /// Optional. Error information if the request has failed.
        /// </summary>
        [DataMember]
        public ErrorInfo ErrorInfo { get; private set; }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="state">Supply the state.</param>
        /// <param name="status">Supply the status.</param>
        /// <param name="errorInfo">Supply the error info.</param>
        public RequestStatus(string state, string status, ErrorInfo errorInfo)
        {
            this.State = state;
            this.Status = status;
            this.ErrorInfo = errorInfo;
        }

        /// <summary>
        /// ToString implementation.
        /// </summary>
        /// <returns>String representation of the RequestStatus object.</returns>
        public override string ToString()
        {
            return $"State:{this.State} Status:{this.Status} ErrorInfo:{this.ErrorInfo}";
        }
    }
}
