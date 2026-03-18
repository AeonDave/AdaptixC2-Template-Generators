// __NAME__ Agent — BOF Loader (stub)
//
// Placeholder for the BOF (Beacon Object File) in-memory COFF loader.
// The full implementation is generated into the output directory.
// This stub declares the types and API surface only.

#pragma once

#ifdef _WIN32
#include <winsock2.h>
#endif
#include <windows.h>
#include <stdint.h>

// ── BOF limits ─────────────────────────────────────────────────────────────────

#define MAX_BOF_SECTIONS    25
#define MAX_BOF_FUNCTIONS   512
#define BOF_GOT_SIZE        4096   // Global Offset Table for import pointers

// ── COFF structures (match PE/COFF spec) ───────────────────────────────────────

#define SIZEOF_FILE_HEADER    20
#define SIZEOF_SECTION_HEADER 40
#define SIZEOF_RELOCATION     10
#define SIZEOF_SYMBOL         18

// Machine types.
#define IMAGE_FILE_MACHINE_I386_  0x14c
#define IMAGE_FILE_MACHINE_AMD64_ 0x8664

// AMD64 relocation types.
#define IMAGE_REL_AMD64_ABSOLUTE 0x0000
#define IMAGE_REL_AMD64_ADDR64   0x0001
#define IMAGE_REL_AMD64_ADDR32NB 0x0003
#define IMAGE_REL_AMD64_REL32    0x0004
#define IMAGE_REL_AMD64_REL32_1  0x0005
#define IMAGE_REL_AMD64_REL32_2  0x0006
#define IMAGE_REL_AMD64_REL32_3  0x0007
#define IMAGE_REL_AMD64_REL32_4  0x0008
#define IMAGE_REL_AMD64_REL32_5  0x0009

// Symbol storage classes.
#define IMAGE_SYM_CLASS_EXTERNAL_ 2
#define IMAGE_SYM_CLASS_STATIC_   3
#define IMAGE_SYM_CLASS_LABEL_    6

// Section characteristics.
#define IMAGE_SCN_CNT_CODE_                0x00000020
#define IMAGE_SCN_CNT_INITIALIZED_DATA_    0x00000040
#define IMAGE_SCN_CNT_UNINITIALIZED_DATA_  0x00000080
#define IMAGE_SCN_MEM_EXECUTE_             0x20000000
#define IMAGE_SCN_MEM_READ_                0x40000000
#define IMAGE_SCN_MEM_WRITE_               0x80000000

#pragma pack(push, 1)

typedef struct {
    uint16_t Machine;
    uint16_t NumberOfSections;
    uint32_t TimeDateStamp;
    uint32_t PointerToSymbolTable;
    uint32_t NumberOfSymbols;
    uint16_t SizeOfOptionalHeader;
    uint16_t Characteristics;
} CoffFileHeader;

typedef struct {
    char     Name[8];
    uint32_t VirtualSize;
    uint32_t VirtualAddress;
    uint32_t SizeOfRawData;
    uint32_t PointerToRawData;
    uint32_t PointerToRelocations;
    uint32_t PointerToLineNumbers;
    uint16_t NumberOfRelocations;
    uint16_t NumberOfLineNumbers;
    uint32_t Characteristics;
} CoffSectionHeader;

typedef struct {
    uint32_t VirtualAddress;
    uint32_t SymbolTableIndex;
    uint16_t Type;
} CoffRelocation;

typedef struct {
    union {
        char     ShortName[8];
        struct {
            uint32_t Zeroes;
            uint32_t Offset;
        };
    };
    uint32_t Value;
    int16_t  SectionNumber;
    uint16_t Type;
    uint8_t  StorageClass;
    uint8_t  NumberOfAuxSymbols;
} CoffSymbol;

#pragma pack(pop)

// ── Section allocation tracker ─────────────────────────────────────────────────

typedef struct {
    void*    address;
    int      size;
    uint32_t characteristics;
} SectionAlloc;

// ── Beacon API hash table entry (for symbol resolution) ───────────────────────

typedef struct {
    ULONG  hash;
    LPVOID proc;
} BOF_API;

// ── BOF output message (single callback entry) ────────────────────────────────

typedef struct {
    int    type;      // CALLBACK_OUTPUT, CALLBACK_ERROR, CALLBACK_AX_*, BOF_ERROR_*
    BYTE*  data;
    int    dataLen;
} BofMsg;

