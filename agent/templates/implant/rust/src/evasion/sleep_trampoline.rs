// __NAME__ Agent — PIC Sleep Trampoline (stub)
//
// TODO: Implement your own position-independent sleep obfuscation blob.
//
// Typical approach:
//   1. NtProtectVirtualMemory → PAGE_READWRITE
//   2. Encrypt .text (and tracked heap regions)
//   3. NtDelayExecution (sleep)
//   4. Decrypt .text (and tracked heap regions)
//   5. NtProtectVirtualMemory → PAGE_EXECUTE_READ
//
// The blob is compiled into .text via global_asm!(), then copied to a
// separate RX page at init time so it survives .text encryption.

#![allow(bad_asm_style)]

use core::arch::global_asm;

global_asm!(
    ".intel_syntax noprefix",

    ".globl sleep_trampoline",
    "sleep_trampoline:",
    "ret",

    ".globl sleep_trampoline_end",
    "sleep_trampoline_end:",

    ".att_syntax prefix",
);

extern "C" {
    pub fn sleep_trampoline();
    fn sleep_trampoline_end();
}

/// Returns (start_address, byte_size) of the trampoline blob.
pub fn trampoline_blob() -> (usize, usize) {
    let start = sleep_trampoline as *const () as usize;
    let end = sleep_trampoline_end as *const () as usize;
    (start, end - start)
}
