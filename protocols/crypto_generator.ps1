<#
.SYNOPSIS
    Generate or replace the crypto module for an existing protocol.

.DESCRIPTION
    Lists available crypto implementations and injects the chosen one into
    a protocol's crypto.go.tmpl file.

.PARAMETER Protocol
    Target protocol name (must exist in protocols/).

.PARAMETER Crypto
    Crypto implementation: "aes-gcm" (default) or "xchacha20".

.EXAMPLE
    cd extenders\templates\protocols
    .\crypto_generator.ps1

.EXAMPLE
    .\crypto_generator.ps1 -Protocol myproto -Crypto xchacha20
#>
param(
    [string]$Protocol = "",
    [string]$Crypto   = ""
)

$ErrorActionPreference = "Stop"

$ScriptDir    = Split-Path -Parent $MyInvocation.MyCommand.Path

# ─── Banner ─────────────────────────────────────────────────────────────────────

Write-Host ""
Write-Host "╔═══════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║   AdaptixC2 Crypto Generator                  ║" -ForegroundColor Cyan
Write-Host "╚═══════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""

# ─── Discover protocols ─────────────────────────────────────────────────────────

$availableProtocols = @()
Get-ChildItem -Path $ScriptDir -Directory | Where-Object { $_.Name -ne '_scaffold' -and (Test-Path (Join-Path $_.FullName 'meta.yaml')) } | ForEach-Object {
    $availableProtocols += $_.Name
}
if ($availableProtocols.Count -eq 0) {
    Write-Host "[-] No protocols found. Create one first with protocols\generator.ps1" -ForegroundColor Red
    exit 1
}

# ─── Select protocol ────────────────────────────────────────────────────────────

if ([string]::IsNullOrEmpty($Protocol)) {
    Write-Host "Available protocols:" -ForegroundColor Cyan
    for ($i = 0; $i -lt $availableProtocols.Count; $i++) {
        Write-Host "  [$($i+1)] $($availableProtocols[$i])"
    }
    Write-Host ""
    $choice = Read-Host "Select protocol"
    $idx = [int]$choice
    if ($idx -lt 1 -or $idx -gt $availableProtocols.Count) {
        Write-Host "[-] Invalid choice." -ForegroundColor Red; exit 1
    }
    $Protocol = $availableProtocols[$idx - 1]
}
$ProtoDir = Join-Path $ScriptDir $Protocol
if (-not (Test-Path $ProtoDir)) {
    Write-Host "[-] Protocol '$Protocol' not found." -ForegroundColor Red; exit 1
}

# ─── Discover crypto templates ──────────────────────────────────────────────────

$cryptoDir = Join-Path $ScriptDir "_crypto"
$cryptoOptions = @()
if (Test-Path $cryptoDir) {
    Get-ChildItem -Path $cryptoDir -Filter "*.go.tmpl" | Sort-Object Name | ForEach-Object {
        $key  = $_.BaseName -replace '\.go$', ''
        $desc = ""
        $firstLine = (Get-Content -Path $_.FullName -TotalCount 1 -Encoding UTF8)
        if ($firstLine -match '^\s*//\s*(.+)$') { $desc = $Matches[1].Trim() }
        $cryptoOptions += @{ Key = $key; Desc = $desc; File = $_.FullName }
    }
}
if ($cryptoOptions.Count -eq 0) {
    Write-Host "[-] No crypto templates found in _crypto/. Add .go.tmpl files there." -ForegroundColor Red
    exit 1
}

# ─── Select crypto ──────────────────────────────────────────────────────────────

