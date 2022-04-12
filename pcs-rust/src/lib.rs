extern crate core;

use std::ops::{Add, Mul};
use std::str::FromStr;

use bigdecimal::BigDecimal;
use hex;
use substreams::{log, proto, state};

use eth::{address_pretty, decode_string, decode_uint32};

use crate::event::pcs_event::Event;
use crate::event::{PcsEvent, Wrapper};
use crate::pb::pcs;
use crate::pcs::event::Type;
use crate::utils::zero_big_decimal;

mod db;
mod eth;
mod event;
mod macros;
mod pb;
mod rpc;
mod state_helper;
mod utils;

#[no_mangle]
pub extern "C" fn map_pairs(block_ptr: *mut u8, block_len: usize) {
    substreams::register_panic_hook();

    let blk: pb::eth::Block = proto::decode_ptr(block_ptr, block_len).unwrap();
    let mut pairs = pb::pcs::Pairs { pairs: vec![] };

    log::println(format!(
        "transaction traces count: {}, len: {}",
        blk.transaction_traces.len(),
        block_len
    ));

    for trx in blk.transaction_traces {
        /* PCS Factory address */
        if hex::encode(&trx.to) != "ca143ce32fe78f1f7019d7d551a6402fc5350c73" {
            continue;
        }

        for log in trx.receipt.unwrap().logs {
            let sig = hex::encode(&log.topics[0]);

            if !event::is_pair_created_event(sig.as_str()) {
                continue;
            }

            pairs.pairs.push(pb::pcs::Pair {
                address: address_pretty(&log.data[12..32]),
                token0_address: address_pretty(&log.topics[1][12..]),
                token1_address: address_pretty(&log.topics[2][12..]),
                creation_transaction_id: address_pretty(&trx.hash),
                block_num: blk.number,
                log_ordinal: log.block_index as u64,
            })
        }
    }

    if pairs.pairs.len() != 0 {
        substreams::output(pairs);
    }
}

#[no_mangle]
pub extern "C" fn build_pairs_state(pairs_ptr: *mut u8, pairs_len: usize) {
    substreams::register_panic_hook();

    let pairs: pb::pcs::Pairs = proto::decode_ptr(pairs_ptr, pairs_len).unwrap();

    for pair in pairs.pairs {
        state::set(
            pair.log_ordinal as i64,
            format!("pair:{}", pair.address),
            &proto::encode(&pair).unwrap(),
        );
        state::set(
            pair.log_ordinal as i64,
            format!(
                "tokens:{}",
                utils::generate_tokens_key(
                    pair.token0_address.as_str(),
                    pair.token1_address.as_str(),
                )
            ),
            &Vec::from(pair.address),
        )
    }
}

#[no_mangle]
pub extern "C" fn map_reserves(
    block_ptr: *mut u8,
    block_len: usize,
    pairs_store_idx: u32,
    tokens_store_idx: u32,
) {
    substreams::register_panic_hook();

    let blk: pb::eth::Block = proto::decode_ptr(block_ptr, block_len).unwrap();

    let mut reserves = pb::pcs::Reserves { reserves: vec![] };

    for trx in blk.transaction_traces {
        for log in trx.receipt.unwrap().logs {
            let addr = address_pretty(&log.address);
            match state::get_last(pairs_store_idx, &format!("pair:{}", addr)) {
                None => continue,
                Some(pair_bytes) => {
                    let sig = hex::encode(&log.topics[0]);

                    if !event::is_pair_sync_event(sig.as_str()) {
                        continue;
                    }

                    // Unmarshall pair
                    let pair: pb::pcs::Pair = proto::decode(pair_bytes).unwrap();

                    // reserve
                    let token0: pb::tokens::Token =
                        utils::get_last_token(tokens_store_idx, &pair.token0_address);
                    let reserve0 =
                        utils::convert_token_to_decimal(&log.data[0..32], &token0.decimals);
                    let token1: pb::tokens::Token =
                        utils::get_last_token(tokens_store_idx, &pair.token1_address);
                    let reserve1 =
                        utils::convert_token_to_decimal(&log.data[32..64], &token1.decimals);

                    // token_price
                    let token0_price = utils::get_token_price(reserve0.clone(), reserve1.clone());
                    let token1_price = utils::get_token_price(reserve1.clone(), reserve0.clone());

                    reserves.reserves.push(pb::pcs::Reserve {
                        pair_address: pair.address,
                        reserve0: reserve0.to_string(),
                        reserve1: reserve1.to_string(), // need to trim leading zeros
                        log_ordinal: log.block_index as u64,
                        token0_price: token0_price.to_string(),
                        token1_price: token1_price.to_string(),
                    });
                }
            }
        }
    }
    if reserves.reserves.len() != 0 {
        substreams::output(reserves)
    }
}

