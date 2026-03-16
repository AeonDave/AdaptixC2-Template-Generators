<#
.SYNOPSIS
    Scaffold a new AdaptixC2 service plugin.

.DESCRIPTION
    Creates <name>_service/ (or <name>_wrapper/) with all files ready to implement.
    Output goes to -OutputDir (or ADAPTIX_OUTPUT_DIR env var, or ./output).

    When -Wrapper is set (or the user answers yes at the interactive prompt),
    the generator includes the post-build wrapper pipeline: an event hook on
    agent.generate, pl_wrapper.go (stage engine), and a wrapper-specific UI.

.PARAMETER Name
    Service name (lowercase alphanumeric). Skips interactive prompt when provided.

.PARAMETER Wrapper
    Include the post-build wrapper pipeline.  When omitted in interactive mode,
    the generator asks.

.PARAMETER OutputDir
    Directory where <name>_service/ or <name>_wrapper/ will be created.
    Default: ADAPTIX_OUTPUT_DIR env var, or ./output.

.EXAMPLE
    .\generator.ps1

.EXAMPLE
    .\generator.ps1 -Name telegram

.EXAMPLE
    .\generator.ps1 -Name crystalpalace -Wrapper

.EXAMPLE
    .\generator.ps1 -Name telegram -OutputDir ..\my-adaptix\extenders
#>
param(
    [string]$Name      = "",
    [string]$OutputDir = "",
    [switch]$Wrapper
)

$ErrorActionPreference = "Stop"

$ScriptDir     = Split-Path -Parent $MyInvocation.MyCommand.Path
$TemplateDir   = Join-Path $ScriptDir "templates"
$TemplatesRoot = Split-Path -Parent $ScriptDir

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
Write-Host "║   AdaptixC2 Template Service Generator        ║" -ForegroundColor Cyan
Write-Host "╚═══════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

# ─── Input: Service name ────────────────────────────────────────────────────────

$IsWrapper = $Wrapper.IsPresent

if (-not [string]::IsNullOrEmpty($Name)) {
    $ServiceName = ($Name.ToLower() -replace '[^a-z0-9_]', '')
    if ([string]::IsNullOrEmpty($ServiceName)) { Write-Host "[-] Invalid name." -ForegroundColor Red; exit 1 }
} else {
    while ($true) {
        $ServiceName = Read-Host "Service name (lowercase, e.g. telegram)"
        $ServiceName = ($ServiceName.ToLower() -replace '[^a-z0-9_]', '')
        if ([string]::IsNullOrEmpty($ServiceName)) {
            Write-Host "[!] Name cannot be empty." -ForegroundColor Yellow
            continue
        }
        break
    }
}

# ─── Input: Wrapper option ──────────────────────────────────────────────────────

if (-not $IsWrapper -and [string]::IsNullOrEmpty($Name)) {
    $answer = Read-Host "Include post-build wrapper pipeline? [y/N]"
    if ($answer -match '^[Yy]') {
        $IsWrapper = $true
    }
}

# Determine suffix and output directory
$Suffix = if ($IsWrapper) { "wrapper" } else { "service" }
$OutDir = Join-Path $ExtendersDir "${ServiceName}_${Suffix}"
if (Test-Path $OutDir) { Write-Host "[-] Directory ${ServiceName}_${Suffix} already exists!" -ForegroundColor Red; exit 1 }

# Capitalize first letter
$ServiceNameCap = $ServiceName.Substring(0,1).ToUpper() + $ServiceName.Substring(1)

Write-Host ""
Write-Host "[*] Creating ${Suffix}: ${ServiceName}_${Suffix}" -ForegroundColor Cyan
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
    $content = $content -replace '__NAME_CAP__', $ServiceNameCap
    $content = $content -replace '__NAME__', $ServiceName
    [System.IO.File]::WriteAllText($Destination, $content, [System.Text.UTF8Encoding]::new($false))
}

# ─── Select template source ────────────────────────────────────────────────────
# Wrapper templates override the base when the wrapper option is active.

function Resolve-Template {
    param([string]$FileName)
    if ($IsWrapper) {
        $override = Join-Path $TemplateDir "wrapper\$FileName"
        if (Test-Path $override) { return $override }
    }
    return (Join-Path $TemplateDir $FileName)
}

# ─── Copy template files ───────────────────────────────────────────────────────

Write-Host "[*] Generating $Suffix files..." -ForegroundColor Cyan
Substitute-Template (Resolve-Template "config.yaml")   (Join-Path $OutDir "config.yaml")
Substitute-Template (Resolve-Template "go.mod")        (Join-Path $OutDir "go.mod")
Substitute-Template (Resolve-Template "go.sum")        (Join-Path $OutDir "go.sum")
Substitute-Template (Resolve-Template "Makefile")      (Join-Path $OutDir "Makefile")
Substitute-Template (Resolve-Template "pl_main.go")    (Join-Path $OutDir "pl_main.go")
Substitute-Template (Resolve-Template "ax_config.axs") (Join-Path $OutDir "ax_config.axs")

if ($IsWrapper) {
    $wrapperSrc = Join-Path $TemplateDir "wrapper\pl_wrapper.go"
    if (Test-Path $wrapperSrc) {
        Substitute-Template $wrapperSrc (Join-Path $OutDir "pl_wrapper.go")
    }
}

# ─── Summary ────────────────────────────────────────────────────────────────────

Write-Host ""
Write-Host "[+] ${Suffix} '${ServiceName}_${Suffix}' scaffolded successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "Directory structure:" -ForegroundColor Cyan
Write-Host ""
Write-Host "  ${ServiceName}_${Suffix}\"
Write-Host "  |-- config.yaml          # Service manifest"
Write-Host "  |-- go.mod               # Go module"
Write-Host "  |-- Makefile             # Build targets"
Write-Host "  |-- pl_main.go           # Plugin entry$(if ($IsWrapper) { ' + event hooks + handlers' } else { ' + Call handler' })"
if ($IsWrapper) {
    Write-Host "  |-- pl_wrapper.go        # Pipeline engine (stages)"
}
Write-Host "  \-- ax_config.axs        # Service UI form"
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "  1. cd $OutDir"
if ($IsWrapper) {
    Write-Host "  2. Edit pl_main.go — register stages in initStages()"
    Write-Host "  3. Add stage functions (e.g. stageEncrypt, stagePack) in pl_wrapper.go or new files"
} else {
    Write-Host "  2. Edit pl_main.go — add function handlers in the Call() switch"
    Write-Host "  3. Edit ax_config.axs — add your functions to the combo + UI fields"
}
Write-Host "  4. go mod tidy"
Write-Host "  5. make plugin"
Write-Host ""
