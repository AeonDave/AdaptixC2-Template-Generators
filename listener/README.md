# Template Listener Generator

Scaffold new AdaptixC2 **listener** plugins from templates, with selectable wire-format **protocols** shared between agents and listeners.

## Quick Start

```powershell
.\generator.ps1 -Mode listener -Name telegram -Protocol adaptix_default -ListenerType external
```

```bash
MODE=listener NAME=telegram PROTOCOL=adaptix_default LISTENER_TYPE=external ./generator.sh
```

Or run interactively: `.\generator.ps1 -Mode listener` (prompts for name, protocol, type).
Direct: `cd listener && .\generator.ps1`

## Protocols

Protocols define shared crypto, constants, and wire types between agents and listeners.
Bundled public protocols: `adaptix_gopher` (AES-256-GCM + msgpack) and `adaptix_default` (RC4 + binary packing).
Private/internal protocol overlays may also exist in `protocols/`, but they are not documented as public options here.

The base listener template is intentionally protocol-agnostic. Protocol-specific behavior should be supplied via
protocol-owned override files under `protocols/<name>/` instead of adding name-based branching to the core listener generator.

See the root README for protocol creation, crypto swap, and file layout docs.

## Generated Listener Structure

```
<name>_listener/
├── config.yaml          # Listener manifest
├── go.mod               # Go module (axc2 v1.2.0)
├── Makefile             # Build targets: plugin, dist
├── pl_main.go           # InitPlugin + Create/Start/Stop/Edit/GetProfile
├── pl_internal.go       # Internal listener registration parser hook
├── pl_transport.go      # Transport loop: accept → handleConnection
├── pl_crypto.go         # EncryptData / DecryptData (from protocol)
├── pl_utils.go          # Wire types + constants (merged from protocol)
├── map.go               # Thread-safe concurrent map
└── ax_config.axs        # Listener UI form (AxScript)
```

## Template Placeholders

| Placeholder | Replaced with | Example |
|---|---|---|
| `__NAME__` | Listener name (lowercase) | `telegram` |
| `__NAME_CAP__` | Name capitalised | `Telegram` |
| `__PROTOCOL__` | Protocol name (lowercase) | `adaptix_default` |
| `__PROTOCOL_CAP__` | Protocol capitalised | `Adaptix_default` |
| `__LISTENER_TYPE__` | `external` or `internal` | `external` |

## Build & Deploy

```bash
cd <name>_listener/
go mod tidy
make plugin        # builds .so
make dist          # copies .so + config.yaml + ax_config.axs to dist/
```

Copy the `dist/` contents into `AdaptixServer/data/extenders/<name>_listener/`.
Alternatively, generate directly into the extenders directory with `-OutputDir`.

## External vs internal listeners

- `external` listeners own a bind socket / transport loop in `pl_transport.go`
- `internal` listeners do not open a socket; they expose `InternalHandler()` for agent-provided registration traffic

If a protocol supports internal listeners, it should provide:

- `pl_internal.go.tmpl`

That file owns the parsing of the decrypted first registration packet into:

- agent type
- agent id
- beat payload

The base `pl_internal.go` is only a stub and should remain generic.

## Agent Compatibility

When an agent and a listener use the **same protocol**, they share identical:
- Encryption (EncryptData / DecryptData)
- Wire-type definitions (structs, constants)
- Framing (4-byte big-endian length prefix)

To pair a custom agent with this listener, generate both with the same protocol:

```powershell
# Listener
cd listener
.\generator.ps1 -Name myc2 -Protocol adaptix_default

# Agent
cd ..\agent
.\generator.ps1 -Name myc2 -Protocol adaptix_default
```

Both will use the same crypto and wire types from `protocols/adaptix_default/`.

If the agent and listener share the same basename and protocol, the agent generator auto-populates
`config.yaml -> listeners:` with the listener name (`<NameCap><ProtocolCap>`), so the pair is build-selectable in Adaptix without manual YAML edits.
