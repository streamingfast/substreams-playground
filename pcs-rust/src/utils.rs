use std::ops::{Add, Div, Mul};
use std::str;
use std::str::FromStr;

use bigdecimal::{BigDecimal, One, Zero};
use num_bigint::BigUint;
use pad::PadStr;
use substreams::{proto, state};

use crate::{pb, Wrapper};

pub const WBNB_ADDRESS: &str = "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c";
pub const BUSD_WBNB_PAIR: &str = "0x58f876857a02d6762e0101bb5c46a8c1ed44dc16";
pub const USDT_WBNB_PAIR: &str = "0x16b9a82891338f9ba80e2d6970fdda79d1eb0dae";
pub const BUSD_PRICE_KEY: &str =
    "price:0xe9e7cea3dedca5984780bafc599bd69add087d56:0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c";
pub const USDT_PRICE_KEY: &str =
    "price:0x55d398326f99059ff775485246999027b3197955:0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c";

const WHITELIST_TOKENS: [&str; 6] = [
    "0xe9e7cea3dedca5984780bafc599bd69add087d56", // BUSD
    "0x55d398326f99059ff775485246999027b3197955", // USDT
    "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d", // USDC
    "0x23396cf899ca06c4472205fc903bdb4de249d6fc", // UST
    "0x7130d2a12b9bcbfae4f2634d864a1ee1ce3ead9c", // BTCB
    "0x2170ed0880ac9a755fd29b2688956bd959f933f8", // WETH
];

pub fn convert_token_to_decimal(amount: &[u8], decimals: &u64) -> BigDecimal {
    let big_uint_amount = BigUint::from_bytes_be(amount);
    let big_float_amount = BigDecimal::from_str(big_uint_amount.to_string().as_str())
        .unwrap()
        .with_prec(100);

    return divide_by_decimals(big_float_amount, decimals);
}

pub fn get_token_price(bf0: BigDecimal, bf1: BigDecimal) -> BigDecimal {
    return bf0.div(bf1).with_prec(100);
}

pub fn generate_tokens_key(token0: &str, token1: &str) -> String {
    if token0 > token1 {
        return format!("{}:{}", token1, token0);
    }
    return format!("{}:{}", token0, token1);
}

// not sure about the & in front of reserve
pub fn compute_usd_price(reserve: &pb::pcs::Reserve, reserves_store_idx: u32) -> BigDecimal {
    let busd_bnb_reserve_big_decimal;
    let usdt_bnb_reserve_big_decimal;

    match state::get_at(
        reserves_store_idx,
        reserve.log_ordinal as i64,
        format!("reserve:{}:{}", BUSD_WBNB_PAIR, WBNB_ADDRESS),
    ) {
        None => busd_bnb_reserve_big_decimal = zero_big_decimal(),
        Some(reserve_bytes) => {
            busd_bnb_reserve_big_decimal = decode_reserve_bytes_to_big_decimal(reserve_bytes)
        }
    }

    match state::get_at(
        reserves_store_idx,
        reserve.log_ordinal as i64,
        format!("reserve:{}:{}", USDT_WBNB_PAIR, WBNB_ADDRESS),
    ) {
        None => usdt_bnb_reserve_big_decimal = zero_big_decimal(),
        Some(reserve_bytes) => {
            usdt_bnb_reserve_big_decimal = decode_reserve_bytes_to_big_decimal(reserve_bytes)
        }
    }

    let mut total_liquidity_bnb = zero_big_decimal();
    total_liquidity_bnb = total_liquidity_bnb
        .clone()
        .add(busd_bnb_reserve_big_decimal.clone());
    total_liquidity_bnb = total_liquidity_bnb
        .clone()
        .add(usdt_bnb_reserve_big_decimal.clone());

    let zero = zero_big_decimal();

    if total_liquidity_bnb.eq(&zero) {
        return zero;
    }

    if busd_bnb_reserve_big_decimal.eq(&zero) {
        return match state::get_at(
            reserves_store_idx,
            reserve.log_ordinal as i64,
            USDT_PRICE_KEY.to_string(),
        ) {
            None => zero,
            Some(reserve_bytes) => decode_reserve_bytes_to_big_decimal(reserve_bytes),
        };
    } else if usdt_bnb_reserve_big_decimal.eq(&zero) {
        return match state::get_at(
            reserves_store_idx,
            reserve.log_ordinal as i64,
            BUSD_PRICE_KEY.to_string(),
        ) {
            None => zero,
            Some(reserve_bytes) => decode_reserve_bytes_to_big_decimal(reserve_bytes),
        };
    }

    // both found and not equal to zero, average out
    let busd_weight = busd_bnb_reserve_big_decimal
        .div(total_liquidity_bnb.clone())
        .with_prec(100);
    let usdt_weight = usdt_bnb_reserve_big_decimal
        .div(total_liquidity_bnb)
        .with_prec(100);

    let busd_price = match state::get_at(
        reserves_store_idx,
        reserve.log_ordinal as i64,
        USDT_PRICE_KEY.to_string(),
    ) {
        None => zero_big_decimal(),
        Some(reserve_bytes) => decode_reserve_bytes_to_big_decimal(reserve_bytes),
    };

    let usdt_price = match state::get_at(
        reserves_store_idx,
        reserve.log_ordinal as i64,
        BUSD_PRICE_KEY.to_string(),
    ) {
        None => zero_big_decimal(),
        Some(reserve_bytes) => decode_reserve_bytes_to_big_decimal(reserve_bytes),
    };

    let busd_price_over_weight = busd_price.mul(busd_weight).with_prec(100);
    let usdt_price_over_weight = usdt_price.mul(usdt_weight).with_prec(100);

    let mut usd_price = zero_big_decimal();
    usd_price = usd_price.add(busd_price_over_weight);
    usd_price = usd_price.add(usdt_price_over_weight);

    usd_price
}

