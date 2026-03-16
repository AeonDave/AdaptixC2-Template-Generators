// __NAME__ Agent — Default Evasion Gate (Panic Placeholder)
//
// Every method panics — forcing you to provide a real EvasionGate
// implementation before the agent performs any OS interaction.

use super::EvasionGate;

/// DefaultGate is the panicking placeholder.
/// Replace it with your own struct that implements EvasionGate.
pub struct DefaultGate;

impl DefaultGate {
    pub fn new() -> Self {
        DefaultGate
    }
}

impl EvasionGate for DefaultGate {
    fn init(&mut self) -> Result<(), String> {
        panic!("evasion: EvasionGate::init() not implemented — provide your own EvasionGate");
    }

    fn syscall(&self, _num: u16, _args: &[usize]) -> Result<u32, String> {
        panic!("evasion: EvasionGate::syscall() not implemented — provide your own EvasionGate");
    }

    fn resolve_fn(&self, _module: &str, _function: &str) -> Result<usize, String> {
        panic!("evasion: EvasionGate::resolve_fn() not implemented — provide your own EvasionGate");
    }

    fn call(&self, _func: usize, _args: &[usize]) -> Result<usize, String> {
        panic!("evasion: EvasionGate::call() not implemented — provide your own EvasionGate");
    }

    fn close(&mut self) {
        // no-op in placeholder — nothing to clean up
    }
}
