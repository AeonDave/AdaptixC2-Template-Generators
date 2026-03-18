#![allow(dead_code)]

use std::env;
use std::net::UdpSocket;
use std::path::Path;
use std::thread;
use std::time::{Duration, SystemTime, UNIX_EPOCH};

pub fn sleep_with_jitter(base_ms: u64, jitter_pct: u32, minimum_ms: u64) {
    let clamped = jitter_pct.clamp(0, 90) as u64;
    let actual_ms = if clamped == 0 {
        base_ms
    } else {
        let delta = base_ms.saturating_mul(clamped) / 100;
        let spread = delta.saturating_mul(2).saturating_add(1);
        let roll = (random_u32() as u64) % spread;
        base_ms.saturating_sub(delta).saturating_add(roll)
    };
    thread::sleep(Duration::from_millis(actual_ms.max(minimum_ms)));
}

pub fn should_exit(kill_date: i64) -> bool {
    kill_date > 0 && now_unix() >= kill_date
}

pub fn wait_for_working_hours(work_start: i32, work_end: i32) {
    if work_start == 0 && work_end == 0 {
        return;
    }

    loop {
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs();
        let hhmm = ((now % 86400) / 3600) as i32 * 100 + (((now % 86400) % 3600) / 60) as i32;
        let in_window = if work_start <= work_end {
            hhmm >= work_start && hhmm < work_end
        } else {
            hhmm >= work_start || hhmm < work_end
        };
        if in_window {
            return;
        }
        thread::sleep(Duration::from_secs(60));
    }
}

pub fn encode_working_time(start: i32, end: i32) -> u32 {
    if start == 0 && end == 0 {
        return 0;
    }
    let start_hour = start / 100;
    let start_min = start % 100;
    let end_hour = end / 100;
    let end_min = end % 100;
    ((start_hour as u32) << 24)
        | ((start_min as u32) << 16)
        | ((end_hour as u32) << 8)
        | (end_min as u32)
}

pub fn decode_working_time(value: u32) -> (i32, i32) {
    if value == 0 {
        return (0, 0);
    }
    let start_hour = ((value >> 24) & 0xff) as i32;
    let start_min = ((value >> 16) & 0xff) as i32;
    let end_hour = ((value >> 8) & 0xff) as i32;
    let end_min = (value & 0xff) as i32;
    (start_hour * 100 + start_min, end_hour * 100 + end_min)
}

pub fn process_name(fallback: &str) -> String {
    env::current_exe()
        .ok()
        .as_deref()
        .and_then(Path::file_name)
        .map(|s| s.to_string_lossy().to_string())
        .unwrap_or_else(|| fallback.to_string())
}

pub fn hostname() -> String {
    env::var("COMPUTERNAME")
        .or_else(|_| env::var("HOSTNAME"))
        .unwrap_or_default()
}

pub fn local_ipv4() -> String {
    UdpSocket::bind("0.0.0.0:0")
        .and_then(|sock| {
            sock.connect("8.8.8.8:80")?;
            sock.local_addr()
        })
        .map(|addr| addr.ip().to_string())
        .unwrap_or_default()
}

pub fn is_elevated() -> bool {
    #[cfg(unix)]
    {
        unsafe { libc::geteuid() == 0 }
    }
    #[cfg(not(unix))]
    {
        false
    }
}

pub fn now_unix() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_secs() as i64
}

pub fn random_u32() -> u32 {
    let nanos = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_nanos();
    (nanos as u64 ^ ((nanos >> 32) as u64)) as u32
}