// ── BOF execution context ──────────────────────────────────────────────────────

typedef struct {
    BofMsg* msgs;
    int     msgCount;
    int     msgCapacity;
} BofContext;

// ── Beacon types (CS BOF ABI) ──────────────────────────────────────────────────

typedef struct _DATA_STORE_OBJECT {
    void* ptr;
    int   size;
} DATA_STORE_OBJECT, *PDATA_STORE_OBJECT;

typedef struct _BEACON_INFO {
    UINT  version;
    PVOID sleepMaskPtr;
    UINT  sleepMaskTextSize;
    UINT  sleepMaskTotalSize;
    PVOID beaconPtr;
    PVOID heapRecords;
    BYTE  mask[13];
    BYTE  allocatedMemory[1968];
} BEACON_INFO;

typedef struct _BEACON_SYSCALLS {
    UINT version;
} BEACON_SYSCALLS, *PBEACON_SYSCALLS;

// ── Data parser (matches Cobalt Strike datap) ──────────────────────────────────

typedef struct {
    char*  original;
    char*  buffer;
    int    length;
    int    size;
} datap;

// ── Format buffer (matches Cobalt Strike formatp) ──────────────────────────────

typedef struct {
    char*  original;
    char*  buffer;
    int    length;
    int    size;
} formatp;

// ── BOF Loader API ─────────────────────────────────────────────────────────────

// Execute a COFF object file in-memory.
// Returns a BofContext* with collected output messages, or nullptr on error.
// Caller owns the returned BofContext and must free it with BofContextFree().
BofContext* ObjectExecute(
    unsigned char* coffFile, unsigned int coffFileSize,
    unsigned char* args, int argsSize
);

void BofContextFree(BofContext* ctx);

// Execute a COFF object file asynchronously (runs in a new thread).
// Caller owns the returned BofContext and must free it with BofContextFree().
BofContext* ObjectExecuteAsync(
    unsigned char* coffFile, unsigned int coffFileSize,
    unsigned char* args, int argsSize
);

// Pack BOF arguments according to a format string.
// Format: "z" = null-terminated string, "Z" = wide string,
//         "i" = 4-byte int, "s" = 2-byte short, "b" = binary blob.
// Returns allocated buffer (caller frees), sets outLen.
BYTE* PackArgs(const char* format, int* outLen, ...);

// ── Beacon API Functions ───────────────────────────────────────────────────────

