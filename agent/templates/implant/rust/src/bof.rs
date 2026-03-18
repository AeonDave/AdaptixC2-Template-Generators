// __NAME__ Agent — BOF Loader (stub)
//
// Placeholder for the BOF (Beacon Object File) in-memory COFF loader.
// Implement the COFF parser, relocations, and Beacon API to enable BOF support.
//
// Reference the project's BOF loader implementation or the original Adaptix
// beacon BOF loader for structure and callback behavior.
//
// Architecture overview (from bof_engine):
//
//   ┌──────────────┐    ┌──────────────┐    ┌──────────────────┐
//   │ load()       │───►│ COFF Loader  │───►│ Beacon API       │
//   │ load_async() │    │ parse/reloc/ │    │ (callbacks)      │
//   │              │    │ resolve/exec │    │ output → Vec     │
//   └──────────────┘    └──────────────┘    └──────────────────┘

#![allow(dead_code)]

use crate::protocol;
use std::collections::HashMap;
use std::sync::Mutex;

// ── COFF structures (match PE/COFF spec — see bof_engine/coff/windef.go) ──────

pub const SIZEOF_FILE_HEADER: usize = 20;
pub const SIZEOF_SECTION_HEADER: usize = 40;
pub const SIZEOF_RELOCATION: usize = 10;
pub const SIZEOF_SYMBOL: usize = 18;

/// Machine types.
pub const IMAGE_FILE_MACHINE_I386: u16 = 0x14c;
pub const IMAGE_FILE_MACHINE_AMD64: u16 = 0x8664;

/// AMD64 relocation types.
pub const IMAGE_REL_AMD64_ABSOLUTE: u16 = 0x0000;
pub const IMAGE_REL_AMD64_ADDR64: u16 = 0x0001;
pub const IMAGE_REL_AMD64_ADDR32NB: u16 = 0x0003;
pub const IMAGE_REL_AMD64_REL32: u16 = 0x0004;
pub const IMAGE_REL_AMD64_REL32_1: u16 = 0x0005;
pub const IMAGE_REL_AMD64_REL32_2: u16 = 0x0006;
pub const IMAGE_REL_AMD64_REL32_3: u16 = 0x0007;
pub const IMAGE_REL_AMD64_REL32_4: u16 = 0x0008;
pub const IMAGE_REL_AMD64_REL32_5: u16 = 0x0009;

/// Symbol storage classes.
pub const IMAGE_SYM_CLASS_EXTERNAL: u8 = 2;
pub const IMAGE_SYM_CLASS_STATIC: u8 = 3;

/// Section characteristics.
pub const IMAGE_SCN_CNT_UNINITIALIZED_DATA: u32 = 0x0000_0080;
pub const IMAGE_SCN_MEM_EXECUTE: u32 = 0x2000_0000;
pub const IMAGE_SCN_MEM_READ: u32 = 0x4000_0000;
pub const IMAGE_SCN_MEM_WRITE: u32 = 0x8000_0000;

/// COFF file header (20 bytes).
#[repr(C, packed)]
#[derive(Clone, Copy)]
pub struct FileHeader {
    pub machine: u16,
    pub number_of_sections: u16,
    pub time_date_stamp: u32,
    pub pointer_to_symbol_table: u32,
    pub number_of_symbols: u32,
    pub size_of_optional_header: u16,
    pub characteristics: u16,
}

/// COFF section header (40 bytes).
#[repr(C, packed)]
#[derive(Clone, Copy)]
pub struct SectionHeader {
    pub name: [u8; 8],
    pub virtual_size: u32,
    pub virtual_address: u32,
    pub size_of_raw_data: u32,
    pub pointer_to_raw_data: u32,
    pub pointer_to_relocations: u32,
    pub pointer_to_line_numbers: u32,
    pub number_of_relocations: u16,
    pub number_of_line_numbers: u16,
    pub characteristics: u32,
}

/// COFF relocation entry (10 bytes).
#[repr(C, packed)]
#[derive(Clone, Copy)]
pub struct Relocation {
    pub virtual_address: u32,
    pub symbol_table_index: u32,
    pub reloc_type: u16,
}

/// COFF symbol table entry (18 bytes).
#[repr(C, packed)]
#[derive(Clone, Copy)]
pub struct CoffSymbol {
    pub name: [u8; 8],
    pub value: u32,
    pub section_number: i16,
    pub sym_type: u16,
    pub storage_class: u8,
    pub number_of_aux_symbols: u8,
}

