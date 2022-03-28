use hex::ToHex;

mod contracts;
mod pb;
pub mod util;

use substreams::{log, proto};

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
        let pool_created_events = tx
            .receipt
            .as_ref()
            .unwrap()
            .logs
            .iter()
            .filter(|event| factory::PoolCreatedEvent::matches(event));

        for _event in pool_created_events {
            // log::println(format!(
            //     "POOL CREATED EVENT: block #{}, tx {}",
            //     block.number,
            //     tx.hash.encode_hex::<String>()
            // ));

            let token0 = String::from("0");
            let token1 = String::from("1");

            pools.pools.push(Pool { token0, token1 });
        }
    }

    substreams::output(pools);
}
