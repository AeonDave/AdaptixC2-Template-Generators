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

# ─── Select crypto ──────────────────────────────────────────────────────────────

$cryptoOptions = @(
    @{ Key = "aes-gcm";    Desc = "AES-256-GCM (standard, fast, widely supported)" },
    @{ Key = "xchacha20";  Desc = "XChaCha20-Poly1305 (modern, nonce-misuse resistant)" }
)

if ([string]::IsNullOrEmpty($Crypto)) {
    Write-Host "Available crypto implementations:" -ForegroundColor Cyan
    for ($i = 0; $i -lt $cryptoOptions.Count; $i++) {
        Write-Host "  [$($i+1)] $($cryptoOptions[$i].Key) - $($cryptoOptions[$i].Desc)"
    }
    Write-Host ""
    $choice = Read-Host "Select crypto [default: 1]"
    if ([string]::IsNullOrEmpty($choice)) { $choice = "1" }
    $idx = [int]$choice
    if ($idx -lt 1 -or $idx -gt $cryptoOptions.Count) {
        Write-Host "[-] Invalid choice." -ForegroundColor Red; exit 1
    }
    $Crypto = $cryptoOptions[$idx - 1].Key
}

# ─── Generate crypto template ───────────────────────────────────────────────────

$destFile = Join-Path $ProtoDir "crypto.go.tmpl"

switch ($Crypto) {
    "aes-gcm" {
        $content = @'
package __PACKAGE__

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

// SKey is set from the embedded profile at startup (agent-side).
var SKey []byte

// EncryptData encrypts data with AES-256-GCM using key.
// The nonce is prepended to the ciphertext.
func EncryptData(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, data, nil), nil
}

// DecryptData decrypts data with AES-256-GCM using key.
// Expects nonce prepended as per EncryptData.
func DecryptData(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(data) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ct := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ct, nil)
}
'@
    }
    "xchacha20" {
        $content = @'
package __PACKAGE__

import (
	"crypto/rand"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
)

// SKey is set from the embedded profile at startup (agent-side).
var SKey []byte

// EncryptData encrypts data with XChaCha20-Poly1305 using key.
// The 24-byte nonce is prepended to the ciphertext.
func EncryptData(data, key []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return aead.Seal(nonce, nonce, data, nil), nil
}

// DecryptData decrypts data with XChaCha20-Poly1305 using key.
// Expects 24-byte nonce prepended as per EncryptData.
func DecryptData(data, key []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	if len(data) < aead.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ct := data[:aead.NonceSize()], data[aead.NonceSize():]
	return aead.Open(nil, nonce, ct, nil)
}
'@
    }
    default {
        Write-Host "[-] Unknown crypto: $Crypto" -ForegroundColor Red
        exit 1
    }
}

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
