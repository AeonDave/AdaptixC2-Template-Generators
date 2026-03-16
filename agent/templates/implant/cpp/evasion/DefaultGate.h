// __NAME__ Agent — Default Evasion Gate (Panic Placeholder)
//
// Every method aborts — forcing you to provide a real IEvasionGate subclass
// before the agent performs any OS interaction.
//
// Subclass IEvasionGate and implement:
//   - Init():      PEB walk → SSN enumeration → gadget discovery → spoof context
//   - Syscall():   indirect syscall dispatch (e.g. RecycleGate / HellsGate)
//   - ResolveFn(): manual API resolution via PEB/export table walking
//   - Call():      function pointer invocation (optionally with spoofed stack)
//   - Close():     cleanup
//
// Then pass your implementation to Agent instead of DefaultGate.

#pragma once

#include "IEvasionGate.h"
#include <cstdio>
#include <cstdlib>

class DefaultGate : public IEvasionGate
{
public:
    BOOL Init() override;
    uint32_t Syscall(uint16_t num, uintptr_t* args, int argCount) override;
    uintptr_t ResolveFn(const char* module, const char* function) override;
    uintptr_t Call(uintptr_t fn, uintptr_t* args, int argCount) override;
    void Close() override;
};
