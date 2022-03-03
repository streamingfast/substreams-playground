extern "C" {
    fn state_set(ord: i64, key_ptr: *const u8, key_len: u32, value_ptr: *const u8, value_len: u32);
    fn state_get_first(store_idx: u32, key_ptr: *const u8, key_len: u32) -> (*mut u8, u32, bool);
    fn state_get_last(store_idx: u32, key_ptr: *const u8, key_len: u32) -> (*mut u8, u32, bool);
    fn state_get_at(store_idx: u32, ord: i64, key_ptr: *const u8, key_len: u32);

}

// type RetVal struct {
//     ptr: *mut u8,
//     len: u32,
//     found: bool,
// }

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
        // let (ptr, len, found) = state_get_at(store_idx, ord, key.as_ptr(), key.len() as u32);
        // let input_data = Vec::from_raw_parts(ptr, len as usize, len as usize);
        // if !found {
        return None;
        // }

        // return Some(input_data);
    }
}

pub fn get_last(store_idx: u32, key: String) -> Option<Vec<u8>> {
    unsafe {
        let (ptr, len, found) = state_get_last(store_idx, key.as_ptr(), key.len() as u32);
        let input_data = Vec::from_raw_parts(ptr, len as usize, len as usize);
        if !found {
            return None;
        }

        return Some(input_data);
    }
}

pub fn get_first(store_idx: u32, key: String) -> Option<Vec<u8>> {
    unsafe {
        let (ptr, len, found) = state_get_first(store_idx, key.as_ptr(), key.len() as u32);
        let input_data = Vec::from_raw_parts(ptr, len as usize, len as usize);
        if !found {
            return None;
        }

        return Some(input_data);
    }
}
