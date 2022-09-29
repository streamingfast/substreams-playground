use std::ops::Div;
use std::str::FromStr;
use bigdecimal::BigDecimal;
use num_bigint::BigInt;
use pad::PadStr;
use substreams::errors::Error;
use substreams::{log, proto};
use substreams::store::StoreGet;
use crate::{keyer, pb};

pub fn get_account(account_store: &StoreGet, account_address: &String) -> Result<pb::spl::Account, Error> {
    return match &account_store.get_last(&keyer::account_key(&account_address)) {
        None => {
            log::info!("ERROR");
            Err(Error::Unexpected(format!("account {} not found", account_address).to_string()))
        },
        Some(bytes) => Ok(proto::decode(bytes).unwrap()),
    };
}


pub fn get_mint(mint_store: &StoreGet, mint_address: &String) -> Result<pb::spl::Mint, Error> {
    return match &mint_store.get_last(&keyer::mint_key(&mint_address)) {
        None => Err(Error::Unexpected(format!("mint {} not found", mint_address).to_string())),
        Some(bytes) => Ok(proto::decode(bytes).unwrap()),
    };
}


pub fn convert_token_to_decimal(amount: &BigInt, decimals: u64) -> BigDecimal {
    let big_float_amount = BigDecimal::from_str(amount.to_string().as_str())
        .unwrap()
        .with_prec(100);

    return divide_by_decimals(big_float_amount, decimals);
}


pub fn divide_by_decimals(big_float_amount: BigDecimal, decimals: u64) -> BigDecimal {
    let bd = BigDecimal::from_str(
        "1".pad_to_width_with_char((decimals + 1) as usize, '0')
            .as_str(),
    )
        .unwrap()
        .with_prec(100);
    return big_float_amount.div(bd);
}
