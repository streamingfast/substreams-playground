use std::ops::Div;
use std::str::FromStr;
use bigdecimal::BigDecimal;
use num_bigint::BigUint;
use pad::PadStr;
use substreams::log;

pub fn is_pair_created_event(sig: String) -> bool {
    /* keccak value for PairCreated(address,address,address,uint256) */
    return sig == "0d3648bd0f6ba80134a33ba9275ac585d9d315f0ad8355cddefde31afa28d0e9";
}

pub fn is_new_pair_sync_event(sig: String) -> bool {
    /* keccak value for Sync(uint112,uint112) */
    return sig == "1c411e9a96e071241c2f21f7726b17ae89e3cab4c78be50e062b03a9fffbbad1";
}

pub fn convert_token_to_decimal(amount: &[u8], decimals: u64) -> BigDecimal {
    let big_uint_amount = BigUint::from_bytes_be(amount);
    log::println(format!("bigUint: {:?}", big_uint_amount));
    log::println(format!("bigUint as String: {:?}", big_uint_amount.to_string()));
    log::println(format!("bigUint as str: {:?}", big_uint_amount.to_string().as_str()));

    let big_float_amount = BigDecimal::from_str(big_uint_amount.to_string().as_str()).unwrap().with_prec(100);
    log::println(format!("big_float_amount {:?}", big_float_amount));

    return divide_by_decimals(big_float_amount, decimals);
}

pub fn get_token_price(bf0: BigDecimal, bf1: BigDecimal) -> BigDecimal {
    return bf0.div(bf1).with_prec(100);
}

pub fn generate_tokens_key(token0: String, token1: String) -> String {
    if token0 > token1 {
        return format!("{}:{}", token1, token0);
    }
    return format!("{}:{}", token0, token1);
}

fn divide_by_decimals(big_float_amount: BigDecimal, decimals: u64) -> BigDecimal{
    let bd = BigDecimal::from_str("1".pad_to_width_with_char((decimals + 1) as usize, '0').as_str()).unwrap().with_prec(100);
    log::println(format!("bd with path: {:?}", bd));

    return big_float_amount.div(bd).with_prec(100)
}
