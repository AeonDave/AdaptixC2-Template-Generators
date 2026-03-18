package main

import "fmt"

// ParseInternalAgentRegistration decodes the decrypted first packet of an internal listener
// into the Teamserver registration tuple: agent type, agent id, and beat payload.
//
// Protocols that support internal listeners should override this file with
// `protocols/<name>/pl_internal.go.tmpl`.
func ParseInternalAgentRegistration(decrypted []byte) (string, string, []byte, error) {
	_ = decrypted
	return "", "", nil, fmt.Errorf("internal listener registration is not implemented for protocol %q", "__PROTOCOL__")
}
