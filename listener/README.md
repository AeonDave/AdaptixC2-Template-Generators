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
Bundled: `gopher` (AES-256-GCM + msgpack), `adaptix_default` (RC4 + binary packing).

See the root README for protocol creation, crypto swap, and file layout docs.

## Generated Listener Structure

```
<name>_listener_<protocol>/
├── config.yaml          # Listener manifest
├── go.mod               # Go module (axc2 v1.2.0)
├── Makefile             # Build targets: plugin, dist
├── pl_main.go           # InitPlugin + Create/Start/Stop/Edit/GetProfile
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
cd <name>_listener_<protocol>/
go mod tidy
make plugin        # builds .so
make dist          # copies .so + config.yaml + ax_config.axs to dist/
```

Copy the `dist/` contents into `AdaptixServer/data/extenders/<name>_listener_<protocol>/`.
Alternatively, generate directly into the extenders directory with `-OutputDir`.

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
