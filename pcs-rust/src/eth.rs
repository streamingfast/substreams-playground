use std::convert::TryInto;

// should be named: encode_hex(), the encode_address should trim extra zeroes..
pub fn decode_address(input: &[u8]) -> String {
    format!("0x{}", hex::encode(input))
}

pub fn decode_uint32(input: &[u8]) -> u32 {
    let as_array: [u8; 4] = input[28..32].try_into().unwrap();
    u32::from_be_bytes(as_array)
}

pub fn decode_string(input: &[u8]) -> String {
    if input.len() < 96 {
       panic!("input length too small: {}", input.len()); 
    }

    let next = decode_uint32(&input[0..32]);
    if next != 32 {
         panic!("invalid input, first part should be 32"); 
    };

    let size : usize = decode_uint32(&input[32..64]) as usize;
    let end: usize = (size) + 64;

    if end > input.len() {
          panic!("invalid input: end {:?}, length: {:?}, next: {:?}, size: {:?}, whole: {:?}", end, input.len(), next, size, hex::encode(&input[32..64])); 
    }
    
    std::str::from_utf8(&input[64..end]).expect("invalid utf-8 sequence").to_string()
}

