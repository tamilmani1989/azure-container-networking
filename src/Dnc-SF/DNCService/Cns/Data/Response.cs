//-----------------------------------------------------------------------
// <copyright file="Response.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// 
//-----------------------------------------------------------------------

using Newtonsoft.Json;

namespace Cns.Data
{
    /// <summary>
    /// Defines the response.
    /// </summary>
    public class Response
    {
        /// <summary>
        /// Return code.
        /// </summary>
        [JsonProperty("ReturnCode")]
        public int ReturnCode { get; private set; }

        /// <summary>
        /// Message string.
        /// </summary>
        [JsonProperty("Message")]
        public string Message { get; private set; }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="returncode"></param>
        /// <param name="message"></param>
        public Response(int returncode, string message)
        {
            this.ReturnCode = returncode;
            this.Message = message;
        }

        /// <summary>
        /// ToString implementation
        /// </summary>
        /// <returns>String representation of the object.</returns>
        public override string ToString()
        {
            return $"[ReturnCode:{this.ReturnCode} Message:{this.Message}]";
        }

    }
}