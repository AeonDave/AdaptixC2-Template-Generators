<#
.SYNOPSIS
    Scaffold a new modular AdaptixC2 agent.

.DESCRIPTION
    Creates <name>_agent/ with all files ready to implement.
    Output goes to -OutputDir (or ADAPTIX_OUTPUT_DIR env var, or ./output).

.PARAMETER Name
    Agent name (lowercase alphanumeric). Skips interactive prompt when provided.

.PARAMETER Watermark
    8-char hex watermark. Skips interactive prompt when provided.

.PARAMETER Protocol
    Protocol from protocols/ (default: "adaptix_default"). Overrides crypto, constants, and wire types.

.PARAMETER Language
    Implant language: go, cpp, rust. Default: go.
    Controls which template set is used for the implant source.

.PARAMETER Toolchain
    Toolchain name matching a YAML manifest under agent/toolchains/.
    Defaults: go → go-standard, cpp → mingw, rust → cargo.

.PARAMETER OutputDir
    Directory where <name>_agent/ will be created.
    Default: ADAPTIX_OUTPUT_DIR env var, or ./output.

.EXAMPLE
    .\generator.ps1

.EXAMPLE
    .\generator.ps1 -Name phantom -Watermark a1b2c3d4 -Protocol adaptix_default

.EXAMPLE
    .\generator.ps1 -Name phantom -Watermark a1b2c3d4 -Language cpp -Toolchain mingw

.EXAMPLE
    .\generator.ps1 -Name phantom -Watermark a1b2c3d4 -OutputDir ..\my-adaptix\extenders
