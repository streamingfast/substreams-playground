mod eth;
mod pb;
mod rpc;
mod utils;

use std::ops::Mul;
use std::str::FromStr;
use bigdecimal::BigDecimal;
use eth::{address_pretty, decode_string, decode_uint32};
use hex;

use substreams::{log, proto, state};

#[no_mangle]
pub extern "C" fn map_pairs(block_ptr: *mut u8, block_len: usize) {
    substreams::register_panic_hook();

    let blk: pb::eth::Block = proto::decode_ptr(block_ptr, block_len).unwrap();
    let mut pairs = pb::pcs::Pairs { pairs: vec![] };

    let msg = format!(
        "transaction traces count: {}, len: {}",
        blk.transaction_traces.len(),
        block_len
    );

    log::println(msg.to_string());

    for trx in blk.transaction_traces {
        /* PCS Factory address */
        if hex::encode(&trx.to) != "ca143ce32fe78f1f7019d7d551a6402fc5350c73" {
            continue;
        }

        for log in trx.receipt.unwrap().logs {
            let sig = hex::encode(&log.topics[0]);

            if !utils::is_pair_created_event(sig) {
                continue;
            }

            pairs.pairs.push(pb::pcs::Pair {
                address: address_pretty(&log.data[12..32]),
                erc20_token0: rpc::get_token(Vec::from(&log.topics[1][12..])),
                erc20_token1: rpc::get_token(Vec::from(&log.topics[2][12..])),
                creation_transaction_id: address_pretty(&trx.hash),
                block_num: blk.number,
                log_ordinal: log.block_index as u64,
            })
        }
    }

    substreams::output(pairs);
}

#[no_mangle]
pub extern "C" fn build_pairs_state(pairs_ptr: *mut u8, pairs_len: usize) {
    substreams::register_panic_hook();

    let pairs: pb::pcs::Pairs = proto::decode_ptr(pairs_ptr, pairs_len).unwrap();

    for pair in pairs.pairs {
        state::set(pair.log_ordinal as i64,
                   format!("pair:{}", pair.address),
                   proto::encode(&pair).unwrap()
        );
        state::set(pair.log_ordinal as i64,
                   format!("tokens:{}", utils::generate_tokens_key(pair.erc20_token0.unwrap().address, pair.erc20_token1.unwrap().address)),
                    Vec::from(pair.address)
        )
    }
}

#[no_mangle]
pub extern "C" fn map_reserves(block_ptr: *mut u8, block_len: usize, pairs_store_idx: u32) {
    substreams::register_panic_hook();

    let blk: pb::eth::Block = proto::decode_ptr(block_ptr, block_len).unwrap();

    let mut reserves = pb::pcs::Reserves { reserves: vec![] };

    for trx in blk.transaction_traces {
        for log in trx.receipt.unwrap().logs {

            let addr = address_pretty(&log.address);
            match state::get_last(pairs_store_idx, format!("pair:{}", addr)) {
                None => continue,
                Some(pair_bytes) => {
                    let sig = hex::encode(&log.topics[0]);

                    if !utils::is_new_pair_sync_event(sig) {
                        continue;
                    }

                    // Unmarshall pair
                    let pair: pb::pcs::Pair = proto::decode(pair_bytes).unwrap();

                    // reserve
                    let reserve0 = utils::convert_token_to_decimal(&log.data[0..32], pair.erc20_token0.unwrap().decimals);
                    let reserve1 = utils::convert_token_to_decimal(&log.data[32..64], pair.erc20_token1.unwrap().decimals);

                    // token_price
                    let token0_price = utils::get_token_price(reserve0.clone(), reserve1.clone());
                    let token1_price = utils::get_token_price(reserve1.clone(), reserve0.clone());

                    reserves.reserves.push(pb::pcs::Reserve {
                        pair_address: pair.address,
                        reserve0: reserve0.to_string(),
                        reserve1: reserve1.to_string(), // need to trim leading zeros
                        log_ordinal: log.block_index as u64,
                        token0_price: token0_price.to_string(),
                        token1_price: token1_price.to_string()
                    });
                }
            }
        }
    }
    substreams::output(reserves)
}

