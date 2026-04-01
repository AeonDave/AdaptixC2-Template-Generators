<#
.SYNOPSIS
    Create a new protocol definition for AdaptixC2 agents and listeners.

.DESCRIPTION
    Scaffolds a new protocol directory in templates/protocols/<name>/ with
    crypto, constants, types, and metadata template files.

.PARAMETER Name
    Protocol name (lowercase alphanumeric + underscore).

.EXAMPLE
    cd extenders\templates\protocols
    .\generator.ps1

.EXAMPLE
    .\generator.ps1 -Name chacha
#>
param(
    [string]$Name = ""
)

$ErrorActionPreference = "Stop"

$ScriptDir    = Split-Path -Parent $MyInvocation.MyCommand.Path
$ScaffoldDir  = Join-Path $ScriptDir "_scaffold"

# ─── Banner ─────────────────────────────────────────────────────────────────────

Write-Host ""
Write-Host "╔═══════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║   AdaptixC2 Protocol Generator                ║" -ForegroundColor Cyan
Write-Host "╚═══════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

# ─── Validate scaffold ──────────────────────────────────────────────────────────

if (-not (Test-Path $ScaffoldDir)) {
    Write-Host "[-] Scaffold directory not found: $ScaffoldDir" -ForegroundColor Red
    exit 1
}

# ─── Input ──────────────────────────────────────────────────────────────────────

if (-not [string]::IsNullOrEmpty($Name)) {
    $ProtoName = ($Name.ToLower() -replace '[^a-z0-9_]', '')
    if ([string]::IsNullOrEmpty($ProtoName)) { Write-Host "[-] Invalid name." -ForegroundColor Red; exit 1 }
} else {
    while ($true) {
        $ProtoName = Read-Host "Protocol name (lowercase, e.g. chacha)"
        $ProtoName = ($ProtoName.ToLower() -replace '[^a-z0-9_]', '')
        if ([string]::IsNullOrEmpty($ProtoName)) {
            Write-Host "[!] Name cannot be empty." -ForegroundColor Yellow
            continue
        }
        break
    }
}

$ProtocolDirName = $ProtoName
$OutDir = Join-Path $ScriptDir $ProtocolDirName
if (Test-Path $OutDir) {
    Write-Host "[-] Protocol '$ProtocolDirName' already exists at $OutDir" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "[*] Creating protocol: $ProtocolDirName" -ForegroundColor Cyan
Write-Host "      Directory : $OutDir\" -ForegroundColor Cyan
Write-Host ""

# ─── Scaffold ───────────────────────────────────────────────────────────────────

New-Item -ItemType Directory -Path $OutDir | Out-Null

# Copy top-level scaffold files (Go templates + meta.yaml)
foreach ($f in Get-ChildItem -Path $ScaffoldDir -File) {
    $content = Get-Content -Path $f.FullName -Raw -Encoding UTF8
    $content = $content -replace '__PROTO_NAME__', $ProtoName
    $dest = Join-Path $OutDir $f.Name
    [System.IO.File]::WriteAllText($dest, $content, [System.Text.UTF8Encoding]::new($false))
}

# Copy implant overlay stubs (C++ and Rust) if present in scaffold
$ImplantScaffold = Join-Path $ScaffoldDir "implant"
if (Test-Path $ImplantScaffold) {
    $ImplantOut = Join-Path $OutDir "implant"
    foreach ($tmplFile in Get-ChildItem -Path $ImplantScaffold -Recurse -File) {
        $relPath = $tmplFile.FullName.Substring($ImplantScaffold.Length)
        $destPath = Join-Path $ImplantOut $relPath
        $destDir  = Split-Path -Parent $destPath
        if (-not (Test-Path $destDir)) {
            New-Item -ItemType Directory -Path $destDir -Force | Out-Null
        }
        $content = Get-Content -Path $tmplFile.FullName -Raw -Encoding UTF8
        $content = $content -replace '__PROTO_NAME__', $ProtoName
        [System.IO.File]::WriteAllText($destPath, $content, [System.Text.UTF8Encoding]::new($false))
    }
    Write-Host "[+] C++ and Rust implant overlay stubs created." -ForegroundColor Green
}

# ─── Summary ────────────────────────────────────────────────────────────────────

Write-Host ""
Write-Host "[+] Protocol '$ProtocolDirName' created!" -ForegroundColor Green
Write-Host "    Scaffold only: crypto/codec templates contain placeholders until you implement them." -ForegroundColor Yellow
Write-Host ""
Write-Host "Files:" -ForegroundColor Cyan
Write-Host "  $ProtocolDirName\"
Write-Host "  |-- meta.yaml                           # Protocol metadata"
Write-Host "  |-- crypto.go.tmpl                      # Go EncryptData / DecryptData"
Write-Host "  |-- constants.go.tmpl                   # Go COMMAND_* / RESP_* constants"
Write-Host "  |-- types.go.tmpl                       # Go wire types, framing helpers"
Write-Host "  \-- implant\"
Write-Host "      |-- cpp\crypto\crypto.{h,cpp}.tmpl  # C++ crypto stubs"
Write-Host "      |-- cpp\protocol\protocol.{h,cpp}.tmpl"
Write-Host "      |-- rust\src\crypto.rs.tmpl         # Rust crypto stub"
Write-Host "      \-- rust\src\protocol.rs.tmpl"
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "  1. Edit crypto.go.tmpl                - Go encryption (listener + Go agent)"
Write-Host "  2. Edit types.go.tmpl                 - Go wire types and framing"
Write-Host "  3. Edit constants.go.tmpl             - Go command/response constants"
Write-Host "  4. Edit implant\cpp\crypto\*          - C++ encryption (must match Go)"
Write-Host "  5. Edit implant\cpp\protocol\*        - C++ constants + wire types"
Write-Host "  6. Edit implant\rust\src\crypto.rs    - Rust encryption (must match Go)"
Write-Host "  7. Edit implant\rust\src\protocol.rs  - Rust constants + wire types"
Write-Host "  8. Use with: generator.ps1 -Protocol $ProtocolDirName"
Write-Host ""
Write-Host "Note:" -ForegroundColor Yellow
Write-Host "  The generated protocol scaffold is intentionally non-functional until the crypto and wire codec templates are completed."
Write-Host ""
