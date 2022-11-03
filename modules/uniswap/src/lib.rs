mod abi;
mod pb;

use hex_literal::hex;
use pb::uniswap::{Pool, Pools};
use substreams::{log, Hex};
use substreams_ethereum::pb::eth::v2 as eth;

const FACTORY_CONTRACT: [u8; 20] = hex!("5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f");

substreams_ethereum::init!();

#[substreams::handlers::map]
fn map_pools(blk: eth::Block) -> Result<Pools, substreams::errors::Error> {
    Ok(Pools {
        pools: blk
            .events::<abi::factory::events::PoolCreated>(&[&FACTORY_CONTRACT])
            .map(|(pool_created, _log)| {
                log::info!("PoolCreated event seen");

                Pool {
                    created_at_timestamp: blk.timestamp_seconds(),
                    created_at_block_number: blk.number,
                    token0: Hex(pool_created.token0).to_string(),
                    token1: Hex(pool_created.token1).to_string(),
                }
            })
            .collect(),
    })
}
