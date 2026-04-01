// __NAME__ Agent — Compile-time MBA String Obfuscation (Rust)
//
// Usage:
//   use crate::obf::obf;
//
//   let s: String = obf!(0xAA, "my secret string");
//
// The string literal is encoded at compile time using Mixed Boolean Arithmetic
// (MBA). obf!() decodes into a fresh String at runtime, so the plaintext
// never appears in .rodata.  The MBA transform defeats simple XOR-signature
// scanners because the operation is algebraically equivalent to XOR but uses
// add/sub/and — a pattern that blends into normal arithmetic.
//
// No proc_macro or external dependencies required — uses Rust 2021 const generics.

/// MBA-decode a compile-time encrypted byte array into a `String`.
/// Identity: a ^ b ≡ (a + b) − 2·(a & b)
#[inline(always)]
pub fn decode(enc: &[u8], key: u8) -> String {
    enc.iter().map(|&b| {
        b.wrapping_add(key)
         .wrapping_sub((b & key).wrapping_mul(2)) as char
    }).collect()
}

/// Compile-time MBA string obfuscation.
///
/// ```rust
/// let s: String = obf!(0x55, "hello");
/// assert_eq!(s, "hello");
/// ```
#[macro_export]
macro_rules! obf {
    ($key:expr, $s:expr) => {{
        const _KEY: u8 = $key;
        const _LEN: usize = $s.len();
        const _ENC: [u8; _LEN] = {
            let src = $s.as_bytes();
            let mut out = [0u8; _LEN];
            let mut i = 0;
            while i < _LEN {
                // MBA: src[i] ^ _KEY = (src[i] + _KEY) - 2*(src[i] & _KEY)
                out[i] = (src[i].wrapping_add(_KEY))
                    .wrapping_sub((src[i] & _KEY).wrapping_mul(2));
                i += 1;
            }
            out
        };
        $crate::obf::decode(&_ENC, _KEY)
    }};
}
