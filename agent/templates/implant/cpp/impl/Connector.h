// __NAME__ Agent — Abstract Connector Interface
//
// All transport connectors must inherit from this class and implement
// the pure virtual methods. Modeled on beacon_agent's Connector pattern.
//
// To add a new transport (HTTP, DNS, SMB, etc.), create a new class
// that inherits Connector and implements all pure virtual methods.

#pragma once

#ifdef _WIN32
#include <winsock2.h>
#endif
#include <windows.h>
#include <stdlib.h>

class Connector
{
public:
    // Set the connection profile extracted from config blob.
    virtual BOOL SetProfile(void* profile, BYTE* beat, ULONG beatSize) = 0;

    // Wait for a connection opportunity (e.g. named pipe accept).
    virtual BOOL WaitForConnection() { return TRUE; }

    // Check if currently connected.
    virtual BOOL IsConnected() { return TRUE; }

    // Disconnect current session.
    virtual void Disconnect() {}

    // Exchange data: send plainData (encrypted with sessionKey), receive response.
    virtual void Exchange(BYTE* plainData, ULONG plainSize, BYTE* sessionKey) = 0;

    // Access received data after Exchange().
    virtual BYTE* RecvData() = 0;
    virtual int   RecvSize() = 0;
    virtual void  RecvClear() = 0;

    // Sleep between check-ins. Override for custom sleep behavior.
    virtual void Sleep(ULONG sleepMs, ULONG jitter)
    {
        ULONG ms = sleepMs;
        if (jitter > 0 && jitter <= 90) {
            int pct = (rand() % (jitter * 2 + 1)) - (int)jitter;
            int adj = (int)sleepMs * pct / 100;
            int total = (int)sleepMs + adj;
            if (total < 0) total = 0;
            ms = (ULONG)total;
        }
        ::Sleep(ms);
    }

    // Cleanup resources.
    virtual void CloseConnector() = 0;

    virtual ~Connector() {}
};
