#!/bin/bash

# ============================================================================
# setup.sh — Deploy extenders into an AdaptixC2 installation
#
# Auto-discovers agents, listeners, and services from subdirectories by
# reading config.yaml → extender_type.
#
# Supports three target modes (auto-detected from -o):
#
#   source       AdaptixServer/ present, no dist/
#                → copy sources to AdaptixServer/extenders/, build plugins
#
#   source+dist  AdaptixServer/ present AND dist/ exists
#                → same as source, PLUS deploy runtime files to dist/extenders/
#
#   compiled     adaptixserver binary + extenders/ (no AdaptixServer/)
#                → no build, deploy only runtime files to extenders/
# ============================================================================

set -euo pipefail

# ════════════════════════════════════════════════════════════════════════════
# Constants & messaging
# ════════════════════════════════════════════════════════════════════════════

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BLUE='\033[0;34m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

error_exit() { echo -e "${RED}[ERROR]${NC} $1" >&2; exit 1; }
info_msg()   { echo -e "${GREEN}[+]${NC} $1"; }
warn_msg()   { echo -e "${YELLOW}[!]${NC} $1"; }
step_msg()   { echo -e "${CYAN}[*]${NC} $1"; }

# ════════════════════════════════════════════════════════════════════════════
# Defaults
# ════════════════════════════════════════════════════════════════════════════

ADAPTIX_DIR=""
INPUT_DIR=""
GO_BIN=""
PULL_CHANGES=false
ACTION=""
PROFILE_NAME=""
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SERVER_TRIMPATH="-trimpath"

AGENTS=()
LISTENERS=()
SERVICES=()

# Derived paths — set after argument validation
EXTENDERS_DIR=""
DIST_DIR=""
GO_WORK_FILE=""
USE_DIST=false

# Target mode: "source" (build from Go source) or "compiled" (deploy-only)
TARGET_MODE=""

# ════════════════════════════════════════════════════════════════════════════
# Usage
# ════════════════════════════════════════════════════════════════════════════

usage() {
    cat <<EOF
Usage: $0 -o <AdaptixC2_dir> [-a <action>] [-i <input_dir>] [-g <go_binary>] [-p <profile>] [--pull]

Required:
  -o, --ax <dir>       Path to AdaptixC2 installation or compiled server folder

                       Accepted layouts:
                         Source checkout — contains AdaptixServer/ (optionally dist/)
                         Compiled server — contains adaptixserver binary + extenders/

Optional:
  -a, --action <act>   Action to perform (omit for interactive selector)
  -i, --input <dir>    Folder containing extender subdirectories
                       (default: same directory as this script)
  -g, --go <path>      Go binary for building plugins (source mode only;
                       auto-detected from AdaptixServer/Makefile if omitted)
  -p, --profile <file> Update a profile YAML to register all installed extenders
                       (resolved relative to the server root; e.g. "profile.yaml")
  --pull               git pull before installation

Actions:
  all        Install all discovered extenders
  agents     Agents only
  listeners  Listeners only
  services   Services only
  clean      Remove all discovered extenders from AdaptixC2
  (none)     Interactive selector — pick which extenders to install

Examples:
  $0 -o ../AdaptixC2                               # interactive selector (source)
  $0 -o /opt/AdaptixC2 -a all -p profile.yaml       # install everything + update profile
  $0 -o /opt/AdaptixC2 -a agents --pull
  $0 -o ../AdaptixC2 -g /usr/local/go1.25.4/bin/go
  $0 -o /opt/server -a all -p profile.yaml           # deploy to compiled server
  $0 -o ../AdaptixC2 -a clean -p profile.yaml        # clean + update profile
EOF
    exit 1
}

# ════════════════════════════════════════════════════════════════════════════
# Utility functions
# ════════════════════════════════════════════════════════════════════════════

# Read extender_type from a config.yaml file.  Returns: agent|listener|service
# Usage: etype="$(read_extender_type "$path/config.yaml")"
read_extender_type() {
    grep -m1 '^extender_type:' "$1" 2>/dev/null \
        | sed 's/extender_type:[[:space:]]*//;s/"//g;s/[[:space:]]*$//'
}

