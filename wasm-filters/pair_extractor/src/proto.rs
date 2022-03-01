use std::io::Cursor;

pub fn decode<T: std::default::Default + prost::Message>(ptr: *mut u8, size: usize) -> T {
    unsafe {
        let input_data = Vec::from_raw_parts(ptr, size, size);
        let obj: T = ::prost::Message::decode(&mut Cursor::new(&input_data)).unwrap();
        std::mem::forget(input_data); // otherwise tries to free that memory at the end and crashes

        obj
    }
}
