// __NAME__ Agent — BOF Loader Implementation
//
// Stub implementation of the COFF loader and Beacon API functions.
// Fill in the TODOs to enable in-memory BOF execution.
//
// Reference: beacon_agent/src_beacon/beacon/bof_loader.cpp
//            beacon_agent/src_beacon/beacon/beacon_functions.cpp

#include "bof_loader.h"
#include <cstdlib>
#include <cstring>

// ── BofOutput helpers ──────────────────────────────────────────────────────────

static BofOutput* BofOutputAlloc(ULONG initialCapacity) {
    auto* out = (BofOutput*)calloc(1, sizeof(BofOutput));
    if (!out) return nullptr;
    out->buffer   = (BYTE*)malloc(initialCapacity);
    out->size     = 0;
    out->capacity = initialCapacity;
    return out;
}

void BofOutputFree(BofOutput* output) {
    if (output) {
        free(output->buffer);
        free(output);
    }
}

// ── COFF Loader ────────────────────────────────────────────────────────────────

BofOutput* ObjectExecute(
    ULONG taskId,
    unsigned char* coffFile, unsigned int coffFileSize,
    unsigned char* args, int argsSize
) {
    (void)taskId;
    (void)coffFile;
    (void)coffFileSize;
    (void)args;
    (void)argsSize;

    // TODO: Implement the full COFF loading pipeline:
    //
    // 1. Parse COFF headers (FileHeader, SectionHeaders, Symbols, StringTable)
    //
    // 2. Allocate executable memory for each section:
    //      VirtualAlloc(NULL, sectionSize, MEM_COMMIT | MEM_RESERVE,
    //                   PAGE_EXECUTE_READWRITE)
    //
    // 3. Copy raw section data into allocated memory
    //
    // 4. Process relocations for each section:
    //    - IMAGE_REL_AMD64_ADDR64:  *(uint64_t*)target += symbolAddr
    //    - IMAGE_REL_AMD64_ADDR32NB: *(uint32_t*)target += (uint32_t)(symbolAddr - targetAddr)
    //    - IMAGE_REL_AMD64_REL32:   *(uint32_t*)target += (uint32_t)(symbolAddr - targetAddr - 4)
    //
    // 5. Resolve external symbols:
    //    - "__imp_" prefix → LoadLibraryA + GetProcAddress
    //    - Beacon API functions → match against registered function table
    //
    // 6. Find entry point symbol ("go" or "_go")
    //
    // 7. Cast entry to void(*)(char*, int) and call with (args, argsSize)
    //
    // 8. Collect output from BeaconOutput/BeaconPrintf callbacks
    //
    // 9. Clean up: VirtualFree all sections
    //
    // See beacon_agent bof_loader.cpp for the full reference implementation.

    return nullptr;
}

// ── Beacon API Function Stubs ──────────────────────────────────────────────────
// Each function below is called by BOFs via the resolved symbol table.
// Implement them to match the Beacon API specification.

// BeaconOutput — write typed output from a BOF
// type: CALLBACK_OUTPUT, CALLBACK_OUTPUT_OEM, CALLBACK_OUTPUT_UTF8, CALLBACK_ERROR
void BeaconOutput(int type, char* data, int len) {
    // TODO: Append {type, data[0..len]} to the current task's BofOutput
    (void)type; (void)data; (void)len;
}

// BeaconPrintf — formatted output from a BOF
void BeaconPrintf(int type, char* fmt, ...) {
    // TODO: vsnprintf + BeaconOutput
    (void)type; (void)fmt;
}

// DataParser API stubs
void  BeaconDataParse(void* parser, char* buffer, int size)   { (void)parser; (void)buffer; (void)size; }
int   BeaconDataInt(void* parser)                              { (void)parser; return 0; }
short BeaconDataShort(void* parser)                            { (void)parser; return 0; }
int   BeaconDataLength(void* parser)                           { (void)parser; return 0; }
char* BeaconDataExtract(void* parser, int* size)               { (void)parser; if(size) *size = 0; return nullptr; }

// Format API stubs
void  BeaconFormatAlloc(void* format, int maxsz)              { (void)format; (void)maxsz; }
void  BeaconFormatReset(void* format)                          { (void)format; }
void  BeaconFormatAppend(void* format, char* text, int len)   { (void)format; (void)text; (void)len; }
void  BeaconFormatPrintf(void* format, char* fmt, ...)        { (void)format; (void)fmt; }
char* BeaconFormatToString(void* format, int* size)            { (void)format; if(size) *size = 0; return nullptr; }
void  BeaconFormatFree(void* format)                           { (void)format; }
void  BeaconFormatInt(void* format, int value)                 { (void)format; (void)value; }

// Token API stubs
BOOL BeaconUseToken(HANDLE token)  { (void)token; return FALSE; }
void BeaconRevertToken()           {}
BOOL BeaconIsAdmin()               { return FALSE; }

// Key-Value store stubs
void  BeaconAddValue(const char* key, void* value) { (void)key; (void)value; }
void* BeaconGetValue(const char* key)               { (void)key; return nullptr; }
BOOL  BeaconRemoveValue(const char* key)            { (void)key; return FALSE; }

// Adaptix extensions stubs
void AxAddScreenshot(char* note, int noteLen, char* data, int dataLen) {
    // TODO: Pack screenshot data and route via CALLBACK_AX_SCREENSHOT
    (void)note; (void)noteLen; (void)data; (void)dataLen;
}

void AxDownloadMemory(char* filename, int filenameLen, char* data, int dataLen) {
    // TODO: Pack memory download and route via CALLBACK_AX_DOWNLOAD_MEM
    (void)filename; (void)filenameLen; (void)data; (void)dataLen;
}
