// -----------------------------------------------------------------------------
// <copyright file="AmeCertMetadata.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    using System;
    public class RootsInfos
    {
        public string rootName
        {
            get; set;
        }

        public string CaName
        {
            get; set;
        }

        public string Cdp
        {
            get; set;
        }

        public string Apptype
        {
            get; set;
        }

        public string StoreLocation
        {
            get; set;
        }

        public string StoreName
        {
            get; set;
        }

        public string Body
        {
            get; set;
        }

        public string PEM
        {
            get; set;
        }

        public Intermediates[] Intermediates
        {
            get; set;
        }
    }

    public class Intermediates
    {        
        public string IntermediateName
        {
            get; set;
        }

        public string Apptype
        {
            get; set;
        }

        public string Cdp
        {
            get; set;
        }

        public string StoreLocation
        {
            get; set;
        }

        public string StoreName
        {
            get; set;
        }

        public string Body
        {
            get; set;
        }

        public string PEM
        {
            get; set;
        }
    }

    public class AmeCertMetadata
    {
        public RootsInfos[] RootsInfos
        {
            get; set;
        }
    }
}

