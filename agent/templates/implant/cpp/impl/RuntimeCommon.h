#pragma once

#ifdef _WIN32
#include <winsock2.h>
#endif
#include <windows.h>
#include <stdint.h>

#include <string>
#include <vector>

namespace runtime_common {

uint32_t EncodeWorkingTime(int start, int end);
void DecodeWorkingTime(uint32_t wt, int* workStart, int* workEnd);

std::string CurrentProcessName(const char* fallbackName);
uint32_t LocalIPv4();
BOOL IsProcessElevated();

BOOL RunDetachedProcess(const char* program, const char* args, DWORD* outPid);
BOOL RunShellCapture(const char* cmdLine, std::string* outText);
BOOL ZipPathToFile(const char* srcPath, const char* zipPath);
BOOL CaptureScreenshotPng(std::vector<BYTE>* outData);

BOOL MkdirRecursive(const char* path);
BOOL RemoveRecursive(const char* path);

} // namespace runtime_common