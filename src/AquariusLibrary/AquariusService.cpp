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
#include "../Server/RestServer.h"
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
	wchar_t svcName[MAX_PATH];

    get_aquariusSvc_mutex.lock();
    static AquariusService *service=nullptr;
        

    if(!service)
    {
		if(AquariusUtilities::GetServiceName(svcName, sizeof(svcName)) != 0)
		{
			AquariusUtilities::WriteEventLogEntry(L"NetAgent:: Unable to retrieve service name..");
			wprintf(L"Unable to retrieve service name..");
			return NULL;
		}

        service = new AquariusService(svcName);
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
	LOGINFOW(L"Going to send a reset event to AquariusSvc");
    // Create a manual-reset event that is not signaled at first to indicate 
    // the stopped signal of the service.
    m_hStoppedEvent[AQUARIUS_THREAD] = CreateEvent(nullptr, TRUE, FALSE, nullptr);

    if (m_hStoppedEvent[AQUARIUS_THREAD] == nullptr)
    {
		LOGINFOW(L"Got a null event. Going to throw the last error.");
        throw GetLastError();
    }

	m_hStoppedEvent[RESTSERVER_THREAD] = CreateEvent(nullptr, TRUE, FALSE, nullptr);
    
	if (m_hStoppedEvent[RESTSERVER_THREAD] == nullptr)
    {
		LOGINFOW(L"Got a null event. Going to throw the last error.");
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

void postHandler(http_request message)
{
	UNREFERENCED_PARAMETER(message);
	LOGINFOW(L"Post Handler triggered");
	LOGINFO("Exiting ProcessPost. Returning.");

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
		LOGINFOW(L"Plugin Name %s State %d exitStatus %d blockrole %d.",
			pluginData[i].name, pluginData[i].processState, pluginData[i].exitStatus, pluginData[i].blockRole);

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

	if(handlerConfigObject.restServerPort.empty())
	{
		handlerConfigObject.restServerPort = RESTSERVER_PORT;
	}

	RestServer restServer(HTTP_PROTO, LOCALHOST, handlerConfigObject.restServerPort, PLUGIN_STATUS, postHandler, getHandler); 
	DWORD status = restServer.Listen();

	if (status != 0 )
	{
		LOGINFO("Error in rest server listen %d\n", status);
	}

	LOGINFOW(L"NetAgent RestServer listening on %s", handlerConfigObject.restServerPort.c_str());

	while(!m_fStopping)
	{
		Sleep(5000);
	}

	LOGINFOW(L"RestServerThread exited");
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
	LOGINFOW(L"AquariusSvc is going to start.");
	InitializeServices();
    // Queue the main service function for execution in a worker thread.
    CThreadPool::QueueUserWorkItem(&AquariusService::ServiceWorkerThread, this);
	// std::thread restServerThread(AquariusService::RestServerThreadHandler);
	// restServerThread.detach();

	CThreadPool::QueueUserWorkItem(&AquariusService::RestServerThreadHandler, this);
	LOGINFOW(L"Enqueued the worker");
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
	wchar_t exeLocation[MAX_PATH];
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

	AquariusUtilities::PopulateServiceExeLocation(exeLocation);

	service_objects_mutex.lock();
	int ret = ProcessConfig(exeLocation);
	
	if(ret != 0)
	{
		LOGINFO("ProcessConfig Failed\n");
	}

	service_objects_mutex.unlock();
}

// The method performs the main function of the service. It runs 
// on a thread pool worker thread.
void AquariusService::ServiceWorkerThread(void)
{
	int pluginCount = 0;
	PluginMetadata *allPluginsMetadata = nullptr;
	
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
		}
	}
	catch(...)
	{
		service_objects_mutex.unlock();
		throw;
	}
	
	aquariusKernel->LoadPlugins(pluginCount, allPluginsMetadata);
	service_objects_mutex.unlock();

    while (!m_fStopping)
    {
	LOGINFO("# of allocations that are not deleted yet: %d.",
			AquariusUtilities::allocator);

		try 
		{
			if (aquariusKernel != nullptr)
			{
				LOGINFOW(L"Monitor/Start job objects");
				service_objects_mutex.lock();
				aquariusKernel->MonitorPlugins(pluginCount, allPluginsMetadata, handlerConfigObject.pluginsToStart, handlerEnvObject.logFolderPath);
				service_objects_mutex.unlock();
			}
			// Perform main service function here...
			LOGINFOW(L"AquariusSvc doing work");
		}
		catch(const char *err)
		{
			LOGINFO("Load plugins exception :%s", err);
		}

        ::Sleep(5000);  // Simulate some lengthy operations.
    }

	int cnt = GetPluginCount();

	PluginMetadata *pdata = GetPluginInstances(cnt);
	
	for(int i=0; i < cnt; i++)
	{
		pdata[i].cleanup=true;		
	}	
	
	LOGINFOW(L"Call cleanup aqsvc %p.",
			pdata);
	
	aquariusKernel->MonitorPlugins(pluginCount, allPluginsMetadata, handlerConfigObject.pluginsToStart, handlerEnvObject.logFolderPath);
	Sleep(5000);

	AquariusUtilities::ReleasePluginMetadata(pluginCount, allPluginsMetadata);
	
	LOGINFO("Free up Aquarius Service data objects");

	handlerConfigObject.Cleanup();
	handlerEnvObject.Cleanup();

	PluginMetadata *plgData = GetPluginInstances(GetPluginCount());
	delete[] plgData;

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
	LOGINFOW(L"AquariusSvc in going to stop: OnStop is called");
	service_objects_mutex.lock();


    // Indicate that the service is stopping and wait for the finish of the 
    // main service function (ServiceWorkerThread).
    m_fStopping = TRUE;

	LOGINFO("# of allocations that were nto deleted: %d.",
		AquariusUtilities::allocator);
	
	if (WaitForMultipleObjects(NUM_THREADS, m_hStoppedEvent, TRUE, INFINITE) != WAIT_OBJECT_0)
    {
        DWORD error = GetLastError();
        LOGINFOW(L"Going to throw error  0x%08lx", error);
        throw GetLastError();
    }

	try
	{
		if (aquariusKernel != nullptr)
		{
			LOGINFOW(L"Going to first unload Aquarius kernel");
			delete(aquariusKernel);
			aquariusKernel = nullptr;
			AquariusUtilities::allocator--;
		}

		if (aqPackageManager != nullptr)
		{
			LOGINFOW(L"Going to unload Aquarius PackageManager");
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

	delete this;
}

void AquariusService::DisplayPluginsMetadata(PluginMetadata* allPluginsMetadata, int pluginCount)
{
	LOGINFOW(L"Received %d plugins to process", pluginCount);
	for(int i=0; i<pluginCount; i++)
	{
		LOGINFOW(L"Plugin to process. Location: %s", allPluginsMetadata[i].location);
	}
}


int AquariusService::ProcessConfig(wchar_t *exeLocation)
/*++

Routine Description:
	This function process the config file passed by user and sets the plugins to be started by aquarius

Arguments:
	exeLocation - Location of Aquarius exe
	
Return Value:
	return 0 on success else -1

--*/
{	
	int ret = handlerEnvObject.ReadHandlerEnvironmentFile(exeLocation);
		
	if(ret!=0)
	{
		LOGINFOW(L"ReadHandlerEnvironmentFile function failed for exe location %s\n", exeLocation);
		return -1;
	}

	LOGSETUP(handlerEnvObject.logFolderPath, AQUARIUS_LOG_PREFIX);

	LOGINFOW(L"Config Folder Path: %s Log Folder :%s", handlerEnvObject.configFolderPath, handlerEnvObject.logFolderPath);

	std::string str = handlerConfigObject.ReadLatestConfigFile(handlerEnvObject.configFolderPath);
	
	if(str.empty())
	{
		LOGERRW(L"ReadLatestConfigFile failed. Returned empty string");
		return -1;
	}

	ret = handlerConfigObject.ParseConfigFile(str);

	if(ret!=0)
	{
		LOGINFOW(L"ReadHandlerEnvironmentFile function failed for exe location %s\n", exeLocation);
		return -1;
	}
	
	return 0;
}