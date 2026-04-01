# __NAME__ Agent — Rust Implant Build
#
# Usage:
#   make all         Build release binary for all configured targets
#   make linux       Build for x86_64-unknown-linux-gnu
#   make windows     Build for x86_64-pc-windows-gnu
#

BEACON := __NAME__

.PHONY: all linux windows clean

all: linux windows

linux:
	@echo "[*] Building $(BEACON) linux (release)..."
	__BUILD_TOOL__ --target x86_64-unknown-linux-gnu
	@cp target/x86_64-unknown-linux-gnu/release/$(BEACON) $(BEACON)_linux
	@echo "[+] $(BEACON)_linux"

windows:
	@echo "[*] Building $(BEACON) windows (release)..."
	__BUILD_TOOL__ --target x86_64-pc-windows-gnu
	@cp target/x86_64-pc-windows-gnu/release/$(BEACON).exe $(BEACON)_windows.exe
	@echo "[+] $(BEACON)_windows.exe"

clean:
	cargo clean
	rm -f $(BEACON)_linux $(BEACON)_windows.exe
