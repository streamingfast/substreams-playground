use std::time::SystemTime;

use hex::ToHex;

mod contracts;
mod pb;
pub mod util;

use substreams::{log, proto, rpc};

use contracts::factory;
use pb::{
    eth::Log,
    uniswap::{Pool, Pools},
};

#[no_mangle]
pub extern "C" fn pools(block_ptr: *mut u8, block_len: usize) {
    substreams::register_panic_hook();

    let mut pools = Pools { pools: vec![] };

    let block: pb::eth::Block = proto::decode_ptr(block_ptr, block_len).unwrap();

    let factory_txs = block
        .transaction_traces
        .iter()
        .filter(|tx| hex::encode(&tx.to) == factory::ADDRESS);

    for tx in factory_txs {
        log::println(format!("TX"));

        let pool_created_events = tx
            .receipt
            .as_ref()
            .unwrap()
            .logs
            .iter()
            .filter(|event| factory::PoolCreatedEvent::matches(event));

        for event in pool_created_events {
            log::println(format!(
                "POOL CREATED: #{}, tx {}",
                block.number,
                hex::encode(&tx.hash)
            ));

            let mut pool = Pool::default();
            // let header = block.header.as_ref().expect("header");

            // pool.token0 = hex::encode(&event.topics[1][12..]);
            // pool.token1 = hex::encode(&event.topics[2][12..]);

            // let timestamp = header.timestamp.as_ref().expect("timestamp");

            // pool.created_at_timestamp = timestamp.seconds as u64;
            // pool.created_at_block_number = header.number;

            // pools.pools.push(pool);
        }
    }

    if !pools.pools.is_empty() {
        substreams::output(pools);
    }
}
