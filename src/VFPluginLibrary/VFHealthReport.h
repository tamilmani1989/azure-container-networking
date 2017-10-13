#include <Windows.h>
#include<iostream>

using namespace std;

class HealthReport
{
public:
    wstring GuestOS;
    wstring OsVersion;
    bool DriverInstallAttempted;
    bool DriverInstallSucceeded;
    bool DriverInstallFailed;
    int NumNonWorkingDevices;
    int TotalDevices;

    void SendHealthStatusToHostAgent(
        wchar_t *url
    );   

    void Reset(); 
};