#include <cassert>
#include <functional>
#include <iostream>
#include <memory>
#include <stdio.h>
#include <string>
#include <Windows.h>
#include <Winreg.h>

#define BUFSIZE       4096
#define INJECTION_DLL "Injection.dll"
#define REG_KEY    L"SOFTWARE\\Microsoft\\Swift"

typedef DWORD pid_t;
typedef LONG (NTAPI *NtSuspendProcess)(IN HANDLE ProcessHandle);
typedef LONG (NTAPI *NtResumeProcess)(IN HANDLE ProcessHandle);

struct process
{
    HANDLE handle;

    process(pid_t pid)
    {
        HANDLE processHandle(OpenProcess(PROCESS_ALL_ACCESS, FALSE, pid));
        if (processHandle) {
            handle = processHandle;
        } else {
            throw std::invalid_argument("pid");
        }
    }

    ~process()
    {
        CloseHandle(handle);
    }

    template <typename T>
    using process_unique_ptr = std::unique_ptr<T, std::function<void(T*)>>;

    template <typename T>
    T* allocate(size_t sz)
    {
        auto ptr(VirtualAllocEx(
            handle,
            nullptr,
            sz,
            MEM_COMMIT | MEM_RESERVE,
            PAGE_READWRITE));
        if (ptr == NULL) {
            throw std::bad_alloc();
        }

        return (T*)ptr;
    }

    template <typename T>
    process_unique_ptr<T> allocate_unique(size_t sz)
    {
        return process_unique_ptr<T>(allocate<T>(sz), std::bind(&process::free, this, std::placeholders::_1));
    }

    void suspend()
    {
        auto pfnNtSuspendProcess = (NtSuspendProcess) GetProcAddress(GetModuleHandleA("ntdll"),"NtSuspendProcess");
        assert(pfnNtSuspendProcess != NULL);
        pfnNtSuspendProcess(handle);
    }

    void resume()
    {
        auto pfnNtResumeProcess = (NtResumeProcess) GetProcAddress(GetModuleHandleA("ntdll"),"NtResumeProcess");
        assert(pfnNtResumeProcess != NULL);
        pfnNtResumeProcess(handle);
    }

    void writeMemory(void *ptr, const void *data, size_t sz)
    {
        if (!WriteProcessMemory(handle, ptr, data, sz, nullptr)) {
            throw std::system_error(std::error_code(GetLastError(), std::system_category()), "Exception occurred");
        }
    }

    void free(void *ptr)
    {
        VirtualFreeEx(handle, ptr, 0, MEM_RELEASE);
    }

    operator HANDLE() const
    {
        return handle;
    }
};

void showUsage()
{
    std::cout << "Usage: Hijack.exe pid container_id" << std::endl;
}

void inject(pid_t pid, std::string dll)
{
    process p(pid);
    p.suspend();
    LPTHREAD_START_ROUTINE pfnThreadRtn((LPTHREAD_START_ROUTINE)GetProcAddress(
        GetModuleHandle(TEXT("Kernel32")),
        "LoadLibraryA"));
    assert(pfnThreadRtn != NULL);
    using char_type = decltype(dll)::value_type;
    const size_t dll_size_bytes = (dll.size() + 1) * sizeof(char_type);
    auto ptr = p.allocate_unique<char_type>(dll_size_bytes);
    p.writeMemory(ptr.get(), dll.c_str(), dll_size_bytes);

    HANDLE remote_thread(CreateRemoteThread(
        p,
        NULL,
        0,
        pfnThreadRtn,
        ptr.get(),
        0,
        nullptr));

    WaitForSingleObject(remote_thread, INFINITE);
    p.resume();
}

void setRegistryKV(pid_t pid, std::string containerID)
{
    std::wstring pidstr(std::to_wstring(pid));
    std::wstring key(REG_KEY);
    std::wstring wcontainerID(containerID.begin(), containerID.end());
    LSTATUS result = RegSetKeyValueW(
                        HKEY_LOCAL_MACHINE,
                        key.c_str(),
                        pidstr.c_str(),
                        REG_SZ,
                        (LPBYTE)wcontainerID.c_str(),
                        (DWORD)(wcontainerID.size() + 1) * sizeof(wchar_t));
    if (result != ERROR_SUCCESS) {
        throw std::system_error(std::error_code(result, std::system_category()), "Exception occurred");
    }
}

int main(int argc, char* argv[])
{
    if (argc != 3) {
        showUsage();
        exit(EXIT_FAILURE);
    }

    // Parse cmd params.
    pid_t pid(std::stoul(argv[1]));
    std::string containerID(argv[2]);

    // Set registry KV pair with PID and ContainerID.
    setRegistryKV(pid, containerID);

    // Load kernel32.dll so that we can inject pid with LoadLibraryA which 
    // will load our Injection.dll into the remote process.
    CHAR dllPath[BUFSIZE];
    GetFullPathNameA(INJECTION_DLL, BUFSIZE, dllPath, NULL);
    inject(pid, std::string(dllPath));

    return 0;
}