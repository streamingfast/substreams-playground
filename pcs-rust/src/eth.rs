pub fn decode_address(input: &[u8]) -> String {
    format!("0x{}", hex::encode(input))
}
