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

class AquariusService : public AquariusSvcBase
{
public:

    AquariusService(PWSTR pszServiceName, 
        BOOL fCanStop = TRUE, 
        BOOL fCanShutdown = TRUE, 
        BOOL fCanPauseContinue = TRUE);
    virtual ~AquariusService(void);

protected:

	void OnStart(DWORD dwArgc, PWSTR *pszArgv) override;
	void OnStop() override;

    void ServiceWorkerThread(void);

private:

    BOOL m_fStopping;
    HANDLE m_hStoppedEvent;
};