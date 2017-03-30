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

#include "AquariusSvcBase.h"
#include "../Kernel/Kernel.h"
#include "../PackMan/AqPackageManager.h"
#include <mutex>

class AquariusService : public AquariusSvcBase
{
public:

    AquariusService(PWSTR pszServiceName, 
        BOOL fCanStop = TRUE, 
        BOOL fCanShutdown = TRUE, 
        BOOL fCanPauseContinue = TRUE);

	~AquariusService(void);
protected:
	AquariusKernel *aquariusKernel = nullptr;
	AqPackageManager *aqPackageManager = nullptr;

	void OnStart(DWORD dwArgc, PWSTR *pszArgv) override;
	void OnStop() override;
	void DisplayPluginsMetadata(PluginMetadata* allPluginsMetadata, int pluginCount);
	void ProcessPlugins(wchar_t* pluginDirectory);
	void ServiceWorkerThread(void);
	void InitializeServices(void);

private:
	std::mutex service_objects_mutex;
    BOOL m_fStopping;
    HANDLE m_hStoppedEvent;	
	
};
