#include "injection.h"

static int (WINAPI * TrueBindEx)(SOCKET s, const sockaddr *addr, int namelen) = bind;
static int (WINAPI * TrueConnectEx)(SOCKET s, const sockaddr *addr, int namelen) = connect;
static BOOL (WINAPI * TrueWriteFile)(_In_ HANDLE hFile,
    _In_reads_bytes_opt_(nNumberOfBytesToWrite) LPCVOID lpBuffer,
    _In_ DWORD nNumberOfBytesToWrite,
    _Out_opt_ LPDWORD lpNumberOfBytesWritten,
    _Inout_opt_ LPOVERLAPPED lpOverlapped) = WriteFile;

int WINAPI OurBindEx(SOCKET s, const sockaddr *addr, int namelen)
{
    FILE *file;
    fopen_s(&file, "C:\\Users\\acn\\Downloads\\testinject.txt", "a+");
    fprintf(file, "We hijacked bind!!!\n");
    fclose(file);
    return TrueBindEx(s, addr, namelen);
}

int WINAPI OurConnectEx(SOCKET s, const sockaddr *addr, int namelen)
{
    sockaddr_in localaddr = {0};
    localaddr.sin_family = AF_INET;
    InetPton(AF_INET, L"172.19.1.4", &localaddr.sin_addr.s_addr);
    bind(s, (sockaddr*)&localaddr, sizeof(localaddr));
    return TrueConnectEx(s, addr, namelen);
}

BOOL WINAPI OurWrite(_In_ HANDLE hFile,
    _In_reads_bytes_opt_(nNumberOfBytesToWrite) LPCVOID lpBuffer,
    _In_ DWORD nNumberOfBytesToWrite,
    _Out_opt_ LPDWORD lpNumberOfBytesWritten,
    _Inout_opt_ LPOVERLAPPED lpOverlapped)
{
    FILE *file;
    fopen_s(&file, "C:\\Users\\acn\\Downloads\\testinject.txt", "a+");
    fprintf(file, "We hijacked write!!!\n");
    fclose(file);
    return TrueWriteFile(hFile, lpBuffer, nNumberOfBytesToWrite, lpNumberOfBytesWritten, lpOverlapped);
}

void Runner(std::wstring containerID)
{
    int routingTableVersion(0);
    Iface * iface(Iface::GetInstance());
    iface->SetID(containerID);
    for (;;) {
        iface->QueryCNS(routingTableVersion);
        std::this_thread::sleep_for(std::chrono::minutes(1));
    }
}

void Iface::QueryCNS(int& version)
{
    
    // Construct our CNS URL
    wchar_t buffer[128];
    std::swprintf(buffer, 128, IFACE_QUERY, DEFAULT_CNS_ADDR, DEFAULT_CNS_PORT);
    std::wstring url(buffer);

    // Create a client to CNS addr
    web::http::client::http_client client(url);
    uri_builder dummyBuilder(U(""));

    pplx::task<void> request = client.request(web::http::methods::GET, dummyBuilder.to_string())
    .then([&](web::http::http_response response)
    {
        // Parse GetInterfaceForContainer response and update our routing table with the CnetAddressSpace
        json::value jsonBody(response.extract_json().get());
        GetInterfaceForContainerResponse resp(jsonBody.as_object());
        if ((resp.Response.ReturnCode == 0) &&
            (version < std::stoi(resp.NetworkContainerVersion))) {
            for (auto iter(resp.CnetAddressSpace.begin()); iter != resp.CnetAddressSpace.end(); ++iter) {
                cnetSpace.insert(*iter);
            }
        }
    });

    try {
        request.wait();
    } catch (const std::exception) {
        // catch and ignore
    }
}

BOOL InitThread()
{
    // Retrieve Swift NIC/NC ID from registry.
    LONG error(0);
    WCHAR buffer[128];
    DWORD bufSize(128);
    std::wstring key(REG_KEY);
    ZeroMemory(reinterpret_cast<LPVOID>(buffer), sizeof(buffer));
    error = RegGetValueW(
                HKEY_LOCAL_MACHINE,
                key.c_str(),
                std::to_wstring(GetCurrentProcessId()).c_str(),
                RRF_RT_REG_SZ,
                NULL,
                buffer,
                &bufSize);
    if (error == ERROR_SUCCESS) {
        std::thread t(Runner, std::wstring(buffer));
        t.detach();
    }

    return (error == ERROR_SUCCESS);
}

BOOL WINAPI DllMain(HINSTANCE hinst, DWORD dwReason, LPVOID reserved)
{
    //LONG error;
    (void)hinst;
    (void)reserved;

    if (DetourIsHelperProcess()) {
        return TRUE;
    }

    switch (dwReason) {
        case DLL_PROCESS_ATTACH:
            if (InitThread()) {
                DisableThreadLibraryCalls(hinst);
                DetourRestoreAfterWith();
                DetourTransactionBegin();
                DetourUpdateThread(GetCurrentThread());
                DetourAttach(&(PVOID&)TrueBindEx, OurBindEx);
                DetourAttach(&(PVOID&)TrueConnectEx, OurConnectEx);
                DetourTransactionCommit();
            }

            break;
        case DLL_PROCESS_DETACH:
            DetourTransactionBegin();
            DetourUpdateThread(GetCurrentThread());
            DetourDetach(&(PVOID&)TrueBindEx, OurBindEx);
            DetourAttach(&(PVOID&)TrueConnectEx, OurConnectEx);
            DetourTransactionCommit();
            break;
    }

    return TRUE;
}
