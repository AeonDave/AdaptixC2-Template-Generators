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

$OutDir = Join-Path $ScriptDir $ProtoName
if (Test-Path $OutDir) {
    Write-Host "[-] Protocol '$ProtoName' already exists at $OutDir" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "[*] Creating protocol: $ProtoName" -ForegroundColor Cyan
Write-Host "      Directory : $OutDir\" -ForegroundColor Cyan
Write-Host ""

# ─── Scaffold ───────────────────────────────────────────────────────────────────

New-Item -ItemType Directory -Path $OutDir -Force | Out-Null
foreach ($f in Get-ChildItem -Path $ScaffoldDir -File) {
    $content = Get-Content -Path $f.FullName -Raw -Encoding UTF8
    $content = $content -replace '__PROTO_NAME__', $ProtoName
    $dest = Join-Path $OutDir $f.Name
    [System.IO.File]::WriteAllText($dest, $content, [System.Text.UTF8Encoding]::new($false))
}

# ─── Summary ────────────────────────────────────────────────────────────────────

Write-Host ""
Write-Host "[+] Protocol '$ProtoName' created!" -ForegroundColor Green
Write-Host ""
Write-Host "Files:" -ForegroundColor Cyan
Write-Host "  $ProtoName\"
Write-Host "  |-- meta.yaml           # Protocol metadata"
Write-Host "  |-- crypto.go.tmpl      # EncryptData / DecryptData"
Write-Host "  |-- constants.go.tmpl   # COMMAND_* / RESP_* constants"
Write-Host "  \-- types.go.tmpl       # Wire types, framing helpers"
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "  1. Edit crypto.go.tmpl    - implement your encryption"
Write-Host "  2. Edit types.go.tmpl     - define wire types and framing"
Write-Host "  3. Edit constants.go.tmpl - add command/response constants"
Write-Host "  4. Use with: generator.ps1 -Protocol $ProtoName"
Write-Host ""
