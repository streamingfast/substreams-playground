extern crate core;
use std::convert::TryInto;

//use bigdecimal::BigDecimal;
use bs58;
//use hex;
use substreams::{log, proto, store};

mod pb;

#[no_mangle]
pub extern "C" fn spl_transfers(block_ptr: *mut u8, block_len: usize) {
    log::info!("Extracting SPL Token Transfers");
    substreams::register_panic_hook();

    let blk: pb::sol::ConfirmedBlock = proto::decode_ptr(block_ptr, block_len).unwrap();
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

                        // log::info!("trx: {} from: {} to: {} account bytes: 0x{}", bs58::encode(&tt.signatures[0]).into_string(), bs58::encode(from.to_vec()).into_string(), bs58::encode(to.to_vec()).into_string(), hex::encode(am));
                    }
                }
            }
        }

   }
        if xfers.transfers.len() != 0 {
            substreams::output(xfers);
        }
}

#[substreams::handlers::store]
pub fn transfer_store(xfers: pb::spl::TokenTransfers, output: store::StoreSet) {
    log::info!("Building xfer state");
    for xfer in xfers.transfers {
        output.set(
            1,
            format!("xfer:{}", xfer.transaction_id),
            &Vec::from("some-content-maybe"),//&proto::encode(&xfer).unwrap(),
        );
    }
}


