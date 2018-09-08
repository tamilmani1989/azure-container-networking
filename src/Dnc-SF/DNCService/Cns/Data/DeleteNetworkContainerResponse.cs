//-----------------------------------------------------------------------
// <copyright file="DeleteNetworkContainerResponse.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// 
//-----------------------------------------------------------------------

namespace Cns.Data
{
    /// <summary>
    /// Defines the delete container response,
    /// </summary>
    public class DeleteNetworkContainerResponse : Response
    {
        public DeleteNetworkContainerResponse(int returncode, string message) : base(returncode, message)
        {
        }
    }
}
