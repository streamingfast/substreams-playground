use std::ops::{Add, Div, Mul};
use std::str;
use std::str::FromStr;

use bigdecimal::{BigDecimal, FromPrimitive};
use num_bigint::BigUint;
use substreams::store;

use crate::event::pcs_event::Event;
use crate::pcs::event::Type::{Burn, Mint, Swap};
use crate::utils::{convert_token_to_decimal, zero_big_decimal};
use crate::{address_pretty, pb, pcs};

pub fn is_pair_created_event(sig: &str) -> bool {
    /* keccak value for PairCreated(address,address,address,uint256) */
    return sig == "0d3648bd0f6ba80134a33ba9275ac585d9d315f0ad8355cddefde31afa28d0e9";
}

pub fn is_pair_approval_event(sig: &str) -> bool {
    /* keccak value for Approval(address,address,uint256) */
    return sig == "8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925";
}

pub fn is_pair_burn_event(sig: &str) -> bool {
    /* keccak value for Burn(address,uint256,uint256,address) */
    return sig == "dccd412f0b1252819cb1fd330b93224ca42612892bb3f4f789976e6d81936496";
}

pub fn is_pair_mint_event(sig: &str) -> bool {
    /* keccak value for Mint(address,uint256,uint256) */
    return sig == "4c209b5fc8ad50758f13e2e1088ba56a560dff690a1c6fef26394f4c03821c4f";
}

pub fn is_pair_swap_event(sig: &str) -> bool {
    /* keccak value for Swap(address,uint256,uint256,uint256,uint256,address) */
    return sig == "d78ad95fa46c994b6551d0da85fc275fe613ce37657fb8d5e3d130840159d822";
}

pub fn is_pair_sync_event(sig: &str) -> bool {
    /* keccak value for Sync(uint112,uint112) */
    return sig == "1c411e9a96e071241c2f21f7726b17ae89e3cab4c78be50e062b03a9fffbbad1";
}

pub fn is_pair_transfer_event(sig: &str) -> bool {
    /* keccak value for Transfer(address,address,uint256) */
    return sig == "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef";
}

pub fn decode_event(log: pb::eth::Log) -> PcsEvent {
    let sig = hex::encode(&log.topics[0]);

    if is_pair_created_event(&sig) {
        return new_pair_created_event(log);
    }

    if is_pair_approval_event(&sig) {
        return new_pair_approval_event(log);
    }

    if is_pair_burn_event(&sig) {
        return new_pair_burn_event(log);
    }

    if is_pair_mint_event(&sig) {
        return new_pair_mint_event(log);
    }

    if is_pair_swap_event(&sig) {
        return new_pair_swap_event(log);
    }

    if is_pair_sync_event(&sig) {
        return new_pair_sync_event(log);
    }

    if is_pair_transfer_event(&sig) {
        return new_pair_transfer_event(log);
    }

    return PcsEvent { event: None };
}

pub fn process_mint(
    mint_id: &str,
    base_event: &mut pcs::Event,
    prices_store: &store::StoreGet,
    pair: &pcs::Pair,
    tr1: Option<&PairTransferEvent>,
    tr2: Option<&PairTransferEvent>,
    pair_mint_event: &PairMintEvent,
    token0_decimals: u64,
    token1_decimals: u64,
) {
    let log_ordinal = pair_mint_event.log_index;
    let (amount0, amount1, amount_usd) = convert_prices(
        &prices_store,
        &log_ordinal,
        &pair_mint_event.amount0,
        &pair_mint_event.amount1,
        &pair,
        token0_decimals,
        token1_decimals,
    );

    base_event.log_ordinal = log_ordinal;

    let mut mint = pcs::Mint {
        id: mint_id.to_string(),
        sender: address_pretty(pair_mint_event.sender.as_slice()),
        to: address_pretty(tr2.unwrap().to.as_slice()),
        fee_to: "".to_string(),
        amount0: amount0.to_string(),
        amount1: amount1.to_string(),
        amount_usd: amount_usd.to_string(),
        liquidity: convert_token_to_decimal(tr2.unwrap().value.as_slice(), &18).to_string(),
        fee_liquidity: "".to_string(),
    };

    if tr1.is_some() {
        if BigUint::from_bytes_be(tr1.unwrap().value.as_slice())
            .ne(&BigUint::from_i32(10000).unwrap())
        {
            mint.fee_to = address_pretty(tr1.unwrap().to.as_slice());
            mint.fee_liquidity =
                convert_token_to_decimal(tr1.unwrap().value.as_slice(), &18).to_string();
        }
    }

    base_event.r#type = Some(Mint(mint));
}