#[no_mangle]
pub extern "C" fn build_reserves_state(
    // todo: replace with BlockClock
    block_ptr: *mut u8,
    block_len: usize,
    reserves_ptr: *mut u8,
    reserves_len: usize,
    pairs_store_idx: u32,
) {
    substreams::register_panic_hook();

    let block: pb::eth::Block = proto::decode_ptr(block_ptr, block_len).unwrap();
    let timestamp_block_header: pb::eth::BlockHeader = block.header.unwrap();
    let timestamp = timestamp_block_header.timestamp.unwrap();
    let timestamp_seconds = timestamp.seconds;

    let day_id: i64 = timestamp_seconds / 86400;
    let hour_id: i64 = timestamp_seconds / 3600;

    let reserves: pb::pcs::Reserves = proto::decode_ptr(reserves_ptr, reserves_len).unwrap();

    state::delete_prefix(0, &format!("pair_day:{}:", day_id - 1));
    state::delete_prefix(0, &format!("pair_hour:{}:", hour_id - 1));

    for reserve in reserves.reserves {
        match state::get_last(pairs_store_idx, &format!("pair:{}", reserve.pair_address)) {
            None => continue,
            Some(pair_bytes) => {
                let pair: pb::pcs::Pair = proto::decode(pair_bytes).unwrap();

                state::set(
                    reserve.log_ordinal as i64,
                    format!("price:{}:{}", pair.token0_address, pair.token1_address),
                    &Vec::from(reserve.token0_price),
                );
                state::set(
                    reserve.log_ordinal as i64,
                    format!("price:{}:{}", pair.token1_address, pair.token0_address),
                    &Vec::from(reserve.token1_price),
                );

                state_helper::set_many(
                    reserve.log_ordinal,
                    &vec![
                        format!("reserve:{}:{}", reserve.pair_address, pair.token0_address),
                        format!("pair_day:{}:{}:reserve", day_id, pair.token0_address),
                        format!("pair_hour:{}:{}:reserve", hour_id, pair.token0_address),
                    ],
                    &Vec::from(reserve.reserve0),
                );

                state_helper::set_many(
                    reserve.log_ordinal,
                    &vec![
                        format!("reserve:{}:{}", reserve.pair_address, pair.token1_address),
                        format!("pair_day:{}:{}:reserve", day_id, pair.token1_address),
                        format!("pair_hour:{}:{}:reserve", hour_id, pair.token1_address),
                    ],
                    &Vec::from(reserve.reserve1),
                )
            }
        }
    }
}

