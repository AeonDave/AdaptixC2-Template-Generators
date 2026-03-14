// __NAME__ Agent — Agent Module
//
// Main agent logic: connection loop, command dispatch, and transport.

use crate::protocol;

/// Connector trait — implement for each transport (TCP, HTTP, etc.)
pub trait Connector {
    fn connect(&mut self) -> Result<(), String>;
    fn exchange(&mut self, data: &[u8]) -> Result<Vec<u8>, String>;
    fn disconnect(&mut self);
}

/// Main agent state
pub struct Agent {
    // TODO: Add fields: session key, sleep interval, connector, etc.
}

impl Agent {
    pub fn new(_profile: Vec<u8>) -> Self {
        // TODO: Parse profile and initialize
        Agent {}
    }

    pub fn run(&mut self) {
        // TODO: Implement check-in → sleep → task loop
        //
        // loop {
        //     let tasks = self.connector.exchange(&checkin_data)?;
        //     self.dispatch(tasks);
        //     std::thread::sleep(self.sleep_interval);
        // }
        let _ = protocol::WATERMARK;
    }
}
