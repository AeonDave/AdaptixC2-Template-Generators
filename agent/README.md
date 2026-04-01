# AdaptixC2 — Template Agent Framework

A template-based framework for rapidly creating new AdaptixC2 beacon agents in **Go**, **C++**, or **Rust**.  
Instead of copying & modifying existing agents, run the generator and implement only the platform-specific methods you need.

---

## Quick Start

### Via Root Dispatcher (recommended)

```powershell
.\generator.ps1            # then select "1) Generate Agent"
.\generator.ps1 -Mode agent  # or skip the menu
.\generator.ps1 -Mode agent -Language cpp -Toolchain mingw   # C++ implant
.\generator.ps1 -Mode agent -Language rust                   # Rust implant
.\generator.ps1 -Mode agent -Language go -Toolchain go-garble # Go + garble obfuscation
.\generator.ps1 -Mode agent -OutputDir ..\AdaptixC2\AdaptixServer\extenders
```
```bash
./generator.sh             # then select "1) Generate Agent"
MODE=agent ./generator.sh  # or skip the menu
MODE=agent LANGUAGE=cpp ./generator.sh           # C++ implant
MODE=agent LANGUAGE=rust ./generator.sh          # Rust implant
MODE=agent LANGUAGE=go TOOLCHAIN=go-garble ./generator.sh  # Go + garble
MODE=agent OUTPUT_DIR=../AdaptixC2/AdaptixServer/extenders ./generator.sh
```

### Direct
```bash
cd agent
chmod +x generator.sh
./generator.sh
```
```powershell
cd agent
.\generator.ps1
.\generator.ps1 -OutputDir ..\..\AdaptixC2\AdaptixServer\extenders
```

The generator asks for:

| Prompt | Description | Example |
|--------|-------------|---------|
| **Agent name** | Lowercase alphanumeric identifier | `phantom` |
| **Watermark** | 8-char hex (auto-generated if empty) | `a1b2c3d4` |
| **Protocol** | Wire-format from `protocols/` | `adaptix_default` |
| **Language** | Implant language: go, cpp, rust | `go` |
| **Toolchain** | Build toolchain (shown when multiple match) | `go-standard` |

### Languages and Toolchains

When `-Language` / `-Toolchain` are omitted, the generator presents interactive menus.
Languages are discovered from `agent/templates/implant/`; toolchains from `agent/toolchains/*.yaml`.
If only one toolchain matches the language, it is auto-selected.

| Language | Default toolchain | Alternatives |
|----------|------------------|--------------|
| `go` | `go-standard` | `go-garble` |
| `cpp` | `mingw` | -- |
| `rust` | `cargo` | -- |

### Protocol Support

Protocols define shared crypto, constants, and wire types between agents and listeners.
When a protocol is selected, the generator overlays:
- `crypto/crypto.go` — from `protocols/<name>/crypto.go.tmpl`
- `protocol/protocol.go` — merged from `protocols/<name>/types.go.tmpl` + `constants.go.tmpl`
- `pl_utils.go` — merged from the same protocol templates (server-side)

For **C++ and Rust** implants, protocols can also provide language-specific overlays in `protocols/<name>/implant/cpp/` and `protocols/<name>/implant/rust/`. These override the base implant templates for protocol structs, wire format, and tasks.

Bundled public protocols: `adaptix_default` (RC4 + binary) and `adaptix_gopher` (AES-GCM + msgpack).
Private/internal protocol overlays may also exist in `protocols/`, but they are not documented as public options here.
The original implementation language of a protocol family does not imply that the same implant language is the most complete template path in this repository; use the generated code maturity here as the source of truth.
Agents and listeners using the **same protocol** share identical encryption and wire types.
The core agent generator stays protocol-agnostic: when a protocol needs custom behavior, it should provide
protocol-owned override files under `protocols/<name>/` instead of relying on name-based branching in the generator.
See the root README for protocol creation and crypto swap documentation.

---

## Generated Structure

