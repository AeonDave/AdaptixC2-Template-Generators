<#
.SYNOPSIS
    AdaptixC2 unified template generator.

.DESCRIPTION
    Root entry-point that dispatches to the appropriate sub-generator:
      1) Agent     - scaffold a new agent extender
      2) Listener  - scaffold a new listener extender
      3) Protocol  - create a new wire-protocol definition
      4) Crypto    - create/swap the crypto implementation of an existing protocol

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
    [ValidateSet("agent","listener","service","wrapper","protocol","crypto","delete","")]
    [string]$Mode      = "",
    [string]$OutputDir = "",
    [ValidateSet("go","cpp","rust","")]
    [string]$Language  = "",
    [string]$Toolchain = ""
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

# ─── UI helpers ────────────────────────────────────────────────────────────────

function Write-CyberBanner {
    Write-Host ""
    Write-Host "┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓" -ForegroundColor DarkGreen
    Write-Host "┃   █████╗ ██████╗  █████╗ ██████╗ ████████╗██╗██╗  ██╗ ██████╗██████╗   ┃" -ForegroundColor DarkGreen
    Write-Host "┃  ██╔══██╗██╔══██╗██╔══██╗██╔══██╗╚══██╔══╝██║╚██╗██╔╝██╔════╝╚════██╗  ┃" -ForegroundColor DarkGreen
    Write-Host "┃  ███████║██║  ██║███████║██████╔╝   ██║   ██║ ╚███╔╝ ██║      █████╔╝  ┃" -ForegroundColor Green
    Write-Host "┃  ██╔══██║██║  ██║██╔══██║██╔═══╝    ██║   ██║ ██╔██╗ ██║     ██╔═══╝   ┃" -ForegroundColor Green
    Write-Host "┃  ██║  ██║██████╔╝██║  ██║██║        ██║   ██║██╔╝ ██╗╚██████╗███████╗  ┃" -ForegroundColor DarkGreen
    Write-Host "┃  ╚═╝  ╚═╝╚═════╝ ╚═╝  ╚═╝╚═╝        ╚═╝   ╚═╝╚═╝  ╚═╝ ╚═════╝╚══════╝  ┃" -ForegroundColor DarkGreen
    Write-Host "┃                                                                        ┃" -ForegroundColor DarkGreen
    Write-Host "┃          Template Generator // agents • listeners • services           ┃" -ForegroundColor Green
    Write-Host "┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛" -ForegroundColor DarkGreen
    Write-Host ""
}

function Write-CyberSection([string]$Title) {
    Write-Host "[:: $Title ::]" -ForegroundColor Cyan
}

function Write-CyberHint([string]$Message) {
    Write-Host "    $Message" -ForegroundColor DarkGray
}

function Write-CyberMenu([string]$Prompt, [array]$Items) {
    Write-CyberSection $Prompt
    Write-Host ""
    for ($i = 0; $i -lt $Items.Count; $i++) {
        $item = $Items[$i]
        $index = "[{0}]" -f ($i + 1)
        Write-Host "  $index" -ForegroundColor Green -NoNewline
        Write-Host " $($item.Label)" -ForegroundColor Cyan -NoNewline
        if ($item.ContainsKey('Tag') -and -not [string]::IsNullOrWhiteSpace($item.Tag)) {
            Write-Host "  <$($item.Tag)>" -ForegroundColor DarkGreen -NoNewline
        }
        Write-Host ""
        Write-CyberHint $item.Desc
    }
    Write-Host ""
}

function Read-Choice([string]$Prompt, [int]$Min, [int]$Max) {
    $raw = Read-Host $Prompt
    $value = 0
    if (-not [int]::TryParse($raw, [ref]$value) -or $value -lt $Min -or $value -gt $Max) {
        Write-Host "[-] Invalid choice." -ForegroundColor Red
        exit 1
    }
    return $value
}

function Write-LaunchMessage([string]$Name) {
    Write-Host "[>] Launching $Name..." -ForegroundColor Cyan
    Write-Host ""
}

Write-CyberBanner

# ─── Mode selection ─────────────────────────────────────────────────────────────

$modes = @(
    @{ Key = "agent";    Label = "Generate Agent";    Desc = "Scaffold a new agent extender"; Tag = "GO/CPP/RUST" },
    @{ Key = "listener"; Label = "Generate Listener"; Desc = "Scaffold a new listener extender"; Tag = "TRANSPORT" },
    @{ Key = "service";  Label = "Generate Service";  Desc = "Scaffold a new service extender"; Tag = "HOOKS" },
    @{ Key = "wrapper";  Label = "Generate Wrapper";  Desc = "Scaffold a service with wrapper pipeline mode enabled"; Tag = "PIPELINE" },
    @{ Key = "protocol"; Label = "Create Protocol";   Desc = "Create a new wire-protocol definition"; Tag = "WIRE" },
    @{ Key = "crypto";   Label = "Create Crypto";     Desc = "Generate or replace the crypto template for a protocol"; Tag = "CRYPTO" },
    @{ Key = "delete";   Label = "Delete";            Desc = "Remove a crypto template, protocol, or generated output"; Tag = "CLEANUP" }
)

