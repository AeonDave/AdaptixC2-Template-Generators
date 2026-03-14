// __NAME__ Agent — Command Dispatcher
//
// Routes incoming command IDs to handler methods.
// Command IDs are defined in protocol/protocol.h.

#pragma once

#include <windows.h>

class Agent;

class Commander
{
public:
    Agent* agent;

    Commander(Agent* agent);
    ~Commander();

    // Process a buffer of packed command tasks
    void ProcessCommandTasks(BYTE* recv, ULONG recvSize, BYTE* outBuf, ULONG* outSize);

    // Command handlers
    void CmdTerminate(ULONG cmdId, BYTE* data, ULONG dataSize);
    void CmdFsList(ULONG cmdId, BYTE* data, ULONG dataSize);
    void CmdFsUpload(ULONG cmdId, BYTE* data, ULONG dataSize);
    void CmdFsDownload(ULONG cmdId, BYTE* data, ULONG dataSize);
    void CmdFsRemove(ULONG cmdId, BYTE* data, ULONG dataSize);
    void CmdFsMkdirs(ULONG cmdId, BYTE* data, ULONG dataSize);
    void CmdFsCopy(ULONG cmdId, BYTE* data, ULONG dataSize);
    void CmdFsMove(ULONG cmdId, BYTE* data, ULONG dataSize);
    void CmdFsCd(ULONG cmdId, BYTE* data, ULONG dataSize);
    void CmdFsPwd(ULONG cmdId, BYTE* data, ULONG dataSize);
    void CmdFsCat(ULONG cmdId, BYTE* data, ULONG dataSize);
    void CmdOsRun(ULONG cmdId, BYTE* data, ULONG dataSize);
    void CmdOsInfo(ULONG cmdId, BYTE* data, ULONG dataSize);
    void CmdOsPs(ULONG cmdId, BYTE* data, ULONG dataSize);
    void CmdOsScreenshot(ULONG cmdId, BYTE* data, ULONG dataSize);
    void CmdOsShell(ULONG cmdId, BYTE* data, ULONG dataSize);
    void CmdOsKill(ULONG cmdId, BYTE* data, ULONG dataSize);
    void CmdProfileSleep(ULONG cmdId, BYTE* data, ULONG dataSize);
    void CmdProfileKilldate(ULONG cmdId, BYTE* data, ULONG dataSize);
    void CmdProfileWorktime(ULONG cmdId, BYTE* data, ULONG dataSize);
    void CmdExecBof(ULONG cmdId, BYTE* data, ULONG dataSize);
    void CmdJobList(ULONG cmdId, BYTE* data, ULONG dataSize);
    void CmdJobKill(ULONG cmdId, BYTE* data, ULONG dataSize);
};
