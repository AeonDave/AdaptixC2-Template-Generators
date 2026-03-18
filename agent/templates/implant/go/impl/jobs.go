// __NAME__ Agent — Job Management
//
// Tracks long-running background tasks (async BOFs, shell sessions, etc.).
// Use Add() to register a job, Kill() to signal stop, Reap() to clean up finished jobs.

package impl

import "sync"

// ── Job types ──────────────────────────────────────────────────────────────────

const (
	JobTypeBof   = 0x5
	JobTypeShell = 0x6
)

// ── Job states ─────────────────────────────────────────────────────────────────

const (
	JobStatePending  = 0
	JobStateRunning  = 1
	JobStateFinished = 2
	JobStateStopped  = 3
)

// Job represents a single background task.
type Job struct {
	JobId   uint32
	JobType int
	State   int
	StopCh  chan struct{} // close to signal cancellation
}

// JobsController manages running jobs.
type JobsController struct {
	mu     sync.Mutex
	jobs   []Job
	nextId uint32
}

// NewJobsController creates an empty controller.
func NewJobsController() *JobsController {
	return &JobsController{nextId: 1}
}

// Add registers a new job and returns its assigned ID.
func (jc *JobsController) Add(jobType int) uint32 {
	jc.mu.Lock()
	defer jc.mu.Unlock()
	id := jc.nextId
	jc.nextId++
	jc.jobs = append(jc.jobs, Job{
		JobId:   id,
		JobType: jobType,
		State:   JobStateRunning,
		StopCh:  make(chan struct{}),
	})
	return id
}

// AddWithID registers a new job using a caller-provided ID.
func (jc *JobsController) AddWithID(jobId uint32, jobType int) uint32 {
	jc.mu.Lock()
	defer jc.mu.Unlock()
	for i := range jc.jobs {
		if jc.jobs[i].JobId == jobId {
			jc.jobs[i].JobType = jobType
			jc.jobs[i].State = JobStateRunning
			if jc.jobs[i].StopCh == nil {
				jc.jobs[i].StopCh = make(chan struct{})
			}
			return jobId
		}
	}
	if jobId >= jc.nextId {
		jc.nextId = jobId + 1
	}
	jc.jobs = append(jc.jobs, Job{
		JobId:   jobId,
		JobType: jobType,
		State:   JobStateRunning,
		StopCh:  make(chan struct{}),
	})
	return jobId
}

// StopSignal returns the stop channel for the given job.
func (jc *JobsController) StopSignal(jobId uint32) (<-chan struct{}, bool) {
	jc.mu.Lock()
	defer jc.mu.Unlock()
	for i := range jc.jobs {
		if jc.jobs[i].JobId == jobId {
			return jc.jobs[i].StopCh, true
		}
	}
	return nil, false
}

// SetState updates the state of a job.
func (jc *JobsController) SetState(jobId uint32, state int) bool {
	jc.mu.Lock()
	defer jc.mu.Unlock()
	for i := range jc.jobs {
		if jc.jobs[i].JobId == jobId {
			jc.jobs[i].State = state
			return true
		}
	}
	return false
}

// Remove deletes a job by ID.
func (jc *JobsController) Remove(jobId uint32) {
	jc.mu.Lock()
	defer jc.mu.Unlock()
	for i, j := range jc.jobs {
		if j.JobId == jobId {
			jc.jobs = append(jc.jobs[:i], jc.jobs[i+1:]...)
			return
		}
	}
}

// Find returns a pointer to the job with the given ID, or nil.
func (jc *JobsController) Find(jobId uint32) *Job {
	jc.mu.Lock()
	defer jc.mu.Unlock()
	for i := range jc.jobs {
		if jc.jobs[i].JobId == jobId {
			return &jc.jobs[i]
		}
	}
	return nil
}

// Kill signals a job to stop by closing its StopCh.
func (jc *JobsController) Kill(jobId uint32) bool {
	jc.mu.Lock()
	defer jc.mu.Unlock()
	for i := range jc.jobs {
		if jc.jobs[i].JobId == jobId {
			jc.jobs[i].State = JobStateStopped
			select {
			case <-jc.jobs[i].StopCh:
				// already closed
			default:
				close(jc.jobs[i].StopCh)
			}
			return true
		}
	}
	return false
}

// List returns a copy of all active jobs.
func (jc *JobsController) List() []Job {
	jc.mu.Lock()
	defer jc.mu.Unlock()
	out := make([]Job, len(jc.jobs))
	copy(out, jc.jobs)
	return out
}

// Reap removes all finished/stopped jobs and returns them.
func (jc *JobsController) Reap() []Job {
	jc.mu.Lock()
	defer jc.mu.Unlock()
	var reaped []Job
	var kept []Job
	for _, j := range jc.jobs {
		if j.State == JobStateFinished || j.State == JobStateStopped {
			reaped = append(reaped, j)
		} else {
			kept = append(kept, j)
		}
	}
	jc.jobs = kept
	return reaped
}
