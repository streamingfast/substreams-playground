use crate::pb;

use hex;
use std::{mem, str};
use num_bigint::BigUint;

const TRANSFER_TOPIC: &str = "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef";

pub fn is_erc20transfer_event(log: &pb::eth::Log) -> bool {
    if log.topics.len() != 3 || log.data.len() != 32 {
        return false;
    }

    return str::Bytes::eq(str::from_utf8(&log.topics[0].to_vec()).unwrap().bytes(), TRANSFER_TOPIC.bytes())
}

pub fn find_erc20_storage_changes(call: &pb::eth::Call, holder: &Vec<u8>) -> Vec<pb::erc20transfer::Erc20BalanceChange> {
    let mut out: Vec<pb::erc20transfer::Erc20BalanceChange> = Vec::new();
    let keys = erc20storage_keys_from_address(call, holder);

    for key in keys {
        let byte_key = hex::decode(key).unwrap();

        for change in &call.storage_changes {
            if str::Bytes::eq(str::from_utf8(&change.key).unwrap().bytes(), str::from_utf8(&byte_key).unwrap().bytes()) {
                let new_balance = BigUint::new(convert_vec_u8(change.clone().new_value)).to_string();
                let old_balance = BigUint::new(convert_vec_u8(change.clone().old_value)).to_string();

                let erc20_balance_change = &pb::erc20transfer::Erc20BalanceChange {
                    old_balance,
                    new_balance
                };

                out.push(erc20_balance_change.clone())
            }
        }
    }

    return out;

}

pub fn erc20storage_keys_from_address<'a>(call: &'a pb::eth::Call, addr: &'a Vec<u8>) -> Vec<&'a  String> {
    let mut out = Vec::new();
    let addr_as_hex = hex::encode(addr);
    for (hash, pre_image) in &call.keccak_preimages {
        if pre_image.chars().count() != 128 {
            continue;
        }

        // we're sure it's a top=level variable or something near that
        if &pre_image[64..126] != "00000000000000000000000000000000000000000000000000000000000000" {
            // Second part of the keccak should be a top-level
            continue;
        }

        if &pre_image[24..64] == addr_as_hex {
            out.push(hash);
        }
    }
    return out;
}

// https://stackoverflow.com/a/49694475/11389045
pub fn convert_vec_u8(mut vec8: Vec<u8>) -> Vec<u32> {
    let vec32 = unsafe {
        let ratio = mem::size_of::<u8>();

        let length = vec8.len() * ratio;
        let capacity = vec8.capacity() * ratio;
        let ptr = vec8.as_mut_ptr() as *mut u32;

        // Don't run the destructor for vec8
        mem::forget(vec8);

        // Construct new Vec
        Vec::from_raw_parts(ptr, length, capacity)
    };

    return vec32;
}

pub fn vec_u8_to_string(vec: &Vec<u8>) -> String {
    return str::from_utf8(vec).unwrap().to_string()
}
