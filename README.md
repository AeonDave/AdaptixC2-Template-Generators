# AdaptixC2 Template Generators

Standalone scaffolding toolkit for [AdaptixC2](https://github.com/Adaptix-Framework/AdaptixC2) extender development.
Generates ready-to-implement stub projects for **agents**, **listeners**, **services** (optionally with **post-build wrapper pipeline**), and custom **wire protocols** -- all compatible with the `axc2 v1.2.0` plugin API.

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
- [Toolchains and Compilers](#toolchains-and-compilers)
  - [Switching Toolchains](#switching-toolchains)
  - [Customizing Toolchains](#customizing-toolchains)
- [Protocols](#protocols)
  - [Bundled Protocols](#bundled-protocols)
  - [Creating a Custom Protocol](#creating-a-custom-protocol)
  - [Protocol File Layout](#protocol-file-layout)
- [Post-Build Wrappers](#post-build-wrappers)
  - [How the Hook Works](#how-the-hook-works)
  - [Example -- Crystal Palace (StealthPalace)](#example----crystal-palace-stealthpalace)
  - [Writing Pipeline Stages](#writing-pipeline-stages)
- [What Gets Generated](#what-gets-generated)
  - [Agent Output](#agent-output)
  - [Listener Output](#listener-output)
  - [Wrapper Output](#wrapper-output)
- [Agent Interfaces](#agent-interfaces)
- [Workflow Examples](#workflow-examples)
  - [Example 1 -- New Go Agent from Scratch](#example-1----new-go-agent-from-scratch)
  - [Example 2 -- C++ Agent](#example-2----c-agent-windows-pedllshellcode)
  - [Example 3 -- Rust Agent](#example-3----rust-agent-cross-platform)
  - [Example 4 -- Agent + Listener Pair](#example-4----agent--listener-pair)
  - [Example 5 -- Custom Protocol with Protobuf](#example-5----custom-protocol-with-protobuf)
  - [Example 6 -- Post-Build Wrapper + Agent](#example-6----post-build-wrapper--agent)
  - [Example 7 -- Go Agent with Garble Obfuscation](#example-7----go-agent-with-garble-obfuscation)
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
| **Service** | Server-side plugin with Call handler (optionally with post-build wrapper pipeline) | Service logic in `pl_main.go`; pipeline stages when wrapper is enabled |
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
|   |-- generator.ps1          Service sub-generator (supports -Wrapper flag)
|   |-- generator.sh           Service sub-generator (supports WRAPPER=1)
|   |-- README.md              Service + wrapper documentation
|   +-- templates/
|       |-- (base templates)    pl_main.go, ax_config.axs, config.yaml, go.mod, Makefile
|       +-- wrapper/            Wrapper addon overrides (pl_main.go, pl_wrapper.go, ax_config.axs, ...)
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
3) Generate Service   - Scaffold a new service extender (optionally with wrapper pipeline)
4) Create Protocol    - Create a new wire-protocol definition
5) Create Crypto      - Generate or replace the crypto template for a protocol
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
.\generator.ps1 -Mode service -Wrapper   # include post-build wrapper pipeline
.\generator.ps1 -Mode protocol
.\generator.ps1 -Mode crypto
.\generator.ps1 -Mode delete
```

**Bash:**
```bash
MODE=agent    ./generator.sh
MODE=listener ./generator.sh
MODE=service  ./generator.sh
MODE=service WRAPPER=1 ./generator.sh   # include post-build wrapper pipeline
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

Optionally include a **post-build wrapper pipeline** with `-Wrapper` (PowerShell) or
`WRAPPER=1` (Bash). When the wrapper is included, the generated service hooks into
`agent.generate` (post-phase) and applies configurable transformation stages to the
generated payload. When neither flag is set in interactive mode, the generator asks.

**Parameters:**

| Parameter | PowerShell | Bash env var | Required | Default |
|-----------|-----------|-------------|----------|---------|
| Name | `-Name` | `NAME` | Yes (prompted if empty) | -- |
| Wrapper | `-Wrapper` switch | `WRAPPER=1` | No | Prompted interactively |
| Output dir | `-OutputDir` | `OUTPUT_DIR` | No | `./output/` |

```powershell
# Plain service
.\generator.ps1 -Mode service -Name telegram

# Service with wrapper pipeline
.\generator.ps1 -Mode service -Name crystalpalace -Wrapper

# Via sub-generator directly
cd service
.\generator.ps1 -Name crystalpalace -Wrapper
```

```bash
MODE=service NAME=telegram ./generator.sh
MODE=service NAME=crystalpalace WRAPPER=1 ./generator.sh
```

See [Post-Build Wrappers](#post-build-wrappers) for detailed integration guidance.

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

## Toolchains and Compilers

Every agent has a **toolchain** — a YAML manifest in `agent/toolchains/` that defines the
compiler binary, build command, flags, and cross-compile targets. The generator substitutes
the `__BUILD_TOOL__` placeholder in Makefiles and build scripts from the toolchain's `command:` field.

### Switching Toolchains

Pass `-Toolchain` (PowerShell) or `TOOLCHAIN=` (Bash) at generation time:

```powershell
# Go agent with standard compiler (default)
.\generator.ps1 -Mode agent -Name phantom -Language go

# Go agent with garble obfuscation
.\generator.ps1 -Mode agent -Name phantom -Language go -Toolchain go-garble

# C++ agent with MinGW (auto-selected — only toolchain for cpp)
.\generator.ps1 -Mode agent -Name wraith -Language cpp

# Rust agent with cargo (auto-selected — only toolchain for rust)
.\generator.ps1 -Mode agent -Name xxx -Language rust
```

```bash
# Garble
MODE=agent NAME=phantom LANGUAGE=go TOOLCHAIN=go-garble ./generator.sh

# MinGW
MODE=agent NAME=wraith LANGUAGE=cpp ./generator.sh
```

When `-Toolchain` is omitted and multiple toolchains exist for the chosen language,
the generator presents an interactive menu:

```
Available toolchains for 'go':
  [1] go-standard (default)  - Standard Go compiler (CGO_ENABLED=0, cross-platform)
  [2] go-garble              - Garble obfuscator (symbol/string obfuscation)

Select toolchain [default: 1]:
```

If only one toolchain matches (e.g. `mingw` for `cpp`), it is auto-selected.

**Bundled toolchains:**

| Toolchain | Language | Compiler | Description |
|-----------|----------|----------|-------------|
| `go-standard` | go | `go build` | Standard Go compiler, `CGO_ENABLED=0`, `-trimpath`, `-ldflags "-s -w"` |
| `go-garble` | go | `garble -literals -tiny build` | Symbol + string obfuscation (install: `go install mvdan.cc/garble@latest`) |
| `mingw` | cpp | `x86_64-w64-mingw32-g++` | MinGW-w64 cross-compiler with PE/DLL/Shellcode format support |
| `cargo` | rust | `cargo build --release` | Standard Rust/Cargo compiler with cross-compile targets |

### Customizing Toolchains

To add a custom toolchain (e.g. `gobfuscate`, a different MinGW version, or `clang`),
create a new YAML file in `agent/toolchains/`:

**Example — gobfuscate toolchain:**

```yaml
# agent/toolchains/go-gobfuscate.yaml
name: go-gobfuscate
language: go
description: "Gobfuscate (advanced Go obfuscation)"

compiler:
  binary: gobfuscate
  version_check: "gobfuscate --version"

build:
  command: "gobfuscate build"
  env:
    CGO_ENABLED: "0"
    GOWORK: "off"
  flags:
    - "-trimpath"
  ldflags: "-s -w"

targets:
  - { goos: linux,   goarch: amd64, suffix: "_linux_amd64" }
  - { goos: windows, goarch: amd64, suffix: "_windows_amd64.exe" }
```

**Example — MinGW with custom flags (e.g. anti-AV, stack encryption):**

```yaml
# agent/toolchains/mingw-custom.yaml
name: mingw-custom
language: cpp
description: "MinGW-w64 with hardened flags"

compiler:
  x64: x86_64-w64-mingw32-g++
  x86: i686-w64-mingw32-g++
  version_check: "x86_64-w64-mingw32-g++ --version"

cxxflags:
  - "-fno-stack-protector"
  - "-fno-exceptions"
  - "-fno-unwind-tables"
  - "-fno-asynchronous-unwind-tables"
  - "-masm=intel"
  - "-fPIC"
  - "-Os"                           # Optimize for size
  - "-ffunction-sections"           # Dead code stripping
  - "-fdata-sections"               # Dead data stripping
  - "-Wl,--gc-sections"             # Linker removes unused sections

formats:
  exe:
    defines: []
    ldflags: []
    extension: ".exe"
  dll:
    defines: ["BUILD_DLL"]
    ldflags: ["-shared"]
    extension: ".dll"
  shellcode:
    defines: ["BUILD_SHELLCODE"]
    ldflags: []
    extension: ".bin"

architectures:
  - { name: x64, compiler_key: x64 }
  - { name: x86, compiler_key: x86 }
```

Once saved, the toolchain is automatically discovered and available on the next run:

```powershell
.\generator.ps1 -Mode agent -Name wraith -Language cpp -Toolchain mingw-custom
```

**Key fields reference:**

| Field | Purpose |
|-------|---------|
| `name` | Unique identifier (matches filename without `.yaml`) |
| `language` | Which language this toolchain applies to (`go`, `cpp`, `rust`) |
| `build.command` | Replaces the `__BUILD_TOOL__` placeholder |
| `build.env` | Environment variables set during build |
| `build.flags` | Additional compiler flags |
| `targets` | Cross-compile target matrix |
| `cxxflags` | C++ specific compiler flags (MinGW) |
| `formats` | Output format options: exe, dll, shellcode, service (C++ only) |

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

## Post-Build Wrappers

A **wrapper** is a service plugin (generated with `-Wrapper` or `WRAPPER=1`) that hooks
into the `agent.generate` event (post-phase) and transforms the generated payload before
it reaches the operator. This enables post-build processing like RDLL loading, sleep
obfuscation, shellcode encryption, packing, and more — without modifying the agent itself.

The wrapper concept is inspired by [Adaptix-StealthPalace](https://github.com/MaorSabag/Adaptix-StealthPalace)
by MaorSabag, which integrates Crystal Palace (RDLL loader), Ekko sleep obfuscation, and
module stomping as a post-build pipeline.

### How the Hook Works

1. **Operator** clicks "Build Agent" in the Adaptix UI.
2. **Teamserver** compiles the agent and fires `agent.generate` (post-phase).
3. **Wrapper plugin** intercepts the event via `TsEventHookRegister`.
4. The event carries a pointer to a struct with the payload accessible via `reflect`:
   - `FileContent []byte` — the generated payload bytes (modified in-place)
   - `FileName string` — output file name (can be renamed)
   - `BuilderId string` — build log identifier
   - `AgentName string` — agent type name
   - `Config string` — build configuration
5. The wrapper runs all enabled pipeline stages and writes the result back via
   `reflect.SetBytes()` — no file I/O, no disk writes.
6. **Teamserver** returns the wrapped payload to the operator.

The wrapper intercepts **all** agent builds automatically. No per-agent configuration is needed.

### Example -- Crystal Palace (StealthPalace)

A complete end-to-end flow:

```powershell
# 1. Generate the agent
.\generator.ps1 -Mode agent -Name xxx -Protocol adaptix_default -Language go

# 2. Generate the wrapper
.\generator.ps1 -Mode service -Name crystalpalace -Wrapper

# 3. Implement your agent (platform stubs)
cd output\xxx_agent\src_xxx\impl
# Edit agent_windows.go, agent_linux.go, ...

# 4. Implement the wrapper stages
cd ..\..\..\..\output\crystalpalace_wrapper
```

In `pl_main.go`, register your stages:

```go
func initStages() {
    RegisterStage(Stage{
        Name:    "rdll_loader",
        Enabled: true,
        Run:     stageRdllLoader,
    })
    RegisterStage(Stage{
        Name:    "sleep_mask",
        Enabled: true,
        Run:     stageSleepMask,
    })
    RegisterStage(Stage{
        Name:    "module_stomp",
        Enabled: true,
        Run:     stageModuleStomp,
    })
}
```

Add stage logic in a new file (e.g. `pl_stages.go`):

```go
package main

import "fmt"

func stageRdllLoader(payload []byte, cfg map[string]string, ctx *BuildContext) ([]byte, error) {
    logBuild(ctx.BuilderID, BuildLogInfo, fmt.Sprintf("Applying RDLL loader to %s", ctx.FileName))
    // Wrap the payload in an RDLL loader stub...
    return payload, nil
}

func stageSleepMask(payload []byte, cfg map[string]string, ctx *BuildContext) ([]byte, error) {
    logBuild(ctx.BuilderID, BuildLogInfo, "Applying Ekko sleep obfuscation")
    // Patch sleep mask into the payload...
    return payload, nil
}

func stageModuleStomp(payload []byte, cfg map[string]string, ctx *BuildContext) ([]byte, error) {
    logBuild(ctx.BuilderID, BuildLogInfo, "Applying module stomping")
    // Apply module stomping...
    return payload, nil
}
```

```powershell
# 5. Build both plugins
cd output\xxx_agent
go mod tidy && make plugin

cd ..\crystalpalace_wrapper
go mod tidy && make plugin

# 6. Deploy both to the Teamserver
copy output\xxx_agent\dist\*        <AdaptixServer>\data\extenders\xxx_agent\
copy output\crystalpalace_wrapper\dist\* <AdaptixServer>\data\extenders\crystalpalace_wrapper\
```

After deploying both plugins, every time any operator builds the `xxx` agent (or any
other agent), the Crystal Palace wrapper automatically intercepts and transforms the payload.

### Writing Pipeline Stages

Each stage is a function with this signature:

```go
func(payload []byte, cfg map[string]string, ctx *BuildContext) ([]byte, error)
```

| Parameter | Description |
|-----------|-------------|
| `payload` | Current payload bytes (output of previous stage) |
| `cfg` | Key-value configuration persisted via `TsExtenderDataSave` |
| `ctx` | Build context: `AgentName`, `BuilderID`, `FileName`, `ModuleDir`, `Extra` |

**Stage lifecycle:**

- Stages run in registration order.
- A stage can be enabled/disabled at runtime via config keys: `stage.<name>.enabled = "true"` or `"false"`.
- If a stage returns an error, the pipeline stops and the original (unwrapped) payload is returned.
- Stages can store state in `ctx.Extra` (e.g. `ctx.Extra["output_filename"]` to rename the output file).
- Use `logBuild(ctx.BuilderID, BuildLogInfo, "message")` for build log output.

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

### Wrapper Output (Service with `-Wrapper`)

```
<name>_wrapper/
|-- config.yaml              Service manifest (extender_type: "service")
|-- go.mod                   Go module (depends on axc2 v1.2.0)
|-- Makefile                 Build targets: plugin, dist
|-- pl_main.go               Plugin entry + event hook + Call handler  <-- ADD STAGES HERE
|-- pl_wrapper.go            Pipeline engine (Stage registration, RunPipeline)
+-- ax_config.axs            Wrapper UI (status, config save/load)
```

The main files to customize are `pl_main.go` (register stages in `initStages()`) and
optionally a new `pl_stages.go` file with your stage implementations.

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
.\generator.ps1 -Mode agent -Name xxx -Language rust

cd output\xxx_agent\src_xxx
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

### Example 6 -- Post-Build Wrapper + Agent

Generate a Go agent and a Crystal Palace-style wrapper, build both, and deploy:

```powershell
# 1. Generate the agent with the adaptix_default protocol
.\generator.ps1 -Mode agent -Name phoenix -Protocol adaptix_default

# 2. Generate the post-build wrapper
.\generator.ps1 -Mode service -Name crystalpalace -Wrapper

# 3. Implement agent platform stubs
cd output\phoenix_agent\src_phoenix\impl
# Fill in agent_windows.go, agent_linux.go, etc.

# 4. Implement wrapper stages in output\crystalpalace_wrapper\
#    - Edit pl_main.go → initStages() to register your stages
#    - Add stage functions in pl_stages.go (see service/README.md)

# 5. Build both plugins (on Linux with CGO_ENABLED=1)
cd output\phoenix_agent
go mod tidy && make full

cd ..\crystalpalace_wrapper
go mod tidy && make plugin

# 6. Deploy both to the Teamserver extenders directory
#    Both .so plugins load on server startup.
#    The wrapper hooks ALL agent builds — no per-agent config needed.
```

```bash
# Same flow on Linux
MODE=agent NAME=phoenix PROTOCOL=adaptix_default ./generator.sh
MODE=service NAME=crystalpalace WRAPPER=1 ./generator.sh

cd output/phoenix_agent && go mod tidy && make full && cd ..
cd crystalpalace_wrapper && go mod tidy && make plugin && cd ..
```

**What happens at runtime:**
1. Operator clicks "Build Agent" for `phoenix` in the Adaptix UI.
2. Teamserver compiles the agent → fires `agent.generate` post event.
3. `crystalpalace_wrapper` intercepts the event, reads `FileContent` via `reflect`,
   runs the RDLL loader / sleep mask / module stomp pipeline, and writes the
   transformed payload back via `SetBytes()`.
4. Teamserver returns the wrapped binary to the operator.

### Example 7 -- Go Agent with Garble Obfuscation

Use garble to strip symbols and obfuscate string literals:

```powershell
# Prerequisite: install garble
go install mvdan.cc/garble@latest

# Generate with garble toolchain
.\generator.ps1 -Mode agent -Name phantom -Language go -Toolchain go-garble -Protocol adaptix_default

# The generated Makefile uses "garble -literals -tiny build" instead of "go build"
cd output\phantom_agent\src_phantom
# Verify the Makefile:
#   BUILD_TOOL = garble -literals -tiny build
make linux_amd64
```

```bash
MODE=agent NAME=phantom LANGUAGE=go TOOLCHAIN=go-garble PROTOCOL=adaptix_default ./generator.sh
```

**Garble flags applied by the `go-garble` toolchain:**

| Flag | Effect |
|------|--------|
| `-literals` | Obfuscate string literals (replaces plaintext with runtime-decoded equivalents) |
| `-tiny` | Remove extra information not required for runtime (smaller binary) |
| `-trimpath` | Remove file system paths from the binary |
| `-ldflags "-s -w"` | Strip symbol table and debug info |

---

## Architecture

```
              +-------------------+
              |  generator.ps1/sh |  root dispatcher
              +---------+---------+
  +------+------+------+------+------+------+
  v      v      v      v      v      v      v
agent/ listener/ service/   proto/ crypto/ delete
gen.   gen.      gen.       gen.   gen.
  |      |      |           |      |
  v      v      v           v      v
 OUTPUT_DIR/   OUTPUT_DIR/   protocols/
 <name>_       <name>_       <name>/
 agent/        listener_     service/ or
               <proto>/      wrapper/
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

**Post-build pipeline (when a wrapper is deployed):**

```
                              agent.generate event (post)
                                       |
Agent Build  --->  Teamserver  --->  Wrapper Plugin  --->  Teamserver returns wrapped payload
                                       |
                               [rdll_loader] -> [sleep_mask] -> [module_stomp] -> ...
                               Stages run in order, modifying FileContent in-place
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

### Service / Wrapper

```bash
cd <name>_service/       # or <name>_wrapper/
go mod tidy

# Build plugin (.so)
make plugin
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

AdaptixServer/data/extenders/<name>_wrapper/
    service_<name>.so
    config.yaml
    ax_config.axs

AdaptixServer/data/extenders/<name>_service/
    service_<name>.so
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

**How do I change the compiler for a specific agent?**
Use `-Toolchain` at generation time. For example, `go-garble` for obfuscated Go builds,
or create a custom toolchain YAML in `agent/toolchains/` (see [Toolchains and Compilers](#toolchains-and-compilers)).

**Does the wrapper need to know about my specific agent?**
No. The wrapper hooks `agent.generate` (post-phase) and intercepts **all** agent builds
automatically. It receives the payload bytes via `reflect` and transforms them in-place.
No per-agent configuration or linking is required.

**Can I have multiple wrappers active simultaneously?**
Yes. Each wrapper registers its own event hook with a unique name and priority. Hooks
execute in priority order, so you can chain multiple wrappers (e.g. one for encryption,
one for packing).

**How do I integrate StealthPalace / Crystal Palace?**
Generate a service with `-Wrapper` (`-Mode service -Wrapper`), implement the RDLL loader, sleep mask, and module
stomping stages, then deploy alongside your agent. See [Example 6](#example-6----post-build-wrapper--agent)
and the [Post-Build Wrappers](#post-build-wrappers) section for the full walkthrough.

---

## License

This project is provided as a companion tool for AdaptixC2.
See the main AdaptixC2 repository for license terms.
