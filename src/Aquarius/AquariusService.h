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

	void OnStart(DWORD dwArgc, PWSTR *pszArgv) override;
	void OnStop() override;

    void ServiceWorkerThread(void);
	void InitializeServices(void);

private:

    BOOL m_fStopping;
    HANDLE m_hStoppedEvent;	

	wchar_t *defaultPackageStoreLocation = L"C:\\ProgramData\\Microsoft\\Aquarius\\Plugins\\";		
	void GetPackageStoreLocation(wchar_t *packageStorePath, DWORD *size);
	HKEY OpenRegistryKey(HKEY hRootKey, wchar_t* strKey);
	void GetStringFromRegistry(HKEY configKey, wchar_t* key, wchar_t *returnValue, DWORD* size);
	void HandleRegistryError(HKEY rootKey, wchar_t* subKey, LONG nError);
};
