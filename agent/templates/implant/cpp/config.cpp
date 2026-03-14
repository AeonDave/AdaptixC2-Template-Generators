// __NAME__ Agent — Configuration Implementation
//
// Profile data is injected via preprocessor: -DPROFILE="\x..." -DPROFILE_SIZE=N
// If not defined, provides empty defaults for testing.

#include "config.h"

#ifndef PROFILE
#define PROFILE ""
#endif

#ifndef PROFILE_SIZE
#define PROFILE_SIZE 0
#endif

#if defined(BUILD_SVC)
#ifndef SERVICE_NAME
#define SERVICE_NAME "__NAME__Service"
#endif
char* getServiceName()
{
    return (char*)SERVICE_NAME;
}
#endif

char* getProfile()
{
    return (char*)PROFILE;
}

unsigned int getProfileSize()
{
    return PROFILE_SIZE;
}
