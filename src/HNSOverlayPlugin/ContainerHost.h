// ***********************************************************************
// <copyright file="ContainerHost.h" company="Microsoft">
//     Copyright (c) Microsoft Corporation. All rights reserved.
// </copyright>
// <summary></summary>
// ***********************************************************************
#pragma once

#ifndef __PC_CONTAINERHOST_H__
#define __PC_CONTAINERHOST_H__

#include <sstream>

#include "hnsPluginApi.h"
#include "ovPluginException.h"
#include <wil\Resource.h>
#include <wil\Com.h>

namespace Microsoft { namespace Azure { namespace Networking { namespace OverlayPlugin
{
    typedef wil::unique_cotaskmem_string PCStringPtr;
    typedef wil::com_ptr<IHNSObjectList> HNSObjListPtr;
    typedef wil::com_ptr<IHNSObject> HNSObjPtr;

    class ContainerHost
    {
    public:
        /// <summary>Initializes this instance of container host.</summary>
        ContainerHost() {};

        // prevent copying
        ContainerHost(_In_ const ContainerHost& right) = delete;
        ContainerHost& operator=(const ContainerHost& right) = delete;

        ~ContainerHost() throw()
        {}
        /// <summary>Handles HNS network notifications.</summary>
        /// <param name='PluginProvider'>HNS provider object.</param>
        /// <param name='RequestObject'>HNS network object.</param>
        /// <param name='createNetwork'>Indicates the request type.</param>
        void HandleNetworkNotification(
            _Inout_ HNSProviderPtr PluginProvider,
            _Inout_ HNSObjPtr RequestObject,
            _In_ bool createNetwork
            );
		/// <summary>Handles HNS endpoint pre create notifications.</summary>
		/// <param name='PluginProvider'>HNS provider object.</param>
		/// <param name='RequestObject'>HNS endpoint object.</param>
		void HandleEndpointPreCreate(
			_Inout_ HNSProviderPtr PluginProvider,
			_Inout_ HNSObjPtr RequestObject
		);
		/// <summary>Handles HNS endpoint post create notifications.</summary>
		/// <param name='PluginProvider'>HNS provider object.</param>
		/// <param name='RequestObject'>HNS endpoint object.</param>
		void HandleEndpointPostCreate(
			_Inout_ HNSProviderPtr PluginProvider,
			_Inout_ HNSObjPtr RequestObject
		);
		/// <summary>Handles HNS endpoint post delete notifications.</summary>
		/// <param name='PluginProvider'>HNS provider object.</param>
		/// <param name='RequestObject'>HNS endpoint object.</param>
		void HandleEndpointPostDelete(
			_Inout_ HNSProviderPtr PluginProvider,
			_Inout_ HNSObjPtr RequestObject
		);
    };
}}}}

#endif /* __PC_CONTAINERHOST_H__ */
