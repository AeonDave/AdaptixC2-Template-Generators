PLUGIN_DIR := $(CURDIR)
SRC_DIR    := $(PLUGIN_DIR)/src___NAME__
PLUGIN_OUT := $(PLUGIN_DIR)/agent___NAME__.so
GO         ?= go
TRIMPATH   ?= -trimpath

.PHONY: all plugin implant clean

all: plugin

plugin:
	@echo "[*] Building __NAME__ server plugin..."
	cd $(PLUGIN_DIR) && GOEXPERIMENT=jsonv2,greenteagc $(GO) build $(TRIMPATH) -buildmode=plugin -ldflags="-s -w" -o $(PLUGIN_OUT) .
	@echo "[+] Plugin built: $(PLUGIN_OUT)"
	@mkdir -p $(PLUGIN_DIR)/dist
	@cp $(PLUGIN_OUT) $(PLUGIN_DIR)/dist/
	@cp $(PLUGIN_DIR)/config.yaml $(PLUGIN_DIR)/dist/
	@cp $(PLUGIN_DIR)/ax_config.axs $(PLUGIN_DIR)/dist/
	@cp -r $(SRC_DIR) $(PLUGIN_DIR)/dist/

implant:
	@echo "[*] Building __NAME__ implants..."
	$(MAKE) -C $(SRC_DIR) all

full: plugin implant
	@echo "[+] __NAME__ full build complete."

dist: plugin

clean:
	rm -f $(PLUGIN_OUT)
	rm -rf $(PLUGIN_DIR)/dist
	$(MAKE) -C $(SRC_DIR) clean
