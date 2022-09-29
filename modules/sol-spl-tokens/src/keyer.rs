// ------------------------------------------------
//      store_accounts
// ------------------------------------------------
pub fn account_key(
    account_address: &String,
) -> String {
    format!("account:{}", account_address)
}

// ------------------------------------------------
//      store_mints
// ------------------------------------------------
pub fn mint_key(
    mint_address: &String,
) -> String {
    format!("mint:{}", mint_address)
}

// ------------------------------------------------
//      store_mint_volumes
// ------------------------------------------------
pub fn native_mint_volume(
    mint_address: &String,
) -> String {
    format!("volume:{}:native", mint_address)
}

pub fn decimal_mint_volume(
    mint_address: &String,
) -> String {
    format!("volume:{}:dec", mint_address)
}