#[no_mangle]
pub extern "C" fn build_prices_state(
    // todo: replace with BlockClock
    block_ptr: *mut u8,
    block_len: usize,
    reserves_ptr: *mut u8,
    reserves_len: usize,
    pairs_store_idx: u32,
    reserves_store_idx: u32,
) {
    substreams::register_panic_hook();

    let block: pb::eth::Block = proto::decode_ptr(block_ptr, block_len).unwrap();
    let timestamp_block_header: pb::eth::BlockHeader = block.header.unwrap();
    let timestamp = timestamp_block_header.timestamp.unwrap();
    let timestamp_seconds = timestamp.seconds;

    let day_id: i64 = timestamp_seconds / 86400;
    let hour_id: i64 = timestamp_seconds / 3600;

    let reserves: pb::pcs::Reserves = proto::decode_ptr(reserves_ptr, reserves_len).unwrap();

    state::delete_prefix(0, &format!("pair_day:{}:", day_id - 1));
    state::delete_prefix(0, &format!("pair_hour:{}:", hour_id - 1));
    state::delete_prefix(0, &format!("token_day:{}:", day_id - 1));

    for reserve in reserves.reserves {
        match state::get_last(pairs_store_idx, &format!("pair:{}", reserve.pair_address)) {
            None => continue,
            Some(pair_bytes) => {
                let pair: pb::pcs::Pair = proto::decode(pair_bytes).unwrap();

                let latest_usd_price: BigDecimal =
                    utils::compute_usd_price(&reserve, reserves_store_idx);

                if reserve.pair_address.eq(&utils::USDT_WBNB_PAIR)
                    || reserve.pair_address.eq(&utils::BUSD_WBNB_PAIR)
                {
                    state::set(
                        reserve.log_ordinal as i64,
                        format!("dprice:usd:bnb"),
                        &Vec::from(latest_usd_price.to_string()),
                    )
                }

                // sets:
                // * dprice:%s:bnb (tokenA)  - as contributed by any pair's sync to that token
                // * dprice:%s:usd (tokenA)  - same
                // * dreserve:%s:%s:bnb (pair, token)
                // * dreserve:%s:%s:usd (pair, token)
                // * dreserves:%s:bnb (pair)  - sum of both token's reserves
                // derived from:
                // * price:%s:%s (tokenA, tokenB)
                // * reserve:%s:%s (pair, tokenA)
                let usd_price_valid: bool = latest_usd_price.ne(&utils::zero_big_decimal());

                let t0_derived_bnb_price = utils::find_bnb_price_per_token(
                    &reserve.log_ordinal,
                    pair.token0_address.as_str(),
                    pairs_store_idx,
                    reserves_store_idx,
                );

                let t1_derived_bnb_price = utils::find_bnb_price_per_token(
                    &reserve.log_ordinal,
                    pair.token1_address.as_str(),
                    pairs_store_idx,
                    reserves_store_idx,
                );

                let apply = |token_derived_bnb_price: Option<BigDecimal>,
                             token_addr: String,
                             reserve_amount: String|
                 -> BigDecimal {
                    if token_derived_bnb_price.is_none() {
                        return utils::zero_big_decimal();
                    }

                    state::set(
                        reserve.log_ordinal.clone() as i64,
                        format!("dprice:{}:bnb", token_addr),
                        &Vec::from(token_derived_bnb_price.clone().unwrap().to_string()),
                    );
                    let reserve_in_bnb = BigDecimal::from_str(reserve_amount.as_str())
                        .unwrap()
                        .mul(token_derived_bnb_price.clone().unwrap());
                    state::set(
                        reserve.log_ordinal as i64,
                        format!("dreserve:{}:{}:bnb", reserve.pair_address, token_addr),
                        &Vec::from(reserve_in_bnb.clone().to_string()),
                    );

                    if usd_price_valid {
                        let derived_usd_price = token_derived_bnb_price
                            .unwrap()
                            .mul(latest_usd_price.clone());
                        state_helper::set_many(
                            reserve.log_ordinal,
                            &vec![
                                format!("dprice:{}:usd", token_addr),
                                format!("token_day:{}:dprice:{}:usd", day_id, token_addr),
                            ],
                            &Vec::from(derived_usd_price.to_string()),
                        );

                        let reserve_in_usd = reserve_in_bnb.clone().mul(latest_usd_price.clone());

                        state_helper::set_many(
                            reserve.log_ordinal,
                            &vec![
                                format!("dreserve:{}:{}:usd", reserve.pair_address, token_addr),
                                format!("pair_day:{}:dreserve:{}:usd", day_id, pair.token0_address),
                                format!("pair_day:{}:dreserve:{}:usd", day_id, pair.token1_address),
                                format!(
                                    "pair_hour:{}:dreserve:{}:usd",
                                    hour_id, pair.token0_address
                                ),
                                format!(
                                    "pair_hour:{}:dreserve:{}:usd",
                                    hour_id, pair.token1_address
                                ),
                            ],
                            &Vec::from(reserve_in_usd.to_string()),
                        );
                    }

                    return reserve_in_bnb;
                };

                let reserve0_bnb = apply(
                    t0_derived_bnb_price,
                    pair.token0_address.clone(),
                    reserve.reserve0.clone(),
                );
                let reserve1_bnb = apply(
                    t1_derived_bnb_price,
                    pair.token1_address.clone(),
                    reserve.reserve1.clone(),
                );

                let reserves_bnb_sum = reserve0_bnb.mul(reserve1_bnb);
                if reserves_bnb_sum.ne(&utils::zero_big_decimal()) {
                    state::set(
                        reserve.log_ordinal as i64,
                        format!("dreserves:{}:bnb", reserve.pair_address),
                        &Vec::from(reserves_bnb_sum.to_string()),
                    );
                }
            }
        }
    }
}

