// --------------------------------------------------------------------------------------------------------------------
// <copyright file="TopLevelResourceError.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    public class TopLevelResourceError
    {
        public int StatusCode
        {
            get; set;
        }

        public string Error
        {
            get; set;
        }
    }
}
