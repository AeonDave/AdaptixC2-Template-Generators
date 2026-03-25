PLUGIN_DIR := $(CURDIR)
PLUGIN_OUT := $(PLUGIN_DIR)/listener___NAME_____PROTOCOL__.so
GO         ?= go
TRIMPATH   ?= -trimpath

.PHONY: all plugin clean

all: plugin

plugin:
	@echo "[*] Building __NAME__ __PROTOCOL__ listener plugin..."
	cd $(PLUGIN_DIR) && GOEXPERIMENT=jsonv2,greenteagc $(GO) build $(TRIMPATH) -buildmode=plugin -ldflags="-s -w" -o $(PLUGIN_OUT) .
	@echo "[+] Plugin built: $(PLUGIN_OUT)"
	@mkdir -p $(PLUGIN_DIR)/dist
	@cp $(PLUGIN_OUT) $(PLUGIN_DIR)/dist/
	@cp $(PLUGIN_DIR)/config.yaml $(PLUGIN_DIR)/dist/
	@cp $(PLUGIN_DIR)/ax_config.axs $(PLUGIN_DIR)/dist/

dist: plugin

clean:
	rm -f $(PLUGIN_OUT)
	rm -rf $(PLUGIN_DIR)/dist
