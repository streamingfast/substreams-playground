extern crate core;
use std::convert::TryInto;

//use bigdecimal::BigDecimal;
use bs58;
//use hex;
use substreams::{log, proto}; //, state};

mod pb;

#[no_mangle]
pub extern "C" fn spl_transfers(block_ptr: *mut u8, block_len: usize) {
    log::info!("Extracting SPL Token Transfers");
    substreams::register_panic_hook();

    let blk: pb::sol::Block = proto::decode_ptr(block_ptr, block_len).unwrap();
    let mut xfers = pb::spl::TokenTransfers { transfers: vec![] };

    for trx in blk.transactions {
        if trx.failed {
            continue;
        }
        for inst in trx.instructions {
            if bs58::encode(inst.program_id).into_string()
                == "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
            {
                if inst.data[0] != 0x03 {
                    continue;
                }

                if inst.failed {
                    continue;
                }

                let a: [u8; 8] = inst.data[1..9].try_into().unwrap();
                let amount = u64::from_be_bytes(a);

                xfers.transfers.push(pb::spl::TokenTransfer {
                    transaction_id: bs58::encode(&trx.id).into_string(),
                    ordinal: inst.begin_ordinal,
                    from: inst.account_keys[0].clone(),
                    to: inst.account_keys[1].clone(),
                    amount: amount,
                })
            }
        }
    }

    if xfers.transfers.len() != 0 {
        substreams::output(xfers);
    }
}
