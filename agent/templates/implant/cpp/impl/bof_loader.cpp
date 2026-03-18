// __NAME__ Agent — BOF Loader (stub)
//
// Placeholder for the BOF (Beacon Object File) in-memory COFF loader.
// The full implementation is generated into the output directory.
// This stub provides the minimum compilable surface:
//   - ObjectExecute returning nullptr (not implemented)
//   - BofContextFree
//   - All Beacon API functions as no-ops

#include "bof_loader.h"
#include "protocol.h"

#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <cstdarg>

// ── BofContext management ──────────────────────────────────────────────────────

void BofContextFree(BofContext* ctx) {
    if (!ctx) return;
    for (int i = 0; i < ctx->msgCount; i++) {
        free(ctx->msgs[i].data);
    }
    free(ctx->msgs);
    free(ctx);
}

// ── COFF Loader (stub) ────────────────────────────────────────────────────────

BofContext* ObjectExecute(
    unsigned char* coffFile, unsigned int coffFileSize,
    unsigned char* args, int argsSize
) {
    (void)coffFile; (void)coffFileSize;
    (void)args; (void)argsSize;

    // TODO: Implement the full COFF loading pipeline.
    // See the output implementation for the reference.
    return nullptr;
}
BofContext* ObjectExecuteAsync(
    unsigned char* coffFile, unsigned int coffFileSize,
    unsigned char* args, int argsSize
) {
    (void)coffFile; (void)coffFileSize;
    (void)args; (void)argsSize;

    // TODO: Implement async variant — execute in a new thread,
    // stream output via BofContext.
    return nullptr;
}

BYTE* PackArgs(const char* format, int* outLen, ...) {
    (void)format;
    if (outLen) *outLen = 0;
    // TODO: Implement format-string argument packer.
    // Format: "z" = null-terminated string, "Z" = wide string,
    //         "i" = 4-byte int, "s" = 2-byte short, "b" = binary blob.
    return nullptr;
}
// ── Beacon API Function Stubs ──────────────────────────────────────────────────

void BeaconOutput(int type, char* data, int len) {
    (void)type; (void)data; (void)len;
}

void BeaconPrintf(int type, char* fmt, ...) {
    (void)type; (void)fmt;
}

void BeaconDataParse(datap* parser, char* buffer, int size) {
    (void)parser; (void)buffer; (void)size;
}

int BeaconDataInt(datap* parser) {
    (void)parser; return 0;
}

short BeaconDataShort(datap* parser) {
    (void)parser; return 0;
}

int BeaconDataLength(datap* parser) {
    (void)parser; return 0;
}

char* BeaconDataExtract(datap* parser, int* size) {
    (void)parser; if (size) *size = 0; return nullptr;
}

void BeaconFormatAlloc(formatp* format, int maxsz) {
    (void)format; (void)maxsz;
}

void BeaconFormatReset(formatp* format) {
    (void)format;
}

void BeaconFormatAppend(formatp* format, char* text, int len) {
    (void)format; (void)text; (void)len;
}

void BeaconFormatPrintf(formatp* format, char* fmt, ...) {
    (void)format; (void)fmt;
}

char* BeaconFormatToString(formatp* format, int* size) {
    (void)format; if (size) *size = 0; return nullptr;
}

void BeaconFormatFree(formatp* format) {
    (void)format;
}

void BeaconFormatInt(formatp* format, int value) {
    (void)format; (void)value;
}

BOOL BeaconUseToken(HANDLE token) {
    (void)token; return FALSE;
}

void BeaconRevertToken() {
}

BOOL BeaconIsAdmin() {
    return FALSE;
}

void BeaconAddValue(const char* key, void* value) {
    (void)key; (void)value;
}

void* BeaconGetValue(const char* key) {
    (void)key; return nullptr;
}

BOOL BeaconRemoveValue(const char* key) {
    (void)key; return FALSE;
}

char* BeaconDataPtr(datap* parser, int size) {
    (void)parser; (void)size; return nullptr;
}

BOOL BeaconGetSpawnTo(BOOL x86, char* buffer, int length) {
    (void)x86; (void)buffer; (void)length; return FALSE;
}

BOOL BeaconSpawnTemporaryProcess(BOOL x86, BOOL ignoreToken, STARTUPINFOA* sInfo, PROCESS_INFORMATION* pInfo) {
    (void)x86; (void)ignoreToken; (void)sInfo; (void)pInfo; return FALSE;
}

void BeaconInjectProcess(HANDLE hProc, int pid, char* payload, int p_len, int p_offset, char* arg, int a_len) {
    (void)hProc; (void)pid; (void)payload; (void)p_len; (void)p_offset; (void)arg; (void)a_len;
}

void BeaconInjectTemporaryProcess(PROCESS_INFORMATION* pInfo, char* payload, int p_len, int p_offset, char* arg, int a_len) {
    (void)pInfo; (void)payload; (void)p_len; (void)p_offset; (void)arg; (void)a_len;
}

void BeaconCleanupProcess(PROCESS_INFORMATION* pInfo) { (void)pInfo; }