if ([string]::IsNullOrEmpty($Crypto)) {
    Write-Host "Available crypto implementations:" -ForegroundColor Cyan
    for ($i = 0; $i -lt $cryptoOptions.Count; $i++) {
        $line = "  [$($i+1)] $($cryptoOptions[$i].Key)"
        if ($cryptoOptions[$i].Desc) { $line += " - $($cryptoOptions[$i].Desc)" }
        Write-Host $line
    }
    $createIdx = $cryptoOptions.Count + 1
    Write-Host "  [$createIdx] Create new..." -ForegroundColor Yellow
    Write-Host ""
    $choice = Read-Host "Select crypto [default: 1]"
    if ([string]::IsNullOrEmpty($choice)) { $choice = "1" }
    $idx = [int]$choice
    if ($idx -eq $createIdx) {
        # ── Create new crypto scaffold ──
        $newName = Read-Host "Enter new crypto name (lowercase, e.g. my-cipher)"
        $newName = $newName.Trim().ToLower() -replace '[^a-z0-9_-]', ''
        if ([string]::IsNullOrEmpty($newName)) { Write-Host "[-] Invalid name." -ForegroundColor Red; exit 1 }
        $newFile = Join-Path $cryptoDir "$newName.go.tmpl"
        if (Test-Path $newFile) { Write-Host "[-] Crypto '$newName' already exists." -ForegroundColor Red; exit 1 }
        $newDesc = Read-Host "Short description (shown in menu)"
        $scaffold = "// $newDesc`npackage __PACKAGE__`n`nvar SKey []byte`n`n// EncryptData encrypts data with $newName using key.`n// TODO: implement`nfunc EncryptData(data, key []byte) ([]byte, error) {`n`tpanic(`"$newName EncryptData not implemented`")`n}`n`n// DecryptData decrypts data with $newName using key.`n// TODO: implement`nfunc DecryptData(data, key []byte) ([]byte, error) {`n`tpanic(`"$newName DecryptData not implemented`")`n}"
        [System.IO.File]::WriteAllText($newFile, $scaffold, [System.Text.UTF8Encoding]::new($false))
        Write-Host ""
        Write-Host "[+] Created crypto scaffold: _crypto/$newName.go.tmpl" -ForegroundColor Green
        Write-Host "    Implement EncryptData/DecryptData, then re-run this generator to apply it." -ForegroundColor Cyan
        Write-Host ""
        exit 0
    }
    if ($idx -lt 1 -or $idx -gt $cryptoOptions.Count) {
        Write-Host "[-] Invalid choice." -ForegroundColor Red; exit 1
    }
    $Crypto = $cryptoOptions[$idx - 1].Key
}

# ─── Resolve crypto template file ───────────────────────────────────────────────

$selectedOption = $cryptoOptions | Where-Object { $_.Key -eq $Crypto } | Select-Object -First 1
if (-not $selectedOption) {
    Write-Host "[-] Unknown crypto: $Crypto. Available: $($cryptoOptions | ForEach-Object { $_.Key }) " -ForegroundColor Red
    exit 1
}

# ─── Generate crypto template ───────────────────────────────────────────────────

$destFile = Join-Path $ProtoDir "crypto.go.tmpl"
$content  = Get-Content -Path $selectedOption.File -Raw -Encoding UTF8
[System.IO.File]::WriteAllText($destFile, $content, [System.Text.UTF8Encoding]::new($false))

# Update meta.yaml crypto field
$metaFile = Join-Path $ProtoDir "meta.yaml"
if (Test-Path $metaFile) {
    $metaContent = Get-Content -Path $metaFile -Raw -Encoding UTF8
    $metaContent = $metaContent -replace 'crypto:\s*"[^"]*"', "crypto: `"$Crypto`""
    [System.IO.File]::WriteAllText($metaFile, $metaContent, [System.Text.UTF8Encoding]::new($false))
}

Write-Host ""
Write-Host "[+] Crypto '$Crypto' applied to protocol '$Protocol'" -ForegroundColor Green
Write-Host "    Updated: crypto.go.tmpl, meta.yaml" -ForegroundColor Cyan
if ($Crypto -eq "xchacha20") {
    Write-Host ""
    Write-Host "[!] Remember: add 'golang.org/x/crypto' to go.mod in generated projects." -ForegroundColor Yellow
}
Write-Host ""
