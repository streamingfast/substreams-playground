pub fn decode_address(input: &Vec<u8>) -> String {
    if input.len() > 40 {
        "larger".to_string()
    } else {
        hex::encode(input)
    }
}
