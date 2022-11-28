mod pb;

use substreams::errors::Error;
use substreams::errors::Error::Unexpected;
use substreams::{log, store, Hex}; // {hex, log, proto, store, Hex};
use substreams_ethereum::pb::eth as ethpb;

#[substreams::handlers::map]
fn map_coinbase(blk: ethpb::v1::Block) -> Result<pb::block_producers::Coinbase, Error> {
    match blk.header {
        Some(h) => Ok(pb::block_producers::Coinbase {
            address: Hex(h.coinbase).to_string(),
        }),
        None => Err(Unexpected("no header in block".to_string())),
    }
}

#[substreams::handlers::store]
fn store_coinbase(coinbase: pb::block_producers::Coinbase, store: store::StoreAddInt64) {
    let key = coinbase.address;
    store.add(1, key.clone(), 1);
    log::info!("added key: {}", key);
    //set(1, key, &proto::encode(&token).unwrap());
}
