// __NAME__ Agent — Agent Implementation (Stub)
//
// TODO: Implement agent initialization, BuildBeat, and lifecycle.

#include "Agent.h"
#include "Commander.h"

Agent::Agent()
{
    commander = new Commander(this);

    // Generate random session key (16 bytes)
    sessionKey = (BYTE*)LocalAlloc(LPTR, 16);
    // TODO: Fill with cryptographic random bytes
}

Agent::~Agent()
{
    if (commander) { delete commander; commander = nullptr; }
    if (sessionKey) { LocalFree(sessionKey); sessionKey = nullptr; }
}

BOOL Agent::IsActive()
{
    return active;
}

BYTE* Agent::BuildBeat(ULONG* size)
{
    // TODO: Build initial beacon packet with system info
    // Pattern: pack agent ID, OS info, hostname, username, etc.
    *size = 0;
    return nullptr;
}
