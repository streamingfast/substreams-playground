pub mod proto {
    use std::io::Cursor;
    use crate::eth;

    #[no_mangle]
    pub fn decode(ptr: *mut u8, size: usize) -> eth::Block {
        unsafe {
            let input_data = Vec::from_raw_parts(ptr, size, size);
            let blk: eth::Block = ::prost::Message::decode(&mut Cursor::new(&input_data)).unwrap();
            std::mem::forget(input_data); // otherwise tries to free that memory at the end and crashes

            blk
        }
    }
}