pub fn process_burn(
    burn_id: &str,
    base_event: &mut pcs::Event,
    prices_store: &store::StoreGet,
    pair: &pcs::Pair,
    tr1: Option<&PairTransferEvent>,
    tr2: Option<&PairTransferEvent>,
    pair_burn_event: &PairBurnEvent,
    token0_decimals: u64,
    token1_decimals: u64,
) {
    let log_ordinal = pair_burn_event.log_index;
    let (amount0, amount1, amount_usd) = convert_prices(
        &prices_store,
        &log_ordinal,
        &pair_burn_event.amount0,
        &pair_burn_event.amount1,
        &pair,
        token0_decimals,
        token1_decimals,
    );

    let mut burn = pcs::Burn {
        id: burn_id.to_string(),
        sender: address_pretty(tr2.unwrap().from.as_slice()),
        to: address_pretty(tr2.unwrap().to.as_slice()),
        fee_to: "".to_string(),
        amount0: amount0.to_string(),
        amount1: amount1.to_string(),
        amount_usd: amount_usd.to_string(),
        liquidity: convert_token_to_decimal(tr2.unwrap().value.as_slice(), &18).to_string(),
        fee_liquidity: "".to_string(),
    };

    if tr1.is_some() {
        burn.fee_to = address_pretty(tr1.unwrap().to.as_slice());
        burn.fee_liquidity =
            convert_token_to_decimal(tr1.unwrap().value.as_slice(), &18).to_string();
    }

    base_event.r#type = Some(Burn(burn));
    base_event.log_ordinal = log_ordinal;
}

pub fn process_swap(
    swap_id: &str,
    base_event: &mut pcs::Event,
    prices_store: &store::StoreGet,
    pair: &pcs::Pair,
    pair_swap_event: Option<&PairSwapEvent>,
    from_addr: String,
    token0_decimals: u64,
    token1_decimals: u64,
) {
    let log_ordinal = pair_swap_event.unwrap().log_index;
    let swap_event = pair_swap_event.unwrap();

    let amount0_in = convert_token_to_decimal(
        pair_swap_event.unwrap().amount0_in.as_slice(),
        &token0_decimals,
    );
    let amount1_in = convert_token_to_decimal(
        pair_swap_event.unwrap().amount1_in.as_slice(),
        &token1_decimals,
    );
    let amount0_out = convert_token_to_decimal(
        pair_swap_event.unwrap().amount0_out.as_slice(),
        &token0_decimals,
    );
    let amount1_out = convert_token_to_decimal(
        pair_swap_event.unwrap().amount1_out.as_slice(),
        &token1_decimals,
    );

    let amount0_total = amount0_out.clone().add(amount0_in.clone());
    let amount1_total = amount1_out.clone().add(amount1_in.clone());

    let mut big_decimals_bnb = Vec::new();
    big_decimals_bnb.push(get_derived_price(
        &swap_event.log_index,
        &prices_store,
        "bnb".to_string(),
        &amount0_total,
        &pair.token0_address,
    ));
    big_decimals_bnb.push(get_derived_price(
        &swap_event.log_index,
        &prices_store,
        "bnb".to_string(),
        &amount1_total,
        &pair.token1_address,
    ));

    let mut big_decimals_usd = Vec::new();
    big_decimals_usd.push(get_derived_price(
        &swap_event.log_index,
        &prices_store,
        "usd".to_string(),
        &amount0_total,
        &pair.token0_address,
    ));
    big_decimals_usd.push(get_derived_price(
        &swap_event.log_index,
        &prices_store,
        "usd".to_string(),
        &amount1_total,
        &pair.token1_address,
    ));

    let derived_amount_bnb = average_floats(&big_decimals_bnb);
    let tracked_amount_usd = average_floats(&big_decimals_usd);

    let token0_trade_volume: BigDecimal = amount1_in.clone().add(&amount0_out);
    let token1_trade_volume: BigDecimal = amount0_in.clone().add(&amount1_out);

    let token0_trade_volume_usd: &BigDecimal = &tracked_amount_usd;
    let token1_trade_volume_usd: &BigDecimal = &tracked_amount_usd;

    let volume_usd = &tracked_amount_usd;
    let token0_volume = &amount0_total;
    let token1_volume = &amount1_total;

    let swap = pcs::Swap {
        id: swap_id.to_string(),
        sender: address_pretty(pair_swap_event.unwrap().sender.as_slice()),
        to: address_pretty(pair_swap_event.unwrap().to.as_slice()),
        from: from_addr,
        amount0_in: amount0_in.to_string(),
        amount1_in: amount1_in.to_string(),
        amount0_out: amount0_out.to_string(),
        amount1_out: amount1_out.to_string(),
        amount_bnb: derived_amount_bnb.to_string(),
        amount_usd: tracked_amount_usd.to_string(),
        trade_volume0: token0_trade_volume.to_string(),
        trade_volume1: token1_trade_volume.to_string(),
        trade_volume_usd0: token0_trade_volume_usd.to_string(),
        trade_volume_usd1: token1_trade_volume_usd.to_string(),
        volume_usd: volume_usd.to_string(),
        volume_token0: token0_volume.to_string(),
        volume_token1: token1_volume.to_string(),
        log_address: String::from_utf8(swap_event.clone().log_address).unwrap(),
    };

    base_event.r#type = Some(Swap(swap));
    base_event.log_ordinal = log_ordinal;
}

