#include "RuntimeCommon.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <string>
#include <vector>

#include <ws2tcpip.h>

namespace runtime_common {

uint32_t EncodeWorkingTime(int start, int end)
{
    if (start == 0 && end == 0) {
        return 0;
    }
    int startHour = start / 100;
    int startMin  = start % 100;
    int endHour   = end / 100;
    int endMin    = end % 100;
    return ((uint32_t)startHour << 24) | ((uint32_t)startMin << 16) | ((uint32_t)endHour << 8) | (uint32_t)endMin;
}

void DecodeWorkingTime(uint32_t wt, int* workStart, int* workEnd)
{
    if (!workStart || !workEnd) {
        return;
    }
    if (wt == 0) {
        *workStart = 0;
        *workEnd = 0;
        return;
    }
    int startHour = (int)((wt >> 24) & 0xff);
    int startMin  = (int)((wt >> 16) & 0xff);
    int endHour   = (int)((wt >> 8) & 0xff);
    int endMin    = (int)(wt & 0xff);
    *workStart = startHour * 100 + startMin;
    *workEnd = endHour * 100 + endMin;
}

std::string CurrentProcessName(const char* fallbackName)
{
    char path[MAX_PATH] = {0};
    DWORD len = GetModuleFileNameA(NULL, path, MAX_PATH);
    if (len == 0 || len >= MAX_PATH) {
        return fallbackName ? std::string(fallbackName) : std::string();
    }
    for (int i = (int)len - 1; i >= 0; --i) {
        if (path[i] == '\\' || path[i] == '/') {
            return std::string(path + i + 1);
        }
    }
    return std::string(path);
}

uint32_t LocalIPv4()
{
    WSADATA wsa = {};
    if (WSAStartup(MAKEWORD(2, 2), &wsa) != 0) {
        return 0;
    }

    uint32_t resultIp = 0;
    char hostname[256] = {0};
    if (gethostname(hostname, sizeof(hostname)) == 0) {
        addrinfo hints = {};
        hints.ai_family = AF_INET;
        hints.ai_socktype = SOCK_STREAM;

        addrinfo* addrs = nullptr;
        if (getaddrinfo(hostname, NULL, &hints, &addrs) == 0) {
            for (addrinfo* it = addrs; it != nullptr; it = it->ai_next) {
                if (!it->ai_addr || it->ai_addrlen < sizeof(sockaddr_in)) {
                    continue;
                }
                const sockaddr_in* addr = reinterpret_cast<const sockaddr_in*>(it->ai_addr);
                const uint8_t* octets = reinterpret_cast<const uint8_t*>(&addr->sin_addr.S_un.S_addr);
                if (octets[0] == 127) {
                    continue;
                }
                resultIp = ((uint32_t)octets[0] << 24) | ((uint32_t)octets[1] << 16) | ((uint32_t)octets[2] << 8) | (uint32_t)octets[3];
                break;
            }
            freeaddrinfo(addrs);
        }
    }

    WSACleanup();
    return resultIp;
}

BOOL IsProcessElevated()
{
    HANDLE hToken = NULL;
    if (!OpenProcessToken(GetCurrentProcess(), TOKEN_QUERY, &hToken)) {
        return FALSE;
    }
    TOKEN_ELEVATION elevation = {};
    DWORD retLen = 0;
    BOOL elevated = FALSE;
    if (GetTokenInformation(hToken, TokenElevation, &elevation, sizeof(elevation), &retLen)) {
        elevated = elevation.TokenIsElevated ? TRUE : FALSE;
    }
    CloseHandle(hToken);
    return elevated;
}

BOOL RunDetachedProcess(const char* program, const char* args, DWORD* outPid)
{
    if (!program || !program[0]) {
        SetLastError(ERROR_INVALID_PARAMETER);
        return FALSE;
    }

    char cmdLine[2048];
    if (args && args[0]) {
        snprintf(cmdLine, sizeof(cmdLine), "\"%s\" %s", program, args);
    } else {
        snprintf(cmdLine, sizeof(cmdLine), "\"%s\"", program);
    }

    STARTUPINFOA si = {};
    PROCESS_INFORMATION pi = {};
    si.cb = sizeof(si);
    BOOL ok = CreateProcessA(NULL, cmdLine, NULL, NULL, FALSE, CREATE_NO_WINDOW | DETACHED_PROCESS, NULL, NULL, &si, &pi);
    if (!ok) {
        return FALSE;
    }
    if (outPid) {
        *outPid = pi.dwProcessId;
    }
    CloseHandle(pi.hThread);
    CloseHandle(pi.hProcess);
    return TRUE;
}

BOOL RunShellCapture(const char* cmdLine, std::string* outText)
{
    if (!cmdLine || !outText) {
        SetLastError(ERROR_INVALID_PARAMETER);
        return FALSE;
    }

    SECURITY_ATTRIBUTES sa = {};
    sa.nLength = sizeof(sa);
    sa.bInheritHandle = TRUE;

    HANDLE readPipe = NULL;
    HANDLE writePipe = NULL;
    if (!CreatePipe(&readPipe, &writePipe, &sa, 0)) {
        return FALSE;
    }
    SetHandleInformation(readPipe, HANDLE_FLAG_INHERIT, 0);

    char fullCmd[2048];
    snprintf(fullCmd, sizeof(fullCmd), "cmd /C %s", cmdLine);

    STARTUPINFOA si = {};
    PROCESS_INFORMATION pi = {};
    si.cb = sizeof(si);
    si.dwFlags = STARTF_USESHOWWINDOW | STARTF_USESTDHANDLES;
    si.wShowWindow = SW_HIDE;
    si.hStdOutput = writePipe;
    si.hStdError = writePipe;

    BOOL ok = CreateProcessA(NULL, fullCmd, NULL, NULL, TRUE, CREATE_NO_WINDOW, NULL, NULL, &si, &pi);
    CloseHandle(writePipe);
    if (!ok) {
        CloseHandle(readPipe);
        return FALSE;
    }

    std::string output;
    char buffer[4096];
    DWORD bytesRead = 0;
    while (ReadFile(readPipe, buffer, sizeof(buffer), &bytesRead, NULL) && bytesRead > 0) {
        output.append(buffer, buffer + bytesRead);
    }

    WaitForSingleObject(pi.hProcess, INFINITE);
    CloseHandle(pi.hThread);
    CloseHandle(pi.hProcess);
    CloseHandle(readPipe);

    *outText = output;
    return TRUE;
}

BOOL ZipPathToFile(const char* srcPath, const char* zipPath)
{
    if (!srcPath || !zipPath) {
        SetLastError(ERROR_INVALID_PARAMETER);
        return FALSE;
    }

    char psCmd[4096];
    snprintf(psCmd, sizeof(psCmd),
        "powershell -NoProfile -Command \"Compress-Archive -Force -Path '%s' -DestinationPath '%s'\"",
        srcPath, zipPath);

    STARTUPINFOA si = {};
    PROCESS_INFORMATION pi = {};
    si.cb = sizeof(si);
    si.dwFlags = STARTF_USESHOWWINDOW;
    si.wShowWindow = SW_HIDE;

    BOOL ok = CreateProcessA(NULL, psCmd, NULL, NULL, FALSE, CREATE_NO_WINDOW, NULL, NULL, &si, &pi);
    if (!ok) {
        return FALSE;
    }

    WaitForSingleObject(pi.hProcess, INFINITE);
    DWORD exitCode = 1;
    GetExitCodeProcess(pi.hProcess, &exitCode);
    CloseHandle(pi.hThread);
    CloseHandle(pi.hProcess);
    if (exitCode != 0) {
        SetLastError(ERROR_GEN_FAILURE);
        return FALSE;
    }
    return TRUE;
}

BOOL CaptureScreenshotPng(std::vector<BYTE>* outData)
{
    if (!outData) {
        SetLastError(ERROR_INVALID_PARAMETER);
        return FALSE;
    }

    char tempDir[MAX_PATH] = {0};
    if (GetTempPathA(MAX_PATH, tempDir) == 0) {
        return FALSE;
    }

    char tempFile[MAX_PATH] = {0};
    if (GetTempFileNameA(tempDir, "axs", 0, tempFile) == 0) {
        return FALSE;
    }
    DeleteFileA(tempFile);
    strcat_s(tempFile, MAX_PATH, ".png");

    char psCmd[8192];
    snprintf(psCmd, sizeof(psCmd),
        "powershell -NoProfile -Command \"Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing; $b=[System.Windows.Forms.SystemInformation]::VirtualScreen; $bmp=New-Object System.Drawing.Bitmap $b.Width,$b.Height; $g=[System.Drawing.Graphics]::FromImage($bmp); $g.CopyFromScreen($b.Left,$b.Top,0,0,$bmp.Size); $bmp.Save('%s', [System.Drawing.Imaging.ImageFormat]::Png); $g.Dispose(); $bmp.Dispose()\"",
        tempFile);

    STARTUPINFOA si = {};
    PROCESS_INFORMATION pi = {};
    si.cb = sizeof(si);
    si.dwFlags = STARTF_USESHOWWINDOW;
    si.wShowWindow = SW_HIDE;

    BOOL ok = CreateProcessA(NULL, psCmd, NULL, NULL, FALSE, CREATE_NO_WINDOW, NULL, NULL, &si, &pi);
    if (!ok) {
        return FALSE;
    }

    WaitForSingleObject(pi.hProcess, INFINITE);
    DWORD exitCode = 1;
    GetExitCodeProcess(pi.hProcess, &exitCode);
    CloseHandle(pi.hThread);
    CloseHandle(pi.hProcess);
    if (exitCode != 0) {
        DeleteFileA(tempFile);
        SetLastError(ERROR_GEN_FAILURE);
        return FALSE;
    }

    HANDLE hFile = CreateFileA(tempFile, GENERIC_READ, FILE_SHARE_READ, NULL, OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL, NULL);
    if (hFile == INVALID_HANDLE_VALUE) {
        DeleteFileA(tempFile);
        return FALSE;
    }

    DWORD fileSize = GetFileSize(hFile, NULL);
    outData->resize(fileSize);
    DWORD bytesRead = 0;
    ok = ReadFile(hFile, outData->data(), fileSize, &bytesRead, NULL);
    CloseHandle(hFile);
    DeleteFileA(tempFile);

    if (!ok) {
        outData->clear();
        return FALSE;
    }
    outData->resize(bytesRead);
    return TRUE;
}

BOOL MkdirRecursive(const char* path)
{
    if (CreateDirectoryA(path, NULL))
        return TRUE;
    if (GetLastError() == ERROR_ALREADY_EXISTS)
        return TRUE;

    char tmp[MAX_PATH];
    strncpy(tmp, path, MAX_PATH - 1);
    tmp[MAX_PATH - 1] = '\0';
    for (int i = (int)strlen(tmp) - 1; i > 0; i--) {
        if (tmp[i] == '\\' || tmp[i] == '/') {
            tmp[i] = '\0';
            MkdirRecursive(tmp);
            tmp[i] = '\\';
            break;
        }
    }
    if (CreateDirectoryA(path, NULL))
        return TRUE;
    return (GetLastError() == ERROR_ALREADY_EXISTS);
}

BOOL RemoveRecursive(const char* path)
{
    DWORD attrs = GetFileAttributesA(path);
    if (attrs == INVALID_FILE_ATTRIBUTES)
        return FALSE;
    if (!(attrs & FILE_ATTRIBUTE_DIRECTORY))
        return DeleteFileA(path);

    char search[MAX_PATH];
    snprintf(search, MAX_PATH, "%s\\*", path);
    WIN32_FIND_DATAA fd;
    HANDLE hFind = FindFirstFileA(search, &fd);
    if (hFind == INVALID_HANDLE_VALUE)
        return FALSE;
    do {
        if (strcmp(fd.cFileName, ".") == 0 || strcmp(fd.cFileName, "..") == 0)
            continue;
        char child[MAX_PATH];
        snprintf(child, MAX_PATH, "%s\\%s", path, fd.cFileName);
        RemoveRecursive(child);
    } while (FindNextFileA(hFind, &fd));
    FindClose(hFind);
    return RemoveDirectoryA(path);
}

} // namespace runtime_common