#[no_mangle]
pub extern "C" fn build_reserves_state(reserves_ptr: *mut u8, reserves_len: usize, pairs_store_idx: u32) {
    substreams::register_panic_hook();

    let reserves: pb::pcs::Reserves = proto::decode_ptr(reserves_ptr, reserves_len).unwrap();

    for reserve in reserves.reserves {
        match state::get_last(pairs_store_idx, format!("pair:{}", reserve.pair_address)) {
            None => continue,
            Some(mut pair_bytes) => {
                let pair: pb::pcs::Pair = proto::decode_ptr(pair_bytes.as_mut_ptr(), pair_bytes.len()).unwrap();

                state::set(reserve.log_ordinal as i64,
                           format!("price:{}:{}", pair.erc20_token0.as_ref().unwrap().address, pair.erc20_token1.as_ref().unwrap().address),
                           Vec::from(reserve.token0_price)
                );
                state::set(reserve.log_ordinal as i64,
                           format!("price:{}:{}", pair.erc20_token1.as_ref().unwrap().address, pair.erc20_token0.as_ref().unwrap().address),
                           Vec::from(reserve.token1_price)
                );
                state::set(reserve.log_ordinal as i64,
                           format!("reserve:{}:{}", reserve.pair_address, pair.erc20_token0.as_ref().unwrap().address),
                           Vec::from(reserve.reserve0)
                );
                state::set(reserve.log_ordinal as i64,
                           format!("reserve:{}:{}", reserve.pair_address, pair.erc20_token1.as_ref().unwrap().address),
                           Vec::from(reserve.reserve1)
                );
            }
        }
    }
}