// ── BOF output ─────────────────────────────────────────────────────────────────

/// Result of a BOF execution — a list of typed output messages.
pub struct BofMsg {
    pub msg_type: u16,
    pub data: Vec<u8>,
}

/// BofContext collects output messages during a BOF execution.
pub struct BofContext {
    pub msgs: Vec<BofMsg>,
}

impl BofContext {
    pub fn new() -> Self {
        BofContext { msgs: Vec::new() }
    }

    pub fn push(&mut self, msg_type: u16, data: Vec<u8>) {
        self.msgs.push(BofMsg { msg_type, data });
    }
}

/// Section allocation tracker.
pub struct SectionAlloc {
    pub address: usize,
    pub size: usize,
    pub characteristics: u32,
}

// ── Data parser (matches Cobalt Strike datap / bof_engine DataParser) ──────────
//
// BOF argument buffer format:
//   [4B LE: total_length] [4B LE: arg1_len][arg1_data] ...

/// DataParser provides sequential unpacking of BOF arguments (bof_pack format).
pub struct DataParser {
    buffer: Vec<u8>,
    offset: usize,
    size: usize,
}

impl DataParser {
    /// Initialize from a raw argument buffer (skips 4-byte length prefix).
    pub fn new(buffer: &[u8]) -> Self {
        if buffer.len() < 4 {
            return DataParser { buffer: Vec::new(), offset: 0, size: 0 };
        }
        let data = buffer[4..].to_vec();
        let size = data.len();
        DataParser { buffer: data, offset: 0, size }
    }

    /// Read a 4-byte little-endian i32.
    pub fn get_int(&mut self) -> i32 {
        if self.offset + 4 > self.size { return 0; }
        let val = i32::from_le_bytes([
            self.buffer[self.offset],
            self.buffer[self.offset + 1],
            self.buffer[self.offset + 2],
            self.buffer[self.offset + 3],
        ]);
        self.offset += 4;
        val
    }

    /// Read a 2-byte little-endian i16.
    pub fn get_short(&mut self) -> i16 {
        if self.offset + 2 > self.size { return 0; }
        let val = i16::from_le_bytes([
            self.buffer[self.offset],
            self.buffer[self.offset + 1],
        ]);
        self.offset += 2;
        val
    }

    /// Returns remaining bytes.
    pub fn length(&self) -> usize {
        self.size.saturating_sub(self.offset)
    }

    /// Read a length-prefixed blob: [4B LE: len][data].
    pub fn extract(&mut self) -> Option<Vec<u8>> {
        let len = self.get_int() as usize;
        if len == 0 || self.offset + len > self.size {
            return None;
        }
        let data = self.buffer[self.offset..self.offset + len].to_vec();
        self.offset += len;
        Some(data)
    }

    /// Return a raw slice of `size` bytes.
    pub fn get_ptr(&mut self, size: usize) -> Option<&[u8]> {
        if self.offset + size > self.size { return None; }
        let slice = &self.buffer[self.offset..self.offset + size];
        self.offset += size;
        Some(slice)
    }
}

// ── Format buffer (matches Cobalt Strike formatp / bof_engine FormatParser) ────

/// FormatBuffer provides a dynamic byte buffer for BOF output formatting.
pub struct FormatBuffer {
    buffer: Vec<u8>,
}

impl FormatBuffer {
    pub fn new(max_size: usize) -> Self {
        FormatBuffer { buffer: Vec::with_capacity(max_size) }
    }

    pub fn reset(&mut self) {
        self.buffer.clear();
    }

    pub fn append(&mut self, data: &[u8]) {
        self.buffer.extend_from_slice(data);
    }

    /// Append a 4-byte big-endian integer.
    pub fn append_int(&mut self, value: i32) {
        self.buffer.extend_from_slice(&value.to_be_bytes());
    }

    pub fn to_string_lossy(&self) -> String {
        String::from_utf8_lossy(&self.buffer).into_owned()
    }

    pub fn as_bytes(&self) -> &[u8] {
        &self.buffer
    }

    pub fn len(&self) -> usize {
        self.buffer.len()
    }
}

// ── Key-Value store (matches bof_engine lighthouse keyvalue) ───────────────────

/// Thread-safe key-value store for BeaconAddValue/GetValue/RemoveValue.
pub struct KvStore {
    inner: Mutex<HashMap<String, Vec<u8>>>,
}

impl KvStore {
    pub fn new() -> Self {
        KvStore { inner: Mutex::new(HashMap::new()) }
    }

