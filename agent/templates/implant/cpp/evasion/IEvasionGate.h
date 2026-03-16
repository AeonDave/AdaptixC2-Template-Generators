// __NAME__ Agent — Evasion Gate Interface
//
// Abstract base class for pluggable syscall dispatch, manual API resolution,
// and call-stack spoofing.  The generated DefaultGate aborts on every call —
// subclass this with your own implementation before the agent performs any
// OS interaction.
//
// ─── Obfuscated string helper ──────────────────────────────────────────────────
//
// Use char-array construction + MBA (Mixed Boolean-Arithmetic) salt instead of
// string literals to keep sensitive names (DLL names, function names) out of
// the binary's string table.
//
// Example — hide "ntdll.dll" with an MBA decode (equivalent to XOR without ^):
//
//   static const char* ntdll_name() {
//       static char buf[10];
//       const unsigned char salt = 0x37;
//       const unsigned char enc[] = {0x59,0x43,0x53,0x5b,0x5b,0x19,0x53,0x5b,0x5b,0x00};
//       for (int i = 0; i < 9; i++) buf[i] = (enc[i] + salt) - 2*(enc[i] & salt); // MBA: a⊕b = (a+b) − 2(a∧b)
//       buf[9] = 0;
//       return buf;
//   }
//
// Or the simple char-array approach (no XOR, still avoids string table):
//
//   char ntdll[] = {'n','t','d','l','l','.','d','l','l',0};
//
// ═══════════════════════════════════════════════════════════════════════════════

#pragma once

#include <windows.h>
#include <stdint.h>

class IEvasionGate
{
public:
    virtual ~IEvasionGate() {}

    // One-time setup: SSN enumeration, gadget discovery, spoof context, etc.
    virtual BOOL Init() = 0;

    // Raw syscall dispatch by number (SSN).
    // args points to an array of argCount uintptr-sized values.
    virtual uint32_t Syscall(uint16_t num, uintptr_t* args, int argCount) = 0;

    // Manually resolve a function address from module + export name.
    // Must not use LoadLibrary/GetProcAddress (PEB walk, export table parse).
    virtual uintptr_t ResolveFn(const char* module, const char* function) = 0;

    // Invoke an arbitrary function pointer with the given arguments.
    // Implementations may route through a spoofed call-stack trampoline.
    virtual uintptr_t Call(uintptr_t fn, uintptr_t* args, int argCount) = 0;

    // Release any resources acquired during Init.
    virtual void Close() = 0;
};