#>
param(
    [string]$Name      = "",
    [string]$Watermark = "",
    [string]$Protocol  = "",
    [string[]]$ListenerNames = @(),
    [ValidateSet("go","cpp","rust","")]
    [string]$Language  = "",
    [string]$Toolchain = "",
    [string]$OutputDir = "",
    [switch]$Evasion
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
Write-Host "║   AdaptixC2 Template Agent Generator          ║" -ForegroundColor Cyan
Write-Host "╚═══════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

# ─── Input ──────────────────────────────────────────────────────────────────────

# Agent name
if (-not [string]::IsNullOrEmpty($Name)) {
    $AgentName = ($Name.ToLower() -replace '[^a-z0-9_]', '')
    $OutDir = Join-Path $ExtendersDir "stub_${AgentName}_agent"
    if ([string]::IsNullOrEmpty($AgentName)) { Write-Host "[-] Invalid name." -ForegroundColor Red; exit 1 }
    if (Test-Path $OutDir) { Write-Host "[-] Directory stub_${AgentName}_agent already exists!" -ForegroundColor Red; exit 1 }
} else {
    while ($true) {
        $AgentName = Read-Host "Agent name (lowercase, e.g. phantom)"
        $AgentName = ($AgentName.ToLower() -replace '[^a-z0-9_]', '')
        if ([string]::IsNullOrEmpty($AgentName)) {
            Write-Host "[!] Name cannot be empty." -ForegroundColor Yellow
            continue
        }
        $OutDir = Join-Path $ExtendersDir "stub_${AgentName}_agent"
        if (Test-Path $OutDir) {
            Write-Host "[!] Directory stub_${AgentName}_agent already exists!" -ForegroundColor Yellow
            continue
        }
        break
    }
}

$AgentDirName = "stub_${AgentName}_agent"

# Capitalize first letter
$AgentNameCap = $AgentName.Substring(0,1).ToUpper() + $AgentName.Substring(1)
$ProtocolCap = ""

# Watermark
$DefaultWatermark = -join ((1..4) | ForEach-Object { "{0:x2}" -f (Get-Random -Minimum 0 -Maximum 256) })
if ([string]::IsNullOrEmpty($Watermark)) {
    $Watermark = Read-Host "Watermark [$DefaultWatermark]"
    if ([string]::IsNullOrEmpty($Watermark)) { $Watermark = $DefaultWatermark }
}
if ($Watermark -notmatch '^[0-9a-fA-F]{8}$') {
    Write-Host "[-] Watermark must be exactly 8 hex characters (e.g. a1b2c3d4)." -ForegroundColor Red
    exit 1
}

# ─── Protocol selection ─────────────────────────────────────────────────────────

$AvailableProtocols = @()
if (Test-Path $ProtocolsDir) {
    Get-ChildItem -Path $ProtocolsDir -Directory | Where-Object { $_.Name -ne '_scaffold' -and (Test-Path (Join-Path $_.FullName 'meta.yaml')) } | ForEach-Object {
        $AvailableProtocols += $_.Name
    }
}
if ($AvailableProtocols.Count -eq 0) {
    Write-Host "[!] No protocols found in $ProtocolsDir. Using template defaults." -ForegroundColor Yellow
    $Protocol = ""
} elseif ($Protocol -eq 'none') {
    # Explicit 'none' sentinel: skip overlay
    $Protocol = ""
} elseif ([string]::IsNullOrEmpty($Protocol)) {
    Write-Host "Available protocols:" -ForegroundColor Cyan
    for ($i = 0; $i -lt $AvailableProtocols.Count; $i++) {
        $pn = $AvailableProtocols[$i]
        $desc = ""
        $metaPath = Join-Path $ProtocolsDir "$pn\meta.yaml"
        if (Test-Path $metaPath) {
            $metaContent = Get-Content $metaPath -Raw -Encoding UTF8
            if ($metaContent -match 'description:\s*"([^"]+)"') { $desc = " - $($Matches[1])" }
        }
        Write-Host "  [$($i+1)] $pn$desc"
    }
    Write-Host "  [0] None (use template defaults)" -ForegroundColor DarkGray
    Write-Host ""
    $choice = Read-Host "Select protocol [default: 1]"
    if ([string]::IsNullOrEmpty($choice)) { $choice = "1" }
    $idx = [int]$choice
    if ($idx -ge 1 -and $idx -le $AvailableProtocols.Count) {
        $Protocol = $AvailableProtocols[$idx - 1]
    } else {
        $Protocol = ""
    }
}

if (-not [string]::IsNullOrEmpty($Protocol)) {
    $ProtoDir = Join-Path $ProtocolsDir $Protocol
    if (-not (Test-Path $ProtoDir)) {
        Write-Host "[-] Protocol '$Protocol' not found in $ProtocolsDir" -ForegroundColor Red
        exit 1
    }

    $ProtocolCap = $Protocol.Substring(0,1).ToUpper() + $Protocol.Substring(1)
}

# ─── Listener binding defaults ─────────────────────────────────────────────────

$ConfiguredListeners = @()
if ($ListenerNames.Count -gt 0) {
    foreach ($listenerName in $ListenerNames) {
        if ([string]::IsNullOrWhiteSpace($listenerName)) { continue }
        $ConfiguredListeners += $listenerName.Trim()
    }
} elseif (-not [string]::IsNullOrEmpty($Protocol)) {
    # Auto-bind agent/listener generated with the same base name and protocol.
    $ConfiguredListeners += "${AgentNameCap}${ProtocolCap}"
} else {
    # Backward-compatible base scaffold default.
    $ConfiguredListeners += "GopherTCP"
}

if ($ConfiguredListeners.Count -eq 0) {
    $ConfiguredListeners += "GopherTCP"
}

$ListenersYaml = ($ConfiguredListeners | ForEach-Object { "  - `"$_`"" }) -join "`n"
$MultiListeners = if ($ConfiguredListeners.Count -gt 1) { "true" } else { "false" }

Write-Host ""
# ─── Language selection ──────────────────────────────────────────────────────

if ([string]::IsNullOrEmpty($Language)) {
    # Discover available languages from template directories
    $AvailableLangs = @()
    $LangDescs = @{ "go" = "Go implant"; "cpp" = "C/C++ implant"; "rust" = "Rust implant" }
    foreach ($lang in @("go", "cpp", "rust")) {
        if (Test-Path (Join-Path $TemplateDir "implant\$lang")) {
            $AvailableLangs += $lang
        }
    }

    if ($AvailableLangs.Count -eq 0) {
        Write-Host "[-] No implant template directories found." -ForegroundColor Red
        exit 1
    } elseif ($AvailableLangs.Count -eq 1) {
        $Language = $AvailableLangs[0]
    } else {
        Write-Host "Select implant language:" -ForegroundColor Cyan
        for ($i = 0; $i -lt $AvailableLangs.Count; $i++) {
            $l = $AvailableLangs[$i]
            $def = if ($i -eq 0) { " (default)" } else { "" }
            $desc = $LangDescs[$l]
            Write-Host "  [$($i+1)] $l$def" -ForegroundColor White -NoNewline
            Write-Host "  - $desc" -ForegroundColor DarkGray
        }
        Write-Host ""
        $choice = Read-Host "Select language [default: 1]"
        if ([string]::IsNullOrEmpty($choice)) { $choice = "1" }
        $idx = [int]$choice
        if ($idx -ge 1 -and $idx -le $AvailableLangs.Count) {
            $Language = $AvailableLangs[$idx - 1]
        } else {
            $Language = $AvailableLangs[0]
        }
    }
}

$ImplantLangDir = Join-Path $TemplateDir "implant\$Language"
if (-not (Test-Path $ImplantLangDir)) {
    Write-Host "[-] No implant templates for language '$Language' in $ImplantLangDir" -ForegroundColor Red
    exit 1
}

# ─── Toolchain selection ─────────────────────────────────────────────────────

$ToolchainsDir = Join-Path $ScriptDir "toolchains"

# Default toolchain per language
$DefaultToolchainName = switch ($Language) {
    "go"   { "go-standard" }
    "cpp"  { "mingw" }
    "rust" { "cargo" }
}

if ([string]::IsNullOrEmpty($Toolchain)) {
    # Scan available toolchains for the selected language
    $MatchingToolchains = @()
    if (Test-Path $ToolchainsDir) {
        Get-ChildItem -Path $ToolchainsDir -Filter "*.yaml" | ForEach-Object {
            $tcContent = Get-Content -Path $_.FullName -Raw -Encoding UTF8
            $tcLang = ""
            if ($tcContent -match 'language:\s*(\S+)') { $tcLang = $Matches[1] }
            if ($tcLang -eq $Language) {
                $tcDesc = ""
                if ($tcContent -match 'description:\s*"([^"]+)"') { $tcDesc = $Matches[1] }
                $MatchingToolchains += @{ Name = $_.BaseName; Desc = $tcDesc }
            }
        }
    }

    if ($MatchingToolchains.Count -eq 0) {
        $Toolchain = $DefaultToolchainName
    } elseif ($MatchingToolchains.Count -eq 1) {
        $Toolchain = $MatchingToolchains[0].Name
    } else {
        # Find default index
        $defaultIdx = 0
        for ($i = 0; $i -lt $MatchingToolchains.Count; $i++) {
            if ($MatchingToolchains[$i].Name -eq $DefaultToolchainName) { $defaultIdx = $i; break }
        }

        Write-Host "Available toolchains for '$Language':" -ForegroundColor Cyan
        for ($i = 0; $i -lt $MatchingToolchains.Count; $i++) {
            $tc = $MatchingToolchains[$i]
            $def = if ($tc.Name -eq $DefaultToolchainName) { " (default)" } else { "" }
            Write-Host "  [$($i+1)] $($tc.Name)$def" -ForegroundColor White -NoNewline
            Write-Host "  - $($tc.Desc)" -ForegroundColor DarkGray
        }
        Write-Host ""
        $choice = Read-Host "Select toolchain [default: $($defaultIdx + 1)]"
        if ([string]::IsNullOrEmpty($choice)) { $choice = "$($defaultIdx + 1)" }
        $idx = [int]$choice
        if ($idx -ge 1 -and $idx -le $MatchingToolchains.Count) {
            $Toolchain = $MatchingToolchains[$idx - 1].Name
        } else {
            $Toolchain = $DefaultToolchainName
        }
    }
}

$ToolchainFile = Join-Path $ToolchainsDir "$Toolchain.yaml"
if (-not (Test-Path $ToolchainFile)) {
    Write-Host "[!] Toolchain file '$ToolchainFile' not found. Continuing without toolchain overlay." -ForegroundColor Yellow
    $ToolchainFile = $null
}

# ─── Evasion gate prompt ─────────────────────────────────────────────────────

$EnableEvasion = $false
if ($Evasion) {
    $EnableEvasion = $true
} elseif ([string]::IsNullOrEmpty($Name)) {
    # Interactive mode — ask the user
    Write-Host "Include evasion gate scaffold? (syscall/stack-spoof abstraction)" -ForegroundColor Cyan
    $choice = Read-Host "Enable evasion [y/N]"
    if ($choice -match '^[Yy]') {
        $EnableEvasion = $true
    }
}

Write-Host "[*] Creating agent: $AgentName" -ForegroundColor Cyan
Write-Host "      Language  : $Language" -ForegroundColor Cyan
Write-Host "      Toolchain : $Toolchain" -ForegroundColor Cyan
Write-Host "      Watermark : $Watermark" -ForegroundColor Cyan
if ($EnableEvasion) {
    Write-Host "      Evasion   : enabled" -ForegroundColor Cyan
}
if (-not [string]::IsNullOrEmpty($Protocol)) {
    Write-Host "      Protocol  : $Protocol" -ForegroundColor Cyan
}
Write-Host "      Directory : $OutDir\" -ForegroundColor Cyan
Write-Host ""

# ─── Create directory structure ─────────────────────────────────────────────────

$SrcDir = Join-Path $OutDir "src_$AgentName"

New-Item -ItemType Directory -Path $OutDir | Out-Null
New-Item -ItemType Directory -Path $SrcDir | Out-Null
if (Test-Path (Join-Path $ImplantLangDir "impl")) {
    New-Item -ItemType Directory -Path (Join-Path $SrcDir "impl") | Out-Null
}
if (Test-Path (Join-Path $ImplantLangDir "crypto")) {
    New-Item -ItemType Directory -Path (Join-Path $SrcDir "crypto") | Out-Null
}
if (Test-Path (Join-Path $ImplantLangDir "protocol")) {
    New-Item -ItemType Directory -Path (Join-Path $SrcDir "protocol") | Out-Null
}
if ($EnableEvasion -and (Test-Path (Join-Path $ImplantLangDir "evasion"))) {
    New-Item -ItemType Directory -Path (Join-Path $SrcDir "evasion") | Out-Null
}

# ─── Parse toolchain ────────────────────────────────────────────────────────────

$BuildTool = "go build"
if ($null -ne $ToolchainFile) {
    $tcContent = Get-Content -Path $ToolchainFile -Raw -Encoding UTF8
    if ($tcContent -match 'command:\s*"([^"]+)"') {
        $BuildTool = $Matches[1]
    } elseif ($tcContent -match "command:\s*'([^']+)'") {
        $BuildTool = $Matches[1]
    } elseif ($tcContent -match 'command:\s*(\S+.*)') {
        $BuildTool = $Matches[1].Trim()
    }
}

# ─── Copy and substitute templates ─────────────────────────────────────────────

function Substitute-Template {
    param(
        [string]$Source,
        [string]$Destination
    )
    $content = Get-Content -Path $Source -Raw -Encoding UTF8
    $content = $content -replace '__NAME_CAP__', $AgentNameCap
    $content = $content -replace '__NAME__', $AgentName
    $content = $content -replace '__WATERMARK__', $Watermark
    $content = $content -replace '__BUILD_TOOL__', $BuildTool
    $content = $content.Replace('__LISTENER_NAMES__', $ListenersYaml)
    $content = $content -replace '__MULTI_LISTENERS__', $MultiListeners
    # Write with UTF-8 no BOM
    [System.IO.File]::WriteAllText($Destination, $content, [System.Text.UTF8Encoding]::new($false))
}

# Plugin files
Write-Host "[*] Generating plugin files..." -ForegroundColor Cyan
Substitute-Template (Join-Path $TemplateDir "plugin\config.yaml")   (Join-Path $OutDir "config.yaml")
Substitute-Template (Join-Path $TemplateDir "plugin\go.mod")        (Join-Path $OutDir "go.mod")
Substitute-Template (Join-Path $TemplateDir "plugin\go.sum")        (Join-Path $OutDir "go.sum")
Substitute-Template (Join-Path $TemplateDir "plugin\Makefile")      (Join-Path $OutDir "Makefile")
Substitute-Template (Join-Path $TemplateDir "plugin\pl_utils.go")   (Join-Path $OutDir "pl_utils.go")
# pl_main.go — protocol-specific override if present
$protoMain = if ($Protocol) { Join-Path $ProtoDir "pl_main.go.tmpl" } else { $null }
if ($protoMain -and (Test-Path $protoMain)) {
    Write-Host "  -> Using protocol-specific pl_main.go from '$Protocol'" -ForegroundColor Yellow
    Substitute-Template $protoMain (Join-Path $OutDir "pl_main.go")
} else {
    Substitute-Template (Join-Path $TemplateDir "plugin\pl_main.go") (Join-Path $OutDir "pl_main.go")
}

# ax_config.axs — language-specific UI definition
$AxConfigVariant = "ax_config.axs"
if ($Language -ne 'go') {
    $langAxs = "ax_config_$Language.axs"
    if (Test-Path (Join-Path $TemplateDir "plugin\$langAxs")) {
        $AxConfigVariant = $langAxs
    }
}
Substitute-Template (Join-Path $TemplateDir "plugin\$AxConfigVariant") (Join-Path $OutDir "ax_config.axs")

# Plugin build variant (language-specific: pl_build_go.go or pl_build_cpp.go)
$BuildVariantMap = @{ "go" = "pl_build_go.go"; "cpp" = "pl_build_cpp.go"; "rust" = "pl_build_rust.go" }
$BuildVariant = $BuildVariantMap[$Language]
if (-not [string]::IsNullOrEmpty($BuildVariant)) {
    $protoBuild = if ($Protocol) { Join-Path $ProtoDir ("{0}.tmpl" -f $BuildVariant) } else { $null }
    $srcBuild = if ($protoBuild -and (Test-Path $protoBuild)) { $protoBuild } else { Join-Path $TemplateDir "plugin\$BuildVariant" }
    if (Test-Path $srcBuild) {
        if ($protoBuild -and ($srcBuild -eq $protoBuild)) {
            Write-Host "  -> Using protocol-specific $BuildVariant from '$Protocol'" -ForegroundColor Yellow
        }
        Substitute-Template $srcBuild (Join-Path $OutDir "pl_build.go")
    }
}

# sRDI helper (Rust only — DLL→shellcode converter)
if ($Language -eq 'rust') {
    $srdiSrc = Join-Path $TemplateDir "plugin\srdi.go"
    if (Test-Path $srdiSrc) {
        Substitute-Template $srdiSrc (Join-Path $OutDir "srdi.go")
    }
}

# Implant files
Write-Host "[*] Generating implant files ($Language)..." -ForegroundColor Cyan
foreach ($f in (Get-ChildItem -Path $ImplantLangDir -File)) {
    Substitute-Template $f.FullName (Join-Path $SrcDir $f.Name)
}

# Crypto — from protocol .go.tmpl if Go and available, otherwise from language template
if ($Language -eq 'go' -and -not [string]::IsNullOrEmpty($Protocol)) {
    $cryptoTmpl = Join-Path $ProtoDir "crypto.go.tmpl"
    if (Test-Path $cryptoTmpl) {
        Write-Host "[*] Applying protocol '$Protocol' crypto..." -ForegroundColor Cyan
        $content = Get-Content -Path $cryptoTmpl -Raw -Encoding UTF8
        $content = $content -replace '__PACKAGE__', 'crypto'
        [System.IO.File]::WriteAllText((Join-Path $SrcDir "crypto\crypto.go"), $content, [System.Text.UTF8Encoding]::new($false))
    } else {
        foreach ($f in (Get-ChildItem -Path (Join-Path $ImplantLangDir "crypto") -File)) {
            Substitute-Template $f.FullName (Join-Path $SrcDir "crypto\$($f.Name)")
        }
    }
} else {
    # Non-Go or no protocol: copy all files from language's crypto/ dir
    $cryptoSrcDir = Join-Path $ImplantLangDir "crypto"
    if (Test-Path $cryptoSrcDir) {
        foreach ($f in (Get-ChildItem -Path $cryptoSrcDir -File)) {
            Substitute-Template $f.FullName (Join-Path $SrcDir "crypto\$($f.Name)")
        }
    }
}

# Protocol types — from protocol .go.tmpl if Go and available, otherwise from language template
if ($Language -eq 'go' -and -not [string]::IsNullOrEmpty($Protocol)) {
    $typesTmpl = Join-Path $ProtoDir "types.go.tmpl"
    $constTmpl = Join-Path $ProtoDir "constants.go.tmpl"
    if ((Test-Path $typesTmpl) -and (Test-Path $constTmpl)) {
        Write-Host "[*] Applying protocol '$Protocol' types + constants..." -ForegroundColor Cyan
        $typesContent = Get-Content -Path $typesTmpl -Raw -Encoding UTF8
        $constContent = Get-Content -Path $constTmpl -Raw -Encoding UTF8
        $constLines = $constContent -split "`n"
        $constFiltered = ($constLines | Where-Object { $_ -notmatch '^\s*package\s+' }) -join "`n"
        $merged = $typesContent + "`n`n" + $constFiltered
        $merged = $merged -replace '__PACKAGE__', 'protocol'
        [System.IO.File]::WriteAllText((Join-Path $SrcDir "protocol\protocol.go"), $merged, [System.Text.UTF8Encoding]::new($false))
        # Also copy base protocol files that are NOT protocol.go (e.g. agent_types.go)
        foreach ($f in (Get-ChildItem -Path (Join-Path $ImplantLangDir "protocol") -File)) {
            if ($f.Name -eq 'protocol.go') { continue }
            Substitute-Template $f.FullName (Join-Path $SrcDir "protocol\$($f.Name)")
        }
    } else {
        foreach ($f in (Get-ChildItem -Path (Join-Path $ImplantLangDir "protocol") -File)) {
            Substitute-Template $f.FullName (Join-Path $SrcDir "protocol\$($f.Name)")
        }
    }
} else {
    # Non-Go or no protocol: copy all files from language's protocol/ dir
    $protoSrcDir = Join-Path $ImplantLangDir "protocol"
    if (Test-Path $protoSrcDir) {
        foreach ($f in (Get-ChildItem -Path $protoSrcDir -File)) {
            Substitute-Template $f.FullName (Join-Path $SrcDir "protocol\$($f.Name)")
        }
    }
}

# Plugin pl_utils.go — overlay with protocol constants+types if available
if (-not [string]::IsNullOrEmpty($Protocol)) {
    $typesTmpl = Join-Path $ProtoDir "types.go.tmpl"
    $constTmpl = Join-Path $ProtoDir "constants.go.tmpl"
    if ((Test-Path $typesTmpl) -and (Test-Path $constTmpl)) {
        Write-Host "[*] Applying protocol '$Protocol' to pl_utils.go..." -ForegroundColor Cyan
        $typesContent = Get-Content -Path $typesTmpl -Raw -Encoding UTF8
        $constContent = Get-Content -Path $constTmpl -Raw -Encoding UTF8
        $constLines = $constContent -split "`n"
        $constFiltered = ($constLines | Where-Object { $_ -notmatch '^\s*package\s+' }) -join "`n"
        $merged = $typesContent + "`n`n" + $constFiltered
        $merged = $merged -replace '__PACKAGE__', 'main'
        [System.IO.File]::WriteAllText((Join-Path $OutDir "pl_utils.go"), $merged, [System.Text.UTF8Encoding]::new($false))
    }
}

# Impl stubs — copy all subdirectories recursively
Write-Host "[*] Generating interface stubs..." -ForegroundColor Cyan
foreach ($subDir in (Get-ChildItem -Path $ImplantLangDir -Directory)) {
    # Skip crypto/, protocol/, and evasion/ (handled separately)
    if ($subDir.Name -eq 'crypto' -or $subDir.Name -eq 'protocol' -or $subDir.Name -eq 'evasion') { continue }
    $destSubDir = Join-Path $SrcDir $subDir.Name
    if (-not (Test-Path $destSubDir)) {
        New-Item -ItemType Directory -Path $destSubDir -Force | Out-Null
    }
    foreach ($f in (Get-ChildItem -Path $subDir.FullName -File -Recurse)) {
        $relPath = $f.FullName.Substring($subDir.FullName.Length + 1)
        # Skip evasion/ subdirectory files unless evasion is enabled
        if ($relPath -match '(^|[\\/])evasion[\\/]' -and -not $EnableEvasion) { continue }
        $destFile = Join-Path $destSubDir $relPath
        $destFileDir = Split-Path -Parent $destFile
        if (-not (Test-Path $destFileDir)) {
            New-Item -ItemType Directory -Path $destFileDir -Force | Out-Null
        }
        Substitute-Template $f.FullName $destFile
    }
}

# ─── Protocol-specific implant overrides ─────────────────────────────────────────
# If the protocol provides implant/ overrides (e.g. tasks.go.tmpl, protocol/agent_types.go.tmpl),
# apply them on top of the base templates. This lets each protocol override any
# implant source file while keeping the base as a fallback.

if (-not [string]::IsNullOrEmpty($Protocol)) {
    # Language-aware implant override directory.
    # Go:        use implant/ root (backward compat), skip cpp/ and rust/ subdirs.
    # C++/Rust:  use implant/<language>/ if it exists.
    if ($Language -eq 'go') {
        $implantOverrides = Join-Path $ProtoDir "implant"
    } else {
        $implantOverrides = Join-Path $ProtoDir "implant\$Language"
    }

    if (Test-Path $implantOverrides) {
        Write-Host "[*] Applying protocol '$Protocol' implant overrides ($Language)..." -ForegroundColor Cyan
        foreach ($f in (Get-ChildItem -Path $implantOverrides -File -Recurse -Filter "*.tmpl")) {
            $relToOverrides = $f.FullName.Substring($implantOverrides.Length + 1)
            # For Go, skip files under cpp/ and rust/ subdirectories
            if ($Language -eq 'go' -and $relToOverrides -match '^(cpp|rust)[\\/]') { continue }
            # Skip evasion/ overlay files unless evasion is enabled
            if ($relToOverrides -match '(^|[\\/])evasion[\\/]' -and -not $EnableEvasion) { continue }
            $relPath = $relToOverrides -replace '\.tmpl$', ''
            $targetPath = Join-Path $SrcDir $relPath
            $tgtDir = Split-Path -Parent $targetPath
            if (-not (Test-Path $tgtDir)) {
                New-Item -ItemType Directory -Path $tgtDir | Out-Null
            }
            Write-Host "  -> $relPath" -ForegroundColor Yellow
            Substitute-Template $f.FullName $targetPath
        }
    }
}

# ─── Evasion gate scaffold ──────────────────────────────────────────────────────

if ($EnableEvasion) {
    $evasionSrcDir = Join-Path $ImplantLangDir "evasion"
    if (Test-Path $evasionSrcDir) {
        Write-Host "[*] Generating evasion gate scaffold..." -ForegroundColor Cyan
        $evasionDestDir = Join-Path $SrcDir "evasion"
        if (-not (Test-Path $evasionDestDir)) {
            New-Item -ItemType Directory -Path $evasionDestDir -Force | Out-Null
        }
        foreach ($f in (Get-ChildItem -Path $evasionSrcDir -File -Recurse)) {
            $relPath = $f.FullName.Substring($evasionSrcDir.Length + 1)
            $destFile = Join-Path $evasionDestDir $relPath
            $destFileDir = Split-Path -Parent $destFile
            if (-not (Test-Path $destFileDir)) {
                New-Item -ItemType Directory -Path $destFileDir -Force | Out-Null
            }
            Substitute-Template $f.FullName $destFile
        }
    }
}

# ─── Evasion marker post-processing ─────────────────────────────────────────────

# Process all generated source files to replace or strip evasion markers.
# When evasion is enabled → inject real code. When disabled → remove markers.

function Process-EvasionMarkers {
    param([string]$Dir)

    foreach ($f in (Get-ChildItem -Path $Dir -File -Recurse -Include *.go, *.h, *.cpp, *.rs, *.toml, *Makefile, go.mod)) {
        $content = Get-Content -Path $f.FullName -Raw -Encoding UTF8
        $modified = $false

        if ($EnableEvasion) {
            switch ($Language) {
                "go" {
                    if ($content -match '// __EVASION_IMPORT__') {
                        $content = $content -replace '(\s*)// __EVASION_IMPORT__', "`$1`"__NAME__/evasion`""
                        $modified = $true
                    }
                    if ($content -match '// __EVASION_MAIN_IMPORT__') {
                        $content = $content -replace '(\s*)// __EVASION_MAIN_IMPORT__', "`$1`"__NAME__/evasion`""
                        $modified = $true
                    }
                    if ($content -match '// __EVASION_FIELD__') {
                        $content = $content -replace '(\s*)// __EVASION_FIELD__', "`$1Gate evasion.Gate"
                        $modified = $true
                    }
                    if ($content -match '// __EVASION_INIT__') {
                        $initBlock = @"
	// Initialize evasion gate (syscall/stack-spoof abstraction).
	gate := evasion.Default()
	if err := gate.Init(); err != nil {
		os.Exit(1)
	}
	_ = gate // TODO: pass gate to agent or store globally
"@
                        $content = $content -replace '\s*// __EVASION_INIT__', "`n$initBlock"
                        $modified = $true
                    }
                    if ($content -match '// __EVASION_GOMOD__') {
                        $gomodBlock = @"

// Uncomment and adjust the path below to import your evasion module:
// require evasion v0.0.0
// add a local replace directive that points to your evasion module
"@
                        $content = $content -replace '\s*// __EVASION_GOMOD__', $gomodBlock
                        $modified = $true
                    }
                }
                "cpp" {
                    if ($content -match '// __EVASION_FORWARD_DECL__') {
                        $content = $content -replace '// __EVASION_FORWARD_DECL__', "class IEvasionGate;"
                        $modified = $true
                    }
                    if ($content -match '// __EVASION_MEMBER__') {
                        $content = $content -replace '(\s*)// __EVASION_MEMBER__', "`$1IEvasionGate* gate = nullptr;"
                        $modified = $true
                    }
                    if ($content -match '// __EVASION_INCLUDE__') {
                        $content = $content -replace '// __EVASION_INCLUDE__', "#include `"../evasion/DefaultGate.h`""
                        $modified = $true
                    }
                    if ($content -match '// __EVASION_CTOR__') {
                        $ctorBlock = @"
    // Initialize evasion gate (syscall/stack-spoof abstraction)
    gate = new DefaultGate();
    gate->Init();
"@
                        $content = $content -replace '\s*// __EVASION_CTOR__', "`n$ctorBlock"
                        $modified = $true
                    }
                    if ($content -match '# __EVASION_SOURCES__') {
                        $content = $content -replace '# __EVASION_SOURCES__', "SOURCES += `$(wildcard evasion/*.cpp)"
                        $modified = $true
                    }
                }
                "rust" {
                    if ($content -match '// __EVASION_MOD__') {
                        $content = $content -replace '// __EVASION_MOD__', "mod evasion;"
                        $modified = $true
                    }
                    if ($content -match '# __EVASION_FEATURES__') {
                        $content = $content -replace '# __EVASION_FEATURES__', 'evasion = []'
                        $modified = $true
                    }
                }
            }
        } else {
            # Strip all evasion markers
            $content = $content -replace '(?m)^\s*// __EVASION_[A-Z_]+__\s*\r?\n', ''
            $content = $content -replace '(?m)^\s*# __EVASION_[A-Z_]+__\s*\r?\n', ''
            $modified = $true
        }

        if ($modified) {
            # Re-apply name substitution (for injected code that may contain __NAME__)
            $content = $content -replace '__NAME_CAP__', $AgentNameCap
            $content = $content -replace '__NAME__', $AgentName
            [System.IO.File]::WriteAllText($f.FullName, $content, [System.Text.UTF8Encoding]::new($false))
        }
    }
}

Process-EvasionMarkers $SrcDir

# ─── Summary ────────────────────────────────────────────────────────────────────

Write-Host ""
if (-not [string]::IsNullOrEmpty($Protocol)) {
    Write-Host "[+] Agent '$AgentName' scaffolded with protocol '$Protocol' ($Language)!" -ForegroundColor Green
} else {
    Write-Host "[+] Agent '$AgentName' scaffolded successfully ($Language)!" -ForegroundColor Green
}
if ($EnableEvasion) {
    Write-Host "    Evasion gate scaffold included." -ForegroundColor Green
}
Write-Host ""
Write-Host "Directory structure:" -ForegroundColor Cyan
Write-Host ""
Write-Host "  $AgentDirName\"
Write-Host "  |-- config.yaml          # Plugin manifest"
Write-Host "  |-- go.mod               # Plugin module"
Write-Host "  |-- Makefile             # Build targets"
Write-Host "  |-- pl_utils.go          # Wire types & constants"
Write-Host "  |-- pl_main.go           # Plugin logic (server-side)"
Write-Host "  |-- pl_build.go          # Build logic ($Language)"
Write-Host "  |-- ax_config.axs        # UI & command definitions"
Write-Host "  \-- src_${AgentName}\"
# List actual generated files dynamically
Get-ChildItem -Path $SrcDir -Recurse -File | ForEach-Object {
    $rel = $_.FullName.Substring($SrcDir.Length + 1)
    Write-Host "      $rel"
}
Write-Host ""
Write-Host "Language  : $Language" -ForegroundColor Cyan
Write-Host "Toolchain : $Toolchain" -ForegroundColor Cyan
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "  1. cd $OutDir"
Write-Host "  2. Implement the TODO stubs in src_${AgentName}\"
Write-Host "  3. Build: make full"
Write-Host ""
