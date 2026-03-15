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

# ─── Discover crypto templates ──────────────────────────────────────────────────

CRYPTO_DIR="$SCRIPT_DIR/_crypto"
crypto_keys=()
crypto_descs=()
crypto_files=()

if [[ -d "$CRYPTO_DIR" ]]; then
    for f in "$CRYPTO_DIR"/*.go.tmpl; do
        [[ -f "$f" ]] || continue
        key="$(basename "$f" .go.tmpl)"
        desc=""
        first_line="$(head -n1 "$f")"
        if [[ "$first_line" =~ ^[[:space:]]*//(.*) ]]; then
            desc="$(echo "${BASH_REMATCH[1]}" | sed 's/^[[:space:]]*//')"
        fi
        crypto_keys+=("$key")
        crypto_descs+=("$desc")
        crypto_files+=("$f")
    done
fi

if [[ ${#crypto_keys[@]} -eq 0 ]]; then
    echo "[-] No crypto templates found in _crypto/. Add .go.tmpl files there."
    exit 1
fi

# ─── Select crypto ──────────────────────────────────────────────────────────────

if [[ -z "$CRYPTO" ]]; then
    echo "Available crypto implementations:"
    for i in "${!crypto_keys[@]}"; do
        line="  [$((i+1))] ${crypto_keys[$i]}"
        [[ -n "${crypto_descs[$i]}" ]] && line+=" - ${crypto_descs[$i]}"
        echo "$line"
    done
    create_idx=$(( ${#crypto_keys[@]} + 1 ))
    echo "  [$create_idx] Create new..."
    echo ""
    read -rp "Select crypto [default: 1]: " choice
    [[ -z "$choice" ]] && choice="1"
    idx=$((choice - 1))

    if [[ $choice -eq $create_idx ]]; then
        # ── Create new crypto scaffold ──
        read -rp "Enter new crypto name (lowercase, e.g. my-cipher): " new_name
        new_name="$(echo "$new_name" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9_-]//g')"
        if [[ -z "$new_name" ]]; then echo "[-] Invalid name."; exit 1; fi
        new_file="$CRYPTO_DIR/$new_name.go.tmpl"
        if [[ -f "$new_file" ]]; then echo "[-] Crypto '$new_name' already exists."; exit 1; fi
        read -rp "Short description (shown in menu): " new_desc
        cat > "$new_file" << SCAFFOLD
// $new_desc
package __PACKAGE__

var SKey []byte

// EncryptData encrypts data with $new_name using key.
// TODO: implement
func EncryptData(data, key []byte) ([]byte, error) {
	panic("$new_name EncryptData not implemented")
}

// DecryptData decrypts data with $new_name using key.
// TODO: implement
func DecryptData(data, key []byte) ([]byte, error) {
	panic("$new_name DecryptData not implemented")
}
SCAFFOLD
        echo ""
        echo "[+] Created crypto scaffold: _crypto/$new_name.go.tmpl"
        echo "    Implement EncryptData/DecryptData, then re-run this generator to apply it."
        echo ""
        exit 0
    fi

    if [[ $idx -lt 0 || $idx -ge ${#crypto_keys[@]} ]]; then
        echo "[-] Invalid choice."; exit 1
    fi
    CRYPTO="${crypto_keys[$idx]}"
fi

# ─── Resolve crypto template file ───────────────────────────────────────────────

SELECTED_FILE=""
for i in "${!crypto_keys[@]}"; do
    if [[ "${crypto_keys[$i]}" == "$CRYPTO" ]]; then
        SELECTED_FILE="${crypto_files[$i]}"
        break
    fi
done
if [[ -z "$SELECTED_FILE" ]]; then
    echo "[-] Unknown crypto: $CRYPTO. Available: ${crypto_keys[*]}"
    exit 1
fi

# ─── Generate crypto template ───────────────────────────────────────────────────

DEST_FILE="$PROTO_DIR/crypto.go.tmpl"
cp "$SELECTED_FILE" "$DEST_FILE"

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
