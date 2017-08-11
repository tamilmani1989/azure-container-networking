/****************************** Module Header ******************************\
* Module Name:  AquariusService.cpp
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
* 
* Provides a sample service class that derives from the service base class - 
* AquariusSvcBase. The sample service logs the service start and stop 
* information to the Application event log, and shows how to run the main 
* function of the service in a thread pool worker thread.
* 
\***************************************************************************/

#pragma region Includes
#include "AquariusService.h"
#include "ThreadPool.h"
#include <strsafe.h>
#include "../Shared/Utilities.h"
#include "../Kernel/JobManager.h"
#include "../Logging/Logging.h"
#pragma endregion

using namespace Microsoft::Azure::Networking::Aqua::Server;
using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

#define AQUARIUS_THREAD 0
#define RESTSERVER_THREAD 1
#define NUM_THREADS 2

std::mutex get_aquariusSvc_mutex;

AquariusService* GetAquariusService()
/*++

Routine Description:
	This function returns the Aquarius Service object if it exists otherwise creates a new Aquarius Service object

Arguments:

Return Value:
	returns Aquarius Service object on successful case else returns NULL

--*/
{
    get_aquariusSvc_mutex.lock();
    static AquariusService *service=nullptr;

    if(!service)
    {
        service = new AquariusService(SERVICE_NAME);
    }

    get_aquariusSvc_mutex.unlock();
    return service;
}

PluginMetadata* GetPluginInstances(int count)
/*++

Routine Description:
	This function returns plugin instances if exist otherwise creates plugin instances based on count

Arguments:
	count - number of plugins

Return Value:
	returns plugin instances pointer if succeeds else returns NULL

--*/
{
	static PluginMetadata *instances = nullptr;

	if(!instances &&  count > 0)
	{
		instances = new PluginMetadata[count];
		
		if(instances != nullptr)
		{
			memset(instances, 0, count*(sizeof(PluginMetadata)));
		}
		LOGINFO("Creating new plugin metadata objects, count :%d\n", count)
	}

	return instances;
}

AquariusService::AquariusService(PWSTR pszServiceName, 
                               BOOL fCanStop, 
                               BOOL fCanShutdown, 
                               BOOL fCanPauseContinue)
: AquariusSvcBase(pszServiceName, fCanStop, fCanShutdown, fCanPauseContinue)
/*++

Routine Description:
	Aquarius Service constructor to intialize aquarius service object. This function also creates event to communicate
	with service worker thread and rest server thread. 

Arguments:
	pszServiceName - service name

Return Value:

--*/
{
    m_fStopping = FALSE;
	AquariusUtilities::WriteEventLogEntry(L"Going to send a reset event to AquariusSvc");
    // Create a manual-reset event that is not signaled at first to indicate 
    // the stopped signal of the service.
    m_hStoppedEvent[AQUARIUS_THREAD] = CreateEvent(nullptr, TRUE, FALSE, nullptr);

    if (m_hStoppedEvent[AQUARIUS_THREAD] == nullptr)
    {
		AquariusUtilities::WriteEventLogEntry(L"Got a null event. Going to throw the last error.");
        throw GetLastError();
    }

	m_hStoppedEvent[RESTSERVER_THREAD] = CreateEvent(nullptr, TRUE, FALSE, nullptr);
    
	if (m_hStoppedEvent[RESTSERVER_THREAD] == nullptr)
    {
		AquariusUtilities::WriteEventLogEntry(L"Got a null event. Going to throw the last error.");
        throw GetLastError();
}

}

AquariusService::~AquariusService(void)
/*++

Routine Description:
	Aquarius Service destructor. Closes the event handle created during initialization.

Arguments:

Return Value:

--*/
{
    if (m_hStoppedEvent[AQUARIUS_THREAD])
    {
        CloseHandle(m_hStoppedEvent[AQUARIUS_THREAD]);
        m_hStoppedEvent[AQUARIUS_THREAD] = nullptr;
    }

	if (m_hStoppedEvent[RESTSERVER_THREAD])
    {
        CloseHandle(m_hStoppedEvent[RESTSERVER_THREAD]);
        m_hStoppedEvent[RESTSERVER_THREAD] = nullptr;
}

	PluginMetadata *plgData = GetPluginInstances(GetPluginCount());
	delete[] plgData;
	AquariusUtilities::allocator--;
}

int AquariusService::GetPluginCount()
/*++

Routine Description:
	This function returns number of plugins

Arguments:

Return Value:
	returns number of plugins if suceeds else returns -1

--*/
{
	if (aqPackageManager)
	{
		return aqPackageManager->GetPluginCount();
	}

	return -1;
}

void AquariusService::AquariusServiceLock()
/*++

Routine Description:
	This function obtains lock for aquarius service

Arguments:

Return Value:

--*/
{
	service_objects_mutex.lock();
}

