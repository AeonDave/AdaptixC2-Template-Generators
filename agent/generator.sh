#!/usr/bin/env bash
#
# generator.sh — Scaffold a new modular AdaptixC2 agent.
#
# Output goes to OUTPUT_DIR (or ADAPTIX_OUTPUT_DIR env var, or ./output).
#
# Usage:
#   ./generator.sh
#   OUTPUT_DIR=/path/to/extenders ./generator.sh
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TEMPLATE_DIR="$SCRIPT_DIR/templates"
TEMPLATES_ROOT="$(dirname "$SCRIPT_DIR")"
PROTOCOLS_DIR="$TEMPLATES_ROOT/protocols"

# Language & toolchain (env-based for non-interactive)
LANGUAGE="${LANGUAGE:-}"
TOOLCHAIN="${TOOLCHAIN:-}"

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

# ─── Input ──────────────────────────────────────────────────────────────────────

echo ""
echo -e "${CYAN}╔═══════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║   AdaptixC2 Template Agent Generator          ║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════╝${NC}"
echo ""

# Agent name (lowercase, alphanumeric + underscore)
while true; do
    read -rp "Agent name (lowercase, e.g. phantom): " AGENT_NAME
    AGENT_NAME=$(echo "$AGENT_NAME" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9_')
    if [[ -z "$AGENT_NAME" ]]; then
        warn "Name cannot be empty."
        continue
    fi
    if [[ -d "$EXTENDERS_DIR/${AGENT_NAME}_agent" ]]; then
        warn "Directory ${AGENT_NAME}_agent already exists!"
        continue
    fi
    break
done

# Capitalize first letter
AGENT_NAME_CAP="$(echo "${AGENT_NAME:0:1}" | tr '[:lower:]' '[:upper:]')${AGENT_NAME:1}"

# Watermark (8-char hex, auto-generated or custom)
DEFAULT_WATERMARK=$(head -c 4 /dev/urandom | xxd -p)
read -rp "Watermark [${DEFAULT_WATERMARK}]: " WATERMARK
WATERMARK=${WATERMARK:-$DEFAULT_WATERMARK}
# Validate: must be exactly 8 hex chars
if ! echo "$WATERMARK" | grep -qE '^[0-9a-fA-F]{8}$'; then
    fail "Watermark must be exactly 8 hex characters (e.g. a1b2c3d4)."
fi

# ─── Protocol selection ─────────────────────────────────────────────────────────

PROTOCOL="${PROTOCOL:-}"

AVAILABLE_PROTOCOLS=()
if [[ -d "$PROTOCOLS_DIR" ]]; then
    for d in "$PROTOCOLS_DIR"/*/; do
        dname="$(basename "$d")"
        [[ "$dname" == "_scaffold" ]] && continue
        [[ -f "$d/meta.yaml" ]] && AVAILABLE_PROTOCOLS+=("$dname")
    done
fi

if [[ ${#AVAILABLE_PROTOCOLS[@]} -eq 0 ]]; then
    warn "No protocols found in $PROTOCOLS_DIR. Using template defaults."
    PROTOCOL=""
elif [[ -z "$PROTOCOL" ]]; then
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
    echo "  [0] None (use template defaults)"
    echo ""
    while true; do
        read -rp "Select protocol [default: 1]: " choice
        [[ -z "$choice" ]] && choice=1
        if [[ "$choice" =~ ^[0-9]+$ ]]; then
            if (( choice == 0 )); then
                PROTOCOL=""
                break
            elif (( choice >= 1 && choice <= ${#AVAILABLE_PROTOCOLS[@]} )); then
                PROTOCOL="${AVAILABLE_PROTOCOLS[$((choice-1))]}"
                break
            fi
        fi
        warn "Invalid choice."
    done
fi

PROTO_DIR=""
if [[ -n "$PROTOCOL" ]]; then
    PROTO_DIR="$PROTOCOLS_DIR/$PROTOCOL"
    [[ -d "$PROTO_DIR" ]] || fail "Protocol '$PROTOCOL' not found in $PROTOCOLS_DIR"
fi

# ─── Language selection ──────────────────────────────────────────────────────

if [[ -z "$LANGUAGE" ]]; then
    # Discover available languages from template directories
    AVAILABLE_LANGS=()
    for lang in go cpp rust; do
        [[ -d "$TEMPLATE_DIR/implant/$lang" ]] && AVAILABLE_LANGS+=("$lang")
    done

    if [[ ${#AVAILABLE_LANGS[@]} -eq 0 ]]; then
        fail "No implant template directories found."
    elif [[ ${#AVAILABLE_LANGS[@]} -eq 1 ]]; then
        LANGUAGE="${AVAILABLE_LANGS[0]}"
    else
        declare -A LANG_DESCS=( [go]="Go implant" [cpp]="C/C++ implant" [rust]="Rust implant" )
        echo -e "${CYAN}Select implant language:${NC}"
        for i in "${!AVAILABLE_LANGS[@]}"; do
            l="${AVAILABLE_LANGS[$i]}"
            def=""
            (( i == 0 )) && def=" (default)"
            echo "  [$((i+1))] ${l}${def}  - ${LANG_DESCS[$l]}"
        done
        echo ""
        read -rp "Select language [default: 1]: " choice
        [[ -z "$choice" ]] && choice=1
        if [[ "$choice" =~ ^[0-9]+$ ]] && (( choice >= 1 && choice <= ${#AVAILABLE_LANGS[@]} )); then
            LANGUAGE="${AVAILABLE_LANGS[$((choice-1))]}"
        else
            LANGUAGE="${AVAILABLE_LANGS[0]}"
        fi
    fi
fi

case "$LANGUAGE" in
    go|cpp|rust) ;;
    *) fail "Unsupported language: $LANGUAGE. Choose: go, cpp, rust" ;;
esac

IMPLANT_LANG_DIR="$TEMPLATE_DIR/implant/$LANGUAGE"
[[ -d "$IMPLANT_LANG_DIR" ]] || fail "No implant templates for language '$LANGUAGE' in $IMPLANT_LANG_DIR"

# ─── Toolchain selection ─────────────────────────────────────────────────────

TOOLCHAINS_DIR="$SCRIPT_DIR/toolchains"

# Default toolchain per language
case "$LANGUAGE" in
    go)   DEFAULT_TC="go-standard" ;;
    cpp)  DEFAULT_TC="mingw" ;;
    rust) DEFAULT_TC="cargo" ;;
esac

if [[ -z "$TOOLCHAIN" ]]; then
    # Scan available toolchains for the selected language
    MATCHING_TCS=()
    MATCHING_DESCS=()
    if [[ -d "$TOOLCHAINS_DIR" ]]; then
        for tcfile in "$TOOLCHAINS_DIR"/*.yaml; do
            [[ -f "$tcfile" ]] || continue
            tc_lang=$(grep -oP 'language:\s*\K\S+' "$tcfile" 2>/dev/null || true)
            if [[ "$tc_lang" == "$LANGUAGE" ]]; then
                tc_name="$(basename "${tcfile%.yaml}")"
                tc_desc=$(grep -oP 'description:\s*"\K[^"]+' "$tcfile" 2>/dev/null || true)
                MATCHING_TCS+=("$tc_name")
                MATCHING_DESCS+=("$tc_desc")
            fi
        done
    fi

    if [[ ${#MATCHING_TCS[@]} -eq 0 ]]; then
        TOOLCHAIN="$DEFAULT_TC"
    elif [[ ${#MATCHING_TCS[@]} -eq 1 ]]; then
        TOOLCHAIN="${MATCHING_TCS[0]}"
    else
        # Find default index
        default_idx=0
        for i in "${!MATCHING_TCS[@]}"; do
            [[ "${MATCHING_TCS[$i]}" == "$DEFAULT_TC" ]] && { default_idx=$i; break; }
        done

        echo -e "${CYAN}Available toolchains for '$LANGUAGE':${NC}"
        for i in "${!MATCHING_TCS[@]}"; do
            def=""
            [[ "${MATCHING_TCS[$i]}" == "$DEFAULT_TC" ]] && def=" (default)"
            echo "  [$((i+1))] ${MATCHING_TCS[$i]}${def}  - ${MATCHING_DESCS[$i]}"
        done
        echo ""
        read -rp "Select toolchain [default: $((default_idx+1))]: " choice
        [[ -z "$choice" ]] && choice=$((default_idx+1))
        if [[ "$choice" =~ ^[0-9]+$ ]] && (( choice >= 1 && choice <= ${#MATCHING_TCS[@]} )); then
            TOOLCHAIN="${MATCHING_TCS[$((choice-1))]}"
        else
            TOOLCHAIN="$DEFAULT_TC"
        fi
    fi
fi

TOOLCHAIN_FILE="$TOOLCHAINS_DIR/$TOOLCHAIN.yaml"
if [[ ! -f "$TOOLCHAIN_FILE" ]]; then
    warn "Toolchain file '$TOOLCHAIN_FILE' not found. Continuing without toolchain overlay."
    TOOLCHAIN_FILE=""
fi

echo ""
info "Creating agent: ${AGENT_NAME}"
info "  Language  : ${LANGUAGE}"
info "  Toolchain : ${TOOLCHAIN}"
info "  Watermark : ${WATERMARK}"
[[ -n "$PROTOCOL" ]] && info "  Protocol  : ${PROTOCOL}"
info "  Directory : ${EXTENDERS_DIR}/${AGENT_NAME}_agent/"
echo ""

# ─── Create directory structure ─────────────────────────────────────────────────

OUT_DIR="$EXTENDERS_DIR/${AGENT_NAME}_agent"
SRC_DIR="$OUT_DIR/src_${AGENT_NAME}"

mkdir -p "$OUT_DIR"
mkdir -p "$SRC_DIR"
[[ -d "$IMPLANT_LANG_DIR/impl" ]]     && mkdir -p "$SRC_DIR/impl"
[[ -d "$IMPLANT_LANG_DIR/crypto" ]]   && mkdir -p "$SRC_DIR/crypto"
[[ -d "$IMPLANT_LANG_DIR/protocol" ]] && mkdir -p "$SRC_DIR/protocol"

# ─── Parse toolchain ────────────────────────────────────────────────────────────

BUILD_TOOL="go build"
if [[ -n "$TOOLCHAIN_FILE" && -f "$TOOLCHAIN_FILE" ]]; then
    _cmd=$(grep -oP 'command:\s*"?\K[^"]+' "$TOOLCHAIN_FILE" 2>/dev/null | head -1 || true)
    [[ -n "$_cmd" ]] && BUILD_TOOL="$_cmd"
fi

# ─── Copy and substitute templates ─────────────────────────────────────────────

substitute() {
    sed -e "s|__NAME__|${AGENT_NAME}|g" \
        -e "s|__NAME_CAP__|${AGENT_NAME_CAP}|g" \
        -e "s|__WATERMARK__|${WATERMARK}|g" \
        -e "s|__BUILD_TOOL__|${BUILD_TOOL}|g" \
        "$1" > "$2"
}

# Plugin files (top-level)
info "Generating plugin files..."
substitute "$TEMPLATE_DIR/plugin/config.yaml"  "$OUT_DIR/config.yaml"
substitute "$TEMPLATE_DIR/plugin/go.mod"       "$OUT_DIR/go.mod"
substitute "$TEMPLATE_DIR/plugin/go.sum"       "$OUT_DIR/go.sum"
substitute "$TEMPLATE_DIR/plugin/Makefile"     "$OUT_DIR/Makefile"
substitute "$TEMPLATE_DIR/plugin/pl_utils.go"  "$OUT_DIR/pl_utils.go"
substitute "$TEMPLATE_DIR/plugin/pl_main.go"   "$OUT_DIR/pl_main.go"

# ax_config.axs — language-specific UI definition
AX_CONFIG_VARIANT="ax_config.axs"
if [[ "$LANGUAGE" != "go" ]]; then
    _lang_axs="ax_config_${LANGUAGE}.axs"
    if [[ -f "$TEMPLATE_DIR/plugin/$_lang_axs" ]]; then
        AX_CONFIG_VARIANT="$_lang_axs"
    fi
fi
substitute "$TEMPLATE_DIR/plugin/$AX_CONFIG_VARIANT" "$OUT_DIR/ax_config.axs"

# Plugin build variant (language-specific: pl_build_go.go or pl_build_cpp.go)
case "$LANGUAGE" in
    go)   BUILD_VARIANT="pl_build_go.go" ;;
    cpp)  BUILD_VARIANT="pl_build_cpp.go" ;;
    rust) BUILD_VARIANT="pl_build_rust.go" ;;
    *)    BUILD_VARIANT="" ;;
esac
if [[ -n "$BUILD_VARIANT" && -f "$TEMPLATE_DIR/plugin/$BUILD_VARIANT" ]]; then
    substitute "$TEMPLATE_DIR/plugin/$BUILD_VARIANT" "$OUT_DIR/pl_build.go"
fi

# Implant files (all top-level files from the language template dir)
info "Generating implant files ($LANGUAGE)..."
for f in "$IMPLANT_LANG_DIR"/*; do
    [[ -f "$f" ]] || continue
    substitute "$f" "$SRC_DIR/$(basename "$f")"
done

# Crypto — from protocol .go.tmpl if Go and available, otherwise from language template
if [[ "$LANGUAGE" == "go" && -n "$PROTOCOL" && -f "$PROTO_DIR/crypto.go.tmpl" ]]; then
    info "Applying protocol '$PROTOCOL' crypto..."
    sed "s|__PACKAGE__|crypto|g" "$PROTO_DIR/crypto.go.tmpl" > "$SRC_DIR/crypto/crypto.go"
else
    for f in "$IMPLANT_LANG_DIR"/crypto/*; do
        [[ -f "$f" ]] || continue
        substitute "$f" "$SRC_DIR/crypto/$(basename "$f")"
    done
fi

# Protocol types — from protocol .go.tmpl if Go and available, otherwise from language template
if [[ "$LANGUAGE" == "go" && -n "$PROTOCOL" && -f "$PROTO_DIR/types.go.tmpl" && -f "$PROTO_DIR/constants.go.tmpl" ]]; then
    info "Applying protocol '$PROTOCOL' types + constants..."
    {
        cat "$PROTO_DIR/types.go.tmpl"
        echo ""
        sed '1{/^package /d}' "$PROTO_DIR/constants.go.tmpl"
    } | sed "s|__PACKAGE__|protocol|g" > "$SRC_DIR/protocol/protocol.go"
else
    for f in "$IMPLANT_LANG_DIR"/protocol/*; do
        [[ -f "$f" ]] || continue
        substitute "$f" "$SRC_DIR/protocol/$(basename "$f")"
    done
fi

# Plugin pl_utils.go — overlay with protocol if available
if [[ -n "$PROTOCOL" && -f "$PROTO_DIR/types.go.tmpl" && -f "$PROTO_DIR/constants.go.tmpl" ]]; then
    info "Applying protocol '$PROTOCOL' to pl_utils.go..."
    {
        cat "$PROTO_DIR/types.go.tmpl"
        echo ""
        sed '1{/^package /d}' "$PROTO_DIR/constants.go.tmpl"
    } | sed "s|__PACKAGE__|main|g" > "$OUT_DIR/pl_utils.go"
fi

# Impl stubs — copy all subdirectories recursively (except crypto/ and protocol/)
info "Generating interface stubs..."
for sub_dir in "$IMPLANT_LANG_DIR"/*/; do
    [[ -d "$sub_dir" ]] || continue
    dname="$(basename "$sub_dir")"
    [[ "$dname" == "crypto" || "$dname" == "protocol" ]] && continue
    mkdir -p "$SRC_DIR/$dname"
    find "$sub_dir" -type f | while IFS= read -r f; do
        rel="${f#$sub_dir}"
        dest="$SRC_DIR/$dname/$rel"
        mkdir -p "$(dirname "$dest")"
        substitute "$f" "$dest"
    done
done

# ─── Summary ────────────────────────────────────────────────────────────────────

echo ""
if [[ -n "$PROTOCOL" ]]; then
    ok "Agent '${AGENT_NAME}' scaffolded with protocol '${PROTOCOL}' (${LANGUAGE})!"
else
    ok "Agent '${AGENT_NAME}' scaffolded successfully (${LANGUAGE})!"
fi
echo ""
echo -e "${CYAN}Directory structure:${NC}"
echo ""
echo "  ${AGENT_NAME}_agent/"
echo "  ├── config.yaml          # Plugin manifest"
echo "  ├── go.mod               # Plugin module"
echo "  ├── Makefile             # Build targets"
echo "  ├── pl_utils.go          # Wire types & constants"
echo "  ├── pl_main.go           # Plugin logic (server-side)"
echo "  ├── pl_build.go          # Build logic (${LANGUAGE})"
echo "  ├── ax_config.axs        # UI & command definitions"
echo "  └── src_${AGENT_NAME}/"
find "$SRC_DIR" -type f -printf '      %P\n' | sort
echo ""
echo -e "${CYAN}Language  : ${LANGUAGE}${NC}"
echo -e "${CYAN}Toolchain : ${TOOLCHAIN}${NC}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "  1. cd ${EXTENDERS_DIR}/${AGENT_NAME}_agent"
echo "  2. Implement the TODO stubs in src_${AGENT_NAME}/"
echo "  3. Build: make full"
echo ""
