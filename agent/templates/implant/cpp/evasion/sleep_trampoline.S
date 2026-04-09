// __NAME__ Agent — Sleep Trampoline Stub
//
// TODO: Implement your PIC sleep trampoline here.
//
// This file is a placeholder for a position-independent code (PIC) blob
// that runs from an allocated RX page outside .text to safely perform
// sleep obfuscation (e.g. encrypt .text, sleep, decrypt .text).
//
// Typical approach:
//   1. Copy this blob to a separate RX page during init
//   2. During sleep, call the copy (since .text will be encrypted)
//   3. The blob uses direct SYSCALL instructions (no DLL calls)
//
// Example flow:
//   - NtProtectVirtualMemory(.text → RW)
//   - XOR/encrypt .text region
//   - NtDelayExecution(sleep duration)
//   - XOR/decrypt .text region
//   - NtProtectVirtualMemory(.text → RX)
//
// All jumps/calls must be PC-relative for position independence.
// The context struct pointer and sleep duration are passed via Win64 ABI
// (RCX, RDX).

.intel_syntax noprefix

.text

// Export symbols so RecycleGate can measure the blob size and copy it.
.globl sleep_trampoline
sleep_trampoline:
    // TODO: implement your sleep obfuscation PIC blob
    ret

.globl sleep_trampoline_end
sleep_trampoline_end:
