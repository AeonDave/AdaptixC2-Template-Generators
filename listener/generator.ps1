<#
.SYNOPSIS
    Scaffold a new AdaptixC2 listener.

.DESCRIPTION
    Creates <name>_listener/ with all files ready to implement.
    Output goes to -OutputDir (or ADAPTIX_OUTPUT_DIR env var, or ./output).

.PARAMETER Name
    Listener name (lowercase alphanumeric). Skips interactive prompt when provided.

.PARAMETER Protocol
    Protocol name from protocols/. Default: scans available protocols and asks.

.PARAMETER ListenerType
    "external" (default) or "internal". External = agents connect in; internal = agent binds.

.PARAMETER Transport
    Transport variant: "tcp" (default) or "http".
    When "http", uses HTTP-specific template overrides from the protocol.

.PARAMETER OutputDir
    Directory where <name>_listener/ will be created.
    Default: ADAPTIX_OUTPUT_DIR env var, or ./output.

.EXAMPLE
    .\generator.ps1

.EXAMPLE
    .\generator.ps1 -Name telegram -Protocol adaptix_default -ListenerType external

.EXAMPLE
    .\generator.ps1 -Name spectre_http -Protocol spectre -ListenerType external -Transport http

.EXAMPLE
    .\generator.ps1 -Name telegram -Protocol adaptix_default -OutputDir ..\my-adaptix\extenders
#>
param(
    [string]$Name         = "",
    [string]$Protocol     = "",
    [string]$ListenerType = "",
    [string]$Transport    = "",
    [string]$OutputDir    = ""
)

$ErrorActionPreference = "Stop"

$ScriptDir     = Split-Path -Parent $MyInvocation.MyCommand.Path
$TemplateDir   = Join-Path $ScriptDir "templates"
$TemplatesRoot = Split-Path -Parent $ScriptDir
$ProtocolsDir  = Join-Path $TemplatesRoot "protocols"

# Resolve output directory
if ([string]::IsNullOrEmpty($OutputDir)) {
    $OutputDir = $env:ADAPTIX_OUTPUT_DIR
}
if ([string]::IsNullOrEmpty($OutputDir)) {
    $OutputDir = Join-Path $TemplatesRoot "output"
}
if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
}
$ExtendersDir = (Resolve-Path $OutputDir).Path

# ─── Banner ─────────────────────────────────────────────────────────────────────

Write-Host ""
Write-Host "╔═══════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║   AdaptixC2 Template Listener Generator       ║" -ForegroundColor Cyan
Write-Host "╚═══════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

# ─── Discover protocols ─────────────────────────────────────────────────────────

$availableProtocols = @()
if (Test-Path $ProtocolsDir) {
    foreach ($d in Get-ChildItem -Path $ProtocolsDir -Directory) {
        if ($d.Name -ne "_scaffold" -and (Test-Path (Join-Path $d.FullName "meta.yaml"))) {
            $availableProtocols += $d.Name
        }
    }
}
if ($availableProtocols.Count -eq 0) {
    Write-Host "[-] No protocols found in $ProtocolsDir" -ForegroundColor Red
    Write-Host "    Run with -NewProtocol <name> to create one, or create a protocol directory manually." -ForegroundColor Yellow
    exit 1
}

# ─── Input: Protocol ────────────────────────────────────────────────────────────

if ([string]::IsNullOrEmpty($Protocol)) {
    Write-Host "Available protocols:" -ForegroundColor Cyan
    for ($i = 0; $i -lt $availableProtocols.Count; $i++) {
        $pn = $availableProtocols[$i]
        $metaPath = Join-Path (Join-Path $ProtocolsDir $pn) "meta.yaml"
        $desc = ""
        if (Test-Path $metaPath) {
            $metaContent = Get-Content -Path $metaPath -Raw -Encoding UTF8
            if ($metaContent -match 'description:\s*"([^"]*)"') {
                $desc = " - $($Matches[1])"
            }
        }
        Write-Host "  [$($i+1)] $pn$desc"
    }
    Write-Host ""
    while ($true) {
        $choice = Read-Host "Select protocol [1-$($availableProtocols.Count)]"
        $idx = 0
        if ([int]::TryParse($choice, [ref]$idx) -and $idx -ge 1 -and $idx -le $availableProtocols.Count) {
            $Protocol = $availableProtocols[$idx - 1]
            break
        }
        Write-Host "[!] Invalid choice." -ForegroundColor Yellow
    }
}

$protoDir = Join-Path $ProtocolsDir $Protocol
if (-not (Test-Path $protoDir)) {
    Write-Host "[-] Protocol '$Protocol' not found in $ProtocolsDir" -ForegroundColor Red
    exit 1
}

# ─── Input: Listener name ──────────────────────────────────────────────────────

if (-not [string]::IsNullOrEmpty($Name)) {
    $ListenerName = ($Name.ToLower() -replace '[^a-z0-9_]', '')
    $OutDir = Join-Path $ExtendersDir "${ListenerName}_listener"
    if ([string]::IsNullOrEmpty($ListenerName)) { Write-Host "[-] Invalid name." -ForegroundColor Red; exit 1 }
    if (Test-Path $OutDir) { Write-Host "[-] Directory ${ListenerName}_listener already exists!" -ForegroundColor Red; exit 1 }
} else {
    while ($true) {
        $ListenerName = Read-Host "Listener name (lowercase, e.g. telegram)"
        $ListenerName = ($ListenerName.ToLower() -replace '[^a-z0-9_]', '')
        if ([string]::IsNullOrEmpty($ListenerName)) {
            Write-Host "[!] Name cannot be empty." -ForegroundColor Yellow
            continue
        }
        $OutDir = Join-Path $ExtendersDir "${ListenerName}_listener"
        if (Test-Path $OutDir) {
            Write-Host "[!] Directory ${ListenerName}_listener already exists!" -ForegroundColor Yellow
            continue
        }
        break
    }
}

# Capitalize first letter
$ListenerNameCap = $ListenerName.Substring(0,1).ToUpper() + $ListenerName.Substring(1)
$ProtocolCap = $Protocol.Substring(0,1).ToUpper() + $Protocol.Substring(1)

# ─── Input: Listener type ──────────────────────────────────────────────────────

if ([string]::IsNullOrEmpty($ListenerType)) {
    $ListenerType = Read-Host "Listener type [external]"
    if ([string]::IsNullOrEmpty($ListenerType)) { $ListenerType = "external" }
}
if ($ListenerType -ne "external" -and $ListenerType -ne "internal") {
    Write-Host "[-] Listener type must be 'external' or 'internal'." -ForegroundColor Red
    exit 1
}

# ─── Input: Transport variant ───────────────────────────────────────────────────

if ([string]::IsNullOrEmpty($Transport)) {
    $Transport = "tcp"
}
$Transport = $Transport.ToLower()
$validTransports = @("tcp", "http", "telegram", "dropbox", "smb")
if ($validTransports -notcontains $Transport) {
    Write-Host "[-] Transport must be one of: $($validTransports -join ', ')." -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "[*] Creating listener: ${ListenerName}_listener" -ForegroundColor Cyan
Write-Host "      Protocol    : $Protocol" -ForegroundColor Cyan
Write-Host "      Type        : $ListenerType" -ForegroundColor Cyan
Write-Host "      Transport   : $Transport" -ForegroundColor Cyan
Write-Host "      Directory   : $OutDir\" -ForegroundColor Cyan
Write-Host ""

# ─── Create directory ───────────────────────────────────────────────────────────

New-Item -ItemType Directory -Path $OutDir -Force | Out-Null

# ─── Substitute function ───────────────────────────────────────────────────────

function Substitute-Template {
    param(
        [string]$Source,
        [string]$Destination
    )
    $content = Get-Content -Path $Source -Raw -Encoding UTF8
    $content = $content -replace '__NAME_CAP__', $ListenerNameCap
    $content = $content -replace '__NAME__', $ListenerName
    $content = $content -replace '__PROTOCOL_CAP__', $ProtocolCap
    $content = $content -replace '__PROTOCOL__', $Protocol
    $content = $content -replace '__LISTENER_TYPE__', $ListenerType
    [System.IO.File]::WriteAllText($Destination, $content, [System.Text.UTF8Encoding]::new($false))
}

function Substitute-Protocol {
    param(
        [string]$Source,
        [string]$Destination,
        [string]$Package
    )
    $content = Get-Content -Path $Source -Raw -Encoding UTF8
    $content = $content -replace '__PACKAGE__', $Package
    [System.IO.File]::WriteAllText($Destination, $content, [System.Text.UTF8Encoding]::new($false))
}

# ─── Copy template files ───────────────────────────────────────────────────────

Write-Host "[*] Generating listener files..." -ForegroundColor Cyan
Substitute-Template (Join-Path $TemplateDir "config.yaml")      (Join-Path $OutDir "config.yaml")
Substitute-Template (Join-Path $TemplateDir "Makefile")         (Join-Path $OutDir "Makefile")

# go.mod: use protocol override if available, else base template
$protoGoMod = Join-Path $protoDir "go_mod.tmpl"
if (Test-Path $protoGoMod) {
    Write-Host "  [+] Using protocol go.mod override" -ForegroundColor Green
    Substitute-Template $protoGoMod (Join-Path $OutDir "go.mod")
} else {
    Substitute-Template (Join-Path $TemplateDir "go.mod")       (Join-Path $OutDir "go.mod")
    Substitute-Template (Join-Path $TemplateDir "go.sum")       (Join-Path $OutDir "go.sum")
}

# pl_main.go: check for transport-specific listener main override in protocol
$listenerMain = Join-Path $protoDir "listener_main_${Transport}.go.tmpl"
if (Test-Path $listenerMain) {
    Write-Host "  [+] Using protocol listener main override: listener_main_${Transport}.go.tmpl" -ForegroundColor Green
    Substitute-Template $listenerMain (Join-Path $OutDir "pl_main.go")
} else {
    Substitute-Template (Join-Path $TemplateDir "pl_main.go")   (Join-Path $OutDir "pl_main.go")
}

# pl_internal.go: skip for non-socket transports; use protocol override if available, else base
$skipInternal = @("http", "telegram", "dropbox")
if ($skipInternal -notcontains $Transport) {
    $protoInternal = Join-Path $protoDir "pl_internal.go.tmpl"
    if (Test-Path $protoInternal) {
        Write-Host "  [+] Using protocol internal override: pl_internal.go.tmpl" -ForegroundColor Green
        Substitute-Template $protoInternal (Join-Path $OutDir "pl_internal.go")
    } else {
        Substitute-Template (Join-Path $TemplateDir "pl_internal.go") (Join-Path $OutDir "pl_internal.go")
    }
}

# pl_transport.go: check for transport-specific override, else default override, else base
$protoTransportVariant = Join-Path $protoDir "pl_transport_${Transport}.go.tmpl"
$protoTransport = Join-Path $protoDir "pl_transport.go.tmpl"
if (Test-Path $protoTransportVariant) {
    Write-Host "  [+] Using protocol transport override: pl_transport_${Transport}.go.tmpl" -ForegroundColor Green
    Substitute-Template $protoTransportVariant (Join-Path $OutDir "pl_transport.go")
} elseif (Test-Path $protoTransport) {
    Write-Host "  [+] Using protocol transport override: pl_transport.go.tmpl" -ForegroundColor Green
    Substitute-Template $protoTransport (Join-Path $OutDir "pl_transport.go")
} else {
    Substitute-Template (Join-Path $TemplateDir "pl_transport.go") (Join-Path $OutDir "pl_transport.go")
}

# map.go: only needed for transports that use concurrent maps (TCP)
$skipMap = @("http", "telegram", "dropbox", "smb")
if ($skipMap -notcontains $Transport) {
    Substitute-Template (Join-Path $TemplateDir "map.go")       (Join-Path $OutDir "map.go")
}

# ax_config.axs: check for transport-specific override in protocol, else base
$protoAxConfig = Join-Path $protoDir "ax_config_${Transport}.axs.tmpl"
if (Test-Path $protoAxConfig) {
    Write-Host "  [+] Using protocol ax_config override: ax_config_${Transport}.axs.tmpl" -ForegroundColor Green
    Substitute-Template $protoAxConfig (Join-Path $OutDir "ax_config.axs")
} else {
    Substitute-Template (Join-Path $TemplateDir "ax_config.axs") (Join-Path $OutDir "ax_config.axs")
}

# ─── Copy from protocol ────────────────────────────────────────────────────────

Write-Host "[*] Applying protocol: $Protocol" -ForegroundColor Cyan

# pl_crypto.go from protocol's crypto.go.tmpl
$cryptoTmpl = Join-Path $protoDir "crypto.go.tmpl"
if (Test-Path $cryptoTmpl) {
    Substitute-Protocol $cryptoTmpl (Join-Path $OutDir "pl_crypto.go") "main"
} else {
    Write-Host "[!] No crypto.go.tmpl in protocol '$Protocol', using template default." -ForegroundColor Yellow
    Substitute-Template (Join-Path $TemplateDir "pl_crypto.go") (Join-Path $OutDir "pl_crypto.go")
}

# pl_utils.go: merge constants.go.tmpl + types.go.tmpl from protocol
$constantsTmpl = Join-Path $protoDir "constants.go.tmpl"
$typesTmpl = Join-Path $protoDir "types.go.tmpl"
if ((Test-Path $constantsTmpl) -and (Test-Path $typesTmpl)) {
    # Read both, strip duplicate package line from types, merge
    $constContent = Get-Content -Path $constantsTmpl -Raw -Encoding UTF8
    $typesContent = Get-Content -Path $typesTmpl -Raw -Encoding UTF8

    # Remove the package line and imports from constants (we keep the types file's package+imports)
    $constBody = ($constContent -replace '(?ms)^package\s+\S+\s*\n', '')

    # Merge: types file (has package + imports) + constants body
    $merged = $typesContent.TrimEnd() + "`n`n" + $constBody.TrimStart()
    $merged = $merged -replace '__PACKAGE__', 'main'

    [System.IO.File]::WriteAllText((Join-Path $OutDir "pl_utils.go"), $merged, [System.Text.UTF8Encoding]::new($false))
} else {
    Write-Host "[!] Protocol missing constants/types templates, using template default." -ForegroundColor Yellow
    Substitute-Template (Join-Path $TemplateDir "pl_utils.go") (Join-Path $OutDir "pl_utils.go")
}

# ─── Summary ────────────────────────────────────────────────────────────────────

Write-Host ""
Write-Host "[+] Listener '${ListenerName}_listener' scaffolded successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "Directory structure:" -ForegroundColor Cyan
Write-Host ""
Write-Host "  ${ListenerName}_listener\"
Write-Host "  |-- config.yaml          # Listener manifest"
Write-Host "  |-- go.mod               # Go module"
Write-Host "  |-- Makefile             # Build targets"
Write-Host "  |-- pl_main.go           # Plugin entry + Teamserver interface"
Write-Host "  |-- pl_internal.go       # Internal listener registration parser"
Write-Host "  |-- pl_transport.go      # Transport: Start/Stop/handleConnection"
Write-Host "  |-- pl_crypto.go         # Encrypt/Decrypt (from protocol: $Protocol)"
Write-Host "  |-- pl_utils.go          # Wire types + constants (from protocol: $Protocol)"
Write-Host "  |-- map.go               # Thread-safe concurrent map"
Write-Host "  \-- ax_config.axs        # Listener UI form"
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "  1. cd $OutDir"
Write-Host "  2. Edit pl_transport.go to customize handleConnection for your transport"
Write-Host "  3. Edit ax_config.axs if you need different UI fields"
Write-Host "  4. go mod tidy"
Write-Host "  5. make plugin"
Write-Host ""
Write-Host "  Agent compatibility:" -ForegroundColor Cyan
Write-Host "    Agents using protocol '$Protocol' are compatible with this listener."
Write-Host "    Set listeners: [""${ListenerNameCap}${ProtocolCap}""] in your agent's config.yaml."
Write-Host ""
