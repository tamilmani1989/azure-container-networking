//-----------------------------------------------------------------------
// <copyright file="InvalidServicePortException.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//-----------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.Gateway
{
    using System;
    using System.Runtime.Serialization;

    /// <summary>
    /// Exception that should be thrown when driver is initialized twice.
    /// </summary>
    [Serializable]
    public class InvalidServicePortException : Exception
    {
        /// <summary>
        /// Initializes a new instance of the <see cref="InvalidServicePortException"/> class.
        /// </summary>
        public InvalidServicePortException()
        {
        }

        /// <summary>
        /// Initializes a new instance of the <see cref="InvalidServicePortException"/> class
        /// using a custom message.
        /// </summary>
        /// <param name="message">Human-readable message indicating the source of the problem</param>
        public InvalidServicePortException(string message)
            : base(message)
        {
        }

        /// <summary>
        /// Initializes a new instance of the <see cref="InvalidServicePortException"/> class with serialized data.
        /// </summary>
        /// <param name="si">
        /// The SerializationInfo that holds the serialized object data about the exception being thrown.
        /// </param>
        /// <param name="sc">
        /// The StreamingContext that contains contextual information about the source or destination.
        /// </param>
        protected InvalidServicePortException(SerializationInfo si, StreamingContext sc)
            : base(si, sc)
        {
        }
    }
}
