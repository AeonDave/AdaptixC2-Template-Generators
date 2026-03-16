// __NAME__ Agent — Default Evasion Gate (Panic Placeholder)

#include "DefaultGate.h"

BOOL DefaultGate::Init()
{
    fprintf(stderr, "evasion: IEvasionGate::Init() not implemented\n");
    abort();
    return FALSE;
}

uint32_t DefaultGate::Syscall(uint16_t num, uintptr_t* args, int argCount)
{
    (void)num; (void)args; (void)argCount;
    fprintf(stderr, "evasion: IEvasionGate::Syscall() not implemented\n");
    abort();
    return 0;
}

uintptr_t DefaultGate::ResolveFn(const char* module, const char* function)
{
    (void)module; (void)function;
    fprintf(stderr, "evasion: IEvasionGate::ResolveFn() not implemented\n");
    abort();
    return 0;
}

uintptr_t DefaultGate::Call(uintptr_t fn, uintptr_t* args, int argCount)
{
    (void)fn; (void)args; (void)argCount;
    fprintf(stderr, "evasion: IEvasionGate::Call() not implemented\n");
    abort();
    return 0;
}

void DefaultGate::Close()
{
    // no-op in placeholder — nothing to clean up
}
