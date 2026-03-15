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

echo ""
echo "╔═══════════════════════════════════════════════╗"
echo "║   AdaptixC2 Template Generator                ║"
echo "╚═══════════════════════════════════════════════╝"
echo ""

# ─── Mode selection ─────────────────────────────────────────────────────────────

if [[ -z "$MODE" ]]; then
    echo "What do you want to generate?"
    echo ""
    echo "  [1] Generate Agent     - Scaffold a new agent extender"
    echo "  [2] Generate Listener  - Scaffold a new listener extender"
    echo "  [3] Generate Service   - Scaffold a new service extender"
    echo "  [4] Create Protocol    - Create a new wire-protocol definition"
    echo "  [5] Create Crypto      - Generate or replace the crypto template for a protocol"
    echo "  [6] Delete             - Remove a crypto template, protocol, or generated output"
    echo ""
    read -rp "Select option: " choice
    case "$choice" in
        1) MODE="agent" ;;
        2) MODE="listener" ;;
        3) MODE="service" ;;
        4) MODE="protocol" ;;
        5) MODE="crypto" ;;
        6) MODE="delete" ;;
        *) echo "[-] Invalid choice."; exit 1 ;;
    esac
fi

# ─── Dispatch ───────────────────────────────────────────────────────────────────

case "$MODE" in
agent)
    echo "[*] Launching Agent Generator..."
    echo ""
    exec bash "$SCRIPT_DIR/agent/generator.sh" "$@"
    ;;
listener)
    echo "[*] Launching Listener Generator..."
    echo ""
    exec bash "$SCRIPT_DIR/listener/generator.sh" "$@"
    ;;
service)
    echo "[*] Launching Service Generator..."
    echo ""
    exec bash "$SCRIPT_DIR/service/generator.sh" "$@"
    ;;
protocol)
    echo "[*] Launching Protocol Generator..."
    echo ""
    exec bash "$SCRIPT_DIR/protocols/generator.sh" "$@"
    ;;
crypto)
    echo "[*] Launching Crypto Generator..."
    echo ""
    exec bash "$SCRIPT_DIR/protocols/crypto_generator.sh" "$@"
    ;;
delete)
    echo ""
    echo "What do you want to delete?"
    echo ""
    echo "  [1] Crypto template  - Remove a crypto .go.tmpl from _crypto/"
    echo "  [2] Protocol         - Remove an entire protocol definition"
    echo "  [3] Generated output - Remove a generated project from output/"
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
        echo "Available crypto templates:"
        for i in "${!items[@]}"; do
            key="$(basename "${items[$i]}" .go.tmpl)"
            echo "  [$((i+1))] $key"
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
        echo "Available protocols:"
        for i in "${!items[@]}"; do
            echo "  [$((i+1))] $(basename "${items[$i]}")"
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
        echo "Generated projects in $OUT_DIR:"
        for i in "${!items[@]}"; do
            echo "  [$((i+1))] $(basename "${items[$i]}")"
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
        echo "[-] Invalid choice."; exit 1
        ;;
    esac
    ;;
*)
    echo "[-] Unknown mode: $MODE"
    exit 1
    ;;
esac
