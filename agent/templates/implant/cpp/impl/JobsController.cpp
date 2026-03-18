// __NAME__ Agent — Jobs Controller Implementation (Stub)
//
#include "JobsController.h"

JobsController::JobsController()
    : jobs(nullptr), jobCount(0), jobCapacity(0), nextId(1)
{
    InitializeCriticalSection(&lock);
}

JobsController::~JobsController()
{
    EnterCriticalSection(&lock);
    for (int i = 0; i < jobCount; ++i) {
        if (jobs[i].hStopEvent) {
            SetEvent(jobs[i].hStopEvent);
        }
        if (jobs[i].hThread) {
            CloseHandle(jobs[i].hThread);
            jobs[i].hThread = nullptr;
        }
        if (jobs[i].hStopEvent) {
            CloseHandle(jobs[i].hStopEvent);
            jobs[i].hStopEvent = nullptr;
        }
    }
    jobCount = 0;
    LeaveCriticalSection(&lock);

    DeleteCriticalSection(&lock);
    if (jobs) {
        LocalFree(jobs);
        jobs = nullptr;
    }
}

ULONG JobsController::Add(ULONG jobType, HANDLE hThread, DWORD threadId, HANDLE hStopEvent)
{
    EnterCriticalSection(&lock);

    if (jobCount + 1 > jobCapacity) {
        int newCapacity = (jobCapacity > 0) ? jobCapacity : 4;
        while (newCapacity < jobCount + 1) {
            newCapacity *= 2;
        }

        SIZE_T bytes = (SIZE_T)newCapacity * sizeof(Job);
        Job* resized = jobs
            ? (Job*)LocalReAlloc(jobs, bytes, LMEM_MOVEABLE | LMEM_ZEROINIT)
            : (Job*)LocalAlloc(LPTR, bytes);
        if (!resized) {
            LeaveCriticalSection(&lock);
            return 0;
        }

        jobs = resized;
        jobCapacity = newCapacity;
    }

    Job* job = &jobs[jobCount++];
    job->jobId = nextId++;
    job->jobType = jobType;
    job->state = JOB_STATE_RUNNING;
    job->hThread = hThread;
    job->threadId = threadId;
    job->hStopEvent = hStopEvent;

    ULONG id = job->jobId;
    LeaveCriticalSection(&lock);
    return id;
}

BOOL JobsController::Remove(ULONG jobId)
{
    BOOL removed = FALSE;
    EnterCriticalSection(&lock);
    for (int i = 0; i < jobCount; ++i) {
        if (jobs[i].jobId != jobId) {
            continue;
        }

        if (jobs[i].hThread) {
            CloseHandle(jobs[i].hThread);
            jobs[i].hThread = nullptr;
        }
        if (jobs[i].hStopEvent) {
            CloseHandle(jobs[i].hStopEvent);
            jobs[i].hStopEvent = nullptr;
        }

        if (i != jobCount - 1) {
            jobs[i] = jobs[jobCount - 1];
        }
        ZeroMemory(&jobs[jobCount - 1], sizeof(Job));
        --jobCount;
        removed = TRUE;
        break;
    }
    LeaveCriticalSection(&lock);
    return removed;
}

Job* JobsController::Find(ULONG jobId)
{
    Job* found = nullptr;
    EnterCriticalSection(&lock);
    for (int i = 0; i < jobCount; ++i) {
        if (jobs[i].jobId == jobId) {
            found = &jobs[i];
            break;
        }
    }
    LeaveCriticalSection(&lock);
    return found;
}

BOOL JobsController::Kill(ULONG jobId)
{
    HANDLE hThread = nullptr;
    HANDLE hStopEvent = nullptr;

    EnterCriticalSection(&lock);
    for (int i = 0; i < jobCount; ++i) {
        if (jobs[i].jobId != jobId) {
            continue;
        }
        jobs[i].state = JOB_STATE_KILLED;
        hThread = jobs[i].hThread;
        hStopEvent = jobs[i].hStopEvent;
        break;
    }
    LeaveCriticalSection(&lock);

    if (!hThread && !hStopEvent) {
        return FALSE;
    }

    if (hStopEvent) {
        SetEvent(hStopEvent);
    }
    if (hThread) {
        WaitForSingleObject(hThread, 5000);
    }
    return Remove(jobId);
}

int JobsController::List(Job** outJobs)
{
    if (!outJobs) {
        return 0;
    }

    EnterCriticalSection(&lock);
    int count = jobCount;
    if (count <= 0) {
        *outJobs = nullptr;
        LeaveCriticalSection(&lock);
        return 0;
    }

    SIZE_T bytes = (SIZE_T)count * sizeof(Job);
    Job* copy = (Job*)LocalAlloc(LPTR, bytes);
    if (copy) {
        CopyMemory(copy, jobs, bytes);
    }
    *outJobs = copy;
    LeaveCriticalSection(&lock);
    return copy ? count : 0;
}

void JobsController::Reap()
{
    EnterCriticalSection(&lock);
    int i = 0;
    while (i < jobCount) {
        BOOL shouldRemove = FALSE;
        if (jobs[i].state == JOB_STATE_FINISHED || jobs[i].state == JOB_STATE_KILLED) {
            shouldRemove = TRUE;
        } else if (jobs[i].hThread) {
            DWORD wait = WaitForSingleObject(jobs[i].hThread, 0);
            if (wait == WAIT_OBJECT_0) {
                jobs[i].state = JOB_STATE_FINISHED;
                shouldRemove = TRUE;
            }
        }

        if (shouldRemove) {
            if (jobs[i].hThread) {
                CloseHandle(jobs[i].hThread);
                jobs[i].hThread = nullptr;
            }
            if (jobs[i].hStopEvent) {
                CloseHandle(jobs[i].hStopEvent);
                jobs[i].hStopEvent = nullptr;
            }
            if (i != jobCount - 1) {
                jobs[i] = jobs[jobCount - 1];
            }
            ZeroMemory(&jobs[jobCount - 1], sizeof(Job));
            --jobCount;
            continue;
        }
        ++i;
    }
    LeaveCriticalSection(&lock);
}