#ifdef __cplusplus
extern "C" {
#endif

// Data Parser
void   BeaconDataParse(datap* parser, char* buffer, int size);
int    BeaconDataInt(datap* parser);
short  BeaconDataShort(datap* parser);
int    BeaconDataLength(datap* parser);
char*  BeaconDataExtract(datap* parser, int* size);
char*  BeaconDataPtr(datap* parser, int size);

// Output
void   BeaconOutput(int type, char* data, int len);
void   BeaconPrintf(int type, char* fmt, ...);

// Format Buffer
void   BeaconFormatAlloc(formatp* format, int maxsz);
void   BeaconFormatReset(formatp* format);
void   BeaconFormatAppend(formatp* format, char* text, int len);
void   BeaconFormatPrintf(formatp* format, char* fmt, ...);
char*  BeaconFormatToString(formatp* format, int* size);
void   BeaconFormatFree(formatp* format);
void   BeaconFormatInt(formatp* format, int value);

// Token
BOOL   BeaconUseToken(HANDLE token);
void   BeaconRevertToken();
BOOL   BeaconIsAdmin();

// Key-Value Store
void   BeaconAddValue(const char* key, void* value);
void*  BeaconGetValue(const char* key);
BOOL   BeaconRemoveValue(const char* key);

// Process / Injection (CS BOF compat)
BOOL   BeaconGetSpawnTo(BOOL x86, char* buffer, int length);
BOOL   BeaconSpawnTemporaryProcess(BOOL x86, BOOL ignoreToken, STARTUPINFOA* sInfo, PROCESS_INFORMATION* pInfo);
void   BeaconInjectProcess(HANDLE hProc, int pid, char* payload, int p_len, int p_offset, char* arg, int a_len);
void   BeaconInjectTemporaryProcess(PROCESS_INFORMATION* pInfo, char* payload, int p_len, int p_offset, char* arg, int a_len);
void   BeaconCleanupProcess(PROCESS_INFORMATION* pInfo);

// Virtual Memory Wrappers (CS 4.10+ BOF compat — stubs)
LPVOID BeaconVirtualAlloc(LPVOID lpAddress, SIZE_T dwSize, DWORD flAllocationType, DWORD flProtect);
LPVOID BeaconVirtualAllocEx(HANDLE hProcess, LPVOID lpAddress, SIZE_T dwSize, DWORD flAllocationType, DWORD flProtect);
BOOL   BeaconVirtualProtect(LPVOID lpAddress, SIZE_T dwSize, DWORD flNewProtect, PDWORD lpflOldProtect);
BOOL   BeaconVirtualProtectEx(HANDLE hProcess, LPVOID lpAddress, SIZE_T dwSize, DWORD flNewProtect, PDWORD lpflOldProtect);
BOOL   BeaconVirtualFree(LPVOID lpAddress, SIZE_T dwSize, DWORD dwFreeType);

// Thread / Process Handle Wrappers (CS 4.10+ BOF compat — stubs)
BOOL   BeaconGetThreadContext(HANDLE hThread, LPCONTEXT lpContext);
BOOL   BeaconSetThreadContext(HANDLE hThread, LPCONTEXT lpContext);
DWORD  BeaconResumeThread(HANDLE hThread);
HANDLE BeaconOpenProcess(DWORD dwDesiredAccess, BOOL bInheritHandle, DWORD dwProcessId);
HANDLE BeaconOpenThread(DWORD dwDesiredAccess, BOOL bInheritHandle, DWORD dwThreadId);
BOOL   BeaconCloseHandle(HANDLE hObject);
BOOL   BeaconUnmapViewOfFile(LPCVOID lpBaseAddress);
SIZE_T BeaconVirtualQuery(LPCVOID lpAddress, PMEMORY_BASIC_INFORMATION lpBuffer, SIZE_T dwLength);
BOOL   BeaconDuplicateHandle(HANDLE hSourceProcessHandle, HANDLE hSourceHandle, HANDLE hTargetProcessHandle, LPHANDLE lpTargetHandle, DWORD dwDesiredAccess, BOOL bInheritHandle, DWORD dwOptions);
BOOL   BeaconReadProcessMemory(HANDLE hProcess, LPCVOID lpBaseAddress, LPVOID lpBuffer, SIZE_T nSize, SIZE_T* lpNumberOfBytesRead);
BOOL   BeaconWriteProcessMemory(HANDLE hProcess, LPVOID lpBaseAddress, LPCVOID lpBuffer, SIZE_T nSize, SIZE_T* lpNumberOfBytesWritten);

// Info / Misc
void   BeaconDownload(const char* filename, const char* buffer, unsigned int length);
BOOL   BeaconInformation(BEACON_INFO* info);
char*  BeaconGetOutputData(int* size);
unsigned int swap_endianess(unsigned int indata);

// Utility
BOOL   toWideChar(char* src, wchar_t* dst, int max);

// Adaptix Extensions (ABI matches official beacon_agent)
void   AxAddScreenshot(char* note, char* data, int len);
void   AxDownloadMemory(char* filename, char* data, int len);

// CS 4.9+ Data Store
PDATA_STORE_OBJECT BeaconDataStoreGetItem(SIZE_T index);
void   BeaconDataStoreProtectItem(SIZE_T index);
void   BeaconDataStoreUnprotectItem(SIZE_T index);
ULONG  BeaconDataStoreMaxEntries();
char*  BeaconGetCustomUserData();

// Async BOF Thread Callbacks (CS 4.9+)
void   BeaconRegisterThreadCallback(PVOID callbackFunction, PVOID callbackData);
void   BeaconUnregisterThreadCallback();
void   BeaconWakeup();
HANDLE BeaconGetStopJobEvent();

// Beacon Gate (CS 4.10+)
void   BeaconDisableBeaconGate();
void   BeaconEnableBeaconGate();
void   BeaconDisableBeaconGateMasking();
void   BeaconEnableBeaconGateMasking();
BOOL   BeaconGetSyscallInformation(PBEACON_SYSCALLS info, BOOL resolveIfNotInitialized);

#ifdef __cplusplus
}
#endif