fn convert_prices(
    prices_store: &store::StoreGet,
    log_ordinal: &u64,
    amount0: &Vec<u8>,
    amount1: &Vec<u8>,
    pair: &pcs::Pair,
    token0_decimals: u64,
    token1_decimals: u64,
) -> (BigDecimal, BigDecimal, BigDecimal) {
    let token0_amount = convert_token_to_decimal(amount0, &token0_decimals);
    let token1_amount = convert_token_to_decimal(amount1, &token1_decimals);

    let derived_bnb0_big_decimal = match prices_store.get_at(
        *log_ordinal,
        &format!("dprice:{}:bnb", pair.token0_address),
    ) {
        None => zero_big_decimal(),
        Some(derived_bnb0_bytes) => {
            BigDecimal::from_str(str::from_utf8(derived_bnb0_bytes.as_slice()).unwrap()).unwrap()
        }
    };

    let derived_bnb1_big_decimal = match prices_store.get_at(
        *log_ordinal,
        &format!("dprice:{}:bnb", pair.token1_address),
    ) {
        None => zero_big_decimal(),
        Some(derived_bnb1_bytes) => {
            BigDecimal::from_str(str::from_utf8(derived_bnb1_bytes.as_slice()).unwrap()).unwrap()
        }
    };

    let usd_price_big_decimal =
        match prices_store.get_at(*log_ordinal, &format!("dprice:usd:bnb")) {
            None => zero_big_decimal(),
            Some(usd_price_bytes) => {
                BigDecimal::from_str(str::from_utf8(usd_price_bytes.as_slice()).unwrap()).unwrap()
            }
        };

    let derived_bnb0_mul_token0_amount = derived_bnb0_big_decimal.mul(&token0_amount);
    let derived_bnb1_mul_token1_amount = derived_bnb1_big_decimal.mul(&token1_amount);

    let sum_derived_bnb = derived_bnb0_mul_token0_amount.add(derived_bnb1_mul_token1_amount);

    let amount_total_usd = sum_derived_bnb.mul(usd_price_big_decimal);

    return (token0_amount, token1_amount, amount_total_usd);
}

fn get_derived_price(
    ord: &u64,
    prices_store: &store::StoreGet,
    derived_token: String,
    token_amount: &BigDecimal,
    token_addr: &String,
) -> Option<BigDecimal> {
    let usd_price_bytes = prices_store
        .get_at(
            *ord,
            &format!("dprice:{}:{}", *token_addr, derived_token),
        )
        .unwrap();
    let usd_price =
        BigDecimal::from_str(str::from_utf8(usd_price_bytes.as_slice()).unwrap()).unwrap();
    if usd_price.eq(&zero_big_decimal()) {
        return None;
    }

    return Some((token_amount).mul(usd_price));
}

fn average_floats(big_decimals: &Vec<Option<BigDecimal>>) -> BigDecimal {
    let mut sum = zero_big_decimal();
    let mut count: f64 = 0.0;
    for big_decimal_option in big_decimals {
        if (*big_decimal_option).is_none() {
            continue;
        }
        sum = sum.add((*big_decimal_option).as_ref().unwrap());
        count = count + 1.0;
    }

    if count.eq(&0.0) {
        return sum;
    }

    return sum.div(BigDecimal::from_f64(count).unwrap());
}

fn new_pair_created_event(log: pb::eth::Log) -> PcsEvent {
    return PcsEvent {
        event: Some(Event::PairCreatedEvent(PairCreatedEvent {
            log_address: log.address,
            log_index: log.block_index as u64,
            token0: Vec::from(&log.topics[1][12..]),
            token1: Vec::from(&log.topics[2][12..]),
            pair: Vec::from(&log.data[12..44]),
        })),
    };
}

fn new_pair_approval_event(log: pb::eth::Log) -> PcsEvent {
    return PcsEvent {
        event: Some(Event::PairApprovalEvent(PairApprovalEvent {
            log_address: log.address,
            log_index: log.block_index as u64,
            owner: Vec::from(&log.topics[1][12..]),
            spender: Vec::from(&log.topics[2][12..]),
            value: Vec::from(&log.data[0..32]),
        })),
    };
}

