SRC_DIR := $(shell pwd)

LDFLAGS := -s -w
TRIMPATH := -trimpath
BUILD    := CGO_ENABLED=0 __BUILD_TOOL__ $(TRIMPATH) -ldflags="$(LDFLAGS)"

.PHONY: all linux windows darwin clean

all: linux windows darwin

linux:
	GOOS=linux GOARCH=amd64 $(BUILD) -o __NAME___linux_amd64 .
	GOOS=linux GOARCH=arm64 $(BUILD) -o __NAME___linux_arm64 .

windows:
	GOOS=windows GOARCH=amd64 $(BUILD) -o __NAME___windows_amd64.exe .
	GOOS=windows GOARCH=arm64 $(BUILD) -o __NAME___windows_arm64.exe .

darwin:
	GOOS=darwin GOARCH=amd64 $(BUILD) -o __NAME___darwin_amd64 .
	GOOS=darwin GOARCH=arm64 $(BUILD) -o __NAME___darwin_arm64 .

clean:
	rm -f __NAME___linux_* __NAME___windows_* __NAME___darwin_*
