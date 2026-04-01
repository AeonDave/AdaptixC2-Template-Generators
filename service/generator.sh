#!/usr/bin/env bash
#
# generator.sh — Scaffold a new AdaptixC2 service plugin.
#
# When WRAPPER=1 (or the user answers yes interactively), the generator includes
# the post-build wrapper pipeline: event hook, pl_wrapper.go, and wrapper UI.
#
# Output goes to OUTPUT_DIR (or ADAPTIX_OUTPUT_DIR env var, or ./output).
#
# Usage:
#   ./generator.sh
#   NAME=telegram ./generator.sh
#   NAME=crystalpalace WRAPPER=1 ./generator.sh
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
IS_WRAPPER="${WRAPPER:-0}"

if [[ -n "$SERVICE_NAME" ]]; then
    SERVICE_NAME=$(echo "$SERVICE_NAME" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9_')
    [[ -z "$SERVICE_NAME" ]] && fail "Invalid service name."
else
    while true; do
        read -rp "Service name (lowercase, e.g. telegram): " SERVICE_NAME
        SERVICE_NAME=$(echo "$SERVICE_NAME" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9_')
        if [[ -z "$SERVICE_NAME" ]]; then
            warn "Name cannot be empty."
            continue
        fi
        break
    done
fi

# ─── Input: Wrapper option ──────────────────────────────────────────────────────

if [[ "$IS_WRAPPER" != "1" && -z "${NAME:-}" ]]; then
    read -rp "Include post-build wrapper pipeline? [y/N]: " answer
    if [[ "$answer" =~ ^[Yy] ]]; then
        IS_WRAPPER="1"
    fi
fi

# Determine suffix
if [[ "$IS_WRAPPER" == "1" ]]; then
    SUFFIX="wrapper"
else
    SUFFIX="service"
fi

OUT_DIR="$EXTENDERS_DIR/${SERVICE_NAME}_${SUFFIX}"
[[ -d "$OUT_DIR" ]] && fail "Directory ${SERVICE_NAME}_${SUFFIX} already exists!"

# Capitalize first letter
SERVICE_NAME_CAP="$(echo "${SERVICE_NAME:0:1}" | tr '[:lower:]' '[:upper:]')${SERVICE_NAME:1}"

echo ""
info "Creating ${SUFFIX}: ${SERVICE_NAME}_${SUFFIX}"
info "  Directory   : ${OUT_DIR}/"
echo ""

# ─── Create directory ───────────────────────────────────────────────────────────

mkdir "$OUT_DIR"

# ─── Substitute function ───────────────────────────────────────────────────────

substitute() {
    sed -e "s|__NAME__|${SERVICE_NAME}|g" \
        -e "s|__NAME_CAP__|${SERVICE_NAME_CAP}|g" \
        "$1" > "$2"
}

# ─── Select template source ────────────────────────────────────────────────────
# Wrapper templates override the base when the wrapper option is active.

resolve_template() {
    local file="$1"
    if [[ "$IS_WRAPPER" == "1" ]]; then
        local override="$TEMPLATE_DIR/wrapper/$file"
        if [[ -f "$override" ]]; then
            echo "$override"
            return
        fi
    fi
    echo "$TEMPLATE_DIR/$file"
}

# ─── Copy template files ───────────────────────────────────────────────────────

info "Generating ${SUFFIX} files..."
substitute "$(resolve_template config.yaml)"   "$OUT_DIR/config.yaml"
substitute "$(resolve_template go.mod)"        "$OUT_DIR/go.mod"
substitute "$(resolve_template go.sum)"        "$OUT_DIR/go.sum"
substitute "$(resolve_template Makefile)"      "$OUT_DIR/Makefile"
substitute "$(resolve_template pl_main.go)"    "$OUT_DIR/pl_main.go"
substitute "$(resolve_template ax_config.axs)" "$OUT_DIR/ax_config.axs"

if [[ "$IS_WRAPPER" == "1" ]]; then
    wrapper_src="$TEMPLATE_DIR/wrapper/pl_wrapper.go"
    if [[ -f "$wrapper_src" ]]; then
        substitute "$wrapper_src" "$OUT_DIR/pl_wrapper.go"
    fi
fi

# ─── Summary ────────────────────────────────────────────────────────────────────

echo ""
ok "${SUFFIX^} '${SERVICE_NAME}_${SUFFIX}' scaffolded successfully!"
echo ""
echo -e "${CYAN}Directory structure:${NC}"
echo ""
echo "  ${SERVICE_NAME}_${SUFFIX}/"
echo "  ├── config.yaml          # Service manifest"
echo "  ├── go.mod               # Go module"
echo "  ├── Makefile             # Build targets"
if [[ "$IS_WRAPPER" == "1" ]]; then
    echo "  ├── pl_main.go           # Plugin entry + event hooks + handlers"
    echo "  ├── pl_wrapper.go        # Pipeline engine (stages)"
else
    echo "  ├── pl_main.go           # Plugin entry + Call handler"
fi
echo "  └── ax_config.axs        # Service UI form"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "  1. cd ${OUT_DIR}"
if [[ "$IS_WRAPPER" == "1" ]]; then
    echo "  2. Edit pl_main.go — register stages in initStages()"
    echo "  3. Add stage functions (e.g. stageEncrypt, stagePack) in pl_wrapper.go or new files"
else
    echo "  2. Edit pl_main.go — add function handlers in the Call() switch"
    echo "  3. Edit ax_config.axs — add your functions to the combo + UI fields"
fi
echo "  4. go mod tidy"
echo "  5. make plugin"
echo ""
