extern crate core;

use std::ops::{Mul, Neg};
use std::str::FromStr;

use bigdecimal::BigDecimal;
use hex;
use substreams::{log, proto, store};
use substreams::errors::Error;

use eth::{address_decode, address_pretty};

use crate::event::pcs_event::Event;
use crate::event::PcsEvent;
use crate::pb::database::DatabaseChanges;
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
mod utils;

#[substreams::handlers::map]
pub fn map_pairs(blk: pb::eth::Block) -> Result<pcs::Pairs, Error> {
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

    Ok(pairs)
}

#[substreams::handlers::store]
pub fn store_pairs(pairs: pcs::Pairs, output: store::StoreSet) {
    log::info!("Building pair state");
    for pair in pairs.pairs {
        output.set(
            pair.log_ordinal,
            format!("pair:{}", pair.address),
            &proto::encode(&pair).unwrap(),
        );
    }
}

#[substreams::handlers::map]
pub fn map_reserves(blk: pb::eth::Block, pairs: store::StoreGet, tokens: store::StoreGet) -> Result<pcs::Reserves, Error> {
    let mut reserves = pcs::Reserves { reserves: vec![] };

    for trx in blk.transaction_traces {
        for log in trx.receipt.unwrap().logs {
            let addr = address_pretty(&log.address);
            match pairs.get_last(&format!("pair:{}", addr)) {
                None => continue,
                Some(pair_bytes) => {
                    let sig = hex::encode(&log.topics[0]);

                    if !event::is_pair_sync_event(sig.as_str()) {
                        continue;
                    }

                    let pair: pcs::Pair = proto::decode(&pair_bytes).unwrap();

                    let token0: Token = utils::get_last_token(&tokens, &pair.token0_address);
                    let reserve0 =
                        utils::convert_token_to_decimal(&log.data[0..32], &token0.decimals);
                    let token1: Token = utils::get_last_token(&tokens, &pair.token1_address);
                    let reserve1 =
                        utils::convert_token_to_decimal(&log.data[32..64], &token1.decimals);

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

    Ok(reserves)
}

#[substreams::handlers::store]
pub fn store_reserves(clock: substreams::pb::substreams::Clock, reserves: pcs::Reserves, pairs: store::StoreGet, output: store::StoreSet) {
    let timestamp_seconds = clock.timestamp.unwrap().seconds;
    let day_id: i64 = timestamp_seconds / 86400;
    let hour_id: i64 = timestamp_seconds / 3600;

    output.delete_prefix(0, &format!("pair_day:{}:", day_id - 1));
    output.delete_prefix(0, &format!("pair_hour:{}:", hour_id - 1));

    for reserve in reserves.reserves {
        match pairs.get_last(&format!("pair:{}", reserve.pair_address)) {
            None => continue,
            Some(pair_bytes) => {
                let pair: pcs::Pair = proto::decode(&pair_bytes).unwrap();

                output.set(
                    reserve.log_ordinal,
                    format!("price:{}:{}:token0", pair.address, pair.token0_address),
                    &Vec::from(reserve.token0_price),
                );
                output.set(
                    reserve.log_ordinal,
                    format!("price:{}:{}:token1", pair.address, pair.token1_address),
                    &Vec::from(reserve.token1_price),
                );

                output.set_many(
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

                output.set_many(
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

#[substreams::handlers::store]
pub fn store_prices(clock: substreams::pb::substreams::Clock, reserves: pcs::Reserves, pairs: store::StoreGet, reserves_store: store::StoreGet, output: store::StoreSet) {
    let timestamp_seconds = clock.timestamp.unwrap().seconds;
    let day_id: i64 = timestamp_seconds / 86400;
    let hour_id: i64 = timestamp_seconds / 3600;

    output.delete_prefix(0, &format!("pair_day:{}:", day_id - 1));
    output.delete_prefix(0, &format!("pair_hour:{}:", hour_id - 1));
    output.delete_prefix(0, &format!("token_day:{}:", day_id - 1));

    for reserve in reserves.reserves {
        match pairs.get_last(&format!("pair:{}", reserve.pair_address)) {
            None => continue,
            Some(pair_bytes) => {
                let pair: pcs::Pair = proto::decode(&pair_bytes).unwrap();

                let latest_usd_price: BigDecimal =
                    utils::compute_usd_price(&reserves_store, &reserve);

                if reserve.pair_address.eq(&utils::USDT_WBNB_PAIR)
                    || reserve.pair_address.eq(&utils::BUSD_WBNB_PAIR)
                {
                    output.set(
                        reserve.log_ordinal,
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
                    &pairs,
                    &reserves_store,
                );

                let t1_derived_bnb_price = utils::find_bnb_price_per_token(
                    &reserve.log_ordinal,
                    pair.token1_address.as_str(),
                    &pairs,
                    &reserves_store,
                );

                let apply = |token_derived_bnb_price: Option<BigDecimal>,
                             token_addr: String,
                             reserve_amount: String|
                 -> BigDecimal {
                    if token_derived_bnb_price.is_none() {
                        return zero_big_decimal();
                    }

                    output.set(
                        reserve.log_ordinal,
                        format!("dprice:{}:bnb", token_addr),
                        &Vec::from(token_derived_bnb_price.clone().unwrap().to_string()),
                    );
                    let reserve_in_bnb = BigDecimal::from_str(reserve_amount.as_str())
                        .unwrap()
                        .mul(token_derived_bnb_price.clone().unwrap());
                    output.set(
                        reserve.log_ordinal,
                        format!("dreserve:{}:{}:bnb", reserve.pair_address, token_addr),
                        &Vec::from(reserve_in_bnb.clone().to_string()),
                    );

                    if usd_price_valid {
                        let derived_usd_price = token_derived_bnb_price
                            .unwrap()
                            .mul(latest_usd_price.clone());
                        output.set_many(
                            reserve.log_ordinal,
                            &vec![
                                format!("dprice:{}:usd", token_addr),
                                format!("token_day:{}:dprice:{}:usd", day_id, token_addr),
                            ],
                            &Vec::from(derived_usd_price.to_string()),
                        );

                        let reserve_in_usd = reserve_in_bnb.clone().mul(latest_usd_price.clone());

                        output.set_many(
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
                    output.set(
                        reserve.log_ordinal,
                        format!("dreserves:{}:bnb", reserve.pair_address),
                        &Vec::from(reserves_bnb_sum.to_string()),
                    );
                }
            }
        }
    }
}

// pub extern "C" fn build_twap_transient_store(clock, prices_deltas) {
//     let deltas: pcs::StoreDeltas;
//     // TODO: flatten the deltas
//     for delta in deltas.deltas {
// 	if delta.key.starts_with("dprice:") {
// 	    let key = delta.key.split(":");

// 	    let token0 = delta.key.split(":")[1];
// 	    // FLATTEN to per block
// 	    output.set(0, "price:{:012}:{}", clock.number, token0);
// 	    // but then, how to compute the twap? we can't read
// 	    // the store here
// 	}
//     }

//     let del_prev_block = (clock.number - 100) / 100.0;
//     output.delete_prefix("price:{:010}", del_prev_block)
// }

// pub extern "C" fn build_twap_from_dprice(clock, twap_price_store, twap_prices_deltas) {
//     let deltas: pcs::StoreDeltas;
//     // Assumes its flattened
//     for delta in deltas.deltas {
// 	let key = delta.key.split(":");
// 	let block_num = key[1];
// 	let token = key[2];
// 	// loop through previous keys?

// 	let count = 0;
// 	let price_sum = 0.0;
// 	for (i = 0; i < 10 /* or whatever is parameterized for the number of blocks of twap */; i++) {
// 	    let price, found = state::get_last(twap_price_store, format!("price:{}:{}", block_num - i, token));
// 	    if found {
// 		count++;
// 		price_sum += price;
// 	    }
// 	}

// 	// avg = price_sum / count;
// 	output.set(0, format!("price:{}:{}", clock.number, token0), avg);
//     }
// }

#[substreams::handlers::map]
pub fn map_burn_swaps_events(blk: pb::eth::Block, pairs_store: store::StoreGet, prices_store: store::StoreGet, tokens_store: store::StoreGet) -> Result<pcs::Events, Error> {
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
            match pairs_store.get_last(&format!("pair:{}", pair_addr)) {
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
                            &prices_store,
                            &pair,
                            ev_tr1,
                            ev_tr2,
                            pair_mint_event,
                            utils::get_last_token(&tokens_store, pair.token0_address.as_str())
                                .decimals,
                            utils::get_last_token(&tokens_store, pair.token1_address.as_str())
                                .decimals,
                        )
                    }
                    Event::PairBurnEvent(pair_burn_event) => {
                        let burn_id = format!("{}-{}", trx_id, burn_count);
                        burn_count = burn_count + 1;

                        event::process_burn(
                            burn_id.as_str(),
                            &mut base_event,
                            &prices_store,
                            &pair,
                            ev_tr1,
                            ev_tr2,
                            pair_burn_event,
                            utils::get_last_token(&tokens_store, pair.token0_address.as_str())
                                .decimals,
                            utils::get_last_token(&tokens_store, pair.token1_address.as_str())
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
                            &prices_store,
                            &pair,
                            None,
                            ev_tr2,
                            pair_mint_event,
                            utils::get_last_token(&tokens_store, pair.token0_address.as_str())
                                .decimals,
                            utils::get_last_token(&tokens_store, pair.token1_address.as_str())
                                .decimals,
                        )
                    }
                    Event::PairBurnEvent(pair_burn_event) => {
                        let burn_id = format!("{}-{}", trx_id, burn_count);
                        burn_count += 1;

                        event::process_burn(
                            burn_id.as_str(),
                            &mut base_event,
                            &prices_store,
                            &pair,
                            None,
                            ev_tr2,
                            pair_burn_event,
                            utils::get_last_token(&tokens_store, pair.token0_address.as_str())
                                .decimals,
                            utils::get_last_token(&tokens_store, pair.token1_address.as_str())
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
                            &prices_store,
                            &pair,
                            Some(pair_swap_event),
                            address_pretty(trx.from.as_slice()),
                            utils::get_last_token(&tokens_store, &pair.token0_address).decimals,
                            utils::get_last_token(&tokens_store, &pair.token1_address).decimals,
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

    Ok(events)
}

#[substreams::handlers::store]
pub fn totals(
    clock: substreams::pb::substreams::Clock,
    pairs: pcs::Pairs,
    events: pcs::Events,
    output: store::StoreAddInt64,
) {
    let timestamp_seconds = clock.timestamp.unwrap().seconds;
    let day_id: i64 = timestamp_seconds / 86400;

    if events.events.len() == 0 && pairs.pairs.len() == 0 {
        return;
    }

    for pair in pairs.pairs {
        output.add(pair.log_ordinal, "global:pair_count".to_string(), 1);
    }

    for event in events.events {
        output.add_many(
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

                output.add_many(
                    event.log_ordinal,
                    &vec![format!("pair:{}:swap_count", event.pair_address)],
                    1,
                );

                //todo: if we want to set the total transactions for global day we need a
                // key setter store to keep track of the latest computed(summed) values
            }
            Type::Burn(_) => output.add(
                event.log_ordinal,
                format!("pair:{}:burn_count", event.pair_address),
                1,
            ),
            Type::Mint(_) => output.add(
                event.log_ordinal,
                format!("pair:{}:mint_count", event.pair_address),
                1,
            ),
        }
    }
}

#[substreams::handlers::store]
pub fn volumes(
    clock: substreams::pb::substreams::Clock,
    events: pcs::Events,
    output: store::StoreAddBigFloat,
) {
    let timestamp_seconds = clock.timestamp.unwrap().seconds;
    let day_id: i64 = timestamp_seconds / 86400;
    let hour_id: i64 = timestamp_seconds / 3600;

    if events.events.len() == 0 {
        return;
    }

    output.delete_prefix(0, &format!("pair_day:{}:", day_id - 1));
    output.delete_prefix(0, &format!("token_day:{}:", day_id - 1));
    output.delete_prefix(0, &format!("pair_hour:{}:", hour_id - 1));
    output.delete_prefix(0, &format!("global_day:{}", day_id - 1));

    for event in events.events {
        if event.r#type.is_some() {
            match event.r#type.unwrap() {
                Type::Mint(mint) => {
                    let amount_usd = BigDecimal::from_str(mint.amount_usd.as_str()).unwrap();
                    if amount_usd.eq(&zero_big_decimal()) {
                        continue;
                    }
                    output.add(
                        event.log_ordinal,
                        format!("global:liquidity_usd"),
                        &amount_usd,
                    );

                    output.add_many(
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
                    output.add(
                        event.log_ordinal,
                        format!("global:liquidity_usd"),
                        &amount_usd.neg(),
                    );

                    output.add_many(
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

                    output.add_many(
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

                    output.add_many(
                        event.log_ordinal,
                        &vec![format!("global:bnb"), format!("global_day:{}:bnb", day_id)],
                        &amount_bnb,
                    );

                    output.add_many(
                        event.log_ordinal,
                        &vec![
                            format!("pair:{}:token0", event.pair_address),
                            format!("pair_day:{}:{}:token0", day_id, event.pair_address),
                            format!("pair_hour:{}:{}:token0", day_id, event.pair_address),
                        ],
                        &amount_0_total,
                    );

                    output.add_many(
                        event.log_ordinal,
                        &vec![
                            format!("pair:{}:token1", event.pair_address),
                            format!("pair_day:{}:{}:token1", day_id, event.pair_address),
                            format!("pair_hour:{}:{}:token1", day_id, event.pair_address),
                        ],
                        &amount_1_total,
                    );

                    output.add(
                        event.log_ordinal,
                        format!("token:{}:trade", event.token0),
                        &BigDecimal::from_str(swap.trade_volume0.as_str()).unwrap(),
                    );
                    output.add(
                        event.log_ordinal,
                        format!("token:{}:trade", event.token1),
                        &BigDecimal::from_str(swap.trade_volume1.as_str()).unwrap(),
                    );
                    output.add(
                        event.log_ordinal,
                        format!("token:{}:trade_usd", event.token0),
                        &BigDecimal::from_str(swap.trade_volume_usd0.as_str()).unwrap(),
                    );
                    output.add(
                        event.log_ordinal,
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
#[substreams::handlers::store]
pub fn pcs_tokens(
    pairs: pcs::Pairs,
    tokens: store::StoreGet,
    output: store::StoreSetIfNotExists,
) {
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
            tokens.get_last(&format!("token:{}", pair.token0_address));
        if token0_option_from_store.is_none() {
            log::info!(
                "token {} is not in the store, retrying rpc calls",
                pair.token0_address,
            );
            let token0_res = rpc::retry_rpc_calls(&pair.token0_address);
            if token0_res.is_err() {
                continue; // skip to next execution, we don't have a valid token
            }

            token0 = token0_res.unwrap();

            token0_retry = true;
            log::info!(
                "successfully found token {} after rpc calls",
                pair.token0_address
            );
        }

        if !token0_retry {
            // didn't need to retry as we have the token in the store
            token0 = proto::decode(&token0_option_from_store.unwrap()).unwrap();
        }

        output.set_if_not_exists(
            pair.log_ordinal,
            format!("token:{}", token0.address),
            &proto::encode(&token0).unwrap(),
        );

        let token1_option_from_store: Option<Vec<u8>> =
            tokens.get_last(&format!("token:{}", pair.token1_address));
        if token1_option_from_store.is_none() {
            log::info!(
                "token {} is not in the store, retrying rpc calls",
                pair.token1_address
            );
            let token1_res = rpc::retry_rpc_calls(&pair.token1_address);
            if token1_res.is_err() {
                continue; // skip to next execution, we don't have a valid token
            }

            token1 = token1_res.unwrap();

            token1_retry = true;
            log::info!(
                "successfully found token {} after rpc calls",
                pair.token1_address
            );
        }

        if !token1_retry {
            // didn't need to retry as we have the token in the store
            token1 = proto::decode(&token1_option_from_store.unwrap()).unwrap();
        }

        output.set_if_not_exists(
            pair.log_ordinal,
            format!("token:{}", token1.address),
            &proto::encode(&token1).unwrap(),
        );
    }
}

#[substreams::handlers::map]
pub fn db_out(
    block: substreams::pb::substreams::Clock,
    pcs_tokens_deltas: store::Deltas,
    pairs_deltas: store::Deltas,
    totals_deltas: store::Deltas,
    volumes_deltas: store::Deltas,
    reserves_deltas: store::Deltas,
    events: pcs::Events,
    tokens: store::StoreGet,
) -> Result<DatabaseChanges, Error> {
    substreams::register_panic_hook();

    log::info!(
        "map_to_database: pairs deltas:{} {}",
        pcs_tokens_deltas.len(),
        pairs_deltas.len()
    );

    let changes = db::process(
        &block,
        pairs_deltas,
        pcs_tokens_deltas,
        totals_deltas,
        volumes_deltas,
        reserves_deltas,
        events,
        &tokens,
    );

    return Ok(changes);
}
