#!/usr/bin/env bash
#
# generator.sh — Scaffold a new modular AdaptixC2 listener.
#
# Output goes to OUTPUT_DIR (or ADAPTIX_OUTPUT_DIR env var, or ./output).
#
# Usage:
#   ./generator.sh
#   OUTPUT_DIR=/path/to/extenders PROTOCOL=adaptix_default ./generator.sh
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TEMPLATE_DIR="$SCRIPT_DIR/templates"
TEMPLATES_ROOT="$(dirname "$SCRIPT_DIR")"
PROTOCOLS_DIR="$TEMPLATES_ROOT/protocols"

# Resolve output directory
OUTPUT_DIR="${OUTPUT_DIR:-${ADAPTIX_OUTPUT_DIR:-}}"
if [[ -z "$OUTPUT_DIR" ]]; then
    OUTPUT_DIR="$TEMPLATES_ROOT/output"
fi
mkdir -p "$OUTPUT_DIR"
EXTENDERS_DIR="$(cd "$OUTPUT_DIR" && pwd)"

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
echo -e "${CYAN}║   AdaptixC2 Template Listener Generator       ║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════╝${NC}"
echo ""

# ─── Optional: scaffold new protocol ────────────────────────────────────────────

PROTOCOL="${PROTOCOL:-}"
LISTENER_TYPE="${LISTENER_TYPE:-}"
TRANSPORT="${TRANSPORT:-tcp}"

# ─── Discover protocols ─────────────────────────────────────────────────────────

