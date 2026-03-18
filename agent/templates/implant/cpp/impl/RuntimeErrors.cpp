#include "RuntimeErrors.h"

namespace runtime_errors {

uint32_t MapSystemErrorCode(DWORD err)
{
    if (err == ERROR_FILE_NOT_FOUND || err == ERROR_PATH_NOT_FOUND) {
        return 2;
    }
    if (err == ERROR_ACCESS_DENIED) {
        return 5;
    }
    return 31;
}

uint32_t UnsupportedCommandErrorCode()
{
    return 50;
}

} // namespace runtime_errors