#[no_mangle]
pub extern "C" fn map_mint_burn_swaps(
    block_ptr: *mut u8,
    block_len: usize,
    pairs_store_idx: u32,
    prices_store_idx: u32,
    tokens_store_idx: u32,
) {
    substreams::register_panic_hook();

    let blk: pb::eth::Block = proto::decode_ptr(block_ptr, block_len).unwrap();

    let mut events: pb::pcs::Events = pb::pcs::Events { events: vec![] };

    let mut burn_count: i32 = 0;
    let mut mint_count: i32 = 0;
    let mut swap_count: i32 = 0;

    for trx in blk.transaction_traces {
        let trx_id = eth::address_pretty(trx.hash.as_slice());
        for call in trx.calls {
            if call.state_reverted {
                continue;
            }

            if call.logs.len() == 0 {
                continue;
            }

            let pair_addr = eth::address_pretty(call.address.as_slice());

            let pair: pcs::Pair;
            match state::get_last(pairs_store_idx, &format!("pair:{}", pair_addr)) {
                None => continue,
                Some(pair_bytes) => pair = proto::decode(pair_bytes).unwrap(),
            }

            let mut pcs_events: Vec<PcsEvent> = Vec::new();

            for log in call.logs {
                pcs_events.push(event::decode_event(log));
            }

            let mut base_event = pcs::Event {
                log_ordinal: 0,
                pair_address: pair_addr,
                token0: pair.token0_address.clone(),
                token1: pair.token1_address.clone(),
                transaction_id: trx_id.to_string(),
                timestamp: blk
                    .header
                    .as_ref()
                    .unwrap()
                    .timestamp
                    .as_ref()
                    .unwrap()
                    .seconds as u64,
                r#type: None,
            };
            if pcs_events.len() == 4 {
                let ev_tr1 = match pcs_events[0].event.as_ref().unwrap() {
                    Event::PairTransferEvent(pair_transfer_event) => Some(pair_transfer_event),
                    _ => None,
                };

                let ev_tr2 = match pcs_events[1].event.as_ref().unwrap() {
                    Event::PairTransferEvent(pair_transfer_event) => Some(pair_transfer_event),
                    _ => None,
                };

                match pcs_events[3].event.as_ref().unwrap() {
                    Event::PairMintEvent(pair_mint_event) => {
                        let mint_id = format!("{}-{}", trx_id, mint_count);
                        mint_count += 1;

                        event::process_mint(
                            mint_id.as_str(),
                            &mut base_event,
                            prices_store_idx,
                            &pair,
                            ev_tr1,
                            ev_tr2,
                            pair_mint_event,
                            utils::get_last_token(tokens_store_idx, pair.token0_address.as_str())
                                .decimals,
                            utils::get_last_token(tokens_store_idx, pair.token1_address.as_str())
                                .decimals,
                        )
                    }
                    Event::PairBurnEvent(pair_burn_event) => {
                        let burn_id = format!("{}-{}", trx_id, burn_count);
                        burn_count = burn_count + 1;

                        event::process_burn(
                            burn_id.as_str(),
                            &mut base_event,
                            prices_store_idx,
                            &pair,
                            ev_tr1,
                            ev_tr2,
                            pair_burn_event,
                            utils::get_last_token(tokens_store_idx, pair.token0_address.as_str())
                                .decimals,
                            utils::get_last_token(tokens_store_idx, pair.token1_address.as_str())
                                .decimals,
                        );
                    }
                    _ => log::println(format!("Error?! Events len is 4")), // fixme: maybe panic with a different message, not sure if this is good.
                }
            } else if pcs_events.len() == 3 {
                let ev_tr2 = match pcs_events[0].event.as_ref().unwrap() {
                    Event::PairTransferEvent(pair_transfer_event) => Some(pair_transfer_event),
                    _ => None,
                };

                match pcs_events[2].event.as_ref().unwrap() {
                    Event::PairMintEvent(pair_mint_event) => {
                        let mint_id = format!("{}-{}", trx_id, mint_count);
                        mint_count += 1;

                        event::process_mint(
                            mint_id.as_str(),
                            &mut base_event,
                            prices_store_idx,
                            &pair,
                            None,
                            ev_tr2,
                            pair_mint_event,
                            utils::get_last_token(tokens_store_idx, pair.token0_address.as_str())
                                .decimals,
                            utils::get_last_token(tokens_store_idx, pair.token1_address.as_str())
                                .decimals,
                        )
                    }
                    Event::PairBurnEvent(pair_burn_event) => {
                        let burn_id = format!("{}-{}", trx_id, burn_count);
                        burn_count += 1;

                        event::process_burn(
                            burn_id.as_str(),
                            &mut base_event,
                            prices_store_idx,
                            &pair,
                            None,
                            ev_tr2,
                            pair_burn_event,
                            utils::get_last_token(tokens_store_idx, pair.token0_address.as_str())
                                .decimals,
                            utils::get_last_token(tokens_store_idx, pair.token1_address.as_str())
                                .decimals,
                        );
                    }
                    _ => log::println(format!("Error?! Events len is 3")), // fixme: maybe panic with a different message
                }
            } else if pcs_events.len() == 2 {
                match pcs_events[1].event.as_ref().unwrap() {
                    Event::PairSwapEvent(pair_swap_event) => {
                        let swap_id = format!("{}-{}", trx_id, swap_count);
                        swap_count += 1;

                        event::process_swap(
                            swap_id.as_str(),
                            &mut base_event,
                            prices_store_idx,
                            &pair,
                            Some(pair_swap_event),
                            eth::address_pretty(trx.from.as_slice()),
                            utils::get_last_token(tokens_store_idx, &pair.token0_address).decimals,
                            utils::get_last_token(tokens_store_idx, &pair.token1_address).decimals,
                        );
                    }
                    _ => log::println(format!("Error?! Events len is 2")),
                }
            } else if pcs_events.len() == 1 {
                match pcs_events[0].event.as_ref().unwrap() {
                    Event::PairTransferEvent(_) => {
                        log::println("Events len 1, PairTransferEvent".to_string())
                    } // do nothing
                    Event::PairApprovalEvent(_) => {
                        log::println("Events len 1, PairApprovalEvent".to_string())
                    } // do nothing
                    _ => panic!("unhandled event pattern, with 1 event"),
                };
            } else {
                panic!("unhandled event pattern with {} events", pcs_events.len());
            }

            events.events.push(base_event);
        }
    }

    substreams::output(events)
}