fn new_pair_burn_event(log: pb::eth::Log) -> PcsEvent {
    return PcsEvent {
        event: Some(Event::PairBurnEvent(PairBurnEvent {
            log_address: log.address,
            log_index: log.block_index as u64,
            sender: Vec::from(&log.topics[1][12..]),
            amount0: Vec::from(&log.data[0..32]),
            amount1: Vec::from(&log.data[32..64]),
            to: Vec::from(&log.topics[2][12..]),
        })),
    };
}

fn new_pair_mint_event(log: pb::eth::Log) -> PcsEvent {
    return PcsEvent {
        event: Some(Event::PairMintEvent(PairMintEvent {
            log_address: log.address,
            log_index: log.block_index as u64,
            sender: Vec::from(&log.topics[1][12..]),
            amount0: Vec::from(&log.data[0..32]),
            amount1: Vec::from(&log.data[32..64]),
        })),
    };
}

fn new_pair_swap_event(log: pb::eth::Log) -> PcsEvent {
    return PcsEvent {
        event: Some(Event::PairSwapEvent(PairSwapEvent {
            log_address: log.address,
            log_index: log.block_index as u64,
            sender: Vec::from(&log.topics[1][12..]),
            amount0_in: Vec::from(&log.data[0..32]),
            amount1_in: Vec::from(&log.data[32..64]),
            amount0_out: Vec::from(&log.data[64..96]),
            amount1_out: Vec::from(&log.data[96..128]),
            to: Vec::from(&log.topics[2][12..]),
        })),
    };
}

fn new_pair_sync_event(log: pb::eth::Log) -> PcsEvent {
    return PcsEvent {
        event: Some(Event::PairSyncEvent(PairSyncEvent {
            log_address: log.address,
            log_index: log.block_index as u64,
            reserve0: Vec::from(&log.data[0..32]),
            reserve1: Vec::from(&log.data[32..64]),
        })),
    };
}

fn new_pair_transfer_event(log: pb::eth::Log) -> PcsEvent {
    return PcsEvent {
        event: Some(Event::PairTransferEvent(PairTransferEvent {
            log_address: log.address,
            log_index: log.block_index as u64,
            from: Vec::from(&log.topics[1][12..]),
            to: Vec::from(&log.topics[2][12..]),
            value: Vec::from(&log.data[0..32]),
        })),
    };
}

/* ---- Structs definition ---- */
#[derive(Clone, PartialEq)]
pub struct PcsEvent {
    pub event: Option<Event>,
}

pub mod pcs_event {
    #[derive(Clone, PartialEq)]
    pub enum Event {
        PairCreatedEvent(super::PairCreatedEvent),
        PairApprovalEvent(super::PairApprovalEvent),
        PairBurnEvent(super::PairBurnEvent),
        PairMintEvent(super::PairMintEvent),
        PairSwapEvent(super::PairSwapEvent),
        PairSyncEvent(super::PairSyncEvent),
        PairTransferEvent(super::PairTransferEvent),
    }
}

#[derive(Clone, PartialEq)]
pub struct PairCreatedEvent {
    pub log_address: Vec<u8>,
    pub log_index: u64,
    pub token0: Vec<u8>,
    pub token1: Vec<u8>,
    pub pair: Vec<u8>,
}

#[derive(Clone, PartialEq)]
pub struct PairApprovalEvent {
    pub log_address: Vec<u8>,
    pub log_index: u64,
    pub owner: Vec<u8>,
    pub spender: Vec<u8>,
    pub value: Vec<u8>,
}

#[derive(Clone, PartialEq)]
pub struct PairBurnEvent {
    pub log_address: Vec<u8>,
    pub log_index: u64,
    pub sender: Vec<u8>,
    pub amount0: Vec<u8>,
    pub amount1: Vec<u8>,
    pub to: Vec<u8>,
}

#[derive(Clone, PartialEq)]
pub struct PairMintEvent {
    pub log_address: Vec<u8>,
    pub log_index: u64,
    pub sender: Vec<u8>,
    pub amount0: Vec<u8>,
    pub amount1: Vec<u8>,
}

#[derive(Clone, PartialEq)]
pub struct PairSwapEvent {
    pub log_address: Vec<u8>,
    pub log_index: u64,
    pub sender: Vec<u8>,
    pub amount0_in: Vec<u8>,
    pub amount1_in: Vec<u8>,
    pub amount0_out: Vec<u8>,
    pub amount1_out: Vec<u8>,
    pub to: Vec<u8>,
}

#[derive(Clone, PartialEq)]
pub struct PairSyncEvent {
    pub log_address: Vec<u8>,
    pub log_index: u64,
    pub reserve0: Vec<u8>,
    pub reserve1: Vec<u8>,
}

#[derive(Clone, PartialEq)]
pub struct PairTransferEvent {
    pub log_address: Vec<u8>,
    pub log_index: u64,
    pub from: Vec<u8>,
    pub to: Vec<u8>,
    pub value: Vec<u8>,
}
