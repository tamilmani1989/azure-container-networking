/****************************** Module Header ******************************\
* Module Name:  AquariusService.h
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
* 
* Class that derives from the service base class AquariusSvcBase. 
* It logs the service start and stop 
* information to the Application event log, and shows how to run the main 
* function of the service in a thread pool worker thread.
* 
\***************************************************************************/

#pragma once

#ifdef _WIN32
#include <thread>
#define _WINSOCK_DEPRECATED_NO_WARNINGS
#endif

#include "AquariusSvcBase.h"
#include "../Kernel/Kernel.h"
#include "../PackMan/AqPackageManager.h"
#include "ConfigParser.h"
#include "HandlerEnvironment.h"
#include <mutex>

#define RESTSERVER_PORT L"4433"
#define HTTP_PROTO L"http"
#define PLUGIN_STATUS L"PluginStatus"
#define LOCALHOST L"127.0.0.1"

#define SERVICE_NAME  L"AquariusSvc"

// Displayed name of the service
#define SERVICE_DISPLAY_NAME     L"Windows Azure Network Agent"

// Service start options.
#define SERVICE_START_TYPE       SERVICE_AUTO_START

// List of service dependencies - "dep1\0dep2\0\0"
#define SERVICE_DEPENDENCIES     L""

// The name of the account under which the service should run
#define SERVICE_ACCOUNT          nullptr

// The password to the service account name
#define SERVICE_PASSWORD         nullptr

#define AQUARIUS_LOG_PREFIX 	L"NetAgent"

//Display name for Aquarius service.
#define AQUARIUS_DISPLAY_NAME L"Azure Aquarius"

//Name of Aquarius service
#define AQUARIUSSVC L"AquariusSvc"

class AquariusService : public AquariusSvcBase
{

public:

	AquariusKernel *aquariusKernel = nullptr;
	AqPackageManager *aqPackageManager = nullptr;
	HandlerEnvironment handlerEnvObject;
	HandlerConfig handlerConfigObject;

    AquariusService(PWSTR pszServiceName, 
        BOOL fCanStop = TRUE, 
        BOOL fCanShutdown = TRUE, 
        BOOL fCanPauseContinue = TRUE);
	~AquariusService(void);
	int GetPluginCount();
	void AquariusServiceLock();
	void AquariusServiceUnLock();
	void OnStart(DWORD dwArgc, PWSTR *pszArgv) override;
	void OnStop() override;
	void DisplayPluginsMetadata(PluginMetadata* allPluginsMetadata, int pluginCount);
	int ProcessConfig(wchar_t *exeLocaiton);
	void ServiceWorkerThread(void);
	void RestServerThreadHandler(void);
	void InitializeServices(void);

private:

	std::mutex service_objects_mutex;
    BOOL m_fStopping;
    HANDLE m_hStoppedEvent[2];	
};
	
AquariusService* GetAquariusService();

