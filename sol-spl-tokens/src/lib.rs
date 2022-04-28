extern crate core;

use std::fmt::format;
use std::str::FromStr;

use bs58;
use bigdecimal::BigDecimal;
use hex;
use substreams::{log, proto, state};

mod pb;

#[no_mangle]
pub extern "C" fn spl_transfers(block_ptr: *mut u8, block_len: usize) {
    log::println("Pairs mapping".to_string());
    substreams::register_panic_hook();

    let blk: pb::sol::Block = proto::decode_ptr(block_ptr, block_len).unwrap();
    let mut xfers = pb::spl::TokenTransfers { transfers: vec![] };

    for trx in blk.transactions {
        for inst in trx.instructions {
            if bs58::encode(inst.program_id).into_string()
                == "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
            {
                if inst.data[0] != 0x03 {
                    continue;
                }

                let amount = inst.data[1] as u64; // u64

                xfers.transfers.push(pb::spl::TokenTransfer {
                    transaction_id: hex::encode(&trx.id),
                    ordinal: inst.ordinal as u64,
                    from: inst.account_keys[0].clone(),
                    to: inst.account_keys[1].clone(),
                    amount: format!("{:?}", amount),
                })
            }
        }
    }

    if xfers.transfers.len() != 0 {
        substreams::output(xfers);
    }
}