pub fn find_bnb_price_per_token(
    log_ordinal: &u64,
    erc20_token_address: &str,
    pairs_store_idx: u32,
    reserves_store_idx: u32,
) -> Option<BigDecimal> {
    if erc20_token_address.eq(WBNB_ADDRESS) {
        return Option::Some(one_big_decimal()); // BNB price of a BNB is always 1
    }

    let direct_to_bnb_price = match state::get_last(
        reserves_store_idx,
        format!("price:{}:{}", WBNB_ADDRESS, erc20_token_address),
    ) {
        None => zero_big_decimal(),
        Some(reserve_bytes) => decode_reserve_bytes_to_big_decimal(reserve_bytes),
    };

    if direct_to_bnb_price.ne(&zero_big_decimal()) {
        return Option::Some(direct_to_bnb_price);
    }

    // loop all whitelist for a matching pair
    for major_token in WHITELIST_TOKENS {
        let tiny_to_major_pair = match state::get_at(
            pairs_store_idx,
            *log_ordinal as i64,
            format!(
                "tokens:{}",
                generate_tokens_key(erc20_token_address, major_token)
            ),
        ) {
            None => continue,
            Some(pair_bytes) => decode_pair_bytes(pair_bytes),
        };

        let major_to_bnb_price = match state::get_at(
            reserves_store_idx,
            *log_ordinal as i64,
            format!("price:{}:{}", major_token, WBNB_ADDRESS),
        ) {
            None => continue,
            Some(reserve_bytes) => decode_reserve_bytes_to_big_decimal(reserve_bytes),
        };

        let tiny_to_major_price = match state::get_at(
            reserves_store_idx,
            *log_ordinal as i64,
            format!("price:{}:{}", erc20_token_address, major_token),
        ) {
            None => continue,
            Some(reserve_bytes) => decode_reserve_bytes_to_big_decimal(reserve_bytes),
        };

        let major_reserve =
            //todo: not sure about tiny_to_major_pair.erc20_token0.addr, maybe its the token1 ?
            match state::get_at(reserves_store_idx, *log_ordinal as i64, format!("reserve:{}:{}", tiny_to_major_pair, major_token)) {
                None => continue,
                Some(reserve_bytes) => decode_reserve_bytes_to_big_decimal(reserve_bytes)
            };

        let bnb_reserve_in_major_pair = major_to_bnb_price.clone().mul(major_reserve);
        // We're checking for half of it, because `reserves_bnb` would have both sides in it.
        // We could very well check the other reserve's BNB value, would be a bit more heavy, but we can do it.
        if bnb_reserve_in_major_pair.le(&BigDecimal::from_str("5").unwrap()) {
            // todo: little or big ?
            continue; // Not enough liquidity
        }

        return Some(tiny_to_major_price.mul(major_to_bnb_price));
    }

    return None;
}

pub fn zero_big_decimal() -> BigDecimal {
    BigDecimal::zero().with_prec(100)
}

pub fn compute_amount_total(amount1: String, amount2: String) -> BigDecimal {
    let amount1_bd: BigDecimal = BigDecimal::from_str(amount1.as_str()).unwrap();
    let amount2_bd: BigDecimal = BigDecimal::from_str(amount2.as_str()).unwrap();

    amount1_bd.add(amount2_bd)
}

pub fn get_ordinal(all: &Wrapper) -> i64 {
    return match all {
        Wrapper::Event(event) => event.log_ordinal as i64,
        Wrapper::Pair(pair) => pair.log_ordinal as i64,
    };
}

pub fn get_last_token(tokens_store_idx: u32, token_address: &str) -> pb::tokens::Token {
    proto::decode(state::get_last(tokens_store_idx, format!("token:{}", token_address)).unwrap())
        .unwrap()
}

fn one_big_decimal() -> BigDecimal {
    BigDecimal::one().with_prec(100)
}

fn divide_by_decimals(big_float_amount: BigDecimal, decimals: &u64) -> BigDecimal {
    let bd = BigDecimal::from_str(
        "1".pad_to_width_with_char((*decimals + 1) as usize, '0')
            .as_str(),
    )
    .unwrap()
    .with_prec(100);
    return big_float_amount.div(bd).with_prec(100);
}

fn decode_pair_bytes(pair_bytes: Vec<u8>) -> String {
    let pair_from_store_decoded = str::from_utf8(pair_bytes.as_slice()).unwrap();
    return pair_from_store_decoded.to_string();
}

fn decode_reserve_bytes_to_big_decimal(reserve_bytes: Vec<u8>) -> BigDecimal {
    let reserve_from_store_decoded = str::from_utf8(reserve_bytes.as_slice()).unwrap();
    return BigDecimal::from_str(reserve_from_store_decoded)
        .unwrap()
        .with_prec(100);
}
