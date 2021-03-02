// --------------------------------------------------------------------------------------------------------------------
// <copyright file="SubnetJoinRequest.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    public class SubnetJoinRequest
    {
        public string AuthenticationToken;
 
        public SubnetJoinRequest (string authToken)
        {
            this.AuthenticationToken = authToken;
        }
    }
}