    pub fn add(&self, key: &str, value: Vec<u8>) {
        if let Ok(mut map) = self.inner.lock() {
            map.insert(key.to_string(), value);
        }
    }

    pub fn get(&self, key: &str) -> Option<Vec<u8>> {
        self.inner.lock().ok()?.get(key).cloned()
    }

    pub fn remove(&self, key: &str) -> bool {
        self.inner.lock().ok().map_or(false, |mut m| m.remove(key).is_some())
    }
}

// ── COFF Loader ────────────────────────────────────────────────────────────────

/// Execute a COFF object file synchronously.
///
/// Returns a BofContext with collected output messages, or an error string.
///
/// Implementation pipeline (from bof_engine):
///  1. Parse COFF file header, section headers, symbol table, string table
///  2. Calculate GOT size (count import symbols) and BSS size (common data)
///  3. VirtualAlloc RW pages for each section, copy raw section data
///  4. Allocate GOT for imported function pointers
///  5. Resolve external symbols:
///     a. "__imp_<lib>$<func>" → LoadLibrary + GetProcAddress → GOT entry
///     b. Beacon API names → registered callback function table
///     c. Runtime symbols (memset, etc.) → msvcrt/ucrtbase/ntdll, create thunks
///     d. "__C_specific_handler" → resolve from ntdll via PEB walk
///  6. Process relocations: ADDR64 (8B abs), ADDR32NB (4B RIP-rel), REL32 (4B rel)
///  7. Set .text sections to PAGE_EXECUTE_READ
///  8. Find entry symbol ("go" / "_go") and invoke with (arg_ptr, arg_len)
///  9. Collect output → Vec<BofMsg>
///  10. VirtualFree all sections, release GOT
pub fn load(_object: &[u8], _args: &[u8]) -> Result<BofContext, String> {
    let _ = protocol::COMMAND_EXEC_BOF;
    Err("BOF execution requires a native loader extension in the Rust scaffold".to_string())
}

/// Execute a COFF object file asynchronously (spawns a thread).
///
/// Streams output via the returned BofContext.
pub fn load_async(_object: &[u8], _args: &[u8]) -> Result<BofContext, String> {
    let _ = protocol::COMMAND_EXEC_BOF_ASYNC;
    Err("async BOF execution requires a native loader extension in the Rust scaffold".to_string())
}

// ── PackArgs helper (matches bof_engine lighthouse PackArgs) ───────────────────
//
// Format: "z" = null-terminated string, "Z" = wide string,
//         "i" = 4-byte int, "s" = 2-byte short,
//         "b" = binary blob (length-prefixed)

/// Pack arguments according to a format string into a BOF argument buffer.
pub fn pack_args(_format: &str, _args: &[&[u8]]) -> Result<Vec<u8>, String> {
    Err("BOF argument packing requires a loader extension in the Rust scaffold".to_string())
}

// ── Beacon API Stubs ───────────────────────────────────────────────────────────
//
// These functions mirror the Cobalt Strike Beacon API surface (60+ functions).
// In the real implementation each function is registered as a callback and
// resolved by symbol name during COFF loading. Replace the stubs when
// integrating the COFF loader.
//
// Organized by category to match bof_engine/lighthouse/ structure.

/// Beacon API callback functions — grouped trait for testability.
pub trait BeaconApi {
    // Data Parser
    fn beacon_data_parse(&self, buffer: &[u8]) -> DataParser;
    fn beacon_data_int(&self, parser: &mut DataParser) -> i32;
    fn beacon_data_short(&self, parser: &mut DataParser) -> i16;
    fn beacon_data_length(&self, parser: &DataParser) -> usize;
    fn beacon_data_extract(&self, parser: &mut DataParser) -> Option<Vec<u8>>;
    fn beacon_data_ptr(&self, parser: &mut DataParser, size: usize) -> Option<Vec<u8>>;

    // Output
    fn beacon_output(&mut self, callback_type: u16, data: &[u8]);
    fn beacon_printf(&mut self, callback_type: u16, fmt: &str);

    // Format Buffer
    fn beacon_format_alloc(&self, max_size: usize) -> FormatBuffer;
    fn beacon_format_reset(&self, format: &mut FormatBuffer);
    fn beacon_format_append(&self, format: &mut FormatBuffer, data: &[u8]);
    fn beacon_format_printf(&self, format: &mut FormatBuffer, fmt: &str);
    fn beacon_format_to_string(&self, format: &FormatBuffer) -> String;
    fn beacon_format_free(&self, format: &mut FormatBuffer);
    fn beacon_format_int(&self, format: &mut FormatBuffer, value: i32);

