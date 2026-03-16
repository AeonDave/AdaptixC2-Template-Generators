package evasion

// defaultGate is the panicking placeholder.  Every method fatally
// aborts — forcing you to provide a real Gate before any OS call.
type defaultGate struct{}

func (defaultGate) Init() error {
	panic("evasion: Gate.Init() not implemented — provide your own evasion.Gate")
}

func (defaultGate) Syscall(_ uint16, _ ...uintptr) (uint32, error) {
	panic("evasion: Gate.Syscall() not implemented — provide your own evasion.Gate")
}

func (defaultGate) ResolveFn(_, _ string) (uintptr, error) {
	panic("evasion: Gate.ResolveFn() not implemented — provide your own evasion.Gate")
}

func (defaultGate) Call(_ uintptr, _ ...uintptr) (uintptr, error) {
	panic("evasion: Gate.Call() not implemented — provide your own evasion.Gate")
}

func (defaultGate) Close() {}
