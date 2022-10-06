use substreams::errors::Error;
use substreams::store::{ProtoStoreSet, StoreSet};
use substreams::Hex;
use substreams_ethereum::pb::eth::v2::Block;

mod pb;

#[substreams::handlers::map]
pub fn map_fork_handling(block: Block) -> Result<pb::fork::Block, Error> {
    return Ok(pb::fork::Block {
        block_hash: Hex(block.hash.as_slice()).to_string(),
        block_number: block.number,
    });
}

#[substreams::handlers::store]
pub fn store_fork_handling(block: pb::fork::Block, store: ProtoStoreSet<pb::fork::Block>) {
    store.set(1, "block".to_string(), &block);
}
