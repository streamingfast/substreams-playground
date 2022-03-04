use crate::substreams::memory::memory;
use std::convert::TryInto;
use std::slice;

extern "C" {
    fn state_set(ord: i64, key_ptr: *const u8, key_len: u32, value_ptr: *const u8, value_len: u32);
    fn state_get_first(store_idx: u32, key_ptr: *const u8, key_len: u32, output_ptr: u32) -> u32;
    fn state_get_last(store_idx: u32, key_ptr: *const u8, key_len: u32, output_ptr: u32) -> u32;
    fn state_get_at(
        store_idx: u32,
        ord: i64,
        key_ptr: *const u8,
        key_len: u32,
        output_ptr: u32,
    ) -> u32;
}

pub fn set(ord: i64, key: String, value: Vec<u8>) {
    unsafe {
        state_set(
            ord,
            key.as_ptr(),
            key.len() as u32,
            value.as_ptr(),
            value.len() as u32,
        )
    }
}

pub fn get_at(store_idx: u32, ord: i64, key: String) -> Option<Vec<u8>> {
    unsafe {
        let key_bytes = key.as_bytes();
        let output_ptr = memory::alloc(8);
        let found = state_get_at(
            store_idx,
            ord,
            key_bytes.as_ptr(),
            key_bytes.len() as u32,
            output_ptr as u32,
        );
        return if found == 1 {
            Some(get_output_data(output_ptr))
        } else {
            None
        };
    }
}

pub fn get_last(store_idx: u32, key: String) -> Option<Vec<u8>> {
    unsafe {
        let key_bytes = key.as_bytes();
        let output_ptr = memory::alloc(8);
        let found = state_get_last(
            store_idx,
            key_bytes.as_ptr(),
            key_bytes.len() as u32,
            output_ptr as u32,
        );

        return if found == 1{
            Some(get_output_data(output_ptr))
        } else {
            None
        };
    }
}

pub fn get_first(store_idx: u32, key: String) -> Option<Vec<u8>> {
    unsafe {
        let key_bytes = key.as_bytes();
        let output_ptr = memory::alloc(8);
        let found = state_get_first(
            store_idx,
            key_bytes.as_ptr(),
            key_bytes.len() as u32,
            output_ptr as u32,
        );

        return if found == 1{
            Some(get_output_data(output_ptr))
        } else {
            None
        };
    }
}

fn read_u32_from_heap(output_ptr: *mut u8, len: usize) -> u32 {
    unsafe {
        let value_bytes = slice::from_raw_parts(output_ptr, len);
        let value_raw_bytes: [u8; 4] = value_bytes.try_into().expect("error reading raw bytes");
        return u32::from_le_bytes(value_raw_bytes);
    }
}

fn get_output_data(output_ptr: *mut u8) -> Vec<u8> {
    unsafe {
        let value_ptr: u32 = read_u32_from_heap(output_ptr, 4);
        let value_len: u32 = read_u32_from_heap(output_ptr.add(4), 4);

        return Vec::from_raw_parts(value_ptr as *mut u8, value_len as usize, value_len as usize);
    }
}
