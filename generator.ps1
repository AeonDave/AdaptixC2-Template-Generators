<#
.SYNOPSIS
    AdaptixC2 unified template generator.

.DESCRIPTION
    Root entry-point that dispatches to the appropriate sub-generator:
      1) Agent     - scaffold a new agent extender
      2) Listener  - scaffold a new listener extender
      3) Protocol  - create a new wire-protocol definition
      4) Crypto    - swap the crypto implementation of an existing protocol

    Each option forwards to its dedicated generator under agent/, listener/,
    or protocols/.  All parameters are passed through so both interactive and
    non-interactive modes work.

.PARAMETER Mode
    Directly select the generator: "agent", "listener", "protocol", "crypto".
    When omitted an interactive menu is shown.

.PARAMETER OutputDir
    Directory where generated extenders are written.
    Forwarded to agent/listener generators.  Default: ADAPTIX_OUTPUT_DIR env var, or ./output.

.PARAMETER Language
    Implant language for agent generation: "go" (default), "cpp", or "rust".
    Only used with -Mode agent.  Forwarded to the agent sub-generator.

.PARAMETER Toolchain
    Build toolchain for agent generation (e.g. "go-garble", "mingw", "cargo").
    Only used with -Mode agent.  When omitted the toolchain is auto-detected from language.

.EXAMPLE
    .\generator.ps1

.EXAMPLE
    .\generator.ps1 -Mode agent

.EXAMPLE
    .\generator.ps1 -Mode agent -OutputDir ..\my-adaptix\extenders

.EXAMPLE
    .\generator.ps1 -Mode agent -Language cpp -Toolchain mingw
#>
param(
    [ValidateSet("agent","listener","service","protocol","crypto","")]
    [string]$Mode      = "",
    [string]$OutputDir = "",
    [ValidateSet("go","cpp","rust","")]
    [string]$Language  = "",
    [string]$Toolchain = ""
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

# ─── Banner ─────────────────────────────────────────────────────────────────────

Write-Host ""
Write-Host "╔═══════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║   AdaptixC2 Template Generator                ║" -ForegroundColor Cyan
Write-Host "╚═══════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

# ─── Mode selection ─────────────────────────────────────────────────────────────

$modes = @(
    @{ Key = "agent";    Label = "Generate Agent";    Desc = "Scaffold a new agent extender" },
    @{ Key = "listener"; Label = "Generate Listener"; Desc = "Scaffold a new listener extender" },
    @{ Key = "service";  Label = "Generate Service";  Desc = "Scaffold a new service extender" },
    @{ Key = "protocol"; Label = "Create Protocol";   Desc = "Create a new wire-protocol definition" },
    @{ Key = "crypto";   Label = "Swap Crypto";       Desc = "Generate or replace the crypto template for a protocol" }
)

if ([string]::IsNullOrEmpty($Mode)) {
    Write-Host "What do you want to generate?" -ForegroundColor Cyan
    Write-Host ""
    for ($i = 0; $i -lt $modes.Count; $i++) {
        Write-Host "  [$($i+1)] $($modes[$i].Label)" -ForegroundColor White -NoNewline
        Write-Host "  - $($modes[$i].Desc)" -ForegroundColor DarkGray
    }
    Write-Host ""
    $choice = Read-Host "Select option"
    $idx = [int]$choice
    if ($idx -lt 1 -or $idx -gt $modes.Count) {
        Write-Host "[-] Invalid choice." -ForegroundColor Red
        exit 1
    }
    $Mode = $modes[$idx - 1].Key
}

# ─── Dispatch ───────────────────────────────────────────────────────────────────

# Collect remaining arguments to forward
$fwdArgs = @{}
foreach ($key in $PSBoundParameters.Keys) {
    if ($key -ne 'Mode') {
        $fwdArgs[$key] = $PSBoundParameters[$key]
    }
}
# Also forward any unbound positional arguments
$extraArgs = @()
if ($args.Count -gt 0) {
    $extraArgs = $args
}

switch ($Mode) {
    "agent" {
        $target = Join-Path $ScriptDir "agent\generator.ps1"
        Write-Host "[*] Launching Agent Generator..." -ForegroundColor Yellow
        Write-Host ""
        & $target @fwdArgs @extraArgs
    }
    "listener" {
        $target = Join-Path $ScriptDir "listener\generator.ps1"
        Write-Host "[*] Launching Listener Generator..." -ForegroundColor Yellow
        Write-Host ""
        & $target @fwdArgs @extraArgs
    }
    "service" {
        $target = Join-Path $ScriptDir "service\generator.ps1"
        Write-Host "[*] Launching Service Generator..." -ForegroundColor Yellow
        Write-Host ""
        & $target @fwdArgs @extraArgs
    }
    "protocol" {
        $target = Join-Path $ScriptDir "protocols\generator.ps1"
        Write-Host "[*] Launching Protocol Generator..." -ForegroundColor Yellow
        Write-Host ""
        & $target @fwdArgs @extraArgs
    }
    "crypto" {
        $target = Join-Path $ScriptDir "protocols\crypto_generator.ps1"
        Write-Host "[*] Launching Crypto Generator..." -ForegroundColor Yellow
        Write-Host ""
        & $target @fwdArgs @extraArgs
    }
    default {
        Write-Host "[-] Unknown mode: $Mode" -ForegroundColor Red
        exit 1
    }
}