void AquariusService::AquariusServiceUnLock()
/*++

Routine Description:
	This function frees lock for aquarius service

Arguments:

Return Value:

--*/
{

	service_objects_mutex.unlock();
}

void postHandler(http_request request)
{
	AquariusUtilities::WriteEventLogEntry(L"Post Handler triggered");
	UNREFERENCED_PARAMETER(request);
}

void getHandler(http_request request)
/*++

Routine Description:
	GetHandler for RestServer. This handler parses the plugin metadata object and sends the state and exit status of all plugins.

Arguments:
	request - http request received

Return Value:

--*/
{
	LOGINFO("Get Handler triggerd");
	AquariusService *aqSvc = GetAquariusService();
	
	if(!aqSvc)
	{
		LOGINFO("Aquarius Service not initialized\n");
		request.reply(status_codes::status_codes::InternalError, nullptr);		
	}
	
	aqSvc->AquariusServiceLock();
	int pluginCount = aqSvc->GetPluginCount();
	
	if(pluginCount==-1)
	{
		LOGINFO("Aquarius Package Manager not initialized\n");
		request.reply(status_codes::status_codes::InternalError, nullptr);
	}

	LOGINFO("Plugincount:%d\n",pluginCount);	
	PluginMetadata *pluginData = GetPluginInstances(pluginCount);
	
	if(pluginData == NULL)
	{
		LOGINFO("GetInsatances for plugin data returns null\n");
		request.reply(status_codes::status_codes::InternalError, nullptr);
	}

	int i=0;
	json::value obj;
	
	for(i=0;i<pluginCount;i++)
	{		
		wchar_t szMessage[MAX_PATH];

		StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"Plugin Name %s State %d exitStatus %d blockrole %d.",
			pluginData[i].name, pluginData[i].processState, pluginData[i].exitStatus, pluginData[i].blockRole);
		AquariusUtilities::WriteEventLogEntry(szMessage);

		json::value pluginStatus;
		pluginStatus[L"state"] = pluginData[i].processState;
	
		if (pluginData[i].blockRole == 1)
		{
			pluginStatus[L"exitStatus"] = pluginData[i].exitStatus;
		}
		else
		{
			pluginStatus[L"exitStatus"] = 0;
		}

		obj[pluginData[i].name] = pluginStatus;
	}
	
	aqSvc->AquariusServiceUnLock();
	request.reply(status_codes::OK, obj);
}

void AquariusService::RestServerThreadHandler()
/*++

Routine Description:
	Rest Server thread handler. This function creates a rest server on localhost and listens on RESTSERVER_PORT. 

Arguments:

Return Value:

--*/
{
	LOGINFO("RestServerThread Created!!");	
	RestServer restServer(HTTP_PROTO, LOCALHOST, RESTSERVER_PORT, PLUGIN_STATUS, postHandler, getHandler); 
	DWORD status = restServer.Listen();

	if (status != 0 )
	{
		LOGINFO("Error in rest server listen %d\n", status);
	}

	while(!m_fStopping)
	{
		Sleep(5000);
	}

	AquariusUtilities::WriteEventLogEntry(L"RestServerThread exited");
    SetEvent(m_hStoppedEvent[RESTSERVER_THREAD]);
}

//
//   FUNCTION: AquariusService::OnStart(DWORD, LPWSTR *)
//
//   PURPOSE: The function is executed when a Start command is sent to the 
//   service by the SCM or when the operating system starts (for a service 
//   that starts automatically). It specifies actions to take when the 
//   service starts.
//
void AquariusService::OnStart(DWORD dwArgc, LPWSTR *lpszArgv)
{
	UNREFERENCED_PARAMETER(dwArgc);
	UNREFERENCED_PARAMETER(lpszArgv);
    // Log a service start message to the Application log.
	AquariusUtilities::WriteEventLogEntry(L"AquariusSvc is going to start.");
	InitializeServices();
    // Queue the main service function for execution in a worker thread.
    CThreadPool::QueueUserWorkItem(&AquariusService::ServiceWorkerThread, this);
	// std::thread restServerThread(AquariusService::RestServerThreadHandler);
	// restServerThread.detach();

	CThreadPool::QueueUserWorkItem(&AquariusService::RestServerThreadHandler, this);
	AquariusUtilities::WriteEventLogEntry(L"Enqueued the worker");
}

void AquariusService::InitializeServices(void)
/*++

Routine Description:
	This function initializes aquarius package manager and kernel

Arguments:

Return Value:

--*/
{
	// check if initialization is needed
	// take initialization lock
	wchar_t packageStoreLocation[MAX_PATH];
	DWORD size = 1024;
	
	if(aqPackageManager == nullptr)
	{
		aqPackageManager = new AqPackageManager();
		AquariusUtilities::allocator++;
	}
	AquariusUtilities::SetRelativePathFromExe(L"\\Plugins");
	aqPackageManager->GetPackageStoreLocation(packageStoreLocation, &size);

	if (aquariusKernel == nullptr)
	{		
		aquariusKernel = new AquariusKernel(packageStoreLocation);
		AquariusUtilities::allocator++;
	}
}

