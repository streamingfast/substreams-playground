#[link(wasm_import_module = "env")]
extern "C" {
    pub fn output(ptr: *const u8, len: u32);
    pub fn register_panic(
        msg_ptr: *const u8,
        msg_len: u32,
        file_ptr: *const u8,
        file_len: u32,
        line: u32,
        column: u32,
    );
}

#[link(wasm_import_module = "logger")]
extern "C" {
    pub fn debug(ptr: *const u8, len: usize);
    pub fn info(ptr: *const u8, len: usize);
}