The server-side plugin files are identical for all languages.
The implant directory (`src_<name>/`) varies by language.

### Go (default)

```
stub_<name>_agent/
├── config.yaml          # Plugin manifest (name, watermark, listeners)
├── go.mod               # Plugin Go module
├── Makefile             # Plugin build (agent_<name>.so)
├── pl_utils.go          # Wire types & command constants
├── pl_main.go           # Server-side plugin logic
├── pl_build.go          # Build logic (Go)
├── ax_config.axs        # UI & command registration
└── src_<name>/
    ├── go.mod               # Implant Go module
    ├── Makefile             # Cross-platform implant build
    ├── config.go            # Encrypted profile placeholder
    ├── main.go              # Connection loop + C2 transport
    ├── tasks.go             # Command dispatch switch
    ├── async_jobs.go        # Background job management
    ├── runtime_common.go    # Shared runtime helpers (dir listing, process list, shell, screenshot)
    ├── runtime_message.go   # Encrypted message send/receive helpers
    ├── crypto/
    │   └── crypto.go        # Protocol crypto (from protocol overlay)
    ├── protocol/
    │   ├── protocol.go      # Wire types & framing
    │   └── agent_types.go   # Protocol adapter types (DirEntry, ProcessEntry helpers)
    ├── impl/
    │   ├── interfaces.go    # Interface contracts (read-only reference)
    │   ├── agent.go         # Cross-platform: Stealth + Transport
    │   ├── agent_linux.go   # Linux: platform info, process listing, screenshot
    │   ├── agent_windows.go # Windows: platform info, process listing, screenshot
    │   ├── agent_darwin.go  # macOS stubs
    │   ├── bof_loader.go    # COFF/BOF parser + loader
    │   ├── downloader.go    # File download chunking
    │   ├── jobs.go          # Async job state machine
    │   └── shared_runtime.go # Reusable runtime (dir entries, shell, screenshot)
    └── evasion/             # (only with -Evasion flag)
        ├── gate.go          # Gate interface definition
        ├── default.go       # Default panic implementation
        ├── gate_linux.go    # Linux gate stub
        ├── gate_windows.go  # Windows gate stub
        └── gate_darwin.go   # macOS gate stub
```

### C++

```
stub_<name>_agent/
├── (same plugin files)
├── pl_build.go              # Build logic (C++: profile_gen.h + make)
├── ax_config.axs            # C++: arch, format (Exe/DLL/Shellcode), svc_name
└── src_<name>/
    ├── Makefile             # MinGW cross-compile targets
    ├── main.cpp             # Entry point (service/DLL/exe modes)
    ├── config.h / config.cpp # Profile placeholder
    ├── crypto/
    │   └── crypto.h / crypto.cpp   # Protocol crypto
    ├── protocol/
    │   └── protocol.h / protocol.cpp # Wire types & framing
    ├── impl/
    │   ├── Agent.h / Agent.cpp           # Agent lifecycle + C2 loop
    │   ├── Commander.h / Commander.cpp   # Command dispatch
    │   ├── Connector.h                   # Transport interface
    │   ├── ConnectorTCP.h / ConnectorTCP.cpp # TCP transport
    │   ├── Downloader.h / Downloader.cpp # Download chunking
    │   ├── JobsController.h / JobsController.cpp # Async job control
    │   ├── RuntimeCommon.h / RuntimeCommon.cpp   # Shared OS/runtime helpers
    │   ├── RuntimeErrors.h / RuntimeErrors.cpp   # Error code mapping
    │   ├── bof_loader.h / bof_loader.cpp         # COFF/BOF parser + loader
    └── evasion/             # (only with -Evasion flag)
        ├── IEvasionGate.h   # Gate interface
        ├── DefaultGate.h / DefaultGate.cpp  # Default panic implementation
```

### Rust