LPVOID BeaconVirtualAlloc(LPVOID a, SIZE_T s, DWORD t, DWORD p) { (void)a;(void)s;(void)t;(void)p; return nullptr; }
LPVOID BeaconVirtualAllocEx(HANDLE h, LPVOID a, SIZE_T s, DWORD t, DWORD p) { (void)h;(void)a;(void)s;(void)t;(void)p; return nullptr; }
BOOL   BeaconVirtualProtect(LPVOID a, SIZE_T s, DWORD n, PDWORD o) { (void)a;(void)s;(void)n;(void)o; return FALSE; }
BOOL   BeaconVirtualProtectEx(HANDLE h, LPVOID a, SIZE_T s, DWORD n, PDWORD o) { (void)h;(void)a;(void)s;(void)n;(void)o; return FALSE; }
BOOL   BeaconVirtualFree(LPVOID a, SIZE_T s, DWORD t) { (void)a;(void)s;(void)t; return FALSE; }

BOOL   BeaconGetThreadContext(HANDLE h, LPCONTEXT c) { (void)h;(void)c; return FALSE; }
BOOL   BeaconSetThreadContext(HANDLE h, LPCONTEXT c) { (void)h;(void)c; return FALSE; }
DWORD  BeaconResumeThread(HANDLE h) { (void)h; return (DWORD)-1; }
HANDLE BeaconOpenProcess(DWORD a, BOOL i, DWORD p) { (void)a;(void)i;(void)p; return nullptr; }
HANDLE BeaconOpenThread(DWORD a, BOOL i, DWORD t) { (void)a;(void)i;(void)t; return nullptr; }
BOOL   BeaconCloseHandle(HANDLE h) { (void)h; return FALSE; }
BOOL   BeaconUnmapViewOfFile(LPCVOID a) { (void)a; return FALSE; }
SIZE_T BeaconVirtualQuery(LPCVOID a, PMEMORY_BASIC_INFORMATION b, SIZE_T l) { (void)a;(void)b;(void)l; return 0; }
BOOL   BeaconDuplicateHandle(HANDLE sp, HANDLE sh, HANDLE tp, LPHANDLE th, DWORD a, BOOL i, DWORD o) { (void)sp;(void)sh;(void)tp;(void)th;(void)a;(void)i;(void)o; return FALSE; }
BOOL   BeaconReadProcessMemory(HANDLE h, LPCVOID b, LPVOID buf, SIZE_T n, SIZE_T* r) { (void)h;(void)b;(void)buf;(void)n;(void)r; return FALSE; }
BOOL   BeaconWriteProcessMemory(HANDLE h, LPVOID b, LPCVOID buf, SIZE_T n, SIZE_T* w) { (void)h;(void)b;(void)buf;(void)n;(void)w; return FALSE; }

void BeaconDownload(const char* filename, const char* buffer, unsigned int length) {
    (void)filename; (void)buffer; (void)length;
}

BOOL BeaconInformation(BEACON_INFO* info) {
    (void)info;
    return FALSE;
}

char* BeaconGetOutputData(int* size) {
    if (size) *size = 0;
    return nullptr;
}

unsigned int swap_endianess(unsigned int indata) {
    return ((indata >> 24) & 0xFF) | ((indata >> 8) & 0xFF00) |
           ((indata << 8) & 0xFF0000) | ((indata << 24) & 0xFF000000);
}

void AxAddScreenshot(char* note, char* data, int len) {
    (void)note; (void)data; (void)len;
}

void AxDownloadMemory(char* filename, char* data, int len) {
    (void)filename; (void)data; (void)len;
}

BOOL toWideChar(char* src, wchar_t* dst, int max) {
    (void)src; (void)dst; (void)max; return FALSE;
}

// ── CS 4.9+ Data Store (stubs) ───────────────────────────────────────────────────

PDATA_STORE_OBJECT BeaconDataStoreGetItem(SIZE_T index) { (void)index; return nullptr; }
void  BeaconDataStoreProtectItem(SIZE_T index) { (void)index; }
void  BeaconDataStoreUnprotectItem(SIZE_T index) { (void)index; }
ULONG BeaconDataStoreMaxEntries() { return 0; }
char* BeaconGetCustomUserData() { return nullptr; }

// ── Async BOF Thread Callbacks (stubs — CS 4.9+) ────────────────────────────────

void   BeaconRegisterThreadCallback(PVOID callbackFunction, PVOID callbackData) { (void)callbackFunction; (void)callbackData; }
void   BeaconUnregisterThreadCallback() {}
void   BeaconWakeup() {}
HANDLE BeaconGetStopJobEvent() { return nullptr; }

// ── Beacon Gate (stubs — CS 4.10+) ──────────────────────────────────────────────

void  BeaconDisableBeaconGate() {}
void  BeaconEnableBeaconGate() {}
void  BeaconDisableBeaconGateMasking() {}
void  BeaconEnableBeaconGateMasking() {}
BOOL  BeaconGetSyscallInformation(PBEACON_SYSCALLS info, BOOL resolveIfNotInitialized) {
    (void)info; (void)resolveIfNotInitialized; return FALSE;
}