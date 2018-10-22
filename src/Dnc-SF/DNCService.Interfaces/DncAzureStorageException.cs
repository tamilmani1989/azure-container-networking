//-----------------------------------------------------------------------
// <copyright file="DncAzureStorageException.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
//-----------------------------------------------------------------------

using System;
using System.Runtime.Serialization;

namespace DNCService.API
{
    /// <summary>
    /// Class defintition for Azure storage exception.
    /// </summary>
    [DataContract]
    [Serializable]
    public class DncAzureStorageException : DncManagementException, ISerializable
    {
        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="errorInfo">Supply the error Info to be associated with th exception.</param>
        public DncAzureStorageException(ErrorInfo errorInfo) : base(errorInfo)
        {
        }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="info">Supply the serialization info.</param>
        /// <param name="context">Supply the streaming context.</param>
        protected DncAzureStorageException(SerializationInfo info, StreamingContext context)
            : base(info, context)
        {
        }

        /// <summary>
        /// Implements the get object method used for serialization.
        /// </summary>
        /// <param name="info">Supply the serialization info.</param>
        /// <param name="context">Supply the streaming context.</param>
        public override void GetObjectData(SerializationInfo info, StreamingContext context)
        {
            if (info == null)
            {
                throw new ArgumentNullException("info");
            }

            // MUST call through to the base class to let it save its own state
            base.GetObjectData(info, context);
        }

        /// <summary>
        /// ToString implementation.
        /// </summary>
        /// <returns>ToString representation of the DncAzureStorageException object.</returns>
        public override string ToString()
        {
            return $"Error:{base.ToString()}";
        }
    }
}
