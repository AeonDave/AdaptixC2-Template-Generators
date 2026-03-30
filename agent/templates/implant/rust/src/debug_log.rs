/// Debug logging macro — active only when the `debug` feature is enabled.
/// Outputs to stderr via `eprintln!`.

#[cfg(feature = "debug")]
macro_rules! dbg_log {
    ($($arg:tt)*) => {
        eprintln!("[DBG] {}", format!($($arg)*));
    };
}

#[cfg(not(feature = "debug"))]
macro_rules! dbg_log {
    ($($arg:tt)*) => {};
}
