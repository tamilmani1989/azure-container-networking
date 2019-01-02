#pragma once

#include <cassert>
#include <cpprest/http_client.h>
#include <cpprest/filestream.h>
#include <pplx/pplxtasks.h>
#include <iostream>
#include <stdio.h>
#include <string>
#include <thread>
#include <windows.h>
#include <winsock2.h>
#include <Winreg.h>
#include <ws2tcpip.h>
#include <detours.h>

#include "GetInterfaceForContainerResponse.h"

#define REG_KEY          L"SOFTWARE\\Microsoft\\Swift"
#define DEFAULT_CNS_PORT L"10090"
#define DEFAULT_CNS_ADDR L"127.0.0.1"
#define IFACE_QUERY      L"http://%s:%s/network/getinterfaceforcontainer"

class Iface
{
public:
    wstring id;
    wstring ip;

    static Iface* GetInstance()
    {
        static Iface* gm_iface = new Iface();
        return gm_iface;
    }

    void SetID(wstring containerID)
    {
        id = containerID;
    }

    void QueryCNS(int& version);

private:
    set<IPSubnet> cnetSpace;

    Iface()
    {
    }
};