    // Token
    fn beacon_use_token(&self, token: usize) -> bool;
    fn beacon_revert_token(&self);
    fn beacon_is_admin(&self) -> bool;

    // Key-Value Store
    fn beacon_add_value(&self, key: &str, value: Vec<u8>);
    fn beacon_get_value(&self, key: &str) -> Option<Vec<u8>>;
    fn beacon_remove_value(&self, key: &str) -> bool;

    // Process / Injection (CS BOF compat)
    fn beacon_get_spawn_to(&self, x86: bool) -> Option<String>;
    fn beacon_spawn_temporary_process(&self, x86: bool, ignore_token: bool) -> Result<(u32, usize, usize), String>;
    fn beacon_inject_process(&self, h_proc: usize, pid: u32, payload: &[u8], offset: usize, arg: &[u8]);
    fn beacon_inject_temporary_process(&self, h_process: usize, h_thread: usize, payload: &[u8], offset: usize, arg: &[u8]);
    fn beacon_cleanup_process(&self, h_process: usize, h_thread: usize);

    // Syscall Wrappers (CS 4.10+)
    fn beacon_virtual_alloc(&self, addr: usize, size: usize, alloc_type: u32, protect: u32) -> usize;
    fn beacon_virtual_alloc_ex(&self, h_process: usize, addr: usize, size: usize, alloc_type: u32, protect: u32) -> usize;
    fn beacon_virtual_protect(&self, addr: usize, size: usize, new_protect: u32) -> (u32, bool);
    fn beacon_virtual_protect_ex(&self, h_process: usize, addr: usize, size: usize, new_protect: u32) -> (u32, bool);
    fn beacon_virtual_free(&self, addr: usize, size: usize, free_type: u32) -> bool;
    fn beacon_get_thread_context(&self, h_thread: usize, ctx: usize) -> bool;
    fn beacon_set_thread_context(&self, h_thread: usize, ctx: usize) -> bool;
    fn beacon_resume_thread(&self, h_thread: usize) -> u32;
    fn beacon_open_process(&self, desired_access: u32, inherit_handle: bool, pid: u32) -> usize;
    fn beacon_open_thread(&self, desired_access: u32, inherit_handle: bool, tid: u32) -> usize;
    fn beacon_close_handle(&self, h: usize) -> bool;
    fn beacon_unmap_view_of_file(&self, addr: usize) -> bool;
    fn beacon_virtual_query(&self, addr: usize, buf: usize, length: usize) -> usize;
    fn beacon_duplicate_handle(&self, src_process: usize, src_handle: usize, tgt_process: usize, desired_access: u32, inherit_handle: bool, options: u32) -> (usize, bool);
    fn beacon_read_process_memory(&self, h_process: usize, base_addr: usize, buf: &mut [u8]) -> (usize, bool);
    fn beacon_write_process_memory(&self, h_process: usize, base_addr: usize, buf: &[u8]) -> (usize, bool);

    // Downloads
    fn beacon_download(&self, filename: &str, data: &[u8]);

    // Miscellaneous
    fn beacon_information(&self, info: usize);
    fn beacon_get_output_data(&self) -> Option<Vec<u8>>;
    fn swap_endianness(&self, val: u32) -> u32;
    fn to_wide_char(&self, src: &str, max_chars: usize) -> Vec<u16>;

    // Adaptix Extensions
    fn ax_add_screenshot(&mut self, note: &str, data: &[u8]);
    fn ax_download_memory(&mut self, filename: &str, data: &[u8]);

    // CS 4.9+ Data Store (stubs)
    fn beacon_data_store_get_item(&self, index: usize) -> usize;
    fn beacon_data_store_protect_item(&self, index: usize);
    fn beacon_data_store_unprotect_item(&self, index: usize);
    fn beacon_data_store_max_entries(&self) -> usize;
    fn beacon_get_custom_user_data(&self) -> usize;

    // Async BOF Thread Callbacks (CS 4.9+)
    fn beacon_register_thread_callback(&self, callback: usize, data: usize);
    fn beacon_unregister_thread_callback(&self);
    fn beacon_wakeup(&self);
    fn beacon_get_stop_job_event(&self) -> usize;