#[no_mangle]
pub extern "C" fn build_totals_state(
    block_ptr: *mut u8,
    block_len: usize,
    pairs_ptr: *mut u8,
    pairs_len: usize,
    events_ptr: *mut u8,
    events_len: usize,
) {
    substreams::register_panic_hook();

    let block: pb::eth::Block = proto::decode_ptr(block_ptr, block_len).unwrap();
    let timestamp_block_header: pb::eth::BlockHeader = block.header.unwrap();
    let timestamp = timestamp_block_header.timestamp.unwrap();
    let timestamp_seconds = timestamp.seconds;

    let day_id: i64 = timestamp_seconds / 86400;

    if events_len == 0 && pairs_len == 0 {
        return;
    }

    let events: pb::pcs::Events = proto::decode_ptr(events_ptr, events_len).unwrap();
    let pairs: pb::pcs::Pairs = proto::decode_ptr(pairs_ptr, pairs_len).unwrap();

    let mut all_pairs_and_events: Vec<Wrapper> = Vec::new();

    for pair in pairs.pairs {
        all_pairs_and_events.push(Wrapper::Pair(pair));
    }

    for event in events.events {
        all_pairs_and_events.push(Wrapper::Event(event));
    }

    all_pairs_and_events.sort_by(|a, b| utils::get_ordinal(a).cmp(&utils::get_ordinal(b)));

    // move stuff here
    for el in all_pairs_and_events {
        match el {
            Wrapper::Event(event) => {
                state::sum_int64(
                    event.log_ordinal as i64,
                    format!("token:{}:transaction_count", event.token0),
                    1,
                );
                state::sum_int64(
                    event.log_ordinal as i64,
                    format!("token:{}:transaction_count", event.token1),
                    1,
                );
                state::sum_int64(
                    event.log_ordinal as i64,
                    format!("pair:{}:transaction_count", event.pair_address),
                    1,
                );

                match event.r#type.unwrap() {
                    Type::Swap(swap) => {
                        if swap.amount_usd.is_empty() {
                            continue;
                        }

                        state_helper::sum_int64_many(
                            event.log_ordinal,
                            &&vec![
                                format!("pair:{}:swap_count", event.pair_address),
                                format!("pair:{}:total_transactions", event.pair_address),
                                format!("token:{}:total_transactions", event.token0),
                                format!("token:{}:total_transactions", event.token1),
                                format!("global_day:{}:total_transactions", day_id),
                                format!("global:total_transactions"),
                            ],
                            1,
                        );

                        //todo: if we want to set the total transactions for global day we need a
                        // key setter store to keep track of the latest computed(summed) values
                    }
                    Type::Burn(_) => state::sum_int64(
                        event.log_ordinal as i64,
                        format!("pair:{}:burn_count", event.pair_address),
                        1,
                    ),
                    Type::Mint(_) => state::sum_int64(
                        event.log_ordinal as i64,
                        format!("pair:{}:mint_count", event.pair_address),
                        1,
                    ),
                }
            }
            Wrapper::Pair(pair) => {
                state::sum_int64(pair.log_ordinal as i64, "global:pair_count".to_string(), 1);
            }
        }
    }
}