if ([string]::IsNullOrEmpty($Mode)) {
    Write-CyberMenu "Select generation mode" $modes
    $idx = Read-Choice "Select option" 1 $modes.Count
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
        Write-LaunchMessage "Agent Generator"
        & $target @fwdArgs @extraArgs
    }
    "listener" {
        $target = Join-Path $ScriptDir "listener\generator.ps1"
        Write-LaunchMessage "Listener Generator"
        & $target @fwdArgs @extraArgs
    }
    "service" {
        $target = Join-Path $ScriptDir "service\generator.ps1"
        Write-LaunchMessage "Service Generator"
        & $target @fwdArgs @extraArgs
    }
    "wrapper" {
        $target = Join-Path $ScriptDir "service\generator.ps1"
        Write-LaunchMessage "Service Generator (wrapper mode)"
        & $target -Wrapper @fwdArgs @extraArgs
    }
    "protocol" {
        $target = Join-Path $ScriptDir "protocols\generator.ps1"
        Write-LaunchMessage "Protocol Generator"
        & $target @fwdArgs @extraArgs
    }
    "crypto" {
        $target = Join-Path $ScriptDir "protocols\crypto_generator.ps1"
        Write-LaunchMessage "Crypto Generator"
        & $target @fwdArgs @extraArgs
    }
    "delete" {
        $deleteModes = @(
            @{ Key = "1"; Label = "Crypto template"; Desc = "Remove a crypto .go.tmpl from _crypto/"; Tag = "WIPE" },
            @{ Key = "2"; Label = "Protocol"; Desc = "Remove an entire protocol definition"; Tag = "PURGE" },
            @{ Key = "3"; Label = "Generated output"; Desc = "Remove a generated project from output/"; Tag = "SCRUB" }
        )
        Write-CyberMenu "Select deletion target" $deleteModes
        $delChoice = [string](Read-Choice "Select option" 1 $deleteModes.Count)

        switch ($delChoice) {
            "1" {
                # ── Delete crypto template ──
                $cryptoDir = Join-Path $ScriptDir "protocols\_crypto"
                $items = @()
                if (Test-Path $cryptoDir) {
                    $items = @(Get-ChildItem -Path $cryptoDir -Filter "*.go.tmpl" | Sort-Object Name)
                }
                if ($items.Count -eq 0) {
                    Write-Host "[-] No crypto templates found." -ForegroundColor Red; exit 1
                }
                Write-CyberSection "Available crypto templates"
                Write-Host ""
                for ($i = 0; $i -lt $items.Count; $i++) {
                    $key = $items[$i].BaseName -replace '\.go$', ''
                    Write-Host "  [$($i+1)] $key" -ForegroundColor Cyan
                }
                Write-Host ""
                $pIdx = Read-Choice "Select crypto to delete" 1 $items.Count
                $target = $items[$pIdx - 1]
                $targetName = $target.BaseName -replace '\.go$', ''
                $confirm = Read-Host "Delete crypto '$targetName'? [y/N]"
                if ($confirm -ne 'y') { Write-Host "Cancelled."; exit 0 }
                Remove-Item -Path $target.FullName -Force
                Write-Host ""
                Write-Host "[+] Deleted crypto template: _crypto/$($target.Name)" -ForegroundColor Green
                Write-Host ""
            }
            "2" {
                # ── Delete protocol ──
                $protoBase = Join-Path $ScriptDir "protocols"
                $protected = @('_scaffold', '_crypto')
                $items = @()
                Get-ChildItem -Path $protoBase -Directory | Where-Object {
                    $_.Name -notin $protected -and (Test-Path (Join-Path $_.FullName 'meta.yaml'))
                } | Sort-Object Name | ForEach-Object { $items += $_ }
                if ($items.Count -eq 0) {
                    Write-Host "[-] No deletable protocols found." -ForegroundColor Red; exit 1
                }
                Write-CyberSection "Available protocols"
                Write-Host ""
                for ($i = 0; $i -lt $items.Count; $i++) {
                    Write-Host "  [$($i+1)] $($items[$i].Name)" -ForegroundColor Cyan
                }
                Write-Host ""
                $pIdx = Read-Choice "Select protocol to delete" 1 $items.Count
                $target = $items[$pIdx - 1]
                $confirm = Read-Host "Delete protocol '$($target.Name)' and all its files? [y/N]"
                if ($confirm -ne 'y') { Write-Host "Cancelled."; exit 0 }
                Remove-Item -Path $target.FullName -Recurse -Force
                Write-Host ""
                Write-Host "[+] Deleted protocol: $($target.Name)/" -ForegroundColor Green
                Write-Host ""
            }
            "3" {
                # ── Delete generated output ──
                $outDir = $OutputDir
                if ([string]::IsNullOrEmpty($outDir)) {
                    $outDir = $env:ADAPTIX_OUTPUT_DIR
                }
                if ([string]::IsNullOrEmpty($outDir)) {
                    $outDir = Join-Path $ScriptDir "output"
                }
                if (-not (Test-Path $outDir)) {
                    Write-Host "[-] Output directory not found: $outDir" -ForegroundColor Red; exit 1
                }
                $items = @(Get-ChildItem -Path $outDir -Directory | Sort-Object Name)
                if ($items.Count -eq 0) {
                    Write-Host "[-] No generated projects found in $outDir" -ForegroundColor Red; exit 1
                }
                Write-CyberSection "Generated projects in ${outDir}"
                Write-Host ""
                for ($i = 0; $i -lt $items.Count; $i++) {
                    Write-Host "  [$($i+1)] $($items[$i].Name)" -ForegroundColor Cyan
                }
                Write-Host ""
                $pIdx = Read-Choice "Select project to delete" 1 $items.Count
                $target = $items[$pIdx - 1]
                $confirm = Read-Host "Delete '$($target.Name)' and all its contents? [y/N]"
                if ($confirm -ne 'y') { Write-Host "Cancelled."; exit 0 }
                Remove-Item -Path $target.FullName -Recurse -Force
                Write-Host ""
                Write-Host "[+] Deleted: $($target.Name)/" -ForegroundColor Green
                Write-Host ""
            }
            default {
                Write-Host "[-] Invalid choice." -ForegroundColor Red; exit 1
            }
        }
    }
    default {
        Write-Host "[-] Unknown mode: $Mode" -ForegroundColor Red
        exit 1
    }
}