```
stub_<name>_agent/
├── (same plugin files)
├── pl_build.go              # Build logic (Rust: config.rs + cargo build)
├── ax_config.axs            # OS + arch selection
└── src_<name>/
    ├── Cargo.toml           # Release profile (size-optimised)
    ├── Makefile             # cargo build targets
    └── src/
        ├── main.rs              # Entry point
        ├── config.rs            # Profile data (populated at build time)
        ├── crypto.rs            # Protocol crypto
        ├── protocol.rs          # Wire types, framing, watermark
        ├── agent.rs             # Agent lifecycle + C2 loop
        ├── commander.rs         # Command dispatch
        ├── connector_tcp.rs     # TCP transport
        ├── bof.rs               # COFF/BOF parser + loader
        ├── downloader.rs        # Download chunking
        ├── jobs.rs              # Async job control
        ├── runtime_common.rs    # Shared runtime helpers
        ├── runtime_fs.rs        # Filesystem helpers
        └── runtime_response.rs  # Response building helpers
```

---

## Evasion Gate

Add `-Evasion` (PowerShell) or `EVASION=1` (Bash) to scaffold an evasion abstraction layer
in the implant source. This provides a unified `Gate` interface for syscall/stack-spoof
techniques without coupling the agent's command logic to a specific evasion implementation.

```powershell
.\generator.ps1 -Mode agent -Name phantom -Evasion
```
```bash
NAME=phantom EVASION=1 ./generator.sh
```

**Behavior:**

- **With `-Evasion`**: an `evasion/` directory is created in the implant source with a `Gate` interface
  (5 methods: `Init`, `Syscall`, `ResolveFn`, `Call`, `Close`) and a default implementation that panics
  on all methods, forcing you to provide a real implementation. Template marker comments (`// __EVASION_*__`)
  are expanded into real evasion calls.
- **Without `-Evasion`**: no `evasion/` directory is created, and all `// __EVASION_*__` markers are
  stripped from the generated code.

| Language | Evasion files |
|----------|---------------|
| Go | `evasion/gate.go`, `default.go`, `gate_linux.go`, `gate_windows.go`, `gate_darwin.go` |
| C++ | `evasion/IEvasionGate.h`, `DefaultGate.h`, `DefaultGate.cpp` |
| Rust | Not yet implemented |

---

## Interfaces (Go)

All platform-specific behavior is defined via Go interfaces in `impl/interfaces.go`.

### Stealth
```go
type Stealth interface {
    IsDebugged() bool       // Anti-debug check — return true to abort
    Masquerade()            // Process masquerading (e.g. ppid spoofing)
    OnStart()               // One-time setup before main loop
}
```

### Platform
```go
type Platform interface {
    GetCP() uint32                     // Console code page (Windows ACP, 65001 for UTF-8)
    IsElevated() bool                  // Running as root / admin
    GetOsVersion() string              // "Windows 11 23H2" / "Ubuntu 22.04" / etc.
    NormalizePath(path string) string  // Expand ~ or . to absolute path
}
```

### FileSystem
```go
type FileSystem interface {
    GetListing(path string) (string, []protocol.DirEntry, error)
    CopyFile(src, dst string) error
    CopyDir(src, dst string) error
}
```

### Execution
```go
type Execution interface {
    RunShell(cmd string, output bool, wait bool) (string, error)
    ListProcesses() ([]protocol.ProcessEntry, error)
    CaptureScreenshot() ([]byte, error)
}
```

### Transport
```go
type Transport interface {
    Dial(addr string, profile *protocol.Profile) (net.Conn, error)
}
```

A default TCP/TLS `Dial` is provided in `agent.go`. Override it for HTTP, DNS, SMB, or custom channels.

### AgentImpl (composite)
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

## Implementation Guide

### Go

1. **Run the generator**: `.\generator.ps1 -Mode agent -Name <name>`
2. **Edit platform files** in `impl/`:
   - `agent_linux.go`, `agent_windows.go`, `agent_darwin.go` — fill in stubs
   - `agent.go` — customize Stealth methods, override Transport if needed
