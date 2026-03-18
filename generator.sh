#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# AdaptixC2 Unified Template Generator
#
# Root entry-point that dispatches to the appropriate sub-generator:
#   1) Agent     - scaffold a new agent extender
#   2) Listener  - scaffold a new listener extender
#   3) Protocol  - create a new wire-protocol definition
#   4) Crypto    - create/swap the crypto implementation of an existing protocol
#
# Usage:
#   ./generator.sh
#   MODE=agent ./generator.sh
#   MODE=agent OUTPUT_DIR=../my-adaptix/extenders ./generator.sh
#   MODE=agent LANGUAGE=cpp TOOLCHAIN=mingw ./generator.sh
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODE="${MODE:-}"
# OUTPUT_DIR is forwarded as-is to sub-generators via env export
export OUTPUT_DIR="${OUTPUT_DIR:-}"

# ─── UI helpers ────────────────────────────────────────────────────────────────

if [[ -t 1 ]]; then
    C_RESET=$'\033[0m'
    C_CYAN=$'\033[36m'
    C_DCYAN=$'\033[36;2m'
    C_DGREEN=$'\033[32;2m'
    C_GREEN=$'\033[32m'
    C_GRAY=$'\033[90m'
    C_RED=$'\033[31m'
else
    C_RESET=''
    C_CYAN=''
    C_DCYAN=''
    C_DGREEN=''
    C_GREEN=''
    C_GRAY=''
    C_RED=''
fi

render_banner() {
    echo ""
    printf '%s\n' "${C_DGREEN}┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓${C_RESET}"
    printf '%s\n' "${C_DGREEN}┃   █████╗ ██████╗  █████╗ ██████╗ ████████╗██╗██╗  ██╗ ██████╗██████╗   ┃${C_RESET}"
    printf '%s\n' "${C_DGREEN}┃  ██╔══██╗██╔══██╗██╔══██╗██╔══██╗╚══██╔══╝██║╚██╗██╔╝██╔════╝╚════██╗  ┃${C_RESET}"
    printf '%s\n' "${C_GREEN}┃  ███████║██║  ██║███████║██████╔╝   ██║   ██║ ╚███╔╝ ██║      █████╔╝  ┃${C_RESET}"
    printf '%s\n' "${C_GREEN}┃  ██╔══██║██║  ██║██╔══██║██╔═══╝    ██║   ██║ ██╔██╗ ██║     ██╔═══╝   ┃${C_RESET}"
    printf '%s\n' "${C_DGREEN}┃  ██║  ██║██████╔╝██║  ██║██║        ██║   ██║██╔╝ ██╗╚██████╗███████╗  ┃${C_RESET}"
    printf '%s\n' "${C_DGREEN}┃  ╚═╝  ╚═╝╚═════╝ ╚═╝  ╚═╝╚═╝        ╚═╝   ╚═╝╚═╝  ╚═╝ ╚═════╝╚══════╝  ┃${C_RESET}"
    printf '%s\n' "${C_DGREEN}┃                                                                        ┃${C_RESET}"
    printf '%s\n' "${C_GREEN}┃          Template Generator // agents • listeners • services           ┃${C_RESET}"
    printf '%s\n' "${C_DGREEN}┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛${C_RESET}"
    echo ""
}

section() {
    printf '%s\n' "${C_CYAN}[:: $1 ::]${C_RESET}"
}

menu_line() {
    local idx="$1"
    local label="$2"
    local desc="$3"
    local tag="${4:-}"
    printf '  %s[%s]%s %s%s%s' "$C_GREEN" "$idx" "$C_RESET" "$C_CYAN" "$label" "$C_RESET"
    if [[ -n "$tag" ]]; then
        printf '  %s<%s>%s' "$C_DGREEN" "$tag" "$C_RESET"
    fi
    printf '\n'
    printf '    %s%s%s\n' "$C_GRAY" "$desc" "$C_RESET"
}

launch_message() {
    printf '%s\n\n' "${C_CYAN}[>] Launching $1...${C_RESET}"
}

render_banner

# ─── Mode selection ─────────────────────────────────────────────────────────────

if [[ -z "$MODE" ]]; then
    section "Select generation mode"
    echo ""
    menu_line 1 "Generate Agent" "Scaffold a new agent extender" "GO/CPP/RUST"
    menu_line 2 "Generate Listener" "Scaffold a new listener extender" "TRANSPORT"
    menu_line 3 "Generate Service" "Scaffold a new service extender" "HOOKS"
    menu_line 4 "Generate Wrapper" "Scaffold a service with wrapper pipeline mode enabled" "PIPELINE"
    menu_line 5 "Create Protocol" "Create a new wire-protocol definition" "WIRE"
    menu_line 6 "Create Crypto" "Generate or replace the crypto template for a protocol" "CRYPTO"
    menu_line 7 "Delete" "Remove a crypto template, protocol, or generated output" "CLEANUP"
    echo ""
    read -rp "Select option: " choice
    case "$choice" in
        1) MODE="agent" ;;
        2) MODE="listener" ;;
        3) MODE="service" ;;
        4) MODE="wrapper" ;;
        5) MODE="protocol" ;;
        6) MODE="crypto" ;;
        7) MODE="delete" ;;
        *) printf '%s\n' "${C_RED}[-] Invalid choice.${C_RESET}"; exit 1 ;;
    esac
fi

# ─── Dispatch ───────────────────────────────────────────────────────────────────

case "$MODE" in
agent)
    launch_message "Agent Generator"
    exec bash "$SCRIPT_DIR/agent/generator.sh" "$@"
    ;;
listener)
    launch_message "Listener Generator"
    exec bash "$SCRIPT_DIR/listener/generator.sh" "$@"
    ;;
