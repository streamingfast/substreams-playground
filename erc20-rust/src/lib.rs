mod pb;
mod utils;
mod eth;

use substreams::{log, proto};
use num_bigint::BigUint;

#[no_mangle]
pub extern "C" fn map_erc_20_transfer(block_ptr: *mut u8, block_len: usize) {
    substreams::register_panic_hook();

    let block: pb::eth::Block = proto::decode_ptr(block_ptr, block_len).unwrap();

    let mut transfers = pb::erc20::Transfers { transfers: vec![] };
    let mut b = false;
    let mut i = 0;
    for trx in block.transaction_traces {
        for call in trx.calls {
            for log in call.clone().logs {
                if !utils::is_erc20transfer_event(&log) {
                    continue
                }

                let from_addr = &Vec::from(&log.topics[1][12..]);
                let to_addr = &Vec::from(&log.topics[2][12..]);
                let amount = &log.data[0..32];
                let log_ordinal = log.index as u64;

                let transfer_event = pb::erc20::Transfer {
                    from: eth::address_pretty(from_addr.as_slice()),
                    to: eth::address_pretty(to_addr.as_slice()),
                    amount: BigUint::from_bytes_le(amount).to_string(),
                    balance_change_from: utils::find_erc20_storage_changes(&call.clone(), from_addr),
                    balance_change_to: utils::find_erc20_storage_changes(&call.clone(), to_addr),
                    log_ordinal
                };

                i = i + 1;

                if i == 3 {
                    log::println(format!("{:?}", transfer_event.balance_change_from));
                    // log::println(format!("{:?}", transfer_event.balance_change_to));
                    transfers.transfers.push(transfer_event);
                    b = true;
                    break;
                }

            }
            if b {
                break;
            }
        }
        if b {
            break;
        }
    }

    substreams::output(&transfers);
}