3. **Don't modify** `interfaces.go`, `crypto/`, or `protocol/`
4. **Build**:
   ```bash
   go mod tidy
   cd src_<name> && go mod tidy && cd ..
   make full
   ```

### C++

1. **Run the generator**: `.\generator.ps1 -Mode agent -Name <name> -Language cpp`
2. **Edit** `impl/agent_windows.cpp` — platform-specific logic
3. **Build**:
   ```bash
   go mod tidy
   make full
   ```
   Formats: `exe`, `service_exe`, `dll`, `shellcode` (selected in Adaptix UI at build time)

### Rust

1. **Run the generator**: `.\generator.ps1 -Mode agent -Name <name> -Language rust`
2. **Edit** `src/agent.rs` — implement the `Connector` trait
3. **Build**:
   ```bash
   go mod tidy
   make full
   ```
   Cross-compile targets: `rustup target add x86_64-unknown-linux-gnu x86_64-pc-windows-gnu`

### Deploy

Copy `agent_<name>.so`, `config.yaml`, and `ax_config.axs` to the server's extenders directory.

---

## Shared Packages (no modification needed)

| Package | Purpose |
|---------|---------|
| `crypto/` | Encrypt/Decrypt (from protocol, e.g. AES-256-GCM) |
| `protocol/` | Wire types, command constants, serialization helpers (from protocol) |

---

## Adding New Commands

1. Add a `COMMAND_*` constant in both `pl_utils.go` and `protocol/protocol.go`.
2. Add `Params*` / `Ans*` structs in both files.
3. Register the command in `ax_config.axs` using `create_command()`.
4. Handle it in `pl_main.go` → `CreateCommand()` and `ProcessData()`.
5. Implement execution in `tasks.go` → `TaskProcess()`.

---

## Tips

- **Stealth first**: Implement `IsDebugged()` and `Masquerade()` before anything else — they run at startup.
- **Start with Linux**: `agent_linux.go` is the easiest to implement. Test on Linux, then port.
- **RunShell**: The `risky` flag indicates interactive / PTY mode. `piped` means capture output via pipe.
- **Screenshots**: Return raw PNG bytes from `CaptureScreenshot()`.
- **Transport**: The default TCP/TLS dial works for GopherTCP listeners. Override for other protocols.

---

## Listener Binding

AdaptixC2 uses **string name matching** to pair agents with listeners:

1. Your agent declares supported listeners in `config.yaml` → `listeners:` (list of names)
2. Each listener plugin declares its `listener_name:` in its own `config.yaml`
3. **Names must match exactly** — the Adaptix UI only shows matching listeners when building your agent

When `-Protocol <name>` is used, the generator auto-populates `config.yaml -> listeners:` with:

- `<AgentNameCap><ProtocolCap>`

Example:

- `-Name minibind2 -Protocol adaptix_gopher` → `listeners: ["Minibind2Adaptix_gopher"]`

Override this only when one agent should intentionally advertise multiple listener names.

### Examples

| Agent | `listeners:` | Supported protocols |
|-------|-------------|---------------------|
| `gopher_agent` | `["GopherTCP"]` | TCP only |
| `beacon_agent` | `["BeaconHTTP", "BeaconTCP", "BeaconSMB", "BeaconDNS"]` | HTTP, TCP, SMB, DNS |

### `multi_listeners`

- `true` — embed **multiple** listener profiles in a single build (agent tries each at runtime)
- `false` — build accepts only **one** listener profile

### Adding listener support

1. Prefer generating the agent and listener with the same basename + protocol so auto-binding is correct by default
2. If one agent must support multiple listener names, edit `config.yaml -> listeners:` explicitly
3. In `pl_main.go` → `GenerateProfiles()`, add a `switch listenerType` branch only when the listener profile fields differ
4. In the implant's `impl/agent.go`, implement or override `Dial()` if the transport changes (e.g. HTTP polling, DNS tunneling)

Listener selection still happens **in the Adaptix UI at build time**; generation now sets the default listener name to match the generated listener scaffold.
