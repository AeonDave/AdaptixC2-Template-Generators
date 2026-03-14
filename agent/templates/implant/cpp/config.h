// __NAME__ Agent — Configuration Interface
//
// Config is injected at build time via preprocessor macros:
//   -DPROFILE="\x..."  -DPROFILE_SIZE=123
//
// The Go plugin (pl_build_cpp.go) generates these flags when building.

#pragma once

#if defined(BUILD_SVC)
char* getServiceName();
#endif

char* getProfile();
unsigned int getProfileSize();
