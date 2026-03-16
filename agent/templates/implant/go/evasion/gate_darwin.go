//go:build darwin

package evasion

// ═══════════════════════════════════════════════════════════════════════════════
// Darwin Gate Stub
//
// macOS uses the Mach syscall ABI (syscall numbers offset by 0x2000000
// on x86-64).  EDR hooking on macOS is less common than on Windows,
// but the Gate abstraction still applies:
//
//   - Direct SYSCALL instruction with Mach-adjusted number
//   - dlopen/dlsym for dynamic API resolution
//   - No call-stack spoofing analogue exists on macOS currently
//
// ═══════════════════════════════════════════════════════════════════════════════

// TODO: Implement your Darwin-specific Gate here.
//
// Minimal skeleton:
//
//	type DarwinGate struct{}
//
//	func NewDarwinGate() *DarwinGate { return &DarwinGate{} }
//
//	func (g *DarwinGate) Init() error { return nil }
//
//	func (g *DarwinGate) Syscall(num uint16, args ...uintptr) (uint32, error) {
//	    // Mach syscall: number |= 0x2000000 on x86-64
//	    return 0, nil
//	}
//
//	func (g *DarwinGate) ResolveFn(module, function string) (uintptr, error) {
//	    // dlopen(module) + dlsym(handle, function)
//	    return 0, nil
//	}
//
//	func (g *DarwinGate) Call(fn uintptr, args ...uintptr) (uintptr, error) {
//	    return 0, nil
//	}
//
//	func (g *DarwinGate) Close() {}
