# Template Listener Generator

Scaffold new AdaptixC2 **listener** plugins from templates, with selectable wire-format **protocols** shared between agents and listeners.

## Quick Start

### Via Root Dispatcher (recommended)

```powershell
.\generator.ps1             # then select "2) Generate Listener"
.\generator.ps1 -Mode listener  # or skip the menu
.\generator.ps1 -Mode listener -OutputDir ..\AdaptixC2\AdaptixServer\extenders
```
```bash
./generator.sh              # then select "2) Generate Listener"
MODE=listener ./generator.sh  # or skip the menu
MODE=listener OUTPUT_DIR=../AdaptixC2/AdaptixServer/extenders ./generator.sh
```

### Direct

```powershell
cd listener

# Interactive — prompts for name, protocol, type
.\generator.ps1

# Non-interactive
.\generator.ps1 -Name telegram -Protocol default -ListenerType external

# With output dir
.\generator.ps1 -Name telegram -Protocol default -ListenerType external -OutputDir ..\..\AdaptixC2\AdaptixServer\extenders
```

```bash
cd listener

# Interactive
bash generator.sh

# Non-interactive (env vars)
PROTOCOL=default LISTENER_TYPE=external bash generator.sh

# With output dir
PROTOCOL=default LISTENER_TYPE=external OUTPUT_DIR=../../AdaptixC2/AdaptixServer/extenders bash generator.sh
```

## Protocols

Protocols live in `protocols/<name>/` and define crypto, constants, and wire types shared by both agents and listeners.

| Directory | Description |
|-----------|-------------|
| `protocols/default/` | AES-256-GCM + msgpack — compatible with stock gopher |
| `protocols/_scaffold/` | Empty template for new protocols |

### Creating a custom protocol

Use the dedicated protocol generator:

```powershell
cd protocols
.\generator.ps1 -Name myproto
```

```bash
cd protocols
NAME=myproto bash generator.sh
```

Or via root dispatcher:

```powershell
.\generator.ps1 -Mode protocol
```

This copies `_scaffold/` into `protocols/myproto/`. Edit the `.tmpl` files to implement your own crypto and framing.

### Swapping crypto on an existing protocol

```powershell
.\generator.ps1 -Mode crypto
```

Or directly:

```powershell
cd protocols
.\crypto_generator.ps1 -Protocol myproto -Crypto xchacha20
```

This generates or replaces `crypto.go.tmpl` in the protocol directory. Regenerate
your listener after swapping to pick up the new crypto.

### Protocol file layout

```
protocols/<name>/
├── meta.yaml           # Protocol metadata (name, version, description)
├── crypto.go.tmpl      # EncryptData / DecryptData functions
├── constants.go.tmpl   # COMMAND_*, RESP_* constants
└── types.go.tmpl       # Wire types, framing helpers, zip utilities
```

The `__PACKAGE__` placeholder is replaced with the target Go package during generation:
- `main` for listener code (flat package)
- `crypto` for agent crypto package
- `protocol` for agent protocol package

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
| `__PROTOCOL__` | Protocol name (lowercase) | `default` |
| `__PROTOCOL_CAP__` | Protocol capitalised | `Default` |
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
.\generator.ps1 -Name myc2 -Protocol default

# Agent
cd ..\agent
.\generator.ps1 -Name myc2 -Protocol default
```

Both will use the same crypto and wire types from `protocols/default/`.