    // Beacon Gate (CS 4.10+)
    fn beacon_disable_beacon_gate(&self);
    fn beacon_enable_beacon_gate(&self);
    fn beacon_disable_beacon_gate_masking(&self);
    fn beacon_enable_beacon_gate_masking(&self);
    fn beacon_get_syscall_information(&self) -> usize;
}

/// Default (stub) implementation of the Beacon API.
pub struct DefaultBeaconApi {
    pub context: BofContext,
    pub kv_store: KvStore,
}

impl DefaultBeaconApi {
    pub fn new() -> Self {
        DefaultBeaconApi {
            context: BofContext::new(),
            kv_store: KvStore::new(),
        }
    }
}

impl BeaconApi for DefaultBeaconApi {
    // ── Data Parser ────────────────────────────────────────────────────────

    fn beacon_data_parse(&self, buffer: &[u8]) -> DataParser {
        DataParser::new(buffer)
    }

    fn beacon_data_int(&self, parser: &mut DataParser) -> i32 {
        parser.get_int()
    }

    fn beacon_data_short(&self, parser: &mut DataParser) -> i16 {
        parser.get_short()
    }

    fn beacon_data_length(&self, parser: &DataParser) -> usize {
        parser.length()
    }

    fn beacon_data_extract(&self, parser: &mut DataParser) -> Option<Vec<u8>> {
        parser.extract()
    }

    fn beacon_data_ptr(&self, parser: &mut DataParser, size: usize) -> Option<Vec<u8>> {
        parser.get_ptr(size).map(|s| s.to_vec())
    }

    // ── Output ─────────────────────────────────────────────────────────────

    fn beacon_output(&mut self, callback_type: u16, data: &[u8]) {
        self.context.push(callback_type, data.to_vec());
    }

    fn beacon_printf(&mut self, callback_type: u16, fmt: &str) {
        // TODO: Implement C-style printf parsing (handles %x/%d/%s/%ls/%p)
        self.context.push(callback_type, fmt.as_bytes().to_vec());
    }

    // ── Format Buffer ──────────────────────────────────────────────────────

    fn beacon_format_alloc(&self, max_size: usize) -> FormatBuffer {
        FormatBuffer::new(max_size)
    }

    fn beacon_format_reset(&self, format: &mut FormatBuffer) {
        format.reset();
    }

    fn beacon_format_append(&self, format: &mut FormatBuffer, data: &[u8]) {
        format.append(data);
    }

    fn beacon_format_printf(&self, format: &mut FormatBuffer, fmt: &str) {
        // TODO: C printf format parser (see bof_engine lighthouse)
        format.append(fmt.as_bytes());
    }

    fn beacon_format_to_string(&self, format: &FormatBuffer) -> String {
        format.to_string_lossy()
    }

    fn beacon_format_free(&self, format: &mut FormatBuffer) {
        format.reset();
    }

    fn beacon_format_int(&self, format: &mut FormatBuffer, value: i32) {
        format.append_int(value);
    }

    // ── Token ──────────────────────────────────────────────────────────────

    fn beacon_use_token(&self, _token: usize) -> bool { false }
    fn beacon_revert_token(&self) {}
    fn beacon_is_admin(&self) -> bool { false }

    // ── Key-Value Store ────────────────────────────────────────────────────

    fn beacon_add_value(&self, key: &str, value: Vec<u8>) {
        self.kv_store.add(key, value);
    }

    fn beacon_get_value(&self, key: &str) -> Option<Vec<u8>> {
        self.kv_store.get(key)
    }

    fn beacon_remove_value(&self, key: &str) -> bool {
        self.kv_store.remove(key)
    }

    // ── Process / Injection ────────────────────────────────────────────────

    fn beacon_get_spawn_to(&self, _x86: bool) -> Option<String> { None }
    fn beacon_spawn_temporary_process(&self, _x86: bool, _ignore_token: bool) -> Result<(u32, usize, usize), String> {
        Err("not implemented".to_string())
    }
    fn beacon_inject_process(&self, _h_proc: usize, _pid: u32, _payload: &[u8], _offset: usize, _arg: &[u8]) {}
    fn beacon_inject_temporary_process(&self, _h_process: usize, _h_thread: usize, _payload: &[u8], _offset: usize, _arg: &[u8]) {}
    fn beacon_cleanup_process(&self, _h_process: usize, _h_thread: usize) {}

    // ── Syscall Wrappers (CS 4.10+) ────────────────────────────────────────