# Resolve profile filename to absolute path.
# Source mode → AdaptixServer/<name>; compiled mode → <root>/<name>.
# Appends .yaml if no extension is present.
resolve_profile_path() {
    local name="$1"
    [[ "$name" != *.yaml && "$name" != *.yml ]] && name="${name}.yaml"
    if [[ "$TARGET_MODE" == "compiled" ]]; then
        printf '%s\n' "$ADAPTIX_DIR/$name"
    else
        printf '%s\n' "$ADAPTIX_DIR/AdaptixServer/$name"
    fi
}

# ── Extender discovery ──────────────────────────────────────────────────────

discover_extenders() {
    AGENTS=(); LISTENERS=(); SERVICES=()
    for dir in "$INPUT_DIR"/*/; do
        local name; name="$(basename "$dir")"
        [[ -f "$dir/config.yaml" ]] || continue
        local etype; etype="$(read_extender_type "$dir/config.yaml")"
        case "$etype" in
            agent)    AGENTS+=("$name") ;;
            listener) LISTENERS+=("$name") ;;
            service)  SERVICES+=("$name") ;;
            *)        warn_msg "Unknown extender_type '$etype' in $name/config.yaml (skipping)" ;;
        esac
    done
}

# ── Server binary & Go toolchain ────────────────────────────────────────────

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
        exp="$(grep -E 'GOEXPERIMENT=[^[:space:]]+.*build' "$mk" \
               | sed -E 's/.*GOEXPERIMENT=([^[:space:]]+).*/\1/' | head -1)"
        [[ -n "$exp" ]] || continue
        if [[ -z "$detected" ]]; then
            detected="$exp"
        elif [[ "$detected" != "$exp" ]]; then
            error_exit "Multiple GOEXPERIMENT values across Makefiles ($detected vs $exp)."
        fi
    done
    echo "$detected"
}

detect_go_bin() {
    # 1. Explicit -g flag
    if [[ -n "$GO_BIN" ]]; then
        [[ -x "$GO_BIN" ]] || error_exit "Go binary not found or not executable: $GO_BIN"
        info_msg "Using Go binary: $GO_BIN ($("$GO_BIN" version 2>/dev/null | head -1))"
        return
    fi

    # 2. Auto-detect from AdaptixServer/Makefile
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

    # 3. Fall back to go in PATH
    GO_BIN="go"
    local path_ver; path_ver="$(go version 2>/dev/null | awk '{print $3}')"
    local server_bin; server_bin="$(find_server_binary 2>/dev/null || true)"
    if [[ -f "$server_bin" ]]; then
        local srv_ver
        srv_ver="$(go version -m "$server_bin" 2>/dev/null | awk 'NR==1 {print $2; exit}')"
        if [[ -n "$srv_ver" && -n "$path_ver" && "$srv_ver" != "$path_ver" ]]; then
            warn_msg "Go version mismatch!"
            warn_msg "  Server binary built with: $srv_ver"
            warn_msg "  go in PATH is:            $path_ver"
            warn_msg "  Plugins may fail to load.  Use -g to specify the correct Go binary."
        else
            info_msg "Using Go from PATH: $path_ver"
        fi
    else
        info_msg "Using Go from PATH: $path_ver"
    fi
}

