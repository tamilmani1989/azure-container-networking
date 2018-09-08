//-----------------------------------------------------------------------
// <copyright file="CreateNetworkContainerResponse.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// 
//-----------------------------------------------------------------------

using System;
using System.Collections.Generic;
using System.Text;

namespace Cns.Data
{
    /// <summary>
    /// Deinfes the create network container response.
    /// </summary>
    public class CreateNetworkContainerResponse : Response
    {
        public CreateNetworkContainerResponse(int returncode, string message) : base(returncode, message)
        {
        }
    }
}
