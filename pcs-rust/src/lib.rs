mod eth;
mod pb;
mod substreams;

use eth::decode_address;
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

    log::info(msg.to_string());

    for trx in blk.transaction_traces {
        /* PCS Factory address */
        if hex::encode(&trx.to) != "ca143ce32fe78f1f7019d7d551a6402fc5350c73" {
            continue;
        }

        for log in trx.receipt.unwrap().logs {
            let sig = hex::encode(&log.topics[0]);

            if sig != "0d3648bd0f6ba80134a33ba9275ac585d9d315f0ad8355cddefde31afa28d0e9" {
                continue;
            }

            // topics[0] is the event signature
            let pair_token0 = decode_address(&log.topics[1]);
            let pair_token1 = decode_address(&log.topics[2]);
            let pair_addr = decode_address(&log.data);

            pairs.pairs.push(pb::pcs::Pair {
                address: pair_addr.clone(),
                token0: pair_token0,
                token1: pair_token1,
                creation_transaction_id: hex::encode(&trx.hash),
                block_num: blk.number,
                log_ordinal: log.block_index as u64,
            })
        }
    }

    substreams::output(&pairs);
}

#[no_mangle]
pub extern "C" fn build_pairs_state(pairs_ptr: *mut u8, pairs_len: usize) {
    substreams::register_panic_hook();

    let pairs: pb::pcs::Pairs = proto::decode_ptr(pairs_ptr, pairs_len).unwrap();

    for pair in pairs.pairs {
        let key = format!("pair:{}", pair.address);
        state::set(pair.log_ordinal as i64, key, proto::encode(&pair).unwrap());
    }
}

#[no_mangle]
pub extern "C" fn map_reserves(block_ptr: *mut u8, block_len: usize, pairs_store_idx: u32) {
    substreams::register_panic_hook();

    let blk: pb::eth::Block = proto::decode_ptr(block_ptr, block_len).unwrap();

    let mut reserves = pb::pcs::Reserves { reserves: vec![] };

    for trx in blk.transaction_traces {
        for log in trx.receipt.unwrap().logs {
            let addr = hex::encode(log.address);
            match state::get_last(pairs_store_idx, format!("pair:{}", addr)) {
                None => continue,
                Some(pair_bytes) => {
                    let sig = hex::encode(&log.topics[0]);
                    // Sync(uint112,uint112)
                    if sig != "1c411e9a96e071241c2f21f7726b17ae89e3cab4c78be50e062b03a9fffbbad1" {
                        continue;
                    }

                    // Continue handling a Pair's Sync event
                    let pair: pb::pcs::Pair = proto::decode(pair_bytes).unwrap();

                    // TODO: Read the log's Reserve0, and Reserve1
                    // TODO: take the `pair.token0/1.decimals` and add the decimal point on that Reserve0
                    // TODO: do floating point calculations

                    reserves.reserves.push(pb::pcs::Reserve {
                        pair_address: pair.address,
                        reserve0: "123".to_string(),
                        reserve1: "234".to_string(),
                        log_ordinal: log.block_index as u64,
                    });
                }
            }
        }
    }

    substreams::output(&reserves)
}
