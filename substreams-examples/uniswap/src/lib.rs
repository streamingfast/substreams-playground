use std::rc::Rc;

mod contracts;
mod pb;
pub mod util;

use substreams::{log, proto};

use contracts::factory::FactoryContract;
use pb::uniswap::{Pool, Pools};

#[no_mangle]
pub extern "C" fn pools(block_ptr: *mut u8, block_len: usize) {
    substreams::register_panic_hook();

    let mut pools = Pools { pools: vec![] };

    let block: Rc<pb::eth::Block> = Rc::new(proto::decode_ptr(block_ptr, block_len).unwrap());

    let factory = FactoryContract::bind(block.clone(), "1f98431c8ad98523631ae4a59f267346ea31f984");

    for event in factory.pool_created_events() {
        log::info!("Pool created at block #{}", block.number);

        let mut pool = Pool::default();
        let header = block.header.as_ref().expect("header");

        pool.token0 = hex::encode(&event.topics[1][12..]);
        pool.token1 = hex::encode(&event.topics[2][12..]);

        let timestamp = header.timestamp.as_ref().expect("timestamp");

        pool.created_at_timestamp = timestamp.seconds as u64;
        pool.created_at_block_number = header.number;

        pools.pools.push(pool);
    }

    if !pools.pools.is_empty() {
        substreams::output(pools);
    }
}
