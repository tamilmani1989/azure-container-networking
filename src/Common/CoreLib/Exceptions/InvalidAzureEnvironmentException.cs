//-----------------------------------------------------------------------
// <copyright file="InvalidAzureEnvironmentException.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//-----------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using System;
    using System.Runtime.Serialization;

    /// <summary>
    /// Exception that should be thrown when driver is initialized twice.
    /// </summary>
    [Serializable]
    public class InvalidAzureEnvironmentException : Exception
    {
        /// <summary>
        /// Initializes a new instance of the <see cref="InvalidAzureEnvironmentException"/> class.
        /// </summary>
        public InvalidAzureEnvironmentException()
        {
        }

        /// <summary>
        /// Initializes a new instance of the <see cref="InvalidAzureEnvironmentException"/> class
        /// using a custom message.
        /// </summary>
        /// <param name="message">Human-readable message indicating the source of the problem</param>
        public InvalidAzureEnvironmentException(string message)
            : base(message)
        {
        }

        /// <summary>
        /// Initializes a new instance of the <see cref="InvalidAzureEnvironmentException"/> class with serialized data.
        /// </summary>
        /// <param name="si">
        /// The SerializationInfo that holds the serialized object data about the exception being thrown.
        /// </param>
        /// <param name="sc">
        /// The StreamingContext that contains contextual information about the source or destination.
        /// </param>
        protected InvalidAzureEnvironmentException(SerializationInfo si, StreamingContext sc)
            : base(si, sc)
        {
        }
    }
}
