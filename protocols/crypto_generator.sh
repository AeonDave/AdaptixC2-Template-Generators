#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# AdaptixC2 Crypto Generator
#
# Generate or replace the crypto module for an existing protocol.
#
# Usage:
#   cd extenders/templates/protocols
#   ./crypto_generator.sh
#
#   PROTOCOL=myproto CRYPTO=xchacha20 ./crypto_generator.sh
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROTOCOL="${PROTOCOL:-}"
CRYPTO="${CRYPTO:-}"

echo ""
echo "╔═══════════════════════════════════════════════╗"
echo "║   AdaptixC2 Crypto Generator                  ║"
echo "╚═══════════════════════════════════════════════╝"
echo ""

# ─── Discover protocols ─────────────────────────────────────────────────────────

available=()
for d in "$SCRIPT_DIR"/*/; do
    name="$(basename "$d")"
    [[ "$name" == "_scaffold" ]] && continue
    [[ -f "$d/meta.yaml" ]] || continue
    available+=("$name")
done

if [[ ${#available[@]} -eq 0 ]]; then
    echo "[-] No protocols found. Create one first with protocols/generator.sh"
    exit 1
fi

# ─── Select protocol ────────────────────────────────────────────────────────────

if [[ -z "$PROTOCOL" ]]; then
    echo "Available protocols:"
    for i in "${!available[@]}"; do
        echo "  [$((i+1))] ${available[$i]}"
    done
    echo ""
    read -rp "Select protocol: " choice
    idx=$((choice - 1))
    if [[ $idx -lt 0 || $idx -ge ${#available[@]} ]]; then
        echo "[-] Invalid choice."; exit 1
    fi
    PROTOCOL="${available[$idx]}"
fi

PROTO_DIR="$SCRIPT_DIR/$PROTOCOL"
if [[ ! -d "$PROTO_DIR" ]]; then
    echo "[-] Protocol '$PROTOCOL' not found."; exit 1
fi

# ─── Select crypto ──────────────────────────────────────────────────────────────

crypto_keys=("aes-gcm" "xchacha20")
crypto_descs=("AES-256-GCM (standard, fast, widely supported)" "XChaCha20-Poly1305 (modern, nonce-misuse resistant)")

if [[ -z "$CRYPTO" ]]; then
    echo "Available crypto implementations:"
    for i in "${!crypto_keys[@]}"; do
        echo "  [$((i+1))] ${crypto_keys[$i]} - ${crypto_descs[$i]}"
    done
    echo ""
    read -rp "Select crypto [default: 1]: " choice
    [[ -z "$choice" ]] && choice="1"
    idx=$((choice - 1))
    if [[ $idx -lt 0 || $idx -ge ${#crypto_keys[@]} ]]; then
        echo "[-] Invalid choice."; exit 1
    fi
    CRYPTO="${crypto_keys[$idx]}"
fi

# ─── Generate crypto template ───────────────────────────────────────────────────

DEST_FILE="$PROTO_DIR/crypto.go.tmpl"

case "$CRYPTO" in
aes-gcm)
cat > "$DEST_FILE" << 'TMPL'
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
TMPL
;;
xchacha20)
cat > "$DEST_FILE" << 'TMPL'
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
TMPL
;;
*)
    echo "[-] Unknown crypto: $CRYPTO"; exit 1
;;
esac

# Update meta.yaml crypto field
META_FILE="$PROTO_DIR/meta.yaml"
if [[ -f "$META_FILE" ]]; then
    sed -i "s/crypto: \"[^\"]*\"/crypto: \"$CRYPTO\"/" "$META_FILE"
fi

echo ""
echo "[+] Crypto '$CRYPTO' applied to protocol '$PROTOCOL'"
echo "    Updated: crypto.go.tmpl, meta.yaml"
if [[ "$CRYPTO" == "xchacha20" ]]; then
    echo ""
    echo "[!] Remember: add 'golang.org/x/crypto' to go.mod in generated projects."
fi
echo ""
