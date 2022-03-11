pub fn address_pretty(input: &[u8]) -> String {
    format!("0x{}", hex::encode(input))
}
