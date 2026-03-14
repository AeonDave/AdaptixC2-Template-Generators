#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# AdaptixC2 Unified Template Generator
#
# Root entry-point that dispatches to the appropriate sub-generator:
#   1) Agent     - scaffold a new agent extender
#   2) Listener  - scaffold a new listener extender
#   3) Protocol  - create a new wire-protocol definition
#   4) Crypto    - swap the crypto implementation of an existing protocol
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
    echo "  [5] Swap Crypto        - Generate or replace the crypto template for a protocol"
    echo ""
    read -rp "Select option: " choice
    case "$choice" in
        1) MODE="agent" ;;
        2) MODE="listener" ;;
        3) MODE="service" ;;
        4) MODE="protocol" ;;
        5) MODE="crypto" ;;
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
*)
    echo "[-] Unknown mode: $MODE"
    exit 1
    ;;
esac
