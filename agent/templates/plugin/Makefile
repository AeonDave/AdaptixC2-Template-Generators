PLUGIN_DIR := $(CURDIR)
SRC_DIR    := $(PLUGIN_DIR)/src___NAME__
PLUGIN_OUT := $(PLUGIN_DIR)/agent___NAME__.so

.PHONY: all plugin implant clean

all: plugin

plugin:
	@echo "[*] Building __NAME__ server plugin..."
	cd $(PLUGIN_DIR) && GOEXPERIMENT=jsonv2,greenteagc go build -trimpath -buildmode=plugin -ldflags="-s -w" -o $(PLUGIN_OUT) .
	@echo "[+] Plugin built: $(PLUGIN_OUT)"

implant:
	@echo "[*] Building __NAME__ implants..."
	$(MAKE) -C $(SRC_DIR) all

full: plugin implant
	@echo "[+] __NAME__ full build complete."

clean:
	rm -f $(PLUGIN_OUT)
	$(MAKE) -C $(SRC_DIR) clean
