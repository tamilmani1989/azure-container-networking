// -----------------------------------------------------------------------------
// <copyright file="AmeCertValidationSettings.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    public class AmeCertValidationSettings
    {
        public string AmeAllowedSubjectNames
        {
            get; set;
        }

        public bool SkipAmeCertValidation
        {
            get; set;
        }
    }
}
