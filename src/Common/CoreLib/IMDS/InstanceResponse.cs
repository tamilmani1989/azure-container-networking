// -----------------------------------------------------------------------------
// <copyright file="ImdsResponse.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    public class ComputeResponse
    {
        public string azEnvironment
        {
            get; set;
        }
    }

    /// <summary>
    /// Object type that is returned when following IMDS API in invoked
    /// http://169.254.169.254/metadata/instance?api-version=2019-08-15&format=json
    /// </summary>
    public class InstanceResponse
    {
        public ComputeResponse compute
        {
            get; set;
        }
    }
}