#[no_mangle]
pub extern "C" fn build_volumes_state(
    block_ptr: *mut u8,
    block_len: usize,
    events_ptr: *mut u8,
    events_len: usize,
) {
    substreams::register_panic_hook();

    let blk: pb::eth::Block = proto::decode_ptr(block_ptr, block_len).unwrap();
    let timestamp_block_header: pb::eth::BlockHeader = blk.header.unwrap();
    let timestamp = timestamp_block_header.timestamp.unwrap();
    let timestamp_seconds = timestamp.seconds;
    let day_id: i64 = timestamp_seconds / 86400;
    let hour_id: i64 = timestamp_seconds / 3600;

    if events_len == 0 {
        return;
    }

    let events: pb::pcs::Events = proto::decode_ptr(events_ptr, events_len).unwrap();

    state::delete_prefix(0, &format!("pair_day:{}:", day_id - 1));
    state::delete_prefix(0, &format!("token_day:{}:", day_id - 1));
    state::delete_prefix(0, &format!("pair_hour:{}:", hour_id - 1));
    state::delete_prefix(0, &format!("global_day:{}", day_id - 1));

    for event in events.events {
        if event.r#type.is_some() {
            match event.r#type.unwrap() {
                Type::Mint(mint) => {
                    // state::sum("global:liquidity_usd", mint.amount_usd)
                    // sum("token:{}:liquidity_usd", mint.to)
                    //state::sum_bigfloat(event.log_ordinal as i32, format!("pair:{}:total_supply", event.pair_address), mint.liquidity)
                }
                Type::Burn(burn) => {
                    // sum("global:liquidity_usd", /* NEGATIVE */ -burn.amount_usd)
                    // sum(token:{}:liquidity_usd", /* NEGATIVE */ burn.to)
                    //state::sum_bigfloat(event.log_ordinal as i32, format!("pair:{}:total_supply", event.pair_address), mint.liquidity) samue but negative
                }
                Type::Swap(swap) => {
                    if swap.amount_usd.is_empty() {
                        continue;
                    }
                    let amount_usd = BigDecimal::from_str(swap.amount_usd.as_str()).unwrap();
                    if amount_usd.eq(&zero_big_decimal()) {
                        continue;
                    }
                    let amount_bnb = BigDecimal::from_str(swap.amount_bnb.as_str()).unwrap();

                    let amount_0_total: BigDecimal =
                        utils::compute_amount_total(swap.amount0_out, swap.amount0_in);
                    let amount_1_total: BigDecimal =
                        utils::compute_amount_total(swap.amount1_out, swap.amount1_in);

                    state_helper::sum_bigfloat_many(
                        event.log_ordinal,
                        &vec![
                            format!("pair_day:{}:{}:usd", day_id, event.pair_address),
                            format!("pair_hour:{}:{}:usd", hour_id, event.pair_address),
                            format!("pair:{}:usd", event.pair_address),
                            format!("token_day:{}:{}:usd", day_id, event.token0),
                            format!("token_day:{}:{}:usd", day_id, event.token1),
                            format!("global:usd"),
                            format!("global_day:{}:usd", day_id),
                        ],
                        &amount_usd,
                    );

                    state_helper::sum_bigfloat_many(
                        event.log_ordinal,
                        &vec![format!("global:bnb"), format!("global_day:{}:bnb", day_id)],
                        &amount_bnb,
                    );

                    state::sum_bigfloat(
                        event.log_ordinal as i64,
                        format!("pair:{}:token0", event.pair_address),
                        &amount_0_total,
                    );
                    state::sum_bigfloat(
                        event.log_ordinal as i64,
                        format!("pair:{}:token1", event.pair_address),
                        &amount_1_total,
                    );
                    state::sum_bigfloat(
                        event.log_ordinal as i64,
                        format!("token:{}:trade", event.token0),
                        &BigDecimal::from_str(swap.trade_volume0.as_str()).unwrap(),
                    );
                    state::sum_bigfloat(
                        event.log_ordinal as i64,
                        format!("token:{}:trade", event.token1),
                        &BigDecimal::from_str(swap.trade_volume1.as_str()).unwrap(),
                    );
                    state::sum_bigfloat(
                        event.log_ordinal as i64,
                        format!("token:{}:trade_usd", event.token0),
                        &BigDecimal::from_str(swap.trade_volume_usd0.as_str()).unwrap(),
                    );
                    state::sum_bigfloat(
                        event.log_ordinal as i64,
                        format!("token:{}:trade_usd", event.token1),
                        &BigDecimal::from_str(swap.trade_volume_usd1.as_str()).unwrap(),
                    );
                }
            }
        }
    }
}