AVAILABLE_PROTOCOLS=()
if [[ -d "$PROTOCOLS_DIR" ]]; then
    for d in "$PROTOCOLS_DIR"/*/; do
        dname="$(basename "$d")"
        [[ "$dname" == "_scaffold" ]] && continue
        [[ -f "$d/meta.yaml" ]] && AVAILABLE_PROTOCOLS+=("$dname")
    done
fi

if [[ ${#AVAILABLE_PROTOCOLS[@]} -eq 0 ]]; then
    fail "No protocols found in $PROTOCOLS_DIR. Run with NEW_PROTOCOL=<name> to create one."
fi

# ─── Input: Protocol ────────────────────────────────────────────────────────────

if [[ -z "$PROTOCOL" ]]; then
    echo -e "${CYAN}Available protocols:${NC}"
    for i in "${!AVAILABLE_PROTOCOLS[@]}"; do
        pn="${AVAILABLE_PROTOCOLS[$i]}"
        desc=""
        meta_path="$PROTOCOLS_DIR/$pn/meta.yaml"
        if [[ -f "$meta_path" ]]; then
            desc_line=$(grep -oP 'description:\s*"\K[^"]*' "$meta_path" 2>/dev/null || true)
            [[ -n "$desc_line" ]] && desc=" - $desc_line"
        fi
        echo "  [$((i+1))] ${pn}${desc}"
    done
    echo ""
    while true; do
        read -rp "Select protocol [1-${#AVAILABLE_PROTOCOLS[@]}]: " choice
        if [[ "$choice" =~ ^[0-9]+$ ]] && (( choice >= 1 && choice <= ${#AVAILABLE_PROTOCOLS[@]} )); then
            PROTOCOL="${AVAILABLE_PROTOCOLS[$((choice-1))]}"
            break
        fi
        warn "Invalid choice."
    done
fi

PROTO_DIR="$PROTOCOLS_DIR/$PROTOCOL"
[[ -d "$PROTO_DIR" ]] || fail "Protocol '$PROTOCOL' not found in $PROTOCOLS_DIR"

# ─── Input: Listener name ──────────────────────────────────────────────────────

LISTENER_NAME="${NAME:-}"
if [[ -n "$LISTENER_NAME" ]]; then
    LISTENER_NAME=$(echo "$LISTENER_NAME" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9_')
    [[ -z "$LISTENER_NAME" ]] && fail "Invalid name."
    [[ -d "$EXTENDERS_DIR/${LISTENER_NAME}_listener" ]] && fail "Directory ${LISTENER_NAME}_listener already exists!"
else
    while true; do
        read -rp "Listener name (lowercase, e.g. telegram): " LISTENER_NAME
        LISTENER_NAME=$(echo "$LISTENER_NAME" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9_')
        if [[ -z "$LISTENER_NAME" ]]; then
            warn "Name cannot be empty."
            continue
        fi
        if [[ -d "$EXTENDERS_DIR/${LISTENER_NAME}_listener" ]]; then
            warn "Directory ${LISTENER_NAME}_listener already exists!"
            continue
        fi
        break
    done
fi

# Capitalize first letter
LISTENER_NAME_CAP="$(echo "${LISTENER_NAME:0:1}" | tr '[:lower:]' '[:upper:]')${LISTENER_NAME:1}"
PROTOCOL_CAP="$(echo "${PROTOCOL:0:1}" | tr '[:lower:]' '[:upper:]')${PROTOCOL:1}"

# ─── Input: Listener type ──────────────────────────────────────────────────────

if [[ -z "$LISTENER_TYPE" ]]; then
    read -rp "Listener type [external]: " LISTENER_TYPE
    LISTENER_TYPE=${LISTENER_TYPE:-external}
fi
if [[ "$LISTENER_TYPE" != "external" && "$LISTENER_TYPE" != "internal" ]]; then
    fail "Listener type must be 'external' or 'internal'."
fi

# ─── Input: Transport variant ───────────────────────────────────────────────────

TRANSPORT=$(echo "$TRANSPORT" | tr '[:upper:]' '[:lower:]')
case "$TRANSPORT" in
    tcp|http|telegram|dropbox|smb) ;;
    *) fail "Transport must be one of: tcp, http, telegram, dropbox, smb.";;
esac

echo ""
info "Creating listener: ${LISTENER_NAME}_listener"
info "  Protocol    : ${PROTOCOL}"
info "  Type        : ${LISTENER_TYPE}"
info "  Transport   : ${TRANSPORT}"
info "  Directory   : ${EXTENDERS_DIR}/${LISTENER_NAME}_listener/"
echo ""

# ─── Create directory ───────────────────────────────────────────────────────────

OUT_DIR="$EXTENDERS_DIR/${LISTENER_NAME}_listener"
mkdir "$OUT_DIR"

# ─── Substitute functions ───────────────────────────────────────────────────────

substitute() {
    sed -e "s|__NAME__|${LISTENER_NAME}|g" \
        -e "s|__NAME_CAP__|${LISTENER_NAME_CAP}|g" \
        -e "s|__PROTOCOL__|${PROTOCOL}|g" \
        -e "s|__PROTOCOL_CAP__|${PROTOCOL_CAP}|g" \
        -e "s|__LISTENER_TYPE__|${LISTENER_TYPE}|g" \
        "$1" > "$2"
}

substitute_protocol() {
    local src="$1" dst="$2" pkg="$3"
    sed -e "s|__PACKAGE__|${pkg}|g" "$src" > "$dst"
}

# ─── Copy template files ───────────────────────────────────────────────────────

info "Generating listener files..."
substitute "$TEMPLATE_DIR/config.yaml"      "$OUT_DIR/config.yaml"
substitute "$TEMPLATE_DIR/Makefile"         "$OUT_DIR/Makefile"

# go.mod: use protocol override if available, else base template
PROTO_GOMOD="$PROTO_DIR/go_mod.tmpl"
if [[ -f "$PROTO_GOMOD" ]]; then
    ok "Using protocol go.mod override"
    substitute "$PROTO_GOMOD" "$OUT_DIR/go.mod"
else
    substitute "$TEMPLATE_DIR/go.mod"       "$OUT_DIR/go.mod"
    substitute "$TEMPLATE_DIR/go.sum"       "$OUT_DIR/go.sum"
fi

# pl_main.go: check for transport-specific listener main override in protocol
LISTENER_MAIN="$PROTO_DIR/listener_main_${TRANSPORT}.go.tmpl"
if [[ -f "$LISTENER_MAIN" ]]; then
    ok "Using protocol listener main override: listener_main_${TRANSPORT}.go.tmpl"
    substitute "$LISTENER_MAIN" "$OUT_DIR/pl_main.go"
else
    substitute "$TEMPLATE_DIR/pl_main.go"   "$OUT_DIR/pl_main.go"
fi

# pl_internal.go: skip for non-socket transports; use protocol override if available, else base
if [[ "$TRANSPORT" != "http" && "$TRANSPORT" != "telegram" && "$TRANSPORT" != "dropbox" ]]; then
    PROTO_INTERNAL="$PROTO_DIR/pl_internal.go.tmpl"
    if [[ -f "$PROTO_INTERNAL" ]]; then
        ok "Using protocol internal override: pl_internal.go.tmpl"
        substitute "$PROTO_INTERNAL" "$OUT_DIR/pl_internal.go"
    else
        substitute "$TEMPLATE_DIR/pl_internal.go" "$OUT_DIR/pl_internal.go"
    fi
fi

# pl_transport.go: check for transport-specific override, else default override, else base
PROTO_TRANSPORT_VARIANT="$PROTO_DIR/pl_transport_${TRANSPORT}.go.tmpl"
PROTO_TRANSPORT="$PROTO_DIR/pl_transport.go.tmpl"
if [[ -f "$PROTO_TRANSPORT_VARIANT" ]]; then
    ok "Using protocol transport override: pl_transport_${TRANSPORT}.go.tmpl"
    substitute "$PROTO_TRANSPORT_VARIANT" "$OUT_DIR/pl_transport.go"
elif [[ -f "$PROTO_TRANSPORT" ]]; then
    ok "Using protocol transport override: pl_transport.go.tmpl"
    substitute "$PROTO_TRANSPORT" "$OUT_DIR/pl_transport.go"
else
    substitute "$TEMPLATE_DIR/pl_transport.go"  "$OUT_DIR/pl_transport.go"
fi

# map.go: only needed for transports that use concurrent maps (TCP)
if [[ "$TRANSPORT" == "tcp" ]]; then
    substitute "$TEMPLATE_DIR/map.go"       "$OUT_DIR/map.go"
fi

# ax_config.axs: check for transport-specific override in protocol, else base
PROTO_AX_CONFIG="$PROTO_DIR/ax_config_${TRANSPORT}.axs.tmpl"
if [[ -f "$PROTO_AX_CONFIG" ]]; then
    ok "Using protocol ax_config override: ax_config_${TRANSPORT}.axs.tmpl"
    substitute "$PROTO_AX_CONFIG" "$OUT_DIR/ax_config.axs"
else
    substitute "$TEMPLATE_DIR/ax_config.axs" "$OUT_DIR/ax_config.axs"
fi

# ─── Copy from protocol ────────────────────────────────────────────────────────

info "Applying protocol: $PROTOCOL"

# pl_crypto.go from protocol's crypto.go.tmpl
CRYPTO_TMPL="$PROTO_DIR/crypto.go.tmpl"
if [[ -f "$CRYPTO_TMPL" ]]; then
    substitute_protocol "$CRYPTO_TMPL" "$OUT_DIR/pl_crypto.go" "main"
else
    warn "No crypto.go.tmpl in protocol '$PROTOCOL', using template default."
    substitute "$TEMPLATE_DIR/pl_crypto.go" "$OUT_DIR/pl_crypto.go"
fi

# pl_utils.go: merge constants.go.tmpl + types.go.tmpl from protocol
CONSTANTS_TMPL="$PROTO_DIR/constants.go.tmpl"
TYPES_TMPL="$PROTO_DIR/types.go.tmpl"
if [[ -f "$CONSTANTS_TMPL" && -f "$TYPES_TMPL" ]]; then
    # types.go.tmpl has the package line + imports
    # constants.go.tmpl has a duplicate package line — strip it
    {
        cat "$TYPES_TMPL"
        echo ""
        # Strip the package line from constants
        sed '/^package /d' "$CONSTANTS_TMPL"
    } | sed "s|__PACKAGE__|main|g" > "$OUT_DIR/pl_utils.go"
else
    warn "Protocol missing constants/types templates, using template default."
    substitute "$TEMPLATE_DIR/pl_utils.go" "$OUT_DIR/pl_utils.go"
fi

# ─── Summary ────────────────────────────────────────────────────────────────────

echo ""
ok "Listener '${LISTENER_NAME}_listener' scaffolded successfully!"
echo ""
echo -e "${CYAN}Directory structure:${NC}"
echo ""
echo "  ${LISTENER_NAME}_listener/"
echo "  ├── config.yaml          # Listener manifest"
echo "  ├── go.mod               # Go module"
echo "  ├── Makefile             # Build targets"
echo "  ├── pl_main.go           # Plugin entry + Teamserver interface"
echo "  ├── pl_transport.go      # Transport: Start/Stop/handleConnection"
echo "  ├── pl_crypto.go         # Encrypt/Decrypt (from protocol: $PROTOCOL)"
echo "  ├── pl_utils.go          # Wire types + constants (from protocol: $PROTOCOL)"
echo "  ├── map.go               # Thread-safe concurrent map"
echo "  └── ax_config.axs        # Listener UI form"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "  1. cd ${EXTENDERS_DIR}/${LISTENER_NAME}_listener"
echo "  2. Edit pl_transport.go to customize handleConnection for your transport"
echo "  3. Edit ax_config.axs if you need different UI fields"
echo "  4. go mod tidy"
echo "  5. make plugin"
echo ""
echo -e "${CYAN}  Agent compatibility:${NC}"
echo "    Agents using protocol '$PROTOCOL' are compatible with this listener."
echo "    Set listeners: [\"${LISTENER_NAME_CAP}${PROTOCOL_CAP}\"] in your agent's config.yaml."
echo ""