#[no_mangle]
pub extern "C" fn build_prices_state(reserves_ptr: *mut u8, reserves_len: usize, pairs_store_idx: u32, reserves_store_idx: u32) {
    substreams::register_panic_hook();

    let reserves: pb::pcs::Reserves = proto::decode_ptr(reserves_ptr, reserves_len).unwrap();

    for reserve in reserves.reserves {
        match state::get_last(pairs_store_idx, format!("pair:{}", reserve.pair_address)) {
            None => continue,
            Some(mut pair_bytes) => {
                let pair: pb::pcs::Pair = proto::decode_ptr(pair_bytes.as_mut_ptr(), pair_bytes.len()).unwrap();
                log::println("pairs decode ok".to_string());

                let latest_usd_price: BigDecimal = utils::compute_usd_price(&reserve, reserves_store_idx);
                log::println("latest_usd_price ok".to_string());

                if reserve.pair_address.eq(&utils::USDT_WBNB_PAIR) || reserve.pair_address.eq(&utils::BUSD_WBNB_PAIR) {
                    state::set(reserve.log_ordinal as i64, format!("dprice:usd:bnb"), Vec::from(latest_usd_price.to_string()))
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
                let usd_price_valid: bool = latest_usd_price.eq(&utils::zero_big_decimal());

                let t0_derived_bnb_price = utils::find_bnb_price_per_token(&reserve.log_ordinal, pair.erc20_token0.clone().unwrap().address, pairs_store_idx, reserves_store_idx);
                log::println(format!("t0_derived_bnb_price ok"));
                let t1_derived_bnb_price = utils::find_bnb_price_per_token(&reserve.log_ordinal, pair.erc20_token1.clone().unwrap().address, pairs_store_idx, reserves_store_idx);
                log::println(format!("t1_derived_bnb_price ok"));

                let apply = | token_derived_bnb_price: Option<BigDecimal>, token_addr: String, reserve_amount: String | -> BigDecimal {
                    if token_derived_bnb_price.is_none() {
                        return utils::zero_big_decimal()
                    }

                    state::set(reserve.clone().log_ordinal as i64, format!("dprice:{}:bnb", token_addr), Vec::from(token_derived_bnb_price.clone().unwrap().to_string()));
                    let reserve_in_bnb = BigDecimal::from_str(reserve_amount.as_str()).unwrap().mul(token_derived_bnb_price.clone().unwrap());
                    state::set(reserve.clone().log_ordinal as i64, format!("dreserve:{}:{}:bnb", reserve.pair_address, token_addr), Vec::from(reserve_in_bnb.clone().to_string()));

                    if usd_price_valid {
                        let derived_usd_price = token_derived_bnb_price.unwrap().mul(latest_usd_price.clone());
                        state::set(reserve.clone().log_ordinal as i64, format!("dprice:{}:use", token_addr), Vec::from(derived_usd_price.to_string()));
                        let reserve_in_usd = reserve_in_bnb.clone().mul(latest_usd_price.clone());
                        state::set(reserve.clone().log_ordinal as i64, format!("dreserve:{}:{}:usd", reserve.pair_address, token_addr), Vec::from(reserve_in_usd.to_string()));
                    }

                    return reserve_in_bnb;
                };

                let reserve0_bnb = apply(t0_derived_bnb_price, pair.clone().erc20_token0.unwrap().address, reserve.clone().reserve0);
                let reserve1_bnb = apply(t1_derived_bnb_price, pair.clone().erc20_token1.unwrap().address, reserve.clone().reserve1);

                let reserves_bnb_sum = reserve0_bnb.mul(reserve1_bnb);
                if reserves_bnb_sum.ne(&reserves_bnb_sum) {
                    state::set(reserve.log_ordinal as i64, format!("dreserves:{}:bnb", reserve.pair_address), Vec::from(reserves_bnb_sum.to_string()));
                }
            }
        }
    }
}

#[no_mangle]
pub extern "C" fn map_to_database(
    reserves_ptr: *mut u8,
    reserves_len: usize,
    pairs_deltas_ptr: *mut u8,
    pairs_deltas_len: usize,
    _pairs_store_idx: u32,
) {
    substreams::register_panic_hook();

    let reserves: pb::pcs::Reserves = proto::decode_ptr(reserves_ptr, reserves_len).unwrap();
    let pair_deltas: substreams::pb::substreams::StoreDeltas =
        proto::decode_ptr(pairs_deltas_ptr, pairs_deltas_len).unwrap();

    for reserve in reserves.reserves {
        log::println(format!(
            "Reserve: {} {} {} {}",
            reserve.pair_address, reserve.log_ordinal, reserve.reserve0, reserve.reserve1
        ));
    }
    for delta in pair_deltas.deltas {
        log::println(format!(
            "Delta: {} {} {}",
            delta.operation, delta.key, delta.ordinal
        ));
    }
}

#[no_mangle]
pub extern "C" fn block_to_tokens(block_ptr: *mut u8, block_len: usize) {
    substreams::register_panic_hook();

    let mut tokens = pb::tokens::Tokens { tokens: vec![] };
    let blk: pb::eth::Block = proto::decode_ptr(block_ptr, block_len).unwrap();

    for trx in blk.transaction_traces {
        for call in trx.calls {
            if call.call_type == pb::eth::CallType::Create as i32 && !call.state_reverted {
                let rpc_calls = rpc::create_rpc_calls(call.clone().address);

                let rpc_responses_marshalled: Vec<u8> =
                    substreams::rpc::eth_call(substreams::proto::encode(&rpc_calls).unwrap());
                let rpc_responses_unmarshalled: substreams::pb::eth::RpcResponses =
                    substreams::proto::decode(rpc_responses_marshalled).unwrap();

                if rpc_responses_unmarshalled.responses[0].failed
                    || rpc_responses_unmarshalled.responses[1].failed
                    || rpc_responses_unmarshalled.responses[2].failed {
                    continue;
                };

                if !(rpc_responses_unmarshalled.responses[1].raw.len() >= 96)
                    || rpc_responses_unmarshalled.responses[0].raw.len() != 32
                    || !(rpc_responses_unmarshalled.responses[2].raw.len() >= 96) {
                    continue;
                };

                let decoded_address = address_pretty(&call.address);
                let decoded_decimals = decode_uint32(rpc_responses_unmarshalled.responses[0].raw.as_ref());
                let decoded_name = decode_string(rpc_responses_unmarshalled.responses[1].raw.as_ref());
                let decoded_symbol = decode_string(rpc_responses_unmarshalled.responses[2].raw.as_ref());

                let token = pb::tokens::Token{
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
        state::set(1, key, proto::encode(&token).unwrap()); //todo: what about the log ordinal
    }
}
