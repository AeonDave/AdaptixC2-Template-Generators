// __NAME__ Agent — BOF (Beacon Object File) Loader
//
// COFF parser and in-memory loader for executing Beacon Object Files.
// Implements the Beacon API function table that BOFs expect at runtime.
//
// Reference: beacon_agent/src_beacon/beacon/bof_loader.h

#pragma once

#include <windows.h>

// Limits
#define MAX_BOF_SECTIONS    25
#define MAX_BOF_FUNCTIONS   512

// Forward declare the output packer type used by BOF output callbacks.
// Replace with your actual Packer class or buffer type.
struct BofOutput {
    BYTE*  buffer;
    ULONG  size;
    ULONG  capacity;
};

// ── BOF Loader API ─────────────────────────────────────────────────────────────

// Execute a COFF object file in-memory.
// - taskId:       command ID for output routing
// - coffFile:     raw .o file bytes
// - coffFileSize: size of coffFile
// - args:         packed argument buffer (bof_pack format)
// - argsSize:     size of args
//
// Returns a BofOutput* with collected output, or nullptr on error.
// Caller owns the returned BofOutput and must free it.
//
// TODO: Implement COFF section allocation, relocation processing,
//       external symbol resolution (Beacon API), and entry point invocation.
//       See beacon_agent bof_loader.cpp for a full reference implementation.
BofOutput* ObjectExecute(
    ULONG taskId,
    unsigned char* coffFile, unsigned int coffFileSize,
    unsigned char* args, int argsSize
);

// Free a BofOutput returned by ObjectExecute.
void BofOutputFree(BofOutput* output);

// ── Beacon Functions API ───────────────────────────────────────────────────────
//
// BOFs call these via the function resolution table. Each must be implemented
// and registered in the symbol lookup so that ObjectExecute can resolve them.
//
// Data Parser:
//   void   BeaconDataParse(void* parser, char* buffer, int size);
//   int    BeaconDataInt(void* parser);
//   short  BeaconDataShort(void* parser);
//   int    BeaconDataLength(void* parser);
//   char*  BeaconDataExtract(void* parser, int* size);
//
// Output:
//   void   BeaconOutput(int type, char* data, int len);
//   void   BeaconPrintf(int type, char* fmt, ...);
//
// Format buffer:
//   void   BeaconFormatAlloc(void* format, int maxsz);
//   void   BeaconFormatReset(void* format);
//   void   BeaconFormatAppend(void* format, char* text, int len);
//   void   BeaconFormatPrintf(void* format, char* fmt, ...);
//   char*  BeaconFormatToString(void* format, int* size);
//   void   BeaconFormatFree(void* format);
//   void   BeaconFormatInt(void* format, int value);
//
// Token:
//   BOOL   BeaconUseToken(HANDLE token);
//   void   BeaconRevertToken();
//   BOOL   BeaconIsAdmin();
//
// Key-value store:
//   void   BeaconAddValue(const char* key, void* value);
//   void*  BeaconGetValue(const char* key);
//   BOOL   BeaconRemoveValue(const char* key);
//
// Adaptix extensions:
//   void   AxAddScreenshot(char* note, int noteLen, char* data, int dataLen);
//   void   AxDownloadMemory(char* filename, int filenameLen, char* data, int dataLen);
//
// TODO: Implement each function. See beacon_agent beacon_functions.cpp.