verify_go_compatibility() {
    local server_bin; server_bin="$(find_server_binary 2>/dev/null || true)"

    if [[ ! -f "$server_bin" ]]; then
        warn_msg "No local server binary found — cannot verify plugin compatibility."
        warn_msg "Use -g with the toolchain that built the server, or build on that machine."
        return
    fi

    local server_meta; server_meta="$(go version -m "$server_bin" 2>/dev/null)"
    if [[ -z "$server_meta" ]]; then
        warn_msg "Failed to read Go metadata from: $server_bin"
        return
    fi

    # Extract selected toolchain properties
    local selected_ver selected_exp selected_goos selected_goarch selected_goamd64
    selected_ver="$("$GO_BIN" version 2>/dev/null | awk '{print $3}')"
    selected_exp="$(detect_plugin_build_goexperiment)"
    [[ -n "$selected_exp" ]] || selected_exp="$("$GO_BIN" env GOEXPERIMENT 2>/dev/null)"
    selected_goos="$("$GO_BIN" env GOOS 2>/dev/null)"
    selected_goarch="$("$GO_BIN" env GOARCH 2>/dev/null)"
    selected_goamd64="$("$GO_BIN" env GOAMD64 2>/dev/null)"

    # Helper to pull a build setting from metadata
    _meta_val() { printf '%s\n' "$server_meta" | awk -v k="$1=" '$1=="build" && $2~"^"k {sub(k,"",$2); print $2; exit}'; }

    local server_ver server_exp server_goos server_goarch server_goamd64
    server_ver="$(printf '%s\n' "$server_meta" | awk 'NR==1 {print $2; exit}')"
    server_exp="$(_meta_val GOEXPERIMENT)"
    server_goos="$(_meta_val GOOS)"
    server_goarch="$(_meta_val GOARCH)"
    server_goamd64="$(_meta_val GOAMD64)"

    if [[ "$selected_ver"  != "$server_ver"  || "$selected_exp"    != "$server_exp"    ||
          "$selected_goos" != "$server_goos" || "$selected_goarch" != "$server_goarch" ||
          "$selected_goamd64" != "$server_goamd64" ]]; then
        error_exit "Go toolchain incompatible with server binary.
  Server:   ver=$server_ver exp=${server_exp:-<none>} os=$server_goos arch=$server_goarch amd64=${server_goamd64:-<default>}
  Selected: ver=$selected_ver exp=${selected_exp:-<none>} os=$selected_goos arch=$selected_goarch amd64=${selected_goamd64:-<default>}
Use -g with the exact Go toolchain that built the server."
    fi

    # Match server -trimpath setting (ABI must match)
    local srv_tp; srv_tp="$(printf '%s\n' "$server_meta" \
        | awk '$1=="build" && $2=="-trimpath=true" {print "true"; exit}')"
    if [[ "$srv_tp" == "true" ]]; then
        SERVER_TRIMPATH="-trimpath"
    else
        SERVER_TRIMPATH=""
        warn_msg "Server built without -trimpath; plugins will match for ABI compatibility."
    fi

    info_msg "Verified Go compatibility against server binary: $server_bin"
}

# ── go.work management ──────────────────────────────────────────────────────

go_work_exists() { [[ -f "$GO_WORK_FILE" ]]; }

