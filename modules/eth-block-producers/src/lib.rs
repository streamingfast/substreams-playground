mod pb;

use substreams::errors::Error;
use substreams::errors::Error::Unexpected;
use substreams::{log, store, Hex}; // {hex, log, proto, store, Hex};
use substreams_ethereum::pb::eth as ethpb;

#[substreams::handlers::map]
fn map_coinbase(blk: ethpb::v1::Block) -> Result<String, Error> {
    log::info!("patate".to_string());
    match blk.header {
        Some(h) => {
            let stringed = Hex(h.coinbase).to_string();
            log::info!(stringed.clone());
            Ok(stringed)
        }
        None => Err(Unexpected("no header in block".to_string())),
    }
}

#[substreams::handlers::store]
fn store_coinbase(coinbase: String, store: store::StoreAddInt64) {
    let key = coinbase;
    store.add(1, key.clone(), 1);
    log::info!("added key: {}", key);
    //set(1, key, &proto::encode(&token).unwrap());
}
