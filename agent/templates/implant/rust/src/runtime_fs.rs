#![allow(dead_code)]

use std::env;
use std::fs;
use std::path::{Path, PathBuf};

pub fn normalize_path(path: &str) -> PathBuf {
    if path.is_empty() || path == "." {
        env::current_dir().unwrap_or_else(|_| Path::new(".").to_path_buf())
    } else {
        Path::new(path).to_path_buf()
    }
}

pub fn copy_recursive(src: &Path, dst: &Path) -> Result<(), String> {
    if src.is_dir() {
        fs::create_dir_all(dst).map_err(|e| e.to_string())?;
        for entry in fs::read_dir(src).map_err(|e| e.to_string())? {
            let entry = entry.map_err(|e| e.to_string())?;
            copy_recursive(&entry.path(), &dst.join(entry.file_name()))?;
        }
        Ok(())
    } else {
        fs::copy(src, dst).map_err(|e| e.to_string()).map(|_| ())
    }
}