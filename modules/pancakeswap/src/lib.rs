extern crate core;

use std::ops::{Mul, Neg};
use std::str::FromStr;

use bigdecimal::BigDecimal;
use hex;
use substreams::{log, proto, store};

use eth::{address_decode, address_pretty, decode_string, decode_uint32};

use crate::event::pcs_event::Event;
use crate::event::PcsEvent;
use crate::pb::pcs;
use crate::pb::tokens::Token;
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
    log::info!("Pairs mapping");
    substreams::register_panic_hook();

    let blk: pb::eth::Block = proto::decode_ptr(block_ptr, block_len).unwrap();
    let mut pairs = pcs::Pairs { pairs: vec![] };

    for trx in blk.transaction_traces {
        /* PCS Factory address */
        //0xbcfccbde45ce874adcb698cc183debcf17952812
        if hex::encode(&trx.to) != "ca143ce32fe78f1f7019d7d551a6402fc5350c73" {
            continue;
        }

        for log in trx.receipt.unwrap().logs {
            let sig = hex::encode(&log.topics[0]);

            if !event::is_pair_created_event(sig.as_str()) {
                continue;
            }

            pairs.pairs.push(pcs::Pair {
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
    log::info!("Building pair state");
    substreams::register_panic_hook();

    let pairs: pcs::Pairs = proto::decode_ptr(pairs_ptr, pairs_len).unwrap();

    for pair in pairs.pairs {
        state::set(
            pair.log_ordinal as i64,
            format!("pair:{}", pair.address),
            &proto::encode(&pair).unwrap(),
        );
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

    let mut reserves = pcs::Reserves { reserves: vec![] };

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
                    let pair: pcs::Pair = proto::decode(&pair_bytes).unwrap();

                    // reserve
                    let token0: Token =
                        utils::get_last_token(tokens_store_idx, &pair.token0_address);
                    let reserve0 =
                        utils::convert_token_to_decimal(&log.data[0..32], &token0.decimals);
                    let token1: Token =
                        utils::get_last_token(tokens_store_idx, &pair.token1_address);
                    let reserve1 =
                        utils::convert_token_to_decimal(&log.data[32..64], &token1.decimals);

                    // token_price
                    let token0_price = utils::get_token_price(reserve0.clone(), reserve1.clone());
                    let token1_price = utils::get_token_price(reserve1.clone(), reserve0.clone());

                    reserves.reserves.push(pcs::Reserve {
                        pair_address: pair.address,
                        reserve0: reserve0.to_string(),
                        reserve1: reserve1.to_string(),
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

    let reserves: pcs::Reserves = proto::decode_ptr(reserves_ptr, reserves_len).unwrap();

    state::delete_prefix(0, &format!("pair_day:{}:", day_id - 1));
    state::delete_prefix(0, &format!("pair_hour:{}:", hour_id - 1));

    for reserve in reserves.reserves {
        match state::get_last(pairs_store_idx, &format!("pair:{}", reserve.pair_address)) {
            None => continue,
            Some(pair_bytes) => {
                let pair: pcs::Pair = proto::decode(&pair_bytes).unwrap();

                state::set(
                    reserve.log_ordinal as i64,
                    format!("price:{}:{}:token0", pair.address, pair.token0_address),
                    &Vec::from(reserve.token0_price),
                );
                state::set(
                    reserve.log_ordinal as i64,
                    format!("price:{}:{}:token1", pair.address, pair.token1_address),
                    &Vec::from(reserve.token1_price),
                );

                state_helper::set_many(
                    reserve.log_ordinal,
                    &vec![
                        format!(
                            "reserve:{}:{}:reserve0",
                            reserve.pair_address, pair.token0_address
                        ),
                        format!("pair_day:{}:{}:reserve0", day_id, pair.token0_address),
                        format!("pair_hour:{}:{}:reserve0", hour_id, pair.token0_address),
                    ],
                    &Vec::from(reserve.reserve0),
                );

                state_helper::set_many(
                    reserve.log_ordinal,
                    &vec![
                        format!(
                            "reserve:{}:{}:reserve1",
                            reserve.pair_address, pair.token1_address
                        ),
                        format!("pair_day:{}:{}:reserve1", day_id, pair.token1_address),
                        format!("pair_hour:{}:{}:reserve1", hour_id, pair.token1_address),
                    ],
                    &Vec::from(reserve.reserve1),
                )
            }
        }
    }
}

#[no_mangle]
pub extern "C" fn build_prices_state(
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

    let reserves: pcs::Reserves = proto::decode_ptr(reserves_ptr, reserves_len).unwrap();

    state::delete_prefix(0, &format!("pair_day:{}:", day_id - 1));
    state::delete_prefix(0, &format!("pair_hour:{}:", hour_id - 1));
    state::delete_prefix(0, &format!("token_day:{}:", day_id - 1));

    for reserve in reserves.reserves {
        match state::get_last(pairs_store_idx, &format!("pair:{}", reserve.pair_address)) {
            None => continue,
            Some(pair_bytes) => {
                let pair: pcs::Pair = proto::decode(&pair_bytes).unwrap();

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
                let usd_price_valid: bool = latest_usd_price.ne(&zero_big_decimal());

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
                        return zero_big_decimal();
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
                if reserves_bnb_sum.ne(&zero_big_decimal()) {
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

    let mut events: pcs::Events = pcs::Events { events: vec![] };

    let mut burn_count: i32 = 0;
    let mut mint_count: i32 = 0;
    let mut swap_count: i32 = 0;

    for trx in blk.transaction_traces {
        let trx_id = address_pretty(trx.hash.as_slice());
        for call in trx.calls {
            if call.state_reverted {
                continue;
            }

            if call.logs.len() == 0 {
                continue;
            }

            let pair_addr = address_pretty(call.address.as_slice());

            let pair: pcs::Pair;
            match state::get_last(pairs_store_idx, &format!("pair:{}", pair_addr)) {
                None => continue,
                Some(pair_bytes) => pair = proto::decode(&pair_bytes).unwrap(),
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
                    _ => {
                        log::info!("Error?! Events len is 4"); // fixme: should we panic here or just continue?
                        continue;
                    }
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
                    _ => {
                        log::info!("Error?! Events len is 3"); // fixme: should we panic here or just continue?
                        continue;
                    }
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
                            address_pretty(trx.from.as_slice()),
                            utils::get_last_token(tokens_store_idx, &pair.token0_address).decimals,
                            utils::get_last_token(tokens_store_idx, &pair.token1_address).decimals,
                        );
                    }
                    _ => {
                        log::info!("Error?! Events len is 2"); // fixme: should we panic here or just continue?
                        continue;
                    }
                }
            } else if pcs_events.len() == 1 {
                match pcs_events[0].event.as_ref().unwrap() {
                    Event::PairTransferEvent(_) => {
                        log::debug!("Events len 1, PairTransferEvent");
                        continue;
                    } // do nothing
                    Event::PairApprovalEvent(_) => {
                        log::debug!("Events len 1, PairApprovalEvent");
                        continue;
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

    let events: pcs::Events = proto::decode_ptr(events_ptr, events_len).unwrap();
    let pairs: pcs::Pairs = proto::decode_ptr(pairs_ptr, pairs_len).unwrap();

    for pair in pairs.pairs {
        state::sum_int64(pair.log_ordinal as i64, "global:pair_count".to_string(), 1);
    }

    for event in events.events {
        state_helper::sum_int64_many(
            event.log_ordinal,
            &vec![
                format!("token:{}:transaction_count", event.token0),
                format!("token:{}:transaction_count", event.token1),
                format!("pair:{}:transaction_count", event.pair_address),
                format!("global_day:{}:transaction_count", day_id),
                format!("global:transaction_count"),
            ],
            1,
        );

        match event.r#type.unwrap() {
            Type::Swap(swap) => {
                if swap.amount_usd.is_empty() {
                    continue;
                }

                state_helper::sum_int64_many(
                    event.log_ordinal,
                    &vec![format!("pair:{}:swap_count", event.pair_address)],
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
    let timestamp_block_header: pb::eth::BlockHeader = match blk.header {
        Some(block_header) => block_header,
        None => {
            log::info!("block id: {}", address_pretty(blk.hash.as_slice()));
            log::info!("block number: {}", blk.number.to_string());
            panic!("missing header")
        }
    };
    let timestamp = timestamp_block_header.timestamp.unwrap();
    let timestamp_seconds = timestamp.seconds;
    let day_id: i64 = timestamp_seconds / 86400;
    let hour_id: i64 = timestamp_seconds / 3600;

    if events_len == 0 {
        return;
    }

    let events: pcs::Events = proto::decode_ptr(events_ptr, events_len).unwrap();

    state::delete_prefix(0, &format!("pair_day:{}:", day_id - 1));
    state::delete_prefix(0, &format!("token_day:{}:", day_id - 1));
    state::delete_prefix(0, &format!("pair_hour:{}:", hour_id - 1));
    state::delete_prefix(0, &format!("global_day:{}", day_id - 1));

    for event in events.events {
        if event.r#type.is_some() {
            match event.r#type.unwrap() {
                Type::Mint(mint) => {
                    let amount_usd = BigDecimal::from_str(mint.amount_usd.as_str()).unwrap();
                    if amount_usd.eq(&zero_big_decimal()) {
                        continue;
                    }
                    state::sum_bigfloat(
                        event.log_ordinal as i64,
                        format!("global:liquidity_usd"),
                        &amount_usd,
                    );

                    state_helper::sum_bigfloat_many(
                        event.log_ordinal,
                        &vec![
                            format!("token:{}:liquidity", mint.to),
                            format!("pair:{}:total_supply", event.pair_address),
                        ],
                        &BigDecimal::from_str(mint.liquidity.as_str()).unwrap(),
                    );
                }
                Type::Burn(burn) => {
                    let amount_usd = BigDecimal::from_str(burn.amount_usd.as_str()).unwrap();
                    if amount_usd.eq(&zero_big_decimal()) {
                        continue;
                    }
                    state::sum_bigfloat(
                        event.log_ordinal as i64,
                        format!("global:liquidity_usd"),
                        &amount_usd.neg(),
                    );

                    state_helper::sum_bigfloat_many(
                        event.log_ordinal,
                        &vec![
                            format!("token:{}:liquidity", burn.to),
                            format!("pair:{}:total_supply", event.pair_address),
                        ],
                        &BigDecimal::from_str(burn.liquidity.as_str()).unwrap().neg(),
                    );
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
                            format!("pair:{}:usd", event.pair_address),
                            format!("pair_day:{}:{}:usd", day_id, event.pair_address),
                            format!("pair_hour:{}:{}:usd", hour_id, event.pair_address),
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

                    state_helper::sum_bigfloat_many(
                        event.log_ordinal,
                        &vec![
                            format!("pair:{}:token0", event.pair_address),
                            format!("pair_day:{}:{}:token0", day_id, event.pair_address),
                            format!("pair_hour:{}:{}:token0", day_id, event.pair_address),
                        ],
                        &amount_0_total,
                    );

                    state_helper::sum_bigfloat_many(
                        event.log_ordinal,
                        &vec![
                            format!("pair:{}:token1", event.pair_address),
                            format!("pair_day:{}:{}:token1", day_id, event.pair_address),
                            format!("pair_hour:{}:{}:token1", day_id, event.pair_address),
                        ],
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

                    //todo: token[0,1]Day.dailyVolumeToken, tokenDay[0,1].dailyVolumeBnb ? what about these
                }
            }
        }
    }
}

// todo: create pcs-token proto
#[no_mangle]
pub extern "C" fn build_pcs_token_state(pairs_ptr: *mut u8, pairs_len: usize, tokens_idx: u32) {
    substreams::register_panic_hook();

    let pairs: pcs::Pairs = proto::decode_ptr(pairs_ptr, pairs_len).unwrap();

    let mut token0_retry: bool = false;
    let mut token0: Token = Token {
        address: "".to_string(),
        name: "".to_string(),
        symbol: "".to_string(),
        decimals: 0,
    };
    let mut token1_retry: bool = false;
    let mut token1: Token = Token {
        address: "".to_string(),
        name: "".to_string(),
        symbol: "".to_string(),
        decimals: 0,
    };

    for pair in pairs.pairs {
        let token0_option_from_store: Option<Vec<u8>> =
            state::get_last(tokens_idx, &format!("token:{}", pair.token0_address));
        if token0_option_from_store.is_none() {
            log::info!(
                "token {} is not in the store, retrying rpc calls",
                pair.token0_address,
            );
            let token0_option = rpc::retry_rpc_calls(&pair.token0_address);
            if token0_option.is_none() {
                continue; // skip to next execution, we don't have a valid token
            }

            token0 = token0_option.unwrap();

            token0_retry = true;
            log::info!(
                "successfully found token {} after rpc calls",
                pair.token0_address
            );
        }

        if !token0_retry { // didn't need to retry as we have the token in the store
            token0 = proto::decode(&token0_option_from_store.unwrap()).unwrap();
        }

        state::set_if_not_exists(
            pair.log_ordinal as i64,
            format!("token:{}", token0.address),
            &proto::encode(&token0).unwrap(),
        );

        let token1_option_from_store: Option<Vec<u8>> =
            state::get_last(tokens_idx, &format!("token:{}", pair.token1_address));
        if token1_option_from_store.is_none() {
            log::info!(
                "token {} is not in the store, retrying rpc calls",
                pair.token1_address
            );
            let token1_option = rpc::retry_rpc_calls(&pair.token1_address);
            if token1_option.is_none() {
                continue; // skip to next execution, we don't have a valid token
            }

            token1 = token1_option.unwrap();

            token1_retry = true;
            log::info!(
                "successfully found token {} after rpc calls",
                pair.token1_address
            );
        }

        if !token1_retry { // didn't need to retry as we have the token in the store
            token1 = proto::decode(&token1_option_from_store.unwrap()).unwrap();
        }

        state::set_if_not_exists(
            pair.log_ordinal as i64,
            format!("token:{}", token1.address),
            &proto::encode(&token1).unwrap(),
        );
    }
}

#[no_mangle]
pub extern "C" fn map_to_database(
    block_ptr: *mut u8,
    block_len: usize,
    pcs_tokens_deltas_ptr: *mut u8,
    pcs_tokens_deltas_len: usize,
    pairs_deltas_ptr: *mut u8,
    pairs_deltas_len: usize,
    totals_deltas_ptr: *mut u8,
    totals_deltas_len: usize,
    volumes_deltas_ptr: *mut u8,
    volumes_deltas_len: usize,
    reserves_deltas_ptr: *mut u8,
    reserves_deltas_len: usize,
    events_ptr: *mut u8,
    events_len: usize,
    tokens_idx: u32,
) {
    substreams::register_panic_hook();

    let block: pb::eth::Block = proto::decode_ptr(block_ptr, block_len).unwrap();
    log::info!("block {:?}:{}", block_ptr, block_len);

    let pcs_token_deltas: substreams::pb::substreams::StoreDeltas =
        proto::decode_ptr(pcs_tokens_deltas_ptr, pcs_tokens_deltas_len).unwrap();

    let pair_deltas: substreams::pb::substreams::StoreDeltas =
        proto::decode_ptr(pairs_deltas_ptr, pairs_deltas_len).unwrap();

    log::info!(
        "map_to_database: pairs deltas:{} {}",
        pcs_tokens_deltas_len,
        pair_deltas.deltas.len()
    );

    let totals_deltas: substreams::pb::substreams::StoreDeltas =
        proto::decode_ptr(totals_deltas_ptr, totals_deltas_len).unwrap();

    let volumes_deltas: substreams::pb::substreams::StoreDeltas =
        proto::decode_ptr(volumes_deltas_ptr, volumes_deltas_len).unwrap();

    let reserves_deltas: substreams::pb::substreams::StoreDeltas =
        proto::decode_ptr(reserves_deltas_ptr, reserves_deltas_len).unwrap();

    let events: pcs::Events = proto::decode_ptr(events_ptr, events_len).unwrap();

    let changes = db::process(
        &block,
        pair_deltas,
        pcs_token_deltas,
        totals_deltas,
        volumes_deltas,
        reserves_deltas,
        events,
        tokens_idx,
    );

    substreams::output(changes);
}
