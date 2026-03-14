// __NAME__ Agent — TCP Connector Implementation (Stub)
//
// TODO: Implement TCP socket communication.
// See beacon_agent ConnectorTCP.cpp for reference implementation.

#include "ConnectorTCP.h"
#include <winsock2.h>

#pragma comment(lib, "ws2_32.lib")

ConnectorTCP::ConnectorTCP()
{
    // TODO: Initialize Winsock, resolve function pointers if using dynamic API
}

ConnectorTCP::~ConnectorTCP()
{
    CloseConnector();
}

BOOL ConnectorTCP::SetProfile(void* profile, BYTE* beat, ULONG beatSize)
{
    // TODO: Extract host/port from profile struct, store beat data
    return FALSE;
}

void ConnectorTCP::Exchange(BYTE* plainData, ULONG plainSize, BYTE* sessionKey)
{
    // TODO: Encrypt plainData with sessionKey, send, receive response
}

BYTE* ConnectorTCP::RecvData()
{
    return recvBuffer;
}

int ConnectorTCP::RecvSize()
{
    return recvLen;
}

void ConnectorTCP::RecvClear()
{
    if (recvBuffer) {
        LocalFree(recvBuffer);
        recvBuffer = nullptr;
    }
    recvLen = 0;
}

void ConnectorTCP::CloseConnector()
{
    RecvClear();
    // TODO: Close socket, cleanup Winsock
}