#[no_mangle]
pub extern "C" fn map_to_database(
    block_ptr: *mut u8,
    block_len: usize,
    tokens_deltas_ptr: *mut u8,
    tokens_deltas_len: usize,
    pairs_deltas_ptr: *mut u8,
    pairs_deltas_len: usize,
    totals_deltas_ptr: *mut u8,
    totals_deltas_len: usize,
    volumes_deltas_ptr: *mut u8,
    volumes_deltas_len: usize,
    reserves_ptr: *mut u8,
    reserves_len: usize,
    events_ptr: *mut u8,
    events_len: usize,
    tokens_idx: u32,
) {
    substreams::register_panic_hook();

    let block: pb::eth::Block = proto::decode_ptr(block_ptr, block_len).unwrap();

    let token_deltas: substreams::pb::substreams::StoreDeltas =
        proto::decode_ptr(tokens_deltas_ptr, tokens_deltas_len).unwrap();

    let pair_deltas: substreams::pb::substreams::StoreDeltas =
        proto::decode_ptr(pairs_deltas_ptr, pairs_deltas_len).unwrap();

    let totals_deltas: substreams::pb::substreams::StoreDeltas =
        proto::decode_ptr(totals_deltas_ptr, totals_deltas_len).unwrap();

    let volumes_deltas: substreams::pb::substreams::StoreDeltas =
        proto::decode_ptr(volumes_deltas_ptr, volumes_deltas_len).unwrap();

    let reserves: pb::pcs::Reserves = proto::decode_ptr(reserves_ptr, reserves_len).unwrap();

    let events: pb::pcs::Events = proto::decode_ptr(events_ptr, events_len).unwrap();

    let changes = db::process(
        &block,
        pair_deltas,
        token_deltas,
        totals_deltas,
        volumes_deltas,
        reserves,
        events,
        tokens_idx,
    );

    //todo: call join_sort_deltas
    //todo: loop all NameDeltas in a single loop with a huge match statements

    //
    //
    // for pair_delta in pair_deltas.deltas {
    // }
    //
    // for delta in total_deltas {
    //     if startwith(delta.key:"pairs") {
    //         // // TODO: should we do on client side or create a new store ??
    //         // database_changes.table_changes.push(TableChange {
    //         //     table: "pancake_factory".to_string(),
    //         //     pk: "0xcA143Ce32Fe78f1f7019d7d551a6402fC5350c73".to_string(),
    //         //     operation: Operation::Update as i32,
    //         //     fields: vec![
    //         //         Field {
    //         //             key: "total_pairs".to_string(),
    //         //             new_value: "10".to_string(),
    //         //             old_value: "11".to_string()
    //         //         }
    //         //     ]
    //         // });
    //     }
    // }
    //
    //
    //
    // }

    // for reserve in reserves.reserves {
    //     log::println(format!(
    //         "Reserve: {} {} {} {}",
    //         reserve.pair_address, reserve.log_ordinal, reserve.reserve0, reserve.reserve1
    //     ));
    // }

    substreams::output(changes);
}

