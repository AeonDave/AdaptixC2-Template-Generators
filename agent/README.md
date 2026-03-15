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

### Interactive language and toolchain selection

When `-Language` / `-Toolchain` are not passed on the command line, the generator
presents numbered menus:

```
Select implant language:
  [1] go (default)   - Go implant
  [2] cpp            - C/C++ implant
  [3] rust           - Rust implant

Select language [default: 1]:
```

After choosing a language, if more than one toolchain is available for that language,
the toolchain menu appears:

```
Available toolchains for 'go':
  [1] go-standard (default)  - Standard Go compiler (CGO_ENABLED=0, cross-platform)
  [2] go-garble              - Garble obfuscator (symbol/string obfuscation)

Select toolchain [default: 1]:
```

If only one toolchain matches the selected language (e.g. `mingw` for `cpp`, `cargo`
for `rust`), it is auto-selected without prompting.

Languages are discovered dynamically from `agent/templates/implant/` subdirectories.
Toolchains are discovered from `agent/toolchains/*.yaml`, filtered by the `language:` field.

Non-interactive parameters (passed via CLI or env vars):

| Parameter | PowerShell | Bash env var | Default |
|-----------|-----------|-------------|----------|
| Language | `-Language` | `LANGUAGE` | Prompted (default: `go`) |
| Toolchain | `-Toolchain` | `TOOLCHAIN` | Prompted or auto-detected |

Supported languages: `go`, `cpp`, `rust`

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

```powershell
# Use a specific protocol
.\generator.ps1 -Name phantom -Watermark a1b2c3d4 -Protocol adaptix_default

# With output directory
.\generator.ps1 -Name phantom -Protocol adaptix_default -OutputDir ..\..\AdaptixC2\AdaptixServer\extenders

# Or via env var (bash)
PROTOCOL=adaptix_default bash generator.sh
PROTOCOL=adaptix_default OUTPUT_DIR=../../AdaptixC2/AdaptixServer/extenders bash generator.sh
```

Agents and listeners using the **same protocol** share identical encryption and wire types, ensuring compatibility.

Output: `<OutputDir>/<name>_agent/` — a fully scaffolded agent ready to implement.
Default output: `./output/<name>_agent/` when no `-OutputDir` is specified.

---

## Generated Structure

The server-side plugin files are identical for all languages.
The implant directory (`src_<name>/`) varies by language.

### Go (default)

```
<name>_agent/
├── config.yaml          # Plugin manifest (name, watermark, listeners)
├── go.mod               # Plugin Go module
├── Makefile             # Plugin build (agent_<name>.so)
├── pl_utils.go          # Wire types & command constants
├── pl_main.go           # Server-side plugin logic
├── pl_build.go          # Build logic (Go)
├── ax_config.axs        # UI & command registration
└── src_<name>/
    ├── go.mod           # Implant Go module
    ├── Makefile         # Cross-platform implant build
    ├── config.go        # Encrypted profile placeholder
    ├── main.go          # Connection loop
    ├── tasks.go         # Command dispatch
    ├── crypto/
    │   └── crypto.go    # AES-256-GCM (ready to use)
    ├── protocol/
    │   └── protocol.go  # Wire types & framing (ready to use)
    └── impl/
        ├── interfaces.go    # Interface contracts (read-only)
        ├── agent.go         # Cross-platform: Stealth + Transport
        ├── agent_linux.go   # Linux stubs      ← IMPLEMENT
        ├── agent_windows.go # Windows stubs    ← IMPLEMENT
        └── agent_darwin.go  # macOS stubs      ← IMPLEMENT
```

### C++

```
<name>_agent/
├── (same plugin files)
├── pl_build.go          # Build logic (C++: profile_gen.h + make)
├── ax_config.axs        # C++: arch, format (Exe/DLL/Shellcode), svc_name
└── src_<name>/
    ├── Makefile         # MinGW cross-compile targets
    ├── main.cpp / config.h / config.cpp
    ├── agent.h / agent.cpp
    ├── crypto.h / crypto.cpp
    ├── protocol.h / protocol.cpp
    └── impl/
        └── agent_windows.h / agent_windows.cpp  ← IMPLEMENT
```

### Rust

```
<name>_agent/
├── (same plugin files)
├── pl_build.go          # Build logic (Rust: config.rs + cargo build)
├── ax_config.axs        # OS + arch selection
└── src_<name>/
    ├── Cargo.toml       # Release profile (size-optimised)
    ├── Makefile         # cargo build targets
    └── src/
        ├── main.rs      # Entry point
        ├── config.rs    # Profile data (populated at build time)
        ├── crypto.rs    # Encrypt/decrypt stubs
        ├── protocol.rs  # Wire protocol + watermark
        └── agent.rs     # Connector trait + Agent  ← IMPLEMENT
```

---

## Interfaces

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
    GetCP() uint32              // Console code page (Windows ACP, 65001 for UTF-8)
    IsElevated() bool           // Running as root / admin
    GetOsVersion() string       // "Windows 11 23H2" / "Ubuntu 22.04" / etc.
    NormalizePath(p string) string // Expand ~ or . to absolute path
}
```

### FileSystem
```go
type FileSystem interface {
    GetListing(dir string) (string, []protocol.DirEntry, error)
    CopyFile(src, dst string) error
    CopyDir(src, dst string) error
}
```

### Execution
```go
type Execution interface {
    RunShell(cmd string, risky, piped bool) (string, error)
    ListProcesses() ([]protocol.ProcessEntry, error)
    CaptureScreenshot() ([]byte, error)
}
```

### Transport
```go
type Transport interface {
    Dial(address string, profile *protocol.Profile) (net.Conn, error)
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

### Examples

| Agent | `listeners:` | Supported protocols |
|-------|-------------|---------------------|
| `gopher_agent` | `["GopherTCP"]` | TCP only |
| `beacon_agent` | `["BeaconHTTP", "BeaconTCP", "BeaconSMB", "BeaconDNS"]` | HTTP, TCP, SMB, DNS |

### `multi_listeners`

- `true` — embed **multiple** listener profiles in a single build (agent tries each at runtime)
- `false` — build accepts only **one** listener profile

### Adding listener support

1. Edit `config.yaml` → add the listener name to the `listeners:` list
2. In `pl_main.go` → `GenerateProfiles()`, add a `switch listenerType` branch to parse the new listener's protocol-specific fields
3. In the implant's `impl/agent.go`, implement or override `Dial()` if the new protocol requires a different transport (e.g. HTTP polling, DNS tunneling)

The default scaffold comes with `GopherTCP` support. The listener selection happens **in the Adaptix UI at build time**, not during generation.