service)
    launch_message "Service Generator"
    exec bash "$SCRIPT_DIR/service/generator.sh" "$@"
    ;;
wrapper)
    launch_message "Service Generator (wrapper mode)"
    WRAPPER=1 exec bash "$SCRIPT_DIR/service/generator.sh" "$@"
    ;;
protocol)
    launch_message "Protocol Generator"
    exec bash "$SCRIPT_DIR/protocols/generator.sh" "$@"
    ;;
crypto)
    launch_message "Crypto Generator"
    exec bash "$SCRIPT_DIR/protocols/crypto_generator.sh" "$@"
    ;;
delete)
    echo ""
    section "Select deletion target"
    echo ""
    menu_line 1 "Crypto template" "Remove a crypto .go.tmpl from _crypto/" "WIPE"
    menu_line 2 "Protocol" "Remove an entire protocol definition" "PURGE"
    menu_line 3 "Generated output" "Remove a generated project from output/" "SCRUB"
    echo ""
    read -rp "Select option: " del_choice

    case "$del_choice" in
    1)
        # ── Delete crypto template ──
        CRYPTO_DIR="$SCRIPT_DIR/protocols/_crypto"
        items=()
        if [[ -d "$CRYPTO_DIR" ]]; then
            for f in "$CRYPTO_DIR"/*.go.tmpl; do
                [[ -f "$f" ]] || continue
                items+=("$f")
            done
        fi
        if [[ ${#items[@]} -eq 0 ]]; then
            echo "[-] No crypto templates found."; exit 1
        fi
        echo ""
        section "Available crypto templates"
        for i in "${!items[@]}"; do
            key="$(basename "${items[$i]}" .go.tmpl)"
            printf '  %s[%s]%s %s%s%s\n' "$C_GREEN" "$((i+1))" "$C_RESET" "$C_CYAN" "$key" "$C_RESET"
        done
        echo ""
        read -rp "Select crypto to delete: " pick
        idx=$((pick - 1))
        if [[ $idx -lt 0 || $idx -ge ${#items[@]} ]]; then
            echo "[-] Invalid choice."; exit 1
        fi
        target="${items[$idx]}"
        target_name="$(basename "$target" .go.tmpl)"
        read -rp "Delete crypto '$target_name'? [y/N]: " confirm
        if [[ "$confirm" != "y" ]]; then echo "Cancelled."; exit 0; fi
        rm -f "$target"
        echo ""
        echo "[+] Deleted crypto template: _crypto/$(basename "$target")"
        echo ""
        ;;
    2)
        # ── Delete protocol ──
        PROTO_BASE="$SCRIPT_DIR/protocols"
        items=()
        for d in "$PROTO_BASE"/*/; do
            [[ -d "$d" ]] || continue
            name="$(basename "$d")"
            [[ "$name" == "_scaffold" || "$name" == "_crypto" ]] && continue
            [[ -f "$d/meta.yaml" ]] || continue
            items+=("$d")
        done
        if [[ ${#items[@]} -eq 0 ]]; then
            echo "[-] No deletable protocols found."; exit 1
        fi
        echo ""
        section "Available protocols"
        for i in "${!items[@]}"; do
            printf '  %s[%s]%s %s%s%s\n' "$C_GREEN" "$((i+1))" "$C_RESET" "$C_CYAN" "$(basename "${items[$i]}")" "$C_RESET"
        done
        echo ""
        read -rp "Select protocol to delete: " pick
        idx=$((pick - 1))
        if [[ $idx -lt 0 || $idx -ge ${#items[@]} ]]; then
            echo "[-] Invalid choice."; exit 1
        fi
        target="${items[$idx]}"
        target_name="$(basename "$target")"
        read -rp "Delete protocol '$target_name' and all its files? [y/N]: " confirm
        if [[ "$confirm" != "y" ]]; then echo "Cancelled."; exit 0; fi
        rm -rf "$target"
        echo ""
        echo "[+] Deleted protocol: $target_name/"
        echo ""
        ;;
    3)
        # ── Delete generated output ──
        OUT_DIR="${OUTPUT_DIR:-${ADAPTIX_OUTPUT_DIR:-$SCRIPT_DIR/output}}"
        if [[ ! -d "$OUT_DIR" ]]; then
            echo "[-] Output directory not found: $OUT_DIR"; exit 1
        fi
        items=()
        for d in "$OUT_DIR"/*/; do
            [[ -d "$d" ]] || continue
            items+=("$d")
        done
        if [[ ${#items[@]} -eq 0 ]]; then
            echo "[-] No generated projects found in $OUT_DIR"; exit 1
        fi
        echo ""
        section "Generated projects in $OUT_DIR"
        for i in "${!items[@]}"; do
            printf '  %s[%s]%s %s%s%s\n' "$C_GREEN" "$((i+1))" "$C_RESET" "$C_CYAN" "$(basename "${items[$i]}")" "$C_RESET"
        done
        echo ""
        read -rp "Select project to delete: " pick
        idx=$((pick - 1))
        if [[ $idx -lt 0 || $idx -ge ${#items[@]} ]]; then
            echo "[-] Invalid choice."; exit 1
        fi
        target="${items[$idx]}"
        target_name="$(basename "$target")"
        read -rp "Delete '$target_name' and all its contents? [y/N]: " confirm
        if [[ "$confirm" != "y" ]]; then echo "Cancelled."; exit 0; fi
        rm -rf "$target"
        echo ""
        echo "[+] Deleted: $target_name/"
        echo ""
        ;;
    *)
        printf '%s\n' "${C_RED}[-] Invalid choice.${C_RESET}"; exit 1
        ;;
    esac
    ;;
*)
    printf '%s\n' "${C_RED}[-] Unknown mode: $MODE${C_RESET}"
    exit 1
    ;;
esac
