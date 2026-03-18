# AdaptixC2 Template Generators

Standalone scaffolding toolkit for [AdaptixC2](https://github.com/Adaptix-Framework/AdaptixC2) extender development.
Generates ready-to-implement stub projects for **agents**, **listeners**, **services** (optionally with **post-build wrapper pipeline**), and custom **wire protocols** -- all compatible with the `axc2 v1.2.0` plugin API.

Agent implants can be scaffolded in **Go**, **C++**, or **Rust** (extensible to more).
The server-side plugin is always Go (required by AdaptixC2's `plugin.Open()` loader).
Generators produce **interface stubs and template structures** -- you fill in the implementation.

---

## Table of Contents

- [Requirements](#requirements)
- [Project Structure](#project-structure)
- [Getting Started](#getting-started)
- [Generator Reference](#generator-reference)
- [Protocols](#protocols)
- [Post-Build Wrappers](#post-build-wrappers)
- [Workflow Examples](#workflow-examples)
- [Architecture](#architecture)
- [Build and Deploy](#build-and-deploy)
- [FAQ](#faq)

---

## Requirements

- Go 1.21+ (for plugin builds: Linux with CGO enabled)
- PowerShell 5.1+ (Windows) or Bash 4+ (Linux/macOS)
- No Go dependencies at generation time -- generators are pure script

**Additional toolchains (only for the languages you use):**

| Language | Toolchain | Notes |
|----------|-----------|-------|
| Go | `go build` or `garble` | Default; no extra install needed |
| C++ | MinGW (`x86_64-w64-mingw32-g++`) | Cross-compile Windows PE/DLL/shellcode |
| Rust | `cargo build` | Install via rustup; add cross-compile targets |

---

## Project Structure

```
.
|-- generator.ps1 / generator.sh    Root dispatcher
|-- agent/                          Agent sub-generator + templates + toolchains
|   +-- README.md                   Language support, interfaces, toolchains
|-- listener/                       Listener sub-generator + templates
|   +-- README.md                   Listener-specific docs
|-- service/                        Service sub-generator (supports -Wrapper)
|   +-- README.md                   Service + wrapper pipeline docs
|   +-- templates/wrapper/          Wrapper addon overrides
+-- protocols/                      Wire-protocol definitions + crypto templates
    |-- _crypto/                    Crypto library (aes-gcm, rc4, xchacha20, and optional private templates)
    |-- _scaffold/                  Empty starting point (incl. C++/Rust implant stubs)
    |-- adaptix_gopher/             AES-256-GCM + msgpack
    |-- adaptix_default/            RC4 + binary packing
    +-- ...                         Private/internal protocol overlays (not documented publicly)
```

---

## Getting Started

### Interactive Mode

Run the root dispatcher without arguments:

```powershell
.\generator.ps1          # PowerShell
./generator.sh           # Bash
```

Menu options: Generate Agent, Generate Listener, Generate Service, Create Protocol, Create Crypto, Delete.

### Non-Interactive Mode

```powershell
.\generator.ps1 -Mode agent
.\generator.ps1 -Mode listener
.\generator.ps1 -Mode service -Wrapper      # include wrapper pipeline
.\generator.ps1 -Mode protocol
.\generator.ps1 -Mode crypto
.\generator.ps1 -Mode delete
```

```bash
MODE=agent    ./generator.sh
MODE=listener ./generator.sh
MODE=service WRAPPER=1 ./generator.sh
MODE=protocol ./generator.sh
MODE=crypto   ./generator.sh
MODE=delete   ./generator.sh
```

### Output Directory

Default: `./output/`. Override with `-OutputDir` / `OUTPUT_DIR`, or set `ADAPTIX_OUTPUT_DIR`:

```powershell
.\generator.ps1 -Mode agent -OutputDir ..\AdaptixC2\AdaptixServer\extenders
$env:ADAPTIX_OUTPUT_DIR = "..\AdaptixC2\AdaptixServer\extenders"
```

```bash
MODE=agent OUTPUT_DIR=../AdaptixC2/AdaptixServer/extenders ./generator.sh
export ADAPTIX_OUTPUT_DIR=../AdaptixC2/AdaptixServer/extenders
```

---

## Generator Reference

### Agent

Scaffolds a server-side Go plugin + cross-platform implant (Go/C++/Rust) with interface stubs.

| Parameter | PowerShell | Bash | Default |
|-----------|-----------|------|---------|
| Name | `-Name` | `NAME` | Prompted |
| Watermark | `-Watermark` | `WATERMARK` | Auto-generated 8-char hex |
| Protocol | `-Protocol` | `PROTOCOL` | `adaptix_default` |
| Language | `-Language` | `LANGUAGE` | `go` (interactive menu) |
| Toolchain | `-Toolchain` | `TOOLCHAIN` | Auto-detected per language |

```powershell
.\generator.ps1 -Mode agent -Name phantom -Protocol adaptix_default
.\generator.ps1 -Mode agent -Name beacon -Language cpp
.\generator.ps1 -Mode agent -Name xxx -Language rust -Protocol adaptix_gopher
.\generator.ps1 -Mode agent -Name phantom -Language go -Toolchain go-garble
```

See [`agent/README.md`](agent/README.md) for generated structure, interfaces, toolchains, and implementation guide.

### Listener

Scaffolds a listener plugin with transport loop, crypto, and wire types from the selected protocol.
The base listener template stays protocol-agnostic: protocol-specific framing, transport behavior,
and internal registration parsing are supplied by protocol-owned overrides instead of hardcoded logic
in the core templates.

| Parameter | PowerShell | Bash | Default |
|-----------|-----------|------|---------|
| Name | `-Name` | `NAME` | Prompted |
| Protocol | `-Protocol` | `PROTOCOL` | `adaptix_default` |
| Listener type | `-ListenerType` | `LISTENER_TYPE` | `external` |

```powershell
.\generator.ps1 -Mode listener -Name falcon -Protocol adaptix_default -ListenerType external
```

See [`listener/README.md`](listener/README.md) for generated structure and template placeholders.

When `-ListenerType internal` is used, the generated listener does not open a socket. Instead, it exposes
`InternalHandler()` and relies on a protocol-owned `pl_internal.go.tmpl` override to decode the first
decrypted registration packet into `(agent type, agent id, beat)` for `TsAgentCreate(...)`.

### Service / Wrapper

Scaffolds a service plugin. Add `-Wrapper` / `WRAPPER=1` to include a post-build wrapper pipeline.

| Parameter | PowerShell | Bash | Default |
|-----------|-----------|------|---------|
| Name | `-Name` | `NAME` | Prompted |
| Wrapper | `-Wrapper` | `WRAPPER=1` | No (prompted if omitted) |

```powershell
.\generator.ps1 -Mode service -Name telegram                    # plain service
.\generator.ps1 -Mode service -Name crystalpalace -Wrapper      # with wrapper pipeline
```

See [`service/README.md`](service/README.md) for wrapper stage API and addon architecture.

### Protocol

Creates a new wire-protocol definition from `_scaffold/`:

```powershell
.\generator.ps1 -Mode protocol       # prompts for name
```

This creates `protocols/<name>/` with stub `crypto.go.tmpl`, `constants.go.tmpl`, and `types.go.tmpl`.
Edit these to implement your serialization and encryption, then generate agents/listeners with `-Protocol <name>`.

### Crypto Swap

Replace the crypto implementation in an existing protocol without touching constants or wire types:

```powershell
.\generator.ps1 -Mode crypto         # interactive: select protocol + crypto template
```

Bundled crypto templates in `protocols/_crypto/`:

| Template | Algorithm |
|----------|-----------|
| `aes-gcm.go.tmpl` | AES-256-GCM (authenticated) |
| `rc4.go.tmpl` | RC4 stream cipher |
| `xchacha20.go.tmpl` | XChaCha20-Poly1305 (authenticated) |
| private/internal templates | Additional AEAD / stream cipher variants kept out of the public protocol docs |

You can also create custom `.go.tmpl` files in `_crypto/` -- they are discovered automatically.

### Delete

Remove generated output, protocols, or crypto templates:

```powershell
.\generator.ps1 -Mode delete         # interactive sub-menu
```

Presents options to delete: generated output directories, custom protocols, or crypto templates.
The `_scaffold` and `_crypto` directories are protected from deletion. User-created and bundled protocols can be deleted.

---

## Protocols

A protocol defines the shared **crypto**, **constants**, and **wire types** between an agent and its listener.
When both are generated with the same `-Protocol`, they are guaranteed wire-compatible.

### Bundled Protocols

| Name | Crypto | Framing | Description |
|------|--------|---------|-------------|
| `adaptix_gopher` | AES-256-GCM | msgpack + 4-byte BE length prefix | Gopher-agent compatible |
| `adaptix_default` | RC4 | Binary packing + 4-byte length prefix | Wire-compatible with beacon agents/listeners |
| `_scaffold` | (stub) | (stub) | Empty starting point for custom protocols |

### Compatibility, Quality, and Readiness

The bundled protocols do **not** all have the same maturity level. Distinguish between:

- **wire compatibility** — crypto, framing, constants, registration format, and task/response encoding
- **reference implant completeness** — how many server-requested features are already implemented in the generated Go/C++/Rust implants

| Protocol | Wire compatibility target | Listener/plugin status | Generated implant status | Practical readiness |
|----------|---------------------------|------------------------|--------------------------|---------------------|
| `adaptix_default` | Existing Adaptix beacon agents/listeners | Strong: listener transport and server-side task/response handling follow the beacon-compatible RC4 + binary protocol | Go templates cover the public command surface used by the bundled generator flow, including file ops, process/exec paths, screenshots, BOF sync/async plumbing, downloads/uploads, and job control; C++/Rust remain template scaffolds | Ready for Go-based beacon-compatible template generation and interoperability validation |
| `adaptix_gopher` | Existing Adaptix gopher agents/listeners | Strong: framing and command model match the gopher protocol family | Go templates share the completed runtime path for file ops, process/exec paths, screenshots, BOF sync/async plumbing, and job control; C++/Rust remain template scaffolds | Ready for Go-based gopher-compatible template generation and interoperability validation |

These readiness statements are intentionally **template-scoped**:

- the public wire contracts are ready to import into new agent/listener templates
- the generated Go path is the validated reference path for interoperability work
- the generated C++ and Rust paths remain scaffolds that a developer must finish before expecting feature parity
- advanced transport/pivot surfaces are part of the public wire vocabulary, but a generated implant may still choose to leave some of them as explicit unsupported responses until a developer implements the runtime behavior

The historical implementation language of an original Adaptix family does **not** automatically define the most complete template path in this repository. Readiness is based on the generated code that exists here today, not on upstream lineage.

### Original Adaptix Compatibility

#### `adaptix_default`

`adaptix_default` is the repository's **beacon-wire-compatible** protocol. Its metadata and templates are designed to match the original Adaptix beacon family:

- RC4 encryption
- beacon-style binary packing
- big-endian registration header for listener handshake
- little-endian task packing / big-endian response parsing
- beacon-compatible command identifiers and server-side task processing model

What this means in practice:

- a **new listener** generated with `-Protocol adaptix_default` is intended to interoperate with an **agent that implements the original Adaptix beacon wire format**
- a **new agent** generated with `-Protocol adaptix_default` is intended to interoperate with an **original Adaptix beacon-style listener**, provided the runtime behavior needed by the server command set is actually implemented in that generated implant
- a **new adaptix_default agent + new adaptix_default listener** generated from this repository are expected to be wire-compatible with each other

Important caveat: wire compatibility does **not** automatically mean feature parity. In this repository today, the Go reference implant is further along than the C++ and Rust reference implants for `adaptix_default`; for example, the current C++ and Rust adaptix_default commanders still return explicit unsupported responses for commands such as `run`, `shell`, `screenshot`, `zip`, and `exec_bof` until a developer finishes those runtime handlers.

#### `adaptix_gopher`

`adaptix_gopher` is the repository's **gopher-wire-compatible** protocol. Its metadata and templates target the original Adaptix gopher family:

- AES-256-GCM
- msgpack message model
- 4-byte big-endian length prefix
- gopher-style command and response envelope layout

What this means in practice:

- a **new listener** generated with `-Protocol adaptix_gopher` is intended to interoperate with an **agent that implements the original Adaptix gopher wire format**
- a **new agent** generated with `-Protocol adaptix_gopher` is intended to interoperate with an **original Adaptix gopher listener**, provided you stay within the command surface already implemented in the generated implant
- a **new adaptix_gopher agent + new adaptix_gopher listener** generated from this repository are expected to interoperate with each other

The public Go template path now includes BOF sync/async plumbing and background job control, so `adaptix_gopher` can be treated as the supported msgpack/AES-GCM compatibility option for generator-produced Go agents and listeners.

Private/internal protocols may exist in this repository, but they are intentionally not documented as public bundled options here.

### Which protocol should I use?

Use:

- `adaptix_default` when you want **beacon-family interoperability** or a binary protocol that tracks the original Adaptix beacon model
- `adaptix_gopher` when you want **gopher-family interoperability** or a msgpack/AES-GCM protocol with simpler extension ergonomics

### Recommended Usage Pattern

For production-facing work, treat the bundled protocols like this:

1. Choose the protocol for the listener/agent family you need
2. Generate **both** sides with the same `-Protocol`
3. Use the generated code as the authoritative wire contract
4. If you are not using Go for the implant, review the language-specific stubs before assuming feature parity
5. Validate against your target listener/agent pair with the exact commands you expect operators to use

### Protocol File Layout

```
protocols/<name>/
|-- meta.yaml           Protocol metadata
|-- crypto.go.tmpl      EncryptData / DecryptData
|-- constants.go.tmpl   COMMAND_* and RESP_* constants
|-- types.go.tmpl       Wire types, framing, serialization
+-- implant/            Language-specific implant overrides (optional)
    |-- *.go.tmpl       Go implant overrides (root = Go)
    |-- cpp/            C++ implant overlay files
    +-- rust/           Rust implant overlay files
```

Optional overrides: `pl_main.go.tmpl` (replaces plugin logic) and `pl_transport.go.tmpl` (replaces listener transport) for protocols that need different command packing or framing.
If a protocol supports internal listeners, add `pl_internal.go.tmpl` to own the parsing of the decrypted
registration packet. The core listener template must remain protocol-agnostic.
The `implant/` directory provides per-language source overrides for the implant side (protocol structs, tasks, main loop, etc.).

The `__PACKAGE__` placeholder is context-dependent:

| Target | Package value |
|--------|--------------|
| Plugin / listener code | `main` |
| Agent crypto package | `crypto` |
| Agent protocol package | `protocol` |

### Creating a Custom Protocol

1. `.\generator.ps1 -Mode protocol` -- copies `_scaffold/` to `protocols/<name>/` (includes C++/Rust implant stubs)
2. Implement `crypto.go.tmpl`, `constants.go.tmpl`, `types.go.tmpl`
3. Add language-specific implant overrides in `implant/` if needed (Go at root, `cpp/`, `rust/`)
4. Update `meta.yaml`
5. If the protocol supports internal listeners, add `pl_internal.go.tmpl` to parse the decrypted registration packet
6. Generate with `-Protocol <name>` and validate with `go vet`

---

## Post-Build Wrappers

A **wrapper** is a service plugin (generated with `-Wrapper`) that hooks `agent.generate` (post-phase) and transforms the generated payload before it reaches the operator. This enables RDLL loading, sleep obfuscation, shellcode encryption, and more -- without modifying the agent.

Inspired by [Adaptix-StealthPalace](https://github.com/MaorSabag/Adaptix-StealthPalace) by MaorSabag.

### How It Works

1. Operator clicks "Build Agent" in the Adaptix UI
2. Teamserver compiles the agent and fires `agent.generate` (post-phase)
3. Wrapper intercepts via `TsEventHookRegister`, receives the payload via `reflect`
4. Pipeline stages run in order, modifying `FileContent` in-place
5. Teamserver returns the wrapped payload to the operator

The wrapper intercepts **all** agent builds automatically.

### Pipeline Stages

Each stage is a function:

```go
func(payload []byte, cfg map[string]string, ctx *BuildContext) ([]byte, error)
```

- Stages run in registration order
- Enable/disable at runtime via config keys: `stage.<name>.enabled`
- On error, the pipeline stops and the original payload is returned
- Use `logBuild(ctx.BuilderID, BuildLogInfo, "message")` for build log output

See [`service/README.md`](service/README.md) for stage registration and the wrapper template API.

---

## Workflow Examples

### New Go Agent from Scratch

```powershell
.\generator.ps1 -Mode agent -Name phoenix -Protocol adaptix_default

cd output\phoenix_agent\src_phoenix\impl
# Edit agent_linux.go, agent_windows.go -- fill in // TODO: stubs

cd ..\..
go mod tidy && cd src_phoenix && go mod tidy && cd ..
make full

# Deploy: copy agent_phoenix.so, config.yaml, ax_config.axs to server extenders dir
```

### Agent + Listener Pair

```powershell
# Same protocol = wire-compatible
.\generator.ps1 -Mode agent    -Name falcon -Protocol adaptix_default
.\generator.ps1 -Mode listener -Name falcon -Protocol adaptix_default -ListenerType external
```

When the agent and listener share the same base name and protocol, the agent generator auto-binds
`config.yaml -> listeners:` to the generated listener name (`<NameCap><ProtocolCap>`). Override this only when you
intentionally want one agent to advertise multiple listener names.

### Wrapper + Agent (Crystal Palace-style)

```powershell
.\generator.ps1 -Mode agent -Name phoenix -Protocol adaptix_default
.\generator.ps1 -Mode service -Name crystalpalace -Wrapper

# Implement agent stubs in output/phoenix_agent/src_phoenix/impl/
# Register wrapper stages in output/crystalpalace_wrapper/pl_main.go → initStages()

cd output\phoenix_agent && go mod tidy && make full
cd ..\crystalpalace_wrapper && go mod tidy && make plugin
# Deploy both .so plugins -- wrapper hooks all agent builds automatically
```

### Custom Protocol

```powershell
.\generator.ps1 -Mode protocol          # creates protocols/<name>/
# Edit crypto.go.tmpl, types.go.tmpl, constants.go.tmpl

.\generator.ps1 -Mode agent    -Name stealth -Protocol myproto
.\generator.ps1 -Mode listener -Name stealth -Protocol myproto
```

---

## Architecture

```
              +-------------------+
              |  generator.ps1/sh |  root dispatcher
              +---------+---------+
  +------+------+------+------+------+------+
  v      v      v      v      v      v      v
agent/ listener/ service/   proto/ crypto/ delete
```

**Data flow:**

1. Root dispatcher selects sub-generator based on `-Mode` / menu choice
2. Sub-generator reads `.tmpl` files from `templates/` and `protocols/<name>/`
3. Placeholders (`__NAME__`, `__WATERMARK__`, `__PROTOCOL__`, `__PACKAGE__`, etc.) are substituted
4. Rendered files are written to the output directory

**Communication layers:**

```
Agent Implant  <----[protocol]---->  Listener Plugin  <----[axc2 msgpack]---->  AdaptixC2 Server
     ^                                    ^                                          ^
     |                                    |                                          |
  You implement                     You implement                         Fixed API (axc2 v1.2.0)
  (impl/*.go)                       (pl_transport.go)
```

The design rule is extensibility-first:

- core generator logic should stay protocol-agnostic
- protocol-specific behavior belongs in protocol-owned override files
- adding a new protocol should prefer adding files under `protocols/<name>/` over adding name-based branching to the generators

---

## Build and Deploy

### Agent

```bash
cd <name>_agent/
go mod tidy
cd src_<name> && go mod tidy && cd ..   # Go implant only
make full                                # builds plugin .so + implant
```

C++ uses MinGW cross-compilation. Rust requires `rustup target add` for cross-compile targets.

### Listener

```bash
cd <name>_listener/
go mod tidy && make plugin
```

### Service / Wrapper

```bash
cd <name>_service/       # or <name>_wrapper/
go mod tidy && make plugin
```

### Deployment

Copy built artifacts to the AdaptixC2 server extenders directory:

```
AdaptixServer/data/extenders/<name>_agent/
    agent_<name>.so, config.yaml, ax_config.axs

AdaptixServer/data/extenders/<name>_listener/
    listener_<name>.so, config.yaml, ax_config.axs

AdaptixServer/data/extenders/<name>_service/   (or <name>_wrapper/)
    service_<name>.so, config.yaml, ax_config.axs
```

Restart the AdaptixC2 server to load new extenders.

---

## FAQ

**Do I need the AdaptixC2 source code?**
No. Generators are standalone. Generated code depends only on the public `axc2 v1.2.0` module.

**Can I generate on Windows and build on Linux?**
Yes. Generate on any OS, transfer to Linux for `go build -buildmode=plugin`.

**Can I use C++/Rust for the implant but Go for the plugin?**
Yes -- that's exactly how multi-language support works. The plugin is always Go.

**How do I add new commands to a generated agent?**
See [`agent/README.md`](agent/README.md). In short: add constants in `pl_utils.go` + `protocol/protocol.go`, register in `ax_config.axs`, handle in `pl_main.go` + `tasks.go`.

**Can I use protobuf, flatbuffers, JSON?**
Yes. Create a custom protocol with your serialization in `types.go.tmpl`.

**How do I add a new implant language?**
See `AGENTS.md` -- create template dir, toolchain YAML, build variant, and register in the generator.
