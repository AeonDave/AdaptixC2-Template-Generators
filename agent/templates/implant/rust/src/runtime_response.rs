#![allow(dead_code)]

use crate::protocol;

pub fn ok_resp<T: serde::Serialize>(code: u32, id: u32, payload: &T) -> Vec<u8> {
    let data = protocol::marshal(payload).unwrap_or_default();
    protocol::marshal(&protocol::Command { code, id, data }).unwrap_or_default()
}

pub fn complete_resp(code: u32, id: u32) -> Vec<u8> {
    protocol::marshal(&protocol::Command { code, id, data: Vec::new() }).unwrap_or_default()
}

pub fn err_resp(id: u32, msg: &str) -> Vec<u8> {
    let data = protocol::marshal(&protocol::AnsError { error: msg.to_string() }).unwrap_or_default();
    protocol::marshal(&protocol::Command { code: protocol::COMMAND_ERROR, id, data }).unwrap_or_default()
}