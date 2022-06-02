extern crate core;
use std::convert::TryInto;

use bs58;
use substreams::{log, proto, store};
use substreams::errors::Error;

mod pb;

#[substreams::handlers::map]
fn map_tokens(blk: ethpb::v1::Block) -> Result<pb::tokens::Tokens, Error> {
    log::info!("Extracting SPL Token Transfers");
    substreams::register_panic_hook();
    let mut xfers = pb::spl::TokenTransfers { transfers: vec![] };
    for trx in blk.transactions {
        if let Some(meta) = trx.meta {
            if let Some(_err) = meta.err {
                continue;
            }
            if let Some(tt) = trx.transaction {
                if let Some(msg) = tt.message {
                    for inst in  msg.instructions {
                        let cop = &msg.account_keys[inst.program_id_index as usize];
                        if bs58::encode(cop).into_string() != "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA" {
                            continue;
                        }

                        if inst.data[0] != 0x03 {
                            continue;
                        }
    
                        // not sure if this is part of the confirmedBlock model
                        //if inst.failed {
                        //    continue;
                        //}
    
                        let am: [u8; 8] = inst.data[1..9].try_into().unwrap();
                        let from = &msg.account_keys[inst.accounts[0] as usize];
                        let to= &msg.account_keys[inst.accounts[1] as usize];
    
                        xfers.transfers.push(pb::spl::TokenTransfer {
                            transaction_id: bs58::encode(&tt.signatures[0]).into_string(),
                            ordinal: 0,
                            from: from.to_vec(),
                            to: to.to_vec(),
                            amount: u64::from_le_bytes(am),
                        });
                    }
                }
            }
        }
    }
    return Ok(xfers);
}

#[substreams::handlers::store]
pub fn store_transfers(xfers: pb::spl::TokenTransfers, output: store::StoreSet) {
    log::info!("Building transfer state");
    for xfer in xfers.transfers {
        output.set(
            1,
            format!("xfer:{}", xfer.transaction_id),
            &Vec::from("some-content-maybe"),//&proto::encode(&xfer).unwrap(),
        );
    }
}