#[no_mangle]
pub extern "C" fn block_to_tokens(block_ptr: *mut u8, block_len: usize) {
    substreams::register_panic_hook();

    let mut tokens = pb::tokens::Tokens { tokens: vec![] };
    let blk: pb::eth::Block = proto::decode_ptr(block_ptr, block_len).unwrap();

    for trx in blk.transaction_traces {
        for call in trx.calls {
            if call.call_type == pb::eth::CallType::Create as i32 && !call.state_reverted {
                let mut code_change_len = 0;
                for code_change in &call.code_changes {
                    code_change_len += code_change.new_code.len()
                }
                log::println(format!(
                    "found contract creation: {}, caller {}, code change {}, input {}",
                    address_pretty(&call.address),
                    address_pretty(&call.caller),
                    code_change_len,
                    call.input.len(),
                ));

                if code_change_len <= 150 {
                    // optimization to skip none viable SC
                    log::println(format!(
                        "skipping to small code to be a token contract: {}",
                        address_pretty(&call.address)
                    ));
                    continue;
                }
                if address_pretty(&call.caller) == "0x0000000000004946c0e9f43f4dee607b0ef1fa1c"
                    || address_pretty(&call.caller) == "0x00000000687f5b66638856396bee28c1db0178d1"
                {
                    continue;
                }
                let rpc_calls = rpc::create_rpc_calls(call.clone().address);

                let rpc_responses_marshalled: Vec<u8> =
                    substreams::rpc::eth_call(substreams::proto::encode(&rpc_calls).unwrap());
                let rpc_responses_unmarshalled: substreams::pb::eth::RpcResponses =
                    substreams::proto::decode(rpc_responses_marshalled).unwrap();

                if rpc_responses_unmarshalled.responses[0].failed
                    || rpc_responses_unmarshalled.responses[1].failed
                    || rpc_responses_unmarshalled.responses[2].failed
                {
                    log::println(format!(
                        "not a token because of a failure: {}",
                        address_pretty(&call.address)
                    ));
                    continue;
                };

                if !(rpc_responses_unmarshalled.responses[1].raw.len() >= 96)
                    || rpc_responses_unmarshalled.responses[0].raw.len() != 32
                    || !(rpc_responses_unmarshalled.responses[2].raw.len() >= 96)
                {
                    log::println(format!(
                        "not a token because response length: {}",
                        address_pretty(&call.address)
                    ));
                    continue;
                };

                log::println(format!(
                    "found a token: {} {}",
                    address_pretty(&call.address),
                    decode_string(rpc_responses_unmarshalled.responses[1].raw.as_ref()),
                ));
                let decoded_address = address_pretty(&call.address);
                let decoded_decimals =
                    decode_uint32(rpc_responses_unmarshalled.responses[0].raw.as_ref());
                let decoded_name =
                    decode_string(rpc_responses_unmarshalled.responses[1].raw.as_ref());
                let decoded_symbol =
                    decode_string(rpc_responses_unmarshalled.responses[2].raw.as_ref());

                let token = pb::tokens::Token {
                    address: decoded_address.clone(),
                    name: decoded_name,
                    symbol: decoded_symbol,
                    decimals: decoded_decimals as u64,
                };

                tokens.tokens.push(token);
            }
        }
    }

    substreams::output(tokens);
}

#[no_mangle]
pub extern "C" fn build_tokens_state(tokens_ptr: *mut u8, tokens_len: usize) {
    substreams::register_panic_hook();

    let tokens: pb::tokens::Tokens = proto::decode_ptr(tokens_ptr, tokens_len).unwrap();

    for token in tokens.tokens {
        let key = format!("token:{}", token.address);
        state::set(1, key, &proto::encode(&token).unwrap()); //todo: what about the log ordinal
    }
}