normalize_go_work_use() {
    local e="$1"
    e="${e//\"/}"; e="${e%/}"
    [[ "$e" == ./* ]] && e="${e#./}"
    printf '%s\n' "$e"
}

go_work_list_uses() {
    go_work_exists || return 0
    awk '
        /^[[:space:]]*\/\// { next }
        /^[[:space:]]*use[[:space:]]+\(/ { inuse=1; next }
        /^[[:space:]]*use[[:space:]]+/ {
            line=$0
            sub(/^[[:space:]]*use[[:space:]]+/, "", line)
            sub(/[[:space:]]*\/\/.*$/, "", line)
            gsub(/"/, "", line); gsub(/^[[:space:]]+|[[:space:]]+$/, "", line)
            if (line != "" && line != "(") print line
            next
        }
        inuse && /^[[:space:]]*\)/ { inuse=0; next }
        inuse {
            line=$0
            sub(/[[:space:]]*\/\/.*$/, "", line)
            gsub(/"/, "", line); gsub(/^[[:space:]]+|[[:space:]]+$/, "", line)
            if (line != "") print line
        }
    ' "$GO_WORK_FILE"
}

find_go_work_use_entry() {
    local normalized_target; normalized_target="$(normalize_go_work_use "$1")"
    local entry
    while IFS= read -r entry; do
        [[ "$(normalize_go_work_use "$entry")" == "$normalized_target" ]] \
            && { printf '%s\n' "$entry"; return 0; }
    done < <(go_work_list_uses)
    return 1
}

drop_go_work_use() {
    go_work_exists || return 0
    local existing_entry; existing_entry="$(find_go_work_use_entry "$1")" || return 0
    local tmp_file; tmp_file="$(mktemp "${GO_WORK_FILE}.XXXXXX")" \
        || error_exit "mktemp failed for go.work"

    awk -v target="$(normalize_go_work_use "$existing_entry")" '
        function norm(p,  v) {
            v=p; gsub(/"/,"",v); gsub(/^[[:space:]]+|[[:space:]]+$/,"",v)
            sub(/\/$/,"",v); sub(/^\.\//,"",v); return v
        }
        function strip_cmt(l,  v) {
            v=l; sub(/[[:space:]]*\/\/.*$/,"",v); gsub(/^[[:space:]]+|[[:space:]]+$/,"",v)
            return v
        }
        /^[[:space:]]*use[[:space:]]+\(/ { inuse=1; print; next }
        inuse && /^[[:space:]]*\)/        { inuse=0; print; next }
        /^[[:space:]]*use[[:space:]]+/ {
            line=$0; sub(/^[[:space:]]*use[[:space:]]+/,"",line)
            if (norm(strip_cmt(line)) == target) next
            print; next
        }
        inuse {
            line=strip_cmt($0)
            if (line != "" && norm(line) == target) next
            print; next
        }
        { print }
    ' "$GO_WORK_FILE" > "$tmp_file" || { rm -f "$tmp_file"; error_exit "Failed to rewrite go.work"; }
    mv "$tmp_file" "$GO_WORK_FILE" || { rm -f "$tmp_file"; error_exit "Failed to update go.work"; }
}

prune_stale_extender_uses() {
    go_work_exists || return 0
    local entry norm_entry fs_path
    while IFS= read -r entry; do
        norm_entry="$(normalize_go_work_use "$entry")"
        [[ "$norm_entry" == extenders/* ]] || continue
        fs_path="$ADAPTIX_DIR/AdaptixServer/$norm_entry"
        if [[ ! -d "$fs_path" || ! -f "$fs_path/go.mod" ]]; then
            warn_msg "Pruning stale go.work entry: $entry"
            drop_go_work_use "$entry"
        fi
    done < <(go_work_list_uses)
}

# ── Extender operations ────────────────────────────────────────────────────

# Copy only runtime files (no Go source) from src to dest.
# Prefers the Makefile-generated dist/ subfolder when present.
# Fallback: copies everything except *.go, go.mod, go.sum, Makefile, dist/.
copy_runtime_files() {
    local src="$1" dest="$2"
    rm -rf "$dest" 2>/dev/null || true
    mkdir -p "$dest"

    # If the Makefile already staged runtime files in dist/, use those
    if [[ -d "$src/dist" ]]; then
        cp -r "$src/dist/"* "$dest/"
        return
    fi

    # Fallback: copy everything except Go build artifacts
    local item base
    for item in "$src"/*; do
        [[ -e "$item" ]] || continue
        base="$(basename "$item")"
        case "$base" in
            *.go|go.mod|go.sum|Makefile|dist) continue ;;
        esac
        cp -r "$item" "$dest/"
    done
}

clean_extender() {
    local name="$1"
    [[ "$TARGET_MODE" == "source" ]] && drop_go_work_use "./extenders/$name"
    rm -rf "${EXTENDERS_DIR:?}/$name"
    $USE_DIST && rm -rf "${DIST_DIR:?}/$name"
}

copy_extender() {
    local name="$1"
    [[ -d "$INPUT_DIR/$name" ]] || { warn_msg "$name: not found (skipping)"; return 1; }
    rm -rf "${EXTENDERS_DIR:?}/$name" 2>/dev/null || true
    cp -r "$INPUT_DIR/$name" "$EXTENDERS_DIR/" || error_exit "Failed to copy $name"
}

# Deploy runtime files for compiled-mode targets.
# Builds from source in a temp directory if no pre-built .so is found.
deploy_extender_compiled() {
    local name="$1"
    [[ -d "$INPUT_DIR/$name" ]] || { warn_msg "$name: not found (skipping)"; return 1; }

    local src="$INPUT_DIR/$name"

    # Check for pre-built .so
    local has_so=false
    if [[ -d "$src/dist" ]]; then
        ls "$src/dist/"*.so &>/dev/null && has_so=true
    else
        ls "$src/"*.so &>/dev/null && has_so=true
    fi

    if $has_so; then
        # Pre-built: just deploy runtime files
        rm -rf "${EXTENDERS_DIR:?}/$name"
        copy_runtime_files "$src" "$EXTENDERS_DIR/$name"
    elif [[ -f "$src/Makefile" && -f "$src/go.mod" ]]; then
        # Source: build in a temp directory, then deploy runtime files
        local tmpdir
        tmpdir="$(mktemp -d)" || error_exit "Failed to create temp build directory"

        cp -r "$src" "$tmpdir/$name"

        # Patch -trimpath if needed
        [[ -z "$SERVER_TRIMPATH" ]] && sed -i 's/-trimpath //g' "$tmpdir/$name/Makefile"

        step_msg "Building ${BOLD}$name${NC}..."
        local build_log
        build_log="$(GOWORK=off make --no-print-directory -C "$tmpdir/$name" plugin \
            GO="$GO_BIN" TRIMPATH="$SERVER_TRIMPATH" 2>&1)" || \
            { echo "$build_log"; rm -rf "$tmpdir"; error_exit "Failed to build $name"; }
        echo "$build_log" | grep -vE '^\[\*\]|^cd |^\[\+\]|^$' || true

        rm -rf "${EXTENDERS_DIR:?}/$name"
        copy_runtime_files "$tmpdir/$name" "$EXTENDERS_DIR/$name"
        rm -rf "$tmpdir"
    else
        warn_msg "$name: no .so found and no buildable source (skipping)"
        return 1
    fi

    info_msg "$name ${DIM}✓${NC}"
}

go_work_use() {
    cd "$ADAPTIX_DIR/AdaptixServer" || error_exit "Could not enter AdaptixServer"
    [[ -d "extenders/$1" ]] && \
        "$GO_BIN" work use "extenders/$1" || error_exit "go work use failed for $1"
}

go_work_sync_all() {
    cd "$ADAPTIX_DIR/AdaptixServer" || error_exit "Could not enter AdaptixServer"
    "$GO_BIN" work sync || error_exit "go work sync failed"
}

build_plugin() {
    local name="$1"
    cd "$ADAPTIX_DIR/AdaptixServer" || error_exit "Could not enter AdaptixServer"
    [[ -f "extenders/$name/Makefile" ]] || { warn_msg "$name: no Makefile (skipping build)"; return 1; }

    # Patch Makefile to match server -trimpath setting
    [[ -z "$SERVER_TRIMPATH" ]] && sed -i 's/-trimpath //g' "extenders/$name/Makefile"

    step_msg "Building ${BOLD}$name${NC}..."
    local build_log
    build_log="$(make --no-print-directory -C "extenders/$name" plugin \
        GO="$GO_BIN" TRIMPATH="$SERVER_TRIMPATH" 2>&1)" || \
        { echo "$build_log"; error_exit "Failed to build $name"; }
    echo "$build_log" | grep -vE '^\[\*\]|^cd |^\[\+\]|^$' || true
    info_msg "$name ${DIM}✓${NC}"
}

copy_dist() {
    $USE_DIST || return 0
    local name="$1"
    rm -rf "${DIST_DIR:?}/$name"
    copy_runtime_files "$EXTENDERS_DIR/$name" "$DIST_DIR/$name"
}

# ── Install / clean orchestration ──────────────────────────────────────────

install_group() {
    local label="$1"; shift
    local -a names=("$@")
    (( ${#names[@]} == 0 )) && return 0

    echo ""
    step_msg "${BOLD}${label}${NC} ${DIM}(${#names[@]})${NC}"

    if [[ "$TARGET_MODE" == "compiled" ]]; then
        # Compiled mode: deploy pre-built runtime files only
        for name in "${names[@]}"; do
            deploy_extender_compiled "$name"
        done
    else
        # Source mode: copy, register, build, (optionally) dist
        prune_stale_extender_uses
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
    fi
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

# ── Profile update ──────────────────────────────────────────────────────────

# Rewrite Teamserver.extenders in the profile YAML with every extender
# currently present in $EXTENDERS_DIR.  No-op when -p was not given.
update_profile() {
    [[ -n "$PROFILE_NAME" ]] || return 0

    local profile_path; profile_path="$(resolve_profile_path "$PROFILE_NAME")"
    [[ -f "$profile_path" ]] || error_exit "Profile not found: $profile_path"

    step_msg "Updating profile ${BOLD}$(basename "$profile_path")${NC}..."

    # Collect entries grouped by type, then sorted alphabetically within each group
    local -a listeners_e=() agents_e=() services_e=() others_e=()
    local dir name etype
    for dir in "$EXTENDERS_DIR"/*/; do
        name="$(basename "$dir")"
        [[ -f "$dir/config.yaml" ]] || continue
        etype="$(read_extender_type "$dir/config.yaml")"
        local entry="extenders/$name/config.yaml"
        case "$etype" in
            listener) listeners_e+=("$entry") ;;
            agent)    agents_e+=("$entry") ;;
            service)  services_e+=("$entry") ;;
            *)        others_e+=("$entry") ;;
        esac
    done

    local -a sorted=()
    local arr
    for arr in listeners_e agents_e services_e others_e; do
        local -n ref="$arr"
        while IFS= read -r line; do
            [[ -n "$line" ]] && sorted+=("$line")
        done < <(printf '%s\n' "${ref[@]}" 2>/dev/null | sort)
    done

    # Build YAML block
    local new_block=""
    for entry in "${sorted[@]}"; do
        new_block+="    - \"$entry\""$'\n'
    done

    # Rewrite in-place via awk
    local tmp_file; tmp_file="$(mktemp "${profile_path}.XXXXXX")" \
        || error_exit "mktemp failed for profile update"

    awk -v new_entries="$new_block" '
        /^[[:space:]]+extenders:[[:space:]]*$/ {
            print
            printf "%s", new_entries
            while ((getline line) > 0) {
                if (line ~ /^[[:space:]]+- /) continue
                print line; break
            }
            next
        }
        { print }
    ' "$profile_path" > "$tmp_file" || { rm -f "$tmp_file"; error_exit "Failed to rewrite profile"; }
    mv "$tmp_file" "$profile_path"  || { rm -f "$tmp_file"; error_exit "Failed to update profile"; }

    info_msg "Profile updated: $(basename "$profile_path") (${#sorted[@]} extender(s))"
    for entry in "${sorted[@]}"; do
        echo -e "    ${DIM}$entry${NC}"
    done
}

