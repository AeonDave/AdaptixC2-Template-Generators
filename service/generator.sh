#!/usr/bin/env bash
#
# generator.sh — Scaffold a new AdaptixC2 service plugin.
#
# Output goes to OUTPUT_DIR (or ADAPTIX_OUTPUT_DIR env var, or ./output).
#
# Usage:
#   ./generator.sh
#   NAME=telegram ./generator.sh
#   NAME=telegram OUTPUT_DIR=/path/to/extenders ./generator.sh
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TEMPLATE_DIR="$SCRIPT_DIR/templates"
TEMPLATES_ROOT="$(dirname "$SCRIPT_DIR")"

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
echo -e "${CYAN}║   AdaptixC2 Template Service Generator        ║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════╝${NC}"
echo ""

# ─── Input: Service name ────────────────────────────────────────────────────────

SERVICE_NAME="${NAME:-}"

if [[ -n "$SERVICE_NAME" ]]; then
    SERVICE_NAME=$(echo "$SERVICE_NAME" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9_')
    [[ -z "$SERVICE_NAME" ]] && fail "Invalid service name."
    [[ -d "$EXTENDERS_DIR/${SERVICE_NAME}_service" ]] && fail "Directory ${SERVICE_NAME}_service already exists!"
else
    while true; do
        read -rp "Service name (lowercase, e.g. telegram): " SERVICE_NAME
        SERVICE_NAME=$(echo "$SERVICE_NAME" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9_')
        if [[ -z "$SERVICE_NAME" ]]; then
            warn "Name cannot be empty."
            continue
        fi
        if [[ -d "$EXTENDERS_DIR/${SERVICE_NAME}_service" ]]; then
            warn "Directory ${SERVICE_NAME}_service already exists!"
            continue
        fi
        break
    done
fi

# Capitalize first letter
SERVICE_NAME_CAP="$(echo "${SERVICE_NAME:0:1}" | tr '[:lower:]' '[:upper:]')${SERVICE_NAME:1}"

echo ""
info "Creating service: ${SERVICE_NAME}_service"
info "  Directory   : ${EXTENDERS_DIR}/${SERVICE_NAME}_service/"
echo ""

# ─── Create directory ───────────────────────────────────────────────────────────

OUT_DIR="$EXTENDERS_DIR/${SERVICE_NAME}_service"
mkdir -p "$OUT_DIR"

# ─── Substitute function ───────────────────────────────────────────────────────

substitute() {
    sed -e "s|__NAME__|${SERVICE_NAME}|g" \
        -e "s|__NAME_CAP__|${SERVICE_NAME_CAP}|g" \
        "$1" > "$2"
}

# ─── Copy template files ───────────────────────────────────────────────────────

info "Generating service files..."
substitute "$TEMPLATE_DIR/config.yaml"   "$OUT_DIR/config.yaml"
substitute "$TEMPLATE_DIR/go.mod"        "$OUT_DIR/go.mod"
substitute "$TEMPLATE_DIR/go.sum"        "$OUT_DIR/go.sum"
substitute "$TEMPLATE_DIR/Makefile"      "$OUT_DIR/Makefile"
substitute "$TEMPLATE_DIR/pl_main.go"    "$OUT_DIR/pl_main.go"
substitute "$TEMPLATE_DIR/ax_config.axs" "$OUT_DIR/ax_config.axs"

# ─── Summary ────────────────────────────────────────────────────────────────────

echo ""
ok "Service '${SERVICE_NAME}_service' scaffolded successfully!"
echo ""
echo -e "${CYAN}Directory structure:${NC}"
echo ""
echo "  ${SERVICE_NAME}_service/"
echo "  ├── config.yaml          # Service manifest"
echo "  ├── go.mod               # Go module"
echo "  ├── Makefile             # Build targets"
echo "  ├── pl_main.go           # Plugin entry + Call handler"
echo "  └── ax_config.axs        # Service UI form"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "  1. cd ${EXTENDERS_DIR}/${SERVICE_NAME}_service"
echo "  2. Edit pl_main.go — add function handlers in the Call() switch"
echo "  3. Edit ax_config.axs — add your functions to the combo + UI fields"
echo "  4. go mod tidy"
echo "  5. make plugin"
echo ""
