// __NAME__ Agent — TCP Connector (Stub)
//
// Concrete Connector implementation for raw TCP transport.
// TODO: Implement all methods.

#pragma once

#include "Connector.h"

class ConnectorTCP : public Connector
{
private:
    BYTE* recvBuffer = nullptr;
    int   recvLen    = 0;
    WORD  port       = 0;

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
