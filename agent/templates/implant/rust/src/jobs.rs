// __NAME__ Agent — Jobs Controller
//
// Manages async jobs (long-running tasks: BOF async, shells, tunnels, etc.)
// that execute in background threads and report status/output asynchronously.

#![allow(dead_code)]

use std::sync::{Arc, Mutex};

// ── Job types (must match server-side pl_utils.go JOB_TYPE_*) ──────────────────

pub const JOB_TYPE_LOCAL: u32   = 0x1;
pub const JOB_TYPE_REMOTE: u32  = 0x2;
pub const JOB_TYPE_PROCESS: u32 = 0x3;
pub const JOB_TYPE_SHELL: u32   = 0x4;
pub const JOB_TYPE_BOF: u32     = 0x5;

// ── Job states ─────────────────────────────────────────────────────────────────

pub const JOB_STATE_STARTING: u32 = 0x0;
pub const JOB_STATE_RUNNING: u32  = 0x1;
pub const JOB_STATE_FINISHED: u32 = 0x2;
pub const JOB_STATE_KILLED: u32   = 0x3;

// ── Job entry ──────────────────────────────────────────────────────────────────

pub struct Job {
    pub job_id: u32,
    pub job_type: u32,
    pub state: u32,
    // TODO: Add thread handle (JoinHandle), stop signal (Arc<AtomicBool>), etc.
}

// ── JobsController ─────────────────────────────────────────────────────────────

pub struct JobsController {
    jobs: Arc<Mutex<Vec<Job>>>,
    next_id: u32,
}

impl JobsController {
    pub fn new() -> Self {
        JobsController {
            jobs: Arc::new(Mutex::new(Vec::new())),
            next_id: 1,
        }
    }

    /// Add a new job. Returns the assigned job ID.
    pub fn add(&mut self, job_type: u32) -> u32 {
        let id = self.next_id;
        self.next_id += 1;
        let job = Job {
            job_id: id,
            job_type,
            state: JOB_STATE_RUNNING,
        };
        if let Ok(mut v) = self.jobs.lock() {
            v.push(job);
        }
        id
    }

    /// Remove a job by ID. Returns true if found and removed.
    pub fn remove(&mut self, job_id: u32) -> bool {
        if let Ok(mut v) = self.jobs.lock() {
            if let Some(pos) = v.iter().position(|j| j.job_id == job_id) {
                v.swap_remove(pos);
                return true;
            }
        }
        false
    }

    /// Kill a job by ID: set state to killed.
    /// TODO: Signal stop channel / atomic bool when threading is wired.
    pub fn kill(&mut self, job_id: u32) -> bool {
        if let Ok(mut v) = self.jobs.lock() {
            if let Some(j) = v.iter_mut().find(|j| j.job_id == job_id) {
                j.state = JOB_STATE_KILLED;
                return true;
            }
        }
        false
    }

    /// Collect running job entries for reporting to the server.
    pub fn list(&self) -> Vec<(u32, u32, u32)> {
        if let Ok(v) = self.jobs.lock() {
            v.iter()
                .filter(|j| j.state == JOB_STATE_RUNNING || j.state == JOB_STATE_STARTING)
                .map(|j| (j.job_id, j.job_type, j.state))
                .collect()
        } else {
            Vec::new()
        }
    }

    /// Remove finished and killed jobs, returning them.
    pub fn reap(&mut self) -> Vec<Job> {
        let mut reaped = Vec::new();
        if let Ok(mut v) = self.jobs.lock() {
            let mut i = 0;
            while i < v.len() {
                if v[i].state == JOB_STATE_FINISHED || v[i].state == JOB_STATE_KILLED {
                    reaped.push(v.swap_remove(i));
                } else {
                    i += 1;
                }
            }
        }
        reaped
    }
}
