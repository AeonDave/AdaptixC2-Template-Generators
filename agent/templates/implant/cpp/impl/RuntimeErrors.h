#pragma once

#ifdef _WIN32
#include <winsock2.h>
#endif
#include <windows.h>
#include <stdint.h>

namespace runtime_errors {

uint32_t MapSystemErrorCode(DWORD err);
uint32_t UnsupportedCommandErrorCode();

} // namespace runtime_errors