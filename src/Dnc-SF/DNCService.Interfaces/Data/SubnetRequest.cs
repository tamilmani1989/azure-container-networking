//--------------------------------------------------------------------------
// <copyright file="SubnetRequest.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
// Definititon for SubnetRequest.
// </summary>
//--------------------------------------------------------------------------

using System.Runtime.Serialization;
using System.Security.Cryptography;
using System.Text;

namespace DNCService.API
{
    /// <summary>
    /// Class that defines a subnet request.
    /// </summary>
    [DataContract]
    public class SubnetRequest
    {
        /// <summary>
        /// Name of the subnet.
        /// </summary>
        [DataMember]
        public string SubnetName { get; private set; }

        /// <summary>
        /// Access token to be used for delegation.
        /// </summary>
        [DataMember]
        public string AuthenticationToken { get; private set; }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="subnetName">Name of the token.</param>
        /// <param name="authenticationToken">Token for delegation.</param>
        public SubnetRequest(string subnetName, string authenticationToken)
        {
            this.SubnetName = subnetName;
            this.AuthenticationToken = authenticationToken;
        }

        /// <summary>
        /// ToString implementation.
        /// </summary>
        /// <returns>Strong representation of the ToString object.</returns>
        public override string ToString()
        {
            string tokenHashString;
            using (var algorithm = SHA256.Create())
            {
                byte[] tokenHash = algorithm.ComputeHash(Encoding.ASCII.GetBytes(this.AuthenticationToken));
                tokenHashString = System.BitConverter.ToString(tokenHash).Replace("-", "").ToLower();
            }
            return $"SubnetName:{this.SubnetName} AuthenticationToken:{tokenHashString}";
        }
    }
}
