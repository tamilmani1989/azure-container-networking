// -----------------------------------------------------------------------------
// <copyright file="DatastoreSettings.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    public class DatastoreSettings
    {
        public string Region
        {
            get; set;
        }

        public string SubscriptionId
        {
            get; set;
        }

        public string ResourceGroup
        {
            get; set;
        }

        public string DBAccountName
        {
            get; set;
        }
    }
}