// The method performs the main function of the service. It runs 
// on a thread pool worker thread.
void AquariusService::ServiceWorkerThread(void)
{
    // Periodically check if the service is stopping.
	wchar_t szMessage[MAX_PATH];

    while (!m_fStopping)
    {
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"# of allocations that are not deleted yet: %d.",
			AquariusUtilities::allocator);
		AquariusUtilities::WriteEventLogEntry(szMessage);

		try 
		{
			if (aquariusKernel != nullptr)
			{
				AquariusUtilities::WriteEventLogEntry(L"Aquarius Kernel is loaded");
				wchar_t* pluginDirectory = nullptr; 
				ProcessPlugins(pluginDirectory);
			}
			// Perform main service function here...
			AquariusUtilities::WriteEventLogEntry(L"AquariusSvc doing work");
		}
		catch(const char *err)
		{
			AquariusUtilities::WriteErrorLogEntry((PWSTR)(err));
		}

        ::Sleep(5000);  // Simulate some lengthy operations.
    }

    // Signal the stopped event.
    SetEvent(m_hStoppedEvent[AQUARIUS_THREAD]);
}

void AquariusService::OnStop()
/*++

Routine Description:
	This function cleans up aquarius service and waits for service thread and rest server thread to exit.

Arguments:

Return Value:

--*/
{
    // Log a service stop message to the Application log.
	AquariusUtilities::WriteEventLogEntry(L"AquariusSvc in going to stop: OnStop is called");
	service_objects_mutex.lock();

	try
	{
		if (aquariusKernel != nullptr)
		{
			AquariusUtilities::WriteEventLogEntry(L"Going to first unload Aquarius kernel");
			delete(aquariusKernel);
			aquariusKernel = nullptr;
			AquariusUtilities::allocator--;
		}

		if (aqPackageManager != nullptr)
		{
			AquariusUtilities::WriteEventLogEntry(L"Going to unload Aquarius PackageManager");
			delete(aqPackageManager);
			AquariusUtilities::allocator--;
			aqPackageManager = nullptr;
		}
	}
	catch(...)
	{
		service_objects_mutex.unlock();
		throw;
	}

	service_objects_mutex.unlock();

    // Indicate that the service is stopping and wait for the finish of the 
    // main service function (ServiceWorkerThread).
    m_fStopping = TRUE;

	wchar_t szMessage[MAX_PATH];
	StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"# of allocations that were nto deleted: %d.",
		AquariusUtilities::allocator);
	AquariusUtilities::WriteEventLogEntry(szMessage);

    if (WaitForMultipleObjects(NUM_THREADS, m_hStoppedEvent, TRUE, INFINITE) != WAIT_OBJECT_0)
    {
        DWORD error = GetLastError();
        StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"Going to throw error  0x%08lx", error);
        AquariusUtilities::WriteEventLogEntry(szMessage);
        throw GetLastError();
    }
}

void AquariusService::DisplayPluginsMetadata(PluginMetadata* allPluginsMetadata, int pluginCount)
{
	wchar_t szMessage[MAX_PATH];
	StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"Received %d plugins to process", pluginCount);
	AquariusUtilities::WriteEventLogEntry(szMessage);
	for(int i=0; i<pluginCount; i++)
	{
		StringCchPrintf(szMessage, ARRAYSIZE(szMessage), L"Plugin to process. Location: %s", allPluginsMetadata[i].location);
		AquariusUtilities::WriteEventLogEntry(szMessage);
	}
}

void AquariusService::ProcessPlugins(wchar_t* pluginDirectory)
/*++

Routine Description:
	This function populates plugin metadata object and calls load plugin.

Arguments:
	directory - plugin directory
	
Return Value:

--*/
{
	UNREFERENCED_PARAMETER(pluginDirectory);
	int pluginCount = 0;
	PluginMetadata *allPluginsMetadata = nullptr;
	// Get the goal state: 
	// i.e., Get the plugin versions that should be current from Package Manager

	service_objects_mutex.lock();
	
	try 
	{
		if (aqPackageManager != nullptr)
		{
			if(!allPluginsMetadata)
			{
			 	pluginCount = aqPackageManager->RetrievePluginsMetadata(&allPluginsMetadata);
			}
			else
			{
				pluginCount = aqPackageManager->GetPluginCount();	 
			}

			aquariusKernel->LoadPlugins(pluginCount, allPluginsMetadata);
			AquariusUtilities::ReleasePluginMetadata(pluginCount, allPluginsMetadata);
			// Forward these to Kernel (retries are okay)
		}
	}
	catch(...)
	{
		service_objects_mutex.unlock();
		throw;
	}

	service_objects_mutex.unlock();
}