    fn beacon_virtual_alloc(&self, _addr: usize, _size: usize, _alloc_type: u32, _protect: u32) -> usize { 0 }
    fn beacon_virtual_alloc_ex(&self, _h_process: usize, _addr: usize, _size: usize, _alloc_type: u32, _protect: u32) -> usize { 0 }
    fn beacon_virtual_protect(&self, _addr: usize, _size: usize, _new_protect: u32) -> (u32, bool) { (0, false) }
    fn beacon_virtual_protect_ex(&self, _h_process: usize, _addr: usize, _size: usize, _new_protect: u32) -> (u32, bool) { (0, false) }
    fn beacon_virtual_free(&self, _addr: usize, _size: usize, _free_type: u32) -> bool { false }
    fn beacon_get_thread_context(&self, _h_thread: usize, _ctx: usize) -> bool { false }
    fn beacon_set_thread_context(&self, _h_thread: usize, _ctx: usize) -> bool { false }
    fn beacon_resume_thread(&self, _h_thread: usize) -> u32 { 0 }
    fn beacon_open_process(&self, _desired_access: u32, _inherit_handle: bool, _pid: u32) -> usize { 0 }
    fn beacon_open_thread(&self, _desired_access: u32, _inherit_handle: bool, _tid: u32) -> usize { 0 }
    fn beacon_close_handle(&self, _h: usize) -> bool { false }
    fn beacon_unmap_view_of_file(&self, _addr: usize) -> bool { false }
    fn beacon_virtual_query(&self, _addr: usize, _buf: usize, _length: usize) -> usize { 0 }
    fn beacon_duplicate_handle(&self, _src_process: usize, _src_handle: usize, _tgt_process: usize, _desired_access: u32, _inherit_handle: bool, _options: u32) -> (usize, bool) { (0, false) }
    fn beacon_read_process_memory(&self, _h_process: usize, _base_addr: usize, _buf: &mut [u8]) -> (usize, bool) { (0, false) }
    fn beacon_write_process_memory(&self, _h_process: usize, _base_addr: usize, _buf: &[u8]) -> (usize, bool) { (0, false) }

    // ── Downloads ──────────────────────────────────────────────────────────

    fn beacon_download(&self, _filename: &str, _data: &[u8]) {}

    // ── Miscellaneous ──────────────────────────────────────────────────────

    fn beacon_information(&self, _info: usize) {}

    fn beacon_get_output_data(&self) -> Option<Vec<u8>> { None }

    fn swap_endianness(&self, val: u32) -> u32 {
        (val >> 24) & 0xFF | (val >> 8) & 0xFF00 | (val << 8) & 0x00FF_0000 | (val << 24) & 0xFF00_0000
    }

    fn to_wide_char(&self, src: &str, max_chars: usize) -> Vec<u16> {
        let mut wide: Vec<u16> = src.encode_utf16().collect();
        wide.truncate(max_chars.saturating_sub(1));
        wide.push(0); // null terminator
        wide
    }

    // ── Adaptix Extensions ─────────────────────────────────────────────────

    fn ax_add_screenshot(&mut self, _note: &str, data: &[u8]) {
        self.context.push(protocol::CALLBACK_AX_SCREENSHOT as u16, data.to_vec());
    }

    fn ax_download_memory(&mut self, _filename: &str, data: &[u8]) {
        self.context.push(protocol::CALLBACK_AX_DOWNLOAD_MEM as u16, data.to_vec());
    }

    // ── CS 4.9+ Data Store ─────────────────────────────────────────────────

    fn beacon_data_store_get_item(&self, _index: usize) -> usize { 0 }
    fn beacon_data_store_protect_item(&self, _index: usize) {}
    fn beacon_data_store_unprotect_item(&self, _index: usize) {}
    fn beacon_data_store_max_entries(&self) -> usize { 0 }
    fn beacon_get_custom_user_data(&self) -> usize { 0 }

    // ── Async BOF Thread Callbacks (CS 4.9+) ──────────────────────────────

    fn beacon_register_thread_callback(&self, _callback: usize, _data: usize) {}
    fn beacon_unregister_thread_callback(&self) {}
    fn beacon_wakeup(&self) {}
    fn beacon_get_stop_job_event(&self) -> usize { 0 }

    // ── Beacon Gate (CS 4.10+) ─────────────────────────────────────────────

    fn beacon_disable_beacon_gate(&self) {}
    fn beacon_enable_beacon_gate(&self) {}
    fn beacon_disable_beacon_gate_masking(&self) {}
    fn beacon_enable_beacon_gate_masking(&self) {}
    fn beacon_get_syscall_information(&self) -> usize { 0 }
}
