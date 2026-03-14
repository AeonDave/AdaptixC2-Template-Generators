#!/usr/bin/env bash
#
# generator.sh — Create a new protocol definition for AdaptixC2.
#
# Usage:
#   cd extenders/templates/protocols
#   bash generator.sh
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SCAFFOLD_DIR="$SCRIPT_DIR/_scaffold"

# ─── Colors ─────────────────────────────────────────────────────────────────────

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${CYAN}[*]${NC} $1"; }
ok()    { echo -e "${GREEN}[+]${NC} $1"; }
warn()  { echo -e "${YELLOW}[!]${NC} $1"; }
fail()  { echo -e "${RED}[-]${NC} $1"; exit 1; }

# ─── Banner ─────────────────────────────────────────────────────────────────────

echo ""
echo -e "${CYAN}╔═══════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║   AdaptixC2 Protocol Generator                ║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════╝${NC}"
echo ""

[[ -d "$SCAFFOLD_DIR" ]] || fail "Scaffold directory not found: $SCAFFOLD_DIR"

# ─── Input ──────────────────────────────────────────────────────────────────────

PROTO_NAME="${NAME:-}"

if [[ -z "$PROTO_NAME" ]]; then
    while true; do
        read -rp "Protocol name (lowercase, e.g. chacha): " PROTO_NAME
        PROTO_NAME=$(echo "$PROTO_NAME" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9_')
        if [[ -z "$PROTO_NAME" ]]; then
            warn "Name cannot be empty."
            continue
        fi
        break
    done
fi

OUT_DIR="$SCRIPT_DIR/$PROTO_NAME"
[[ -d "$OUT_DIR" ]] && fail "Protocol '$PROTO_NAME' already exists at $OUT_DIR"

echo ""
info "Creating protocol: $PROTO_NAME"
info "  Directory : $OUT_DIR/"
echo ""

# ─── Scaffold ───────────────────────────────────────────────────────────────────

mkdir -p "$OUT_DIR"
for f in "$SCAFFOLD_DIR"/*; do
    [[ -f "$f" ]] || continue
    sed -e "s|__PROTO_NAME__|${PROTO_NAME}|g" "$f" > "$OUT_DIR/$(basename "$f")"
done

# ─── Summary ────────────────────────────────────────────────────────────────────

echo ""
ok "Protocol '$PROTO_NAME' created!"
echo ""
echo -e "${CYAN}Files:${NC}"
echo "  $PROTO_NAME/"
echo "  ├── meta.yaml           # Protocol metadata"
echo "  ├── crypto.go.tmpl      # EncryptData / DecryptData"
echo "  ├── constants.go.tmpl   # COMMAND_* / RESP_* constants"
echo "  └── types.go.tmpl       # Wire types, framing helpers"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "  1. Edit crypto.go.tmpl    - implement your encryption"
echo "  2. Edit types.go.tmpl     - define wire types and framing"
echo "  3. Edit constants.go.tmpl - add command/response constants"
echo "  4. Use with: generator.sh (select protocol '$PROTO_NAME')"
echo ""
