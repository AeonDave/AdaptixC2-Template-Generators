# AdaptixC2 Template Generators

Standalone scaffolding toolkit for [AdaptixC2](https://github.com/Adaptix-Framework/AdaptixC2) extender development.
Generates ready-to-implement stub projects for **agents**, **listeners**, and custom **wire protocols** -- all compatible with the `axc2 v1.2.0` plugin API.

Agent implants can be scaffolded in **Go**, **C++**, or **Rust**.
But can be extended to more languages.
The server-side plugin is always Go (required by AdaptixC2's `plugin.Open()` loader).

The generators do not produce finished, working code.
They produce **interface stubs and template structures** that define the contracts an extender must fulfill.
The developer fills in the platform-specific and protocol-specific logic.

---

## Table of Contents

- [Overview](#overview)
- [Requirements](#requirements)
- [Project Structure](#project-structure)
- [Getting Started](#getting-started)
  - [Interactive Mode](#interactive-mode)
  - [Non-Interactive Mode](#non-interactive-mode)
  - [Output Directory](#output-directory)
- [Generator Reference](#generator-reference)
  - [Agent Generator](#agent-generator)
  - [Listener Generator](#listener-generator)
  - [Service Generator](#service-generator)
  - [Protocol Generator](#protocol-generator)
  - [Crypto Swap](#crypto-swap)
  - [Delete](#delete)
- [Multi-Language Support](#multi-language-support)
- [Protocols](#protocols)
  - [Bundled Protocols](#bundled-protocols)
  - [Creating a Custom Protocol](#creating-a-custom-protocol)
  - [Protocol File Layout](#protocol-file-layout)
- [What Gets Generated](#what-gets-generated)
  - [Agent Output](#agent-output)
  - [Listener Output](#listener-output)
- [Agent Interfaces](#agent-interfaces)
- [Workflow Examples](#workflow-examples)
  - [Example 1 -- New Go Agent from Scratch](#example-1----new-go-agent-from-scratch)
  - [Example 2 -- C++ Agent](#example-2----c-agent-windows-pedllshellcode)
  - [Example 3 -- Rust Agent](#example-3----rust-agent-cross-platform)
  - [Example 4 -- Agent + Listener Pair](#example-4----agent--listener-pair)
  - [Example 5 -- Custom Protocol with Protobuf](#example-5----custom-protocol-with-protobuf)
- [Architecture](#architecture)
- [Build and Deploy](#build-and-deploy)
- [FAQ](#faq)
- [License](#license)

---

## Overview

AdaptixC2 loads extenders as Go plugins (`.so` files built with `-buildmode=plugin`).
Each extender must export an `InitPlugin` function and conform to the interfaces defined in the
[axc2](https://github.com/Adaptix-Framework/axc2) module.

Writing an extender from scratch involves substantial boilerplate: module setup, Makefile,
wire-type definitions, crypto wrappers, UI registration, and platform stubs.
This toolkit automates that scaffolding so you can focus on implementation.

**What the generators create:**

| Component | What you get | What you implement |
|-----------|-------------|-------------------|
| **Agent** | Plugin skeleton + implant with interface stubs | Platform methods in `impl/agent_<os>.go` |
| **Listener** | Full plugin skeleton with transport loop | Connection handling in `pl_transport.go` |
| **Protocol** | Crypto, constants, and wire-type template files | Your serialization and encryption logic |

---

## Requirements

- Go 1.21+ (for plugin builds: Linux with CGO enabled)
- PowerShell 5.1+ (Windows) or Bash 4+ (Linux/macOS)
- No Go dependencies at generation time -- generators are pure script

**Additional toolchains (only for the languages you use):**

| Language | Toolchain | Notes |
|----------|-----------|-------|
| Go | `go build` (standard) or `garble` | Default; no extra install needed |
| C++ | MinGW (`x86_64-w64-mingw32-g++`) | Cross-compile Windows PE/DLL/shellcode |
| Rust | `cargo build` | Install via rustup; add cross-compile targets with `rustup target add` |

---

## Project Structure

```
.
|-- generator.ps1              Root dispatcher (PowerShell)
|-- generator.sh               Root dispatcher (Bash)
|-- README.md
|
|-- agent/
|   |-- generator.ps1          Agent sub-generator
|   |-- generator.sh
|   |-- README.md              Detailed agent documentation
|   |-- toolchains/            Toolchain manifests (YAML)
|   |   |-- go-standard.yaml
|   |   |-- go-garble.yaml
|   |   |-- mingw.yaml
|   |   +-- cargo.yaml
|   +-- templates/
|       |-- plugin/            Server-side Go plugin templates
|       +-- implant/
|           |-- go/             Go implant templates
|           |-- cpp/            C++ implant templates
|           +-- rust/           Rust implant templates
|
|-- listener/
|   |-- generator.ps1          Listener sub-generator
|   |-- generator.sh
|   |-- README.md              Detailed listener documentation
|   +-- templates/             Go template files for listeners
|
|-- service/
|   |-- generator.ps1          Service sub-generator
|   |-- generator.sh
|   |-- README.md              Service documentation
|   +-- templates/             Go template files for services
|
+-- protocols/
    |-- generator.ps1          Protocol scaffold generator
    |-- generator.sh
    |-- crypto_generator.ps1   Crypto implementation swap tool
    |-- crypto_generator.sh
    |-- _crypto/               Crypto template library (discovered at runtime)
    |   |-- aes-gcm.go.tmpl
    |   |-- xchacha20.go.tmpl
    |   |-- rc4.go.tmpl
    |   +-- xor_custom.go.tmpl  (scaffold -- implement yourself)
    |-- _scaffold/             Empty starting point for new protocols
    |-- gopher/                AES-256-GCM + msgpack (gopher-agent compatible)
    +-- adaptix_default/       RC4 + binary packing (wire-compatible with beacon)
```

---

## Getting Started

### Interactive Mode

Run the root dispatcher without arguments. A numbered menu lets you pick the operation:

**PowerShell:**
```powershell
.\generator.ps1
```

**Bash:**
```bash
./generator.sh
```

The menu presents six options:

```
1) Generate Agent     - Scaffold a new agent extender
2) Generate Listener  - Scaffold a new listener extender
3) Generate Service   - Scaffold a new service extender
4) Create Protocol    - Create a new wire-protocol definition
5) Swap Crypto        - Generate or replace the crypto template for a protocol
6) Delete             - Remove a crypto template, protocol, or generated output
```

Each option launches the corresponding sub-generator and prompts for the required parameters
(name, watermark, protocol, language, toolchain, listener type, etc.).

### Non-Interactive Mode

Pass `-Mode` (PowerShell) or `MODE=` (Bash) to skip the menu.
Sub-generator parameters can be passed inline.

**PowerShell:**
```powershell
.\generator.ps1 -Mode agent
.\generator.ps1 -Mode listener
.\generator.ps1 -Mode service
.\generator.ps1 -Mode protocol
.\generator.ps1 -Mode crypto
.\generator.ps1 -Mode delete
```

**Bash:**
```bash
MODE=agent    ./generator.sh
MODE=listener ./generator.sh
MODE=service  ./generator.sh
MODE=protocol ./generator.sh
MODE=crypto   ./generator.sh
MODE=delete   ./generator.sh
```

### Output Directory

By default, generated projects are written to `./output/`.
To write directly into an AdaptixC2 installation, specify the extenders directory:

**Resolution order (first non-empty wins):**

| Priority | PowerShell | Bash |
|----------|-----------|------|
| 1 | `-OutputDir` parameter | `OUTPUT_DIR` env var |
| 2 | `$env:ADAPTIX_OUTPUT_DIR` | `ADAPTIX_OUTPUT_DIR` env var |
| 3 | `./output/` (default) | `./output/` (default) |

**Examples:**

```powershell
# Inline
.\generator.ps1 -Mode agent -OutputDir ..\AdaptixC2\AdaptixServer\extenders

# Environment variable (persists for the session)
$env:ADAPTIX_OUTPUT_DIR = "..\AdaptixC2\AdaptixServer\extenders"
.\generator.ps1 -Mode agent
```

```bash
# Inline
MODE=agent OUTPUT_DIR=../AdaptixC2/AdaptixServer/extenders ./generator.sh

# Environment variable
export ADAPTIX_OUTPUT_DIR=../AdaptixC2/AdaptixServer/extenders
MODE=agent ./generator.sh
```

---

## Generator Reference

### Agent Generator

Scaffolds a complete agent extender: server-side plugin + cross-platform implant with
interface stubs for every supported OS.

**Parameters:**

| Parameter | PowerShell | Bash env var | Required | Default |
|-----------|-----------|-------------|----------|---------|
| Name | `-Name` | `NAME` | Yes (prompted if empty) | -- |
| Watermark | `-Watermark` | `WATERMARK` | No | Auto-generated 8-char hex |
| Protocol | `-Protocol` | `PROTOCOL` | No | Prompted (default: `adaptix_default`) |
| Language | `-Language` | `LANGUAGE` | No | Prompted (default: `go`) |
| Toolchain | `-Toolchain` | `TOOLCHAIN` | No | Prompted when multiple available |
| Output dir | `-OutputDir` | `OUTPUT_DIR` | No | `./output/` |

**Interactive prompts:**

When `-Language` or `-Toolchain` are not passed, the generator presents interactive menus:

```
Select implant language:
  [1] go (default)   - Go implant
  [2] cpp            - C/C++ implant
  [3] rust           - Rust implant

Available toolchains for 'go':
  [1] go-standard (default)  - Standard Go compiler (CGO_ENABLED=0, cross-platform)
  [2] go-garble              - Garble obfuscator (symbol/string obfuscation)
```

Languages are discovered from `agent/templates/implant/`. Toolchains are discovered from
`agent/toolchains/*.yaml`, filtered by the `language:` field in each YAML. If only one
toolchain matches the language, it is auto-selected without prompting.

Supported languages: `go`, `cpp`, `rust`

| Language | Default toolchain | Alternatives |
|----------|------------------|--------------|
| `go` | `go-standard` | `go-garble` |
| `cpp` | `mingw` | -- |
| `rust` | `cargo` | -- |

```powershell
# Fully non-interactive
.\generator.ps1 -Mode agent -Name phantom -Watermark a1b2c3d4 -Protocol adaptix_default

# C++ agent with MinGW
.\generator.ps1 -Mode agent -Name beacon -Language cpp

# Rust agent
.\generator.ps1 -Mode agent -Name stealth -Language rust

# Go agent with garble obfuscation
.\generator.ps1 -Mode agent -Name phantom -Language go -Toolchain go-garble

# Via sub-generator directly
cd agent
.\generator.ps1 -Name phantom -Protocol adaptix_default -OutputDir ..\..\my-adaptix\extenders
```

```bash
MODE=agent NAME=phantom WATERMARK=a1b2c3d4 PROTOCOL=adaptix_default ./generator.sh
MODE=agent NAME=beacon LANGUAGE=cpp ./generator.sh
MODE=agent NAME=stealth LANGUAGE=rust ./generator.sh
```

### Listener Generator

Scaffolds a listener plugin with transport loop, crypto, and wire-type definitions.

**Parameters:**

| Parameter | PowerShell | Bash env var | Required | Default |
|-----------|-----------|-------------|----------|---------|
| Name | `-Name` | `NAME` | Yes (prompted if empty) | -- |
| Protocol | `-Protocol` | `PROTOCOL` | No | `adaptix_default` |
| Listener type | `-ListenerType` | `LISTENER_TYPE` | No | `external` |
| Output dir | `-OutputDir` | `OUTPUT_DIR` | No | `./output/` |

```powershell
.\generator.ps1 -Mode listener -Name telegram -Protocol adaptix_default -ListenerType external
```

```bash
MODE=listener NAME=telegram PROTOCOL=adaptix_default LISTENER_TYPE=external ./generator.sh
```

### Protocol Generator

Creates a new protocol definition from the `_scaffold/` template.
Protocol files live inside this repository under `protocols/<name>/` and are referenced
at generation time -- they are not written to the output directory.

```powershell
.\generator.ps1 -Mode protocol
# or directly:
cd protocols
.\generator.ps1 -Name myprotobuf
```

```bash
MODE=protocol NAME=myprotobuf ./generator.sh
```

### Service Generator

Scaffolds a server-side service plugin. Services are pure server-side logic (no implant,
no transport) — think notifications, integrations, or utilities.

**Parameters:**

| Parameter | PowerShell | Bash env var | Required | Default |
|-----------|-----------|-------------|----------|---------|
| Name | `-Name` | `NAME` | Yes (prompted if empty) | -- |
| Output dir | `-OutputDir` | `OUTPUT_DIR` | No | `./output/` |

```powershell
.\generator.ps1 -Mode service -Name telegram
```

```bash
MODE=service NAME=telegram ./generator.sh
```

### Crypto Swap

Generates or replaces the crypto implementation (`.go.tmpl`) of an existing protocol.

Crypto templates are **discovered dynamically** from `protocols/_crypto/*.go.tmpl`.
The first-line `//` comment in each file is used as the menu description.
Adding a new `.go.tmpl` file to `_crypto/` makes it appear automatically on the next run.

**Bundled crypto templates:**

| Key | Algorithm | Notes |
|-----|-----------|-------|
| `aes-gcm` | AES-256-GCM | Standard, hardware-accelerated on most platforms |
| `xchacha20` | XChaCha20-Poly1305 | 24-byte nonce, requires `golang.org/x/crypto` |
| `rc4` | RC4 | Wire-compatible with existing beacon agents/listeners |
| `xor_custom` | XOR (scaffold) | Stub -- implement `EncryptData`/`DecryptData` yourself |

**Interactive menu:**

```
Available crypto implementations:
  [1] aes-gcm    - AES-256-GCM (standard, fast, widely supported)
  [2] rc4        - RC4 (wire-compatible with existing beacon agents/listeners)
  [3] xchacha20  - XChaCha20-Poly1305 (modern, nonce-misuse resistant)
  [4] xor_custom - XOR Custom (TODO: implement your custom XOR-based crypto)
  [5] Create new...
```

Selecting **Create new...** prompts for a name and description, then scaffolds a new
`.go.tmpl` in `_crypto/` with stub `EncryptData`/`DecryptData` functions.
On the next run the new crypto appears in the menu automatically.

```powershell
.\generator.ps1 -Mode crypto
# or directly:
cd protocols
.\crypto_generator.ps1 -Protocol myprotobuf -Crypto xchacha20
```

```bash
MODE=crypto PROTOCOL=myprotobuf CRYPTO=xchacha20 ./generator.sh
```

**Adding a custom crypto (manual):**

Create `protocols/_crypto/<name>.go.tmpl` with this structure:

```go
// Short description shown in menu
package __PACKAGE__

var SKey []byte

func EncryptData(data, key []byte) ([]byte, error) { /* ... */ }
func DecryptData(data, key []byte) ([]byte, error) { /* ... */ }
```

### Delete

Interactively remove a crypto template, protocol definition, or generated output project.
All deletions require explicit `y` confirmation.

```powershell
.\generator.ps1 -Mode delete
```

```bash
MODE=delete ./generator.sh
```

**Sub-menu:**

```
What do you want to delete?

  [1] Crypto template  - Remove a crypto .go.tmpl from _crypto/
  [2] Protocol         - Remove an entire protocol definition
  [3] Generated output - Remove a generated project from output/
```

| Target | What is removed | Protected items |
|--------|----------------|-----------------|
| Crypto template | Single `.go.tmpl` from `protocols/_crypto/` | None |
| Protocol | Entire `protocols/<name>/` directory | `_scaffold`, `_crypto` (hidden from list) |
| Generated output | Entire `output/<name>/` directory | None |

---

## Multi-Language Support

Agent implants can be scaffolded in multiple languages.
The server-side plugin (`.so`) is always Go -- only the implant source code changes.

| Language | Implant structure | Build tool | Output formats |
|----------|------------------|------------|----------------|
| **Go** (default) | `src_<name>/` with `go.mod`, `impl/`, `crypto/`, `protocol/` | `go build` or `garble` | ELF, PE |
| **C++** | `src_<name>/` with Makefile, `.cpp`/`.h` files | MinGW (`x86_64-w64-mingw32-g++`) | Exe, Service Exe, DLL, Shellcode |
| **Rust** | `src_<name>/` with `Cargo.toml`, `src/` | `cargo build` | ELF, PE |

### Toolchains

Each language has a default toolchain. Toolchain manifests live in `agent/toolchains/` as YAML files.

```yaml
# agent/toolchains/mingw.yaml
name: "MinGW"
language: "cpp"
command: "x86_64-w64-mingw32-g++"
targets:
  - os: windows
    arch: x86_64
```

The `command:` field replaces the `__BUILD_TOOL__` placeholder in Makefiles and build scripts.

### Language-Specific UI

Each language can have its own `ax_config.axs` UI definition:

| Language | Config file | UI fields |
|----------|------------|-----------|
| Go | `ax_config.axs` | OS, Arch, Win7 support |
| C++ | `ax_config_cpp.axs` | Arch (x64/x86), Format (Exe/DLL/Shellcode), Service name |
| Rust | `ax_config.axs` (default) | OS, Arch |

### Build Variants

Build logic is language-specific and lives in `pl_build.go`:

- **Go**: Writes `config.go` with encrypted profiles, runs `go build` with cross-compile env vars
- **C++**: Writes `profile_gen.h` with `#define PROFILE`, runs `make` with format/arch overrides
- **Rust**: Writes `src/config.rs` with profile byte slices, runs `cargo build --target`

---

## Protocols

A protocol defines the shared **crypto**, **constants**, and **wire types** used between an
agent implant and its listener. When both are generated with the same protocol name, they
are guaranteed to be wire-compatible.

### Bundled Protocols

| Name | Crypto | Framing | Description |
|------|--------|---------|-------------|
| `gopher` | AES-256-GCM | msgpack + 4-byte BE length prefix | Gopher-agent compatible |
| `adaptix_default` | RC4 | Binary packing + 4-byte length prefix | Wire-compatible with existing beacon agents/listeners |
| `_scaffold` | (stub) | (stub) | Empty starting point for custom protocols |

### Creating a Custom Protocol

1. Run the protocol generator:
   ```powershell
   .\generator.ps1 -Mode protocol
   ```
   This copies `_scaffold/` to `protocols/<name>/`.

2. Edit the three template files in `protocols/<name>/`:
   - `crypto.go.tmpl` -- implement `EncryptData()` and `DecryptData()`
   - `constants.go.tmpl` -- define `COMMAND_*` and `RESP_*` constants
   - `types.go.tmpl` -- define wire structs, framing helpers, serialization

3. Update `meta.yaml` with a description of your protocol.

4. Generate an agent and/or listener using your new protocol:
   ```powershell
   .\generator.ps1 -Mode agent -Name myagent -Protocol myprotobuf
   .\generator.ps1 -Mode listener -Name mylistener -Protocol myprotobuf
   ```

### Protocol File Layout

```
protocols/<name>/
|-- meta.yaml           Protocol metadata (name, version, description)
|-- crypto.go.tmpl      EncryptData / DecryptData function stubs
|-- constants.go.tmpl   COMMAND_* and RESP_* constant definitions
+-- types.go.tmpl       Wire types, framing helpers, serialization utilities
```

The `__PACKAGE__` placeholder in `.tmpl` files is replaced at generation time:

| Target | Package value |
|--------|--------------|
| Listener code (flat package) | `main` |
| Agent crypto package | `crypto` |
| Agent protocol package | `protocol` |

---

## What Gets Generated

### Agent Output

The server-side plugin files are identical regardless of language.
The implant directory (`src_<name>/`) varies by language:

**Go implant** (`-Language go`, default):

```
<name>_agent/
|-- config.yaml              Plugin manifest (name, watermark, listeners)
|-- go.mod                   Plugin Go module (depends on axc2 v1.2.0)
|-- Makefile                 Build targets: plugin, full
|-- pl_utils.go              Wire types and command constants (from protocol)
|-- pl_main.go               Server-side plugin logic
|-- pl_build.go              Build logic (Go: writes config.go, runs go build)
|-- ax_config.axs            UI and command registration (AxScript)
+-- src_<name>/
    |-- go.mod               Implant Go module
    |-- Makefile             Cross-platform implant build
    |-- config.go            Encrypted connection profile placeholder
    |-- main.go              Connection loop
    |-- tasks.go             Command dispatch
    |-- crypto/
    |   +-- crypto.go        AES-256-GCM (ready to use)
    |-- protocol/
    |   +-- protocol.go      Wire types & framing (ready to use)
    +-- impl/
        |-- interfaces.go    Interface contracts (DO NOT EDIT)
        |-- agent.go         Cross-platform: Stealth + Transport defaults
        |-- agent_linux.go   Linux stubs         <-- IMPLEMENT
        |-- agent_windows.go Windows stubs       <-- IMPLEMENT
        +-- agent_darwin.go  macOS stubs         <-- IMPLEMENT
```

**C++ implant** (`-Language cpp`):

```
<name>_agent/
|-- (same plugin files as above)
|-- pl_build.go              Build logic (C++: writes profile_gen.h, runs make)
|-- ax_config.axs            C++ specific: arch, format (Exe/DLL/Shellcode), svc_name
+-- src_<name>/
    |-- Makefile             MinGW cross-compile (Exe, DLL, Shellcode targets)
    |-- main.cpp             Entry point
    |-- config.h / config.cpp
    |-- agent.h / agent.cpp  Agent logic
    |-- crypto.h / crypto.cpp
    |-- protocol.h / protocol.cpp
    +-- impl/
        +-- agent_windows.h / agent_windows.cpp  <-- IMPLEMENT
```

**Rust implant** (`-Language rust`):

```
<name>_agent/
|-- (same plugin files as above)
|-- pl_build.go              Build logic (Rust: writes src/config.rs, runs cargo build)
|-- ax_config.axs            OS + arch selection (same as Go)
+-- src_<name>/
    |-- Cargo.toml           Rust manifest (release profile optimised for size)
    |-- Makefile             cargo build targets (linux, windows cross-compile)
    +-- src/
        |-- main.rs          Entry point
        |-- config.rs        Profile data (populated at build time)
        |-- crypto.rs        Encrypt/decrypt stubs
        |-- protocol.rs      Wire protocol + watermark
        +-- agent.rs         Connector trait + Agent struct  <-- IMPLEMENT
```

### Listener Output

```
<name>_listener_<protocol>/
|-- config.yaml              Listener manifest
|-- go.mod                   Go module (depends on axc2 v1.2.0)
|-- Makefile                 Build targets: plugin, dist
|-- pl_main.go               InitPlugin + Create/Start/Stop/Edit/GetProfile
|-- pl_transport.go          Transport loop: accept, handleConnection  <-- IMPLEMENT
|-- pl_crypto.go             EncryptData / DecryptData (from protocol)
|-- pl_utils.go              Wire types + constants (from protocol)
|-- map.go                   Thread-safe concurrent map utility
+-- ax_config.axs            Listener UI form (AxScript)
```

The main file to customize is `pl_transport.go`, which contains the connection accept
loop and per-connection handler.

---

## Agent Interfaces

The agent generator produces Go interfaces in `impl/interfaces.go`.
Each platform file (`agent_linux.go`, `agent_windows.go`, `agent_darwin.go`) must
implement these methods. Stubs with `// TODO:` markers are provided.

**Stealth** -- anti-analysis and startup hooks:

```go
type Stealth interface {
    IsDebugged() bool    // Return true to abort execution
    Masquerade()         // Process masquerading (e.g., PPID spoofing)
    OnStart()            // One-time initialization before the main loop
}
```

**Platform** -- OS-level information:

```go
type Platform interface {
    GetCP() uint32                 // Console code page (65001 = UTF-8)
    IsElevated() bool              // Running as root / administrator
    GetOsVersion() string          // e.g., "Windows 11 23H2", "Ubuntu 22.04"
    NormalizePath(p string) string // Expand ~ or . to absolute path
}
```

**FileSystem** -- directory listing, file/directory copy:

```go
type FileSystem interface {
    GetListing(dir string) (string, []protocol.DirEntry, error)
    CopyFile(src, dst string) error
    CopyDir(src, dst string) error
}
```

**Execution** -- shell commands, process listing, screenshots:

```go
type Execution interface {
    RunShell(cmd string, risky, piped bool) (string, error)
    ListProcesses() ([]protocol.ProcessEntry, error)
    CaptureScreenshot() ([]byte, error)
}
```

**Transport** -- network connection to the listener:

```go
type Transport interface {
    Dial(address string, profile *protocol.Profile) (net.Conn, error)
}
```

A default TCP/TLS implementation of `Dial` is provided in `agent.go`.
Override it for HTTP, DNS, SMB, or any custom channel.

**AgentImpl** -- composite interface (all of the above):

```go
type AgentImpl interface {
    Stealth
    Platform
    FileSystem
    Execution
    Transport
}
```

---

## Workflow Examples

### Example 1 -- New Go Agent from Scratch

Generate, implement, build, and deploy a Linux agent (Go, the default):

```powershell
# 1. Generate the scaffold
.\generator.ps1 -Mode agent -Name phoenix -Protocol adaptix_default

# 2. Implement platform methods
cd output\phoenix_agent\src_phoenix\impl
# Edit agent_linux.go -- fill in every // TODO: stub

# 3. Build
cd ..\..
go mod tidy
cd src_phoenix && go mod tidy && cd ..
make full

# 4. Deploy
copy agent_phoenix.so <AdaptixServer>\data\extenders\phoenix_agent\
copy config.yaml      <AdaptixServer>\data\extenders\phoenix_agent\
copy ax_config.axs    <AdaptixServer>\data\extenders\phoenix_agent\
```

### Example 2 -- C++ Agent (Windows PE/DLL/Shellcode)

```powershell
# Generate a C++ agent scaffold
.\generator.ps1 -Mode agent -Name wraith -Language cpp

cd output\wraith_agent\src_wraith
# Implement impl/agent_windows.cpp
# Build targets: make exe, make dll, make shellcode, or make all
```

### Example 3 -- Rust Agent (Cross-Platform)

```powershell
# Generate a Rust agent scaffold
.\generator.ps1 -Mode agent -Name spectre -Language rust

cd output\spectre_agent\src_spectre
# Implement src/agent.rs (Connector trait)
# Build: cargo build --release --target x86_64-unknown-linux-gnu
```

### Example 4 -- Agent + Listener Pair

Generate a matched agent and listener that share the same wire format:

```powershell
# Both use protocol "adaptix_default" -- guaranteed wire-compatible
.\generator.ps1 -Mode agent    -Name falcon -Protocol adaptix_default
.\generator.ps1 -Mode listener -Name falcon -Protocol adaptix_default -ListenerType external
```

After implementing both, link them in the agent's `config.yaml`:

```yaml
listeners: ["FalconAdaptix_default"]
```

The name `FalconAdaptix_default` must match the `listener_name` field in the listener's `config.yaml`.

### Example 5 -- Custom Protocol with Protobuf

Create a protocol that uses Protocol Buffers instead of msgpack:

```powershell
# 1. Create the protocol scaffold
.\generator.ps1 -Mode protocol
# Enter name: protobuf

# 2. Edit the template files
cd protocols\protobuf

# crypto.go.tmpl   -- implement EncryptData/DecryptData (e.g., AES-GCM, ChaCha20)
# types.go.tmpl    -- replace msgpack with protobuf marshal/unmarshal
# constants.go.tmpl -- define your COMMAND_* / RESP_* values

# 3. Update meta.yaml
# name: "protobuf"
# crypto: "AES-256-GCM"
# framing: "protobuf + length-prefix"

# 4. Generate extenders using the new protocol
cd ..\..
.\generator.ps1 -Mode agent    -Name stealth -Protocol protobuf
.\generator.ps1 -Mode listener -Name stealth -Protocol protobuf
```

The generator injects your `crypto.go.tmpl`, `types.go.tmpl`, and `constants.go.tmpl`
into the correct locations within each generated project.

---

## Architecture

```
              +-------------------+
              |  generator.ps1/sh |  root dispatcher
              +---------+---------+
       +------+----+----+----+--------+
       v      v    v    v    v        v
    agent/ listener/ service/ proto/ crypto/
    gen.   gen.      gen.    gen.   gen.
       |      |    |    |    |        |
       v      v    v    v    v        v
  OUTPUT_DIR/  OUTPUT_DIR/  OUTPUT_DIR/ protocols/  protocols/
  <name>_      <name>_     <name>_     <name>/     <proto>/
  agent/       listener_   service/                crypto.go
               <proto>/                            .tmpl
```

**Data flow:**

1. The root dispatcher selects a sub-generator based on `-Mode` or menu choice.
2. The sub-generator reads `.tmpl` files from its `templates/` directory and from `protocols/<name>/`.
3. Placeholders (`__NAME__`, `__WATERMARK__`, `__PROTOCOL__`, `__PACKAGE__`, etc.) are
   replaced with the values provided by the user.
4. The rendered files are written to the output directory.

**Communication layers:**

```
Agent Implant  <----[protocol]---->  Listener Plugin  <----[axc2 msgpack]---->  AdaptixC2 Server
     ^                                    ^                                          ^
     |                                    |                                          |
  You implement                     You implement                         Fixed API (axc2 v1.2.0)
  (impl/*.go)                       (pl_transport.go)
```

The agent-to-listener wire format is defined by the selected protocol and is fully replaceable.
The listener-to-server interface uses the fixed axc2 msgpack API and cannot be changed.

---

## Build and Deploy

### Agent (Go implant)

```bash
cd <name>_agent/
go mod tidy
cd src_<name> && go mod tidy && cd ..

# Build plugin (.so) + implant binaries
make full
```

### Agent (C++ implant)

```bash
cd <name>_agent/
go mod tidy

# Build plugin (.so) + implant (cross-compiled with MinGW)
make full
```

The C++ Makefile uses MinGW (`x86_64-w64-mingw32-g++`).
Format options: Exe, Service Exe, DLL, Shellcode.

### Agent (Rust implant)

```bash
cd <name>_agent/
go mod tidy

# Build plugin (.so) + implant (cross-compiled with cargo)
make full
```

Requires cross-compile targets: `rustup target add x86_64-unknown-linux-gnu x86_64-pc-windows-gnu`

### Listener

```bash
cd <name>_listener_<protocol>/
go mod tidy

# Build plugin (.so)
make plugin

# Package for deployment
make dist
```

### Deployment

Copy the built artifacts into the AdaptixC2 server extenders directory:

```
AdaptixServer/data/extenders/<name>_agent/
    agent_<name>.so
    config.yaml
    ax_config.axs

AdaptixServer/data/extenders/<name>_listener_<protocol>/
    listener_<name>_<protocol>.so
    config.yaml
    ax_config.axs
```

Restart the AdaptixC2 server to load the new extenders.

---

## FAQ

**Do I need the AdaptixC2 source code to use these generators?**
No. The generators are standalone scripts with no compile-time dependency on the server.
Generated code depends only on the public `axc2 v1.2.0` module.

**Can I generate on Windows and build on Linux?**
Yes. Generate on any OS, then transfer the output to a Linux machine for `go build -buildmode=plugin`.

**Can I write the implant in C++ or Rust but keep the server plugin in Go?**
Yes -- that is exactly how multi-language support works. The server-side plugin (`.so`)
is always Go. Only the implant source changes based on `-Language`.

**What if I only need a Linux agent?**
For Go: implement only `agent_linux.go`. For C++/Rust: target only the Linux build.
Unused platform stubs compile but remain non-functional.

**Can I use a different serialization format (protobuf, flatbuffers, JSON)?**
Yes. Create a custom protocol, implement your serialization in `types.go.tmpl`, and generate
agents/listeners with `-Protocol <name>`.

**How do I add new commands to a generated agent?**
See the detailed guide in `agent/README.md` under "Adding New Commands". In short: add constants
in `pl_utils.go` and `protocol/protocol.go`, register in `ax_config.axs`, handle in `pl_main.go`
and `tasks.go`.

**Can I add more languages in the future?**
Yes. Create a new template directory under `agent/templates/implant/<lang>/`, a toolchain
YAML in `agent/toolchains/`, a build variant `pl_build_<lang>.go`, and register it in the
generator's build variant map.

---

## License

This project is provided as a companion tool for AdaptixC2.
See the main AdaptixC2 repository for license terms.