# ── Interactive selector ────────────────────────────────────────────────────
# Returns 0 on confirm (even if nothing selected), 1 on abort (q).
# Modifies AGENTS/LISTENERS/SERVICES in the caller.

interactive_select() {
    local dest_dir="$1"
    local -a all_names=() all_types=()

    for n in "${AGENTS[@]}";    do all_names+=("$n"); all_types+=("agent");    done
    for n in "${LISTENERS[@]}"; do all_names+=("$n"); all_types+=("listener"); done
    for n in "${SERVICES[@]}";  do all_names+=("$n"); all_types+=("service");  done

    local count=${#all_names[@]}
    (( count == 0 )) && error_exit "No extenders found in $INPUT_DIR"

    local -a selected=() colors=()
    for (( i=0; i<count; i++ )); do
        selected+=( 1 )
        if [[ -d "$dest_dir/${all_names[$i]}" ]]; then
            colors+=("$BLUE")    # already installed
        else
            colors+=("$GREEN")   # new
        fi
    done

    local cursor=0 _menu_drawn=0

    # Save/restore terminal state — also via trap so Ctrl-C doesn't break it
    local saved_stty; saved_stty="$(stty -g)"
    _restore_tty() { stty "$saved_stty" 2>/dev/null; tput cnorm 2>/dev/null; }
    trap _restore_tty EXIT
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
            echo -e "${arrow}[${check}] ${colors[$i]}${all_names[$i]}${NC}  ${DIM}${all_types[$i]}${NC}"
        done
        printf '\r\033[K\n'
    }

    _render_menu
    local aborted=false

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
            q|Q)  aborted=true; break ;;
        esac
        _render_menu
    done

    _restore_tty
    trap - EXIT   # remove our EXIT trap so it doesn't fire again later

    if $aborted; then
        echo -e "${YELLOW}Aborted.${NC}"
        return 1
    fi

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
    if (( total == 0 )); then
        echo -e "${YELLOW}Nothing selected for install.${NC}"
    else
        echo -e "${GREEN}Selected ${total} extender(s)${NC}"
    fi
    return 0
}

