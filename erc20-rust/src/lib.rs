mod pb;
mod utils;

use substreams::proto;
use num_bigint::BigUint;

#[no_mangle]
pub extern "C" fn create_erc_20_transfer(block_ptr: *mut u8, block_len: usize) {
    substreams::register_panic_hook();

    let block: pb::eth::Block = proto::decode_ptr(block_ptr, block_len).unwrap();

    for trx in block.transaction_traces {
        for call in trx.calls {
            for log in call.clone().logs {
                if !utils::is_erc20transfer_event(&log) {
                    continue
                }

                let from_addr = &Vec::from(&log.topics[1][12..]);
                let to_addr = &Vec::from(&log.topics[2][12..]);
                let amount = utils::convert_vec_u8(Vec::from(&log.data[0..32]));
                let log_ordinal = log.index as u64;

                let transfer_event = &pb::erc20transfer::Erc20Transfer {
                    from: utils::vec_u8_to_string(from_addr),
                    to: utils::vec_u8_to_string(to_addr),
                    amount: BigUint::new(amount).to_string(),
                    balance_change_from: utils::find_erc20_storage_changes(&call.clone(), from_addr),
                    balance_change_to: utils::find_erc20_storage_changes(&call.clone(), to_addr),
                    log_ordinal
                };

                substreams::output(transfer_event);
            }
        }
    }

}
