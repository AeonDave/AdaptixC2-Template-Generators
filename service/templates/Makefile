PLUGIN_DIR := $(shell pwd)
PLUGIN_OUT := $(PLUGIN_DIR)/service___NAME__.so

.PHONY: all plugin clean

all: plugin

plugin:
	@echo "[*] Building __NAME__ service plugin..."
	cd $(PLUGIN_DIR) && GOEXPERIMENT=jsonv2,greenteagc go build -trimpath -buildmode=plugin -ldflags="-s -w" -o $(PLUGIN_OUT) .
	@echo "[+] Plugin built: $(PLUGIN_OUT)"

dist: plugin
	@mkdir -p $(PLUGIN_DIR)/dist
	@cp $(PLUGIN_OUT) $(PLUGIN_DIR)/dist/
	@cp $(PLUGIN_DIR)/config.yaml $(PLUGIN_DIR)/dist/
	@cp $(PLUGIN_DIR)/ax_config.axs $(PLUGIN_DIR)/dist/
	@echo "[+] Dist ready: $(PLUGIN_DIR)/dist/"

clean:
	rm -f $(PLUGIN_OUT)
	rm -rf $(PLUGIN_DIR)/dist
