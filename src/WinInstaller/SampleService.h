/****************************** Module Header ******************************\
* Module Name:  SampleService.h
* Project:      WinInstaller
* Copyright (c) Microsoft Corporation.
* 
* Class that derives from the service base class CServiceBase. 
* It logs the service start and stop 
* information to the Application event log, and shows how to run the main 
* function of the service in a thread pool worker thread.
* 
\***************************************************************************/

#pragma once

#include "ServiceBase.h"


class CSampleService : public CServiceBase
{
public:

    CSampleService(PWSTR pszServiceName, 
        BOOL fCanStop = TRUE, 
        BOOL fCanShutdown = TRUE, 
        BOOL fCanPauseContinue = FALSE);
    virtual ~CSampleService(void);

protected:

    virtual void OnStart(DWORD dwArgc, PWSTR *pszArgv);
    virtual void OnStop();

    void ServiceWorkerThread(void);

private:

    BOOL m_fStopping;
    HANDLE m_hStoppedEvent;
};