// __NAME__ Agent — Rust Implant Configuration
//
// Profile data is injected at build time. The server-side plugin writes
// this file before compiling.
//
// TODO: Replace with actual profile structure after implementing
// GenerateProfiles in pl_build.go.

/// Encrypted profile blob (populated by build system)
pub static ENC_PROFILES: &[&[u8]] = &[];
