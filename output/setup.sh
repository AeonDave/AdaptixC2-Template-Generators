#!/bin/bash

# ============================================================================
# setup.sh — Deploy extenders into an AdaptixC2 installation
#
# Auto-discovers agents, listeners, and services from subdirectories by
# reading config.yaml → extender_type.  Copies into AdaptixServer/extenders/,
# registers them in the Go workspace, builds Go plugins (.so), and copies
# distribution artifacts to dist/extenders/.
# ============================================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BLUE='\033[0;34m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

error_exit() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }
info_msg()   { echo -e "${GREEN}[+]${NC} $1"; }
warn_msg()   { echo -e "${YELLOW}[!]${NC} $1"; }
step_msg()   { echo -e "${CYAN}[*]${NC} $1"; }

# ── Defaults ────────────────────────────────────────────────────────────────
ADAPTIX_DIR=""
INPUT_DIR=""
GO_BIN=""
PULL_CHANGES=false
ACTION=""
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SERVER_TRIMPATH="-trimpath"

# ── Auto-discover extenders ────────────────────────────────────────────────
# Scans subdirectories for config.yaml with extender_type field
AGENTS=()
LISTENERS=()
SERVICES=()

discover_extenders() {
    for dir in "$INPUT_DIR"/*/; do
        local name
        name="$(basename "$dir")"
        local cfg="$dir/config.yaml"
        [[ -f "$cfg" ]] || continue
        local etype
        etype="$(grep -m1 '^extender_type:' "$cfg" | sed 's/extender_type:[[:space:]]*//;s/"//g;s/[[:space:]]*$//')"
        case "$etype" in
            agent)    AGENTS+=("$name") ;;
            listener) LISTENERS+=("$name") ;;
            service)  SERVICES+=("$name") ;;
            *)        warn_msg "Unknown extender_type '$etype' in $name/config.yaml (skipping)" ;;
        esac
    done
}

# ── Usage ───────────────────────────────────────────────────────────────────
usage() {
    cat <<EOF
Usage: $0 -o <AdaptixC2_dir> [-a <action>] [-i <input_dir>] [-g <go_binary>] [--pull]

Required:
  -o, --ax <dir>      Path to AdaptixC2 output directory

Optional:
  -a, --action <act>  Action to perform (omit for interactive selector)
  -i, --input <dir>   Folder containing extender subdirectories
                      (default: same directory as this script)
  -g, --go <path>     Go binary to use for building plugins
                      (auto-detected from AdaptixServer/Makefile if omitted)
  --pull              Execute git pull before installation

Actions:
  all                 Install all discovered extenders
  agents              Agents only
  listeners           Listeners only
  services            Services only
  clean               Remove all discovered extenders from AdaptixC2
  (none)              Interactive selector — pick which extenders to install

Examples:
  $0 -o ../AdaptixC2                          # interactive selector
  $0 -o /opt/AdaptixC2 -a all                 # install everything
  $0 -o /opt/AdaptixC2 -a agents --pull
  $0 -o ../AdaptixC2 -g /usr/local/go1.25.4/bin/go
  $0 -o ../AdaptixC2 -a clean
EOF
    exit 1
}

# ── Parse arguments ─────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
    case $1 in
        -o|--ax)     ADAPTIX_DIR="$(realpath "$2" 2>/dev/null || echo "$2")"; shift 2 ;;
        -a|--action) ACTION="$2"; shift 2 ;;
        -i|--input)  INPUT_DIR="$(realpath "$2" 2>/dev/null || echo "$2")"; shift 2 ;;
        -g|--go)     GO_BIN="$2"; shift 2 ;;
        --pull)      PULL_CHANGES=true; shift ;;
        *)           error_exit "Unknown parameter: $1" ;;
    esac
done

[[ -z "$ADAPTIX_DIR" ]] && usage
[[ -z "$INPUT_DIR" ]]   && INPUT_DIR="$SCRIPT_DIR"

[[ -d "$INPUT_DIR" ]] || error_exit "Input directory does not exist: $INPUT_DIR"

discover_extenders

# ── Interactive selector ────────────────────────────────────────────────────
# Shows a toggleable checklist: green = new, blue = already installed.
# Space toggles, Enter confirms.  Needs EXTENDERS_DIR, called after dir setup.

interactive_select() {
    local dest_dir="$1"
    local -a all_names=() all_types=()

    # Merge all discovered into flat arrays preserving type
    for n in "${AGENTS[@]}"; do    all_names+=("$n"); all_types+=("agent"); done
    for n in "${LISTENERS[@]}"; do all_names+=("$n"); all_types+=("listener"); done
    for n in "${SERVICES[@]}"; do  all_names+=("$n"); all_types+=("service"); done

    local count=${#all_names[@]}
    (( count == 0 )) && error_exit "No extenders found in $INPUT_DIR"

    # Build selected[] (all on) and color per entry
    local -a selected=() colors=()
    for (( i=0; i<count; i++ )); do
        selected+=( 1 )
        if [[ -d "$dest_dir/${all_names[$i]}" ]]; then
            colors+=( "$BLUE" )    # already installed
        else
            colors+=( "$GREEN" )   # new
        fi
    done

    local cursor=0 _menu_drawn=0

    # Save terminal, hide cursor
    local saved_stty
    saved_stty="$(stty -g)"
    stty -echo -icanon min 1 time 0
    tput civis 2>/dev/null

    _render_menu() {
        (( _menu_drawn )) && printf '\033[%dA' "$((count + 3))"
        _menu_drawn=1
        printf '\r\033[K'
        echo -e "${BOLD}Select extenders to install ${DIM}(Space=toggle  a=all  n=none  Enter=confirm  q=quit)${NC}"
        printf '\r\033[K\n'
        for (( i=0; i<count; i++ )); do
            printf '\r\033[K'
            local arrow="  "; [[ $i -eq $cursor ]] && arrow="> "
            local check=" "; (( selected[i] )) && check="x"
            local tag=""
            case "${all_types[$i]}" in
                agent)    tag="${DIM}agent${NC}" ;;
                listener) tag="${DIM}listener${NC}" ;;
                service)  tag="${DIM}service${NC}" ;;
            esac
            echo -e "${arrow}[${check}] ${colors[$i]}${all_names[$i]}${NC}  ${tag}"
        done
        printf '\r\033[K\n'
    }

    _render_menu

    while true; do
        local key
        IFS= read -rsN1 key
        case "$key" in
            $'\x1b')
                IFS= read -rsN2 rest
                case "$rest" in
                    '[A') (( cursor > 0 )) && (( cursor-- )) ;;
                    '[B') (( cursor < count-1 )) && (( cursor++ )) ;;
                esac ;;
            ' ')  (( selected[cursor] = !selected[cursor] )) ;;
            a|A)  for (( i=0; i<count; i++ )); do selected[$i]=1; done ;;
            n|N)  for (( i=0; i<count; i++ )); do selected[$i]=0; done ;;
            ''|$'\n') break ;;
            q|Q)  stty "$saved_stty"; tput cnorm 2>/dev/null
                  echo -e "${YELLOW}Aborted.${NC}"; exit 0 ;;
        esac
        _render_menu
    done

    stty "$saved_stty"; tput cnorm 2>/dev/null

    # Rebuild arrays from selection
    AGENTS=(); LISTENERS=(); SERVICES=()
    for (( i=0; i<count; i++ )); do
        (( selected[i] )) || continue
        case "${all_types[$i]}" in
            agent)    AGENTS+=("${all_names[$i]}") ;;
            listener) LISTENERS+=("${all_names[$i]}") ;;
            service)  SERVICES+=("${all_names[$i]}") ;;
        esac
    done
    local total=$(( ${#AGENTS[@]} + ${#LISTENERS[@]} + ${#SERVICES[@]} ))
    (( total == 0 )) && { echo -e "${YELLOW}Nothing selected.${NC}"; exit 0; }
    echo -e "${GREEN}Selected ${total} extender(s)${NC}"
}

# ── Validate ────────────────────────────────────────────────────────────────
[[ -d "$ADAPTIX_DIR" ]] || error_exit "Directory does not exist: $ADAPTIX_DIR"
[[ -d "$ADAPTIX_DIR/AdaptixServer" ]] || error_exit "AdaptixServer not found in: $ADAPTIX_DIR"

find_server_binary() {
    local candidate
    for candidate in \
        "$ADAPTIX_DIR/dist/adaptixserver" \
        "$ADAPTIX_DIR/dist/AdaptixServer" \
        "$ADAPTIX_DIR/AdaptixServer/AdaptixServer" \
        "$ADAPTIX_DIR/adaptixserver" \
        "$ADAPTIX_DIR/AdaptixServer"; do
        [[ -f "$candidate" ]] && { echo "$candidate"; return 0; }
    done
    return 1
}

detect_plugin_build_goexperiment() {
    local name mk exp detected=""
    for name in "${AGENTS[@]}" "${LISTENERS[@]}" "${SERVICES[@]}"; do
        mk="$INPUT_DIR/$name/Makefile"
        [[ -f "$mk" ]] || continue
        exp="$(grep -E 'GOEXPERIMENT=[^[:space:]]+.*build' "$mk" | sed -E 's/.*GOEXPERIMENT=([^[:space:]]+).*/\1/' | head -1)"
        [[ -n "$exp" ]] || continue
        if [[ -z "$detected" ]]; then
            detected="$exp"
        elif [[ "$detected" != "$exp" ]]; then
            error_exit "Multiple GOEXPERIMENT values detected across extender Makefiles ($detected vs $exp)."
        fi
    done
    echo "$detected"
}

# ── Detect Go binary ────────────────────────────────────────────────────────
detect_go_bin() {
    # 1. Explicit -g flag → use as-is
    if [[ -n "$GO_BIN" ]]; then
        [[ -x "$GO_BIN" ]] || error_exit "Go binary not found or not executable: $GO_BIN"
        info_msg "Using Go binary: $GO_BIN ($($GO_BIN version 2>/dev/null | head -1))"
        return
    fi

    # 2. Auto-detect from AdaptixServer/Makefile (look for GO ?= or GO =)
    local axs_mk="$ADAPTIX_DIR/AdaptixServer/Makefile"
    if [[ -f "$axs_mk" ]]; then
        local detected
        detected="$(grep -m1 '^GO\s*[?:]*=' "$axs_mk" | sed 's/^GO[^=]*=\s*//' | tr -d ' \t')"
        if [[ -n "$detected" && "$detected" != "go" && -x "$detected" ]]; then
            GO_BIN="$detected"
            info_msg "Auto-detected Go binary from AdaptixServer/Makefile: $GO_BIN"
            return
        fi
    fi

    # 3. Fall back to go in PATH — warn if version differs from server binary
    GO_BIN="go"
    local path_ver
    path_ver="$(go version 2>/dev/null | awk '{print $3}')"
    local server_bin
    server_bin="$(find_server_binary 2>/dev/null || true)"
    if [[ -f "$server_bin" ]]; then
        local srv_ver
        srv_ver="$(go version -m "$server_bin" 2>/dev/null | awk 'NR==1 {print $2; exit}')"
        if [[ -n "$srv_ver" && -n "$path_ver" && "$srv_ver" != "$path_ver" ]]; then
            warn_msg "Go version mismatch detected!"
            warn_msg "  Server binary built with: $srv_ver"
            warn_msg "  go in PATH is:            $path_ver"
            warn_msg "  Plugins may fail to load. Use -g to specify the correct Go binary."
        else
            info_msg "Using Go from PATH: $path_ver"
        fi
    else
        info_msg "Using Go from PATH: $path_ver"
    fi
}

verify_go_compatibility() {
    local server_bin
    server_bin="$(find_server_binary 2>/dev/null || true)"

    if [[ ! -f "$server_bin" ]]; then
        warn_msg "Could not inspect a local server binary for plugin compatibility verification."
        warn_msg "If the real server runs on another Linux machine, build plugins there or use -g with the exact Go toolchain used to build that server."
        return
    fi

    local server_meta selected_ver selected_exp selected_goos selected_goarch selected_goamd64
    local server_ver server_exp server_goos server_goarch server_goamd64

    server_meta="$(go version -m "$server_bin" 2>/dev/null)"
    [[ -n "$server_meta" ]] || {
        warn_msg "Failed to read Go metadata from server binary: $server_bin"
        warn_msg "Plugin compatibility could not be verified automatically."
        return
    }

    selected_ver="$("$GO_BIN" version 2>/dev/null | awk '{print $3}')"
    selected_exp="$(detect_plugin_build_goexperiment)"
    [[ -n "$selected_exp" ]] || selected_exp="$("$GO_BIN" env GOEXPERIMENT 2>/dev/null)"
    selected_goos="$("$GO_BIN" env GOOS 2>/dev/null)"
    selected_goarch="$("$GO_BIN" env GOARCH 2>/dev/null)"
    selected_goamd64="$("$GO_BIN" env GOAMD64 2>/dev/null)"

    server_ver="$(printf '%s\n' "$server_meta" | awk 'NR==1 {print $2; exit}')"
    server_exp="$(printf '%s\n' "$server_meta" | awk '$1=="build" && $2 ~ /^GOEXPERIMENT=/ {sub(/^GOEXPERIMENT=/, "", $2); print $2; exit}')"
    server_goos="$(printf '%s\n' "$server_meta" | awk '$1=="build" && $2 ~ /^GOOS=/ {sub(/^GOOS=/, "", $2); print $2; exit}')"
    server_goarch="$(printf '%s\n' "$server_meta" | awk '$1=="build" && $2 ~ /^GOARCH=/ {sub(/^GOARCH=/, "", $2); print $2; exit}')"
    server_goamd64="$(printf '%s\n' "$server_meta" | awk '$1=="build" && $2 ~ /^GOAMD64=/ {sub(/^GOAMD64=/, "", $2); print $2; exit}')"

    if [[ "$selected_ver" != "$server_ver" || "$selected_exp" != "$server_exp" || "$selected_goos" != "$server_goos" || "$selected_goarch" != "$server_goarch" || "$selected_goamd64" != "$server_goamd64" ]]; then
        error_exit "Selected Go toolchain is incompatible with server binary. Server: ver=$server_ver exp=${server_exp:-<none>} os=$server_goos arch=$server_goarch amd64=${server_goamd64:-<default>} ; Selected: ver=$selected_ver exp=${selected_exp:-<none>} os=$selected_goos arch=$selected_goarch amd64=${selected_goamd64:-<default>}. Use -g with the exact Go toolchain used to build the server, or build the plugins on that Linux machine."
    fi

    # Detect server -trimpath to match in plugin builds (ABI must match)
    local server_trimpath_val
    server_trimpath_val="$(printf '%s\n' "$server_meta" | awk '$1=="build" && $2=="-trimpath=true" {print "true"; exit}')"
    if [[ "$server_trimpath_val" == "true" ]]; then
        SERVER_TRIMPATH="-trimpath"
    else
        SERVER_TRIMPATH=""
        warn_msg "Server was built without -trimpath; plugins will be built without it too for ABI compatibility."
    fi

    info_msg "Verified Go compatibility against server binary: $server_bin"
}

detect_go_bin
verify_go_compatibility

if $PULL_CHANGES; then
    cd "$SCRIPT_DIR" || true
    git pull || warn_msg "git pull failed (continuing anyway)"
    info_msg "Pulled latest changes"
fi

# ── Directories ─────────────────────────────────────────────────────────────
EXTENDERS_DIR="$ADAPTIX_DIR/AdaptixServer/extenders"
DIST_DIR="$ADAPTIX_DIR/dist/extenders"
mkdir -p "$EXTENDERS_DIR" || error_exit "Failed to create extenders directory"
if [[ -d "$ADAPTIX_DIR/dist" ]]; then
    USE_DIST=true
    mkdir -p "$DIST_DIR" || error_exit "Failed to create dist/extenders directory"
else
    USE_DIST=false
fi

# Show interactive selector when no -a flag was given
if [[ -z "$ACTION" ]]; then
    interactive_select "$EXTENDERS_DIR"
    ACTION="all"
fi

# ════════════════════════════════════════════════════════════════════════════
# Helper functions
# ════════════════════════════════════════════════════════════════════════════

clean_extender() {
    local name="$1"
    rm -rf "$EXTENDERS_DIR/$name"
    $USE_DIST && rm -rf "$DIST_DIR/$name"
}

copy_extender() {
    local name="$1"
    [[ -d "$INPUT_DIR/$name" ]] || { warn_msg "$name: not found (skipping)"; return 1; }
    cp -r "$INPUT_DIR/$name" "$EXTENDERS_DIR/" || error_exit "Failed to copy $name"
}

go_work_use() {
    local name="$1"
    cd "$ADAPTIX_DIR/AdaptixServer" || error_exit "Could not enter AdaptixServer"
    if [[ -d "extenders/$name" ]]; then
        "$GO_BIN" work use "extenders/$name" || error_exit "go work use failed for $name"
    fi
}

go_work_sync_all() {
    cd "$ADAPTIX_DIR/AdaptixServer" || error_exit "Could not enter AdaptixServer"
    "$GO_BIN" work sync || error_exit "go work sync failed"
}

build_plugin() {
    local name="$1"
    cd "$ADAPTIX_DIR/AdaptixServer" || error_exit "Could not enter AdaptixServer"
    [[ -f "extenders/$name/Makefile" ]] || { warn_msg "$name: no Makefile (skipping build)"; return 1; }

    # Patch Makefile to match server -trimpath setting (ABI must match)
    if [[ -z "$SERVER_TRIMPATH" ]]; then
        sed -i 's/-trimpath //g' "extenders/$name/Makefile"
    fi

    step_msg "Building ${BOLD}$name${NC}..."
    local build_log
    build_log="$(make --no-print-directory -C "extenders/$name" plugin GO="$GO_BIN" TRIMPATH="$SERVER_TRIMPATH" 2>&1)" || \
        { echo "$build_log"; error_exit "Failed to build $name"; }
    # Show only unexpected output (errors/warnings), suppress known noise
    echo "$build_log" | grep -vE '^\[\*\]|^cd |^\[\+\]|^$' || true
    info_msg "$name ${DIM}✓${NC}"
}

copy_dist() {
    local name="$1"
    $USE_DIST || return 0
    mkdir -p "$DIST_DIR/$name" || error_exit "Failed to create dist dir for $name"

    local ext_dir="$EXTENDERS_DIR/$name"

    # Copy .so plugin
    for so_file in "$ext_dir"/*.so; do
        [[ -f "$so_file" ]] && cp "$so_file" "$DIST_DIR/$name/"
    done

    # Copy config + axs
    [[ -f "$ext_dir/config.yaml" ]]    && cp "$ext_dir/config.yaml"    "$DIST_DIR/$name/"
    [[ -f "$ext_dir/ax_config.axs" ]]  && cp "$ext_dir/ax_config.axs"  "$DIST_DIR/$name/"
}

# ════════════════════════════════════════════════════════════════════════════
# Action blocks
# ════════════════════════════════════════════════════════════════════════════

install_group() {
    local label="$1"; shift
    local -a names=("$@")
    (( ${#names[@]} == 0 )) && return 0

    echo ""
    step_msg "${BOLD}${label}${NC} ${DIM}(${#names[@]})${NC}"

    for name in "${names[@]}"; do
        clean_extender "$name"
        copy_extender "$name" || continue
        go_work_use "$name"
    done
    go_work_sync_all
    for name in "${names[@]}"; do
        [[ -d "$EXTENDERS_DIR/$name" ]] || continue
        build_plugin "$name"
        copy_dist "$name"
    done
}

install_agents()    { install_group "Agents"    "${AGENTS[@]}"; }
install_listeners() { install_group "Listeners" "${LISTENERS[@]}"; }
install_services()  { install_group "Services"  "${SERVICES[@]}"; }

clean_all() {
    step_msg "Cleaning all discovered extenders..."
    for name in "${AGENTS[@]}" "${LISTENERS[@]}" "${SERVICES[@]}"; do
        clean_extender "$name"
    done
    info_msg "All extenders removed"
}

# ════════════════════════════════════════════════════════════════════════════
# Dispatch
# ════════════════════════════════════════════════════════════════════════════

case $ACTION in
    all)
        install_agents
        install_listeners
        install_services
        ;;
    agents)    install_agents ;;
    listeners) install_listeners ;;
    services)  install_services ;;
    clean)     clean_all ;;
    *)         error_exit "Unknown action: $ACTION" ;;
esac

# ── Summary ─────────────────────────────────────────────────────────────────
echo ""
local_total=$(( ${#AGENTS[@]} + ${#LISTENERS[@]} + ${#SERVICES[@]} ))
echo -e "${GREEN}${BOLD}Done.${NC} ${local_total} extender(s) → ${ADAPTIX_DIR}"
[[ ${#AGENTS[@]} -gt 0 ]]    && echo -e "  Agents:    ${AGENTS[*]}"
[[ ${#LISTENERS[@]} -gt 0 ]] && echo -e "  Listeners: ${LISTENERS[*]}"
[[ ${#SERVICES[@]} -gt 0 ]]  && echo -e "  Services:  ${SERVICES[*]}"