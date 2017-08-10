/****************************** Module Header ******************************\
* Module Name:  SampleService.h
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
#include "../Server/RestServer.h"
#include <mutex>

#define RESTSERVER_PORT L"4433"
#define HTTP_PROTO L"http"
#define PLUGIN_STATUS L"PluginStatus"
#define LOCALHOST L"127.0.0.1"

class AquariusService : public AquariusSvcBase
{
public:
    AquariusService(PWSTR pszServiceName, 
        BOOL fCanStop = TRUE, 
        BOOL fCanShutdown = TRUE, 
        BOOL fCanPauseContinue = TRUE);
	~AquariusService(void);
	int GetPluginCount();
	void AquariusServiceLock();
	void AquariusServiceUnLock();

protected:
	AquariusKernel *aquariusKernel = nullptr;
	AqPackageManager *aqPackageManager = nullptr;	
	void OnStart(DWORD dwArgc, PWSTR *pszArgv) override;
	void OnStop() override;
	void DisplayPluginsMetadata(PluginMetadata* allPluginsMetadata, int pluginCount);
	void ProcessPlugins(wchar_t* pluginDirectory);
	void ServiceWorkerThread(void);
	void RestServerThreadHandler(void);
	void InitializeServices(void);
private:
	std::mutex service_objects_mutex;
    BOOL m_fStopping;
    HANDLE m_hStoppedEvent[2];	
};

AquariusService* GetAquariusService();

