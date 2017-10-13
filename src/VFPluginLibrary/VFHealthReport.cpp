#include <cpprest/http_listener.h>
#include <cpprest/http_client.h>
#include <cpprest/uri.h>
#include <cpprest/asyncrt_utils.h>
#include <cpprest/json.h>
#include "..\Logging\Logging.h"
#include "VFHealthReport.h"
#include "VFPluginUtilities.h"

using namespace std;
using namespace web;
using namespace http;
using namespace web::http::client;
using namespace experimental::listener;
using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;

void HealthReport::Reset()
{
    GuestOS.assign(L"Windows");
    DriverInstallAttempted = false;
    DriverInstallSucceeded = false;
    DriverInstallFailed = false;
    NumNonWorkingDevices = 0;
    TotalDevices = 0;
}

void HealthReport::SendHealthStatusToHostAgent(
    wchar_t *url
)
{
    http_client client(url);
    http_request requester;
    json::value obj;

    wchar_t *attempted = DriverInstallAttempted?L"True":L"False";
    wchar_t *succeeded = DriverInstallSucceeded?L"True":L"False";
    wchar_t *failed = DriverInstallFailed?L"True":L"False";
    wchar_t count[32];


    if(swprintf_s(count, sizeof(count), L"%d", NumNonWorkingDevices) < 0)
    {
        LOGERRW(L"swprintf_s failed while converting num non working count to str");
    }

    requester.set_method(methods::POST);
    requester.headers().set_content_type(U("application/json"));

    obj[L"GuestOSFamily"] = json::value::string(GuestOS);
	obj[L"GuestOSVersion"] = json::value::string(OsVersion);
	obj[L"DriverInstallAttempted"] = json::value::string(attempted);
	obj[L"DriverInstallSucceeded"] = json::value::string(succeeded);
	obj[L"DriverInstallFailed"] = json::value::string(failed);

    
    if(swprintf_s(count, sizeof(count), L"%d", NumNonWorkingDevices) < 0)
    {
        LOGERRW(L"swprintf_s failed while converting num non working count to str");
    }
    obj[L"NonFunctioningCount"] = json::value::string(count);


    if(swprintf_s(count, sizeof(count), L"%d", TotalDevices) < 0)
    {
        LOGERRW(L"swprintf_s failed while converting num non working count to str");
    }
    obj[L"TotalDevices"] = json::value::string(count);
    
    requester.set_body(obj);   

     pplx::task<void> requestTask = client.request(requester).then([](http_response response)
    {
        if(response.status_code() == status_codes::OK)
        {
            // auto body = response.extract_string();
            // std::wcout << L"Added new Id: " << body.get().c_str() << std::endl;
            LOGINFOW(L"Health Report status sent success");
        }
        else
        {
            LOGERRW(L"Error:: HTTP post status code %d", response.status_code());
        }
    });
    
    try
    {
        requestTask.wait();
    }
    catch (const std::exception &e)
    {
        LOGINFO("Error exception:%s\n", e.what());
    }
}