# ── Summary ─────────────────────────────────────────────────────────────────

print_summary() {
    echo ""
    local total=$(( ${#AGENTS[@]} + ${#LISTENERS[@]} + ${#SERVICES[@]} ))
    echo -e "${GREEN}${BOLD}Done.${NC} ${total} extender(s) → ${ADAPTIX_DIR}"
    [[ ${#AGENTS[@]} -gt 0 ]]    && echo -e "  Agents:    ${AGENTS[*]}"
    [[ ${#LISTENERS[@]} -gt 0 ]] && echo -e "  Listeners: ${LISTENERS[*]}"
    [[ ${#SERVICES[@]} -gt 0 ]]  && echo -e "  Services:  ${SERVICES[*]}"
}

# ════════════════════════════════════════════════════════════════════════════
# Main execution flow
# ════════════════════════════════════════════════════════════════════════════

# ── 1. Parse arguments ──────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
    case $1 in
        -o|--ax)      ADAPTIX_DIR="$(realpath "$2" 2>/dev/null || echo "$2")"; shift 2 ;;
        -a|--action)  ACTION="$2"; shift 2 ;;
        -i|--input)   INPUT_DIR="$(realpath "$2" 2>/dev/null || echo "$2")"; shift 2 ;;
        -g|--go)      GO_BIN="$2"; shift 2 ;;
        -p|--profile) PROFILE_NAME="$2"; shift 2 ;;
        --pull)       PULL_CHANGES=true; shift ;;
        *)            error_exit "Unknown parameter: $1" ;;
    esac
done

# ── 2. Detect target mode ──────────────────────────────────────────────────
[[ -n "$ADAPTIX_DIR" ]]  || usage
[[ -d "$ADAPTIX_DIR" ]]  || error_exit "Directory does not exist: $ADAPTIX_DIR"

if [[ -d "$ADAPTIX_DIR/AdaptixServer" ]]; then
    # Source checkout (may or may not have dist/)
    TARGET_MODE="source"
    info_msg "Target mode: ${BOLD}source${NC} (AdaptixServer/ detected)"
elif [[ -f "$ADAPTIX_DIR/adaptixserver" || -d "$ADAPTIX_DIR/extenders" ]]; then
    # Compiled server directory
    TARGET_MODE="compiled"
    info_msg "Target mode: ${BOLD}compiled${NC} (standalone server detected)"
else
    error_exit "Cannot detect target layout in: $ADAPTIX_DIR
  Expected one of:
    • Source checkout with AdaptixServer/ directory
    • Compiled server with adaptixserver binary or extenders/ directory"
fi

[[ -z "$INPUT_DIR" ]] && INPUT_DIR="$SCRIPT_DIR"
[[ -d "$INPUT_DIR" ]]  || error_exit "Input directory does not exist: $INPUT_DIR"

# ── 3. Validate profile early (fail fast) ──────────────────────────────────
if [[ -n "$PROFILE_NAME" ]]; then
    _pp="$(resolve_profile_path "$PROFILE_NAME")"
    [[ -f "$_pp" ]] || error_exit "Profile not found: $_pp"
    unset _pp
fi

# ── 4. Discover extenders in input directory ────────────────────────────────
discover_extenders

# ── 5. Detect Go toolchain & verify compatibility ─────────────────────────
detect_go_bin
verify_go_compatibility

# ── 6. Git pull (optional) ──────────────────────────────────────────────────
if $PULL_CHANGES; then
    cd "$SCRIPT_DIR" || true
    git pull || warn_msg "git pull failed (continuing anyway)"
    info_msg "Pulled latest changes"
fi

# ── 7. Setup directories ───────────────────────────────────────────────────
if [[ "$TARGET_MODE" == "source" ]]; then
    EXTENDERS_DIR="$ADAPTIX_DIR/AdaptixServer/extenders"
    DIST_DIR="$ADAPTIX_DIR/dist/extenders"
    GO_WORK_FILE="$ADAPTIX_DIR/AdaptixServer/go.work"
    mkdir -p "$EXTENDERS_DIR" || error_exit "Failed to create extenders directory"
    if [[ -d "$ADAPTIX_DIR/dist" ]]; then
        USE_DIST=true
        mkdir -p "$DIST_DIR" || error_exit "Failed to create dist/extenders directory"
    fi
else
    # Compiled mode — deploy straight into the server's extenders/
    EXTENDERS_DIR="$ADAPTIX_DIR/extenders"
    mkdir -p "$EXTENDERS_DIR" || error_exit "Failed to create extenders directory"
fi

# ── 8. Interactive selector (when no -a flag) ──────────────────────────────
if [[ -z "$ACTION" ]]; then
    if interactive_select "$EXTENDERS_DIR"; then
        # User confirmed — install whatever was selected (may be empty)
        ACTION="all"
    else
        # User aborted (q) — still update profile if requested, then exit
        update_profile
        exit 0
    fi
fi

# ── 9. Dispatch ─────────────────────────────────────────────────────────────
[[ "$TARGET_MODE" == "source" ]] && prune_stale_extender_uses

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

# ── 10. Profile update ─────────────────────────────────────────────────────
update_profile

# ── 11. Summary ─────────────────────────────────────────────────────────────
print_summary
