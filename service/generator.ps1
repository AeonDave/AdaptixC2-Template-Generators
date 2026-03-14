<#
.SYNOPSIS
    Scaffold a new AdaptixC2 service plugin.

.DESCRIPTION
    Creates <name>_service/ with all files ready to implement.
    Output goes to -OutputDir (or ADAPTIX_OUTPUT_DIR env var, or ./output).

.PARAMETER Name
    Service name (lowercase alphanumeric). Skips interactive prompt when provided.

.PARAMETER OutputDir
    Directory where <name>_service/ will be created.
    Default: ADAPTIX_OUTPUT_DIR env var, or ./output.

.EXAMPLE
    .\generator.ps1

.EXAMPLE
    .\generator.ps1 -Name telegram

.EXAMPLE
    .\generator.ps1 -Name telegram -OutputDir ..\my-adaptix\extenders
#>
param(
    [string]$Name      = "",
    [string]$OutputDir = ""
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

if (-not [string]::IsNullOrEmpty($Name)) {
    $ServiceName = ($Name.ToLower() -replace '[^a-z0-9_]', '')
    $OutDir = Join-Path $ExtendersDir "${ServiceName}_service"
    if ([string]::IsNullOrEmpty($ServiceName)) { Write-Host "[-] Invalid name." -ForegroundColor Red; exit 1 }
    if (Test-Path $OutDir) { Write-Host "[-] Directory ${ServiceName}_service already exists!" -ForegroundColor Red; exit 1 }
} else {
    while ($true) {
        $ServiceName = Read-Host "Service name (lowercase, e.g. telegram)"
        $ServiceName = ($ServiceName.ToLower() -replace '[^a-z0-9_]', '')
        if ([string]::IsNullOrEmpty($ServiceName)) {
            Write-Host "[!] Name cannot be empty." -ForegroundColor Yellow
            continue
        }
        $OutDir = Join-Path $ExtendersDir "${ServiceName}_service"
        if (Test-Path $OutDir) {
            Write-Host "[!] Directory ${ServiceName}_service already exists!" -ForegroundColor Yellow
            continue
        }
        break
    }
}

# Capitalize first letter
$ServiceNameCap = $ServiceName.Substring(0,1).ToUpper() + $ServiceName.Substring(1)

Write-Host ""
Write-Host "[*] Creating service: ${ServiceName}_service" -ForegroundColor Cyan
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

# ─── Copy template files ───────────────────────────────────────────────────────

Write-Host "[*] Generating service files..." -ForegroundColor Cyan
Substitute-Template (Join-Path $TemplateDir "config.yaml")   (Join-Path $OutDir "config.yaml")
Substitute-Template (Join-Path $TemplateDir "go.mod")        (Join-Path $OutDir "go.mod")
Substitute-Template (Join-Path $TemplateDir "go.sum")        (Join-Path $OutDir "go.sum")
Substitute-Template (Join-Path $TemplateDir "Makefile")      (Join-Path $OutDir "Makefile")
Substitute-Template (Join-Path $TemplateDir "pl_main.go")    (Join-Path $OutDir "pl_main.go")
Substitute-Template (Join-Path $TemplateDir "ax_config.axs") (Join-Path $OutDir "ax_config.axs")

# ─── Summary ────────────────────────────────────────────────────────────────────

Write-Host ""
Write-Host "[+] Service '${ServiceName}_service' scaffolded successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "Directory structure:" -ForegroundColor Cyan
Write-Host ""
Write-Host "  ${ServiceName}_service\"
Write-Host "  |-- config.yaml          # Service manifest"
Write-Host "  |-- go.mod               # Go module"
Write-Host "  |-- Makefile             # Build targets"
Write-Host "  |-- pl_main.go           # Plugin entry + Call handler"
Write-Host "  \-- ax_config.axs        # Service UI form"
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "  1. cd $OutDir"
Write-Host "  2. Edit pl_main.go — add function handlers in the Call() switch"
Write-Host "  3. Edit ax_config.axs — add your functions to the combo + UI fields"
Write-Host "  4. go mod tidy"
Write-Host "  5. make plugin"
Write-Host ""
