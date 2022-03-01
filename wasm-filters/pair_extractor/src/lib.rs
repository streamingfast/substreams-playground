mod memory;
mod proto;

use std::convert::TryInto;
use std::io::Cursor;
use hex;
use crate::proto::proto::decode;

pub mod eth {
    include!(concat!(env!("OUT_DIR"), "/dfuse.ethereum.codec.v1.rs"));
}
pub mod pcs {
    include!(concat!(env!("OUT_DIR"), "/pcs.types.v1.rs"));
}

extern "C" {
    fn println(ptr: *const u8, len: usize);
    fn output(ptr: *const u8, len: usize);
    fn register_panic(msg_ptr: *const u8, msg_len: u32, file_ptr: *const u8, file_len: u32, line: u32, column: u32);
    fn state_set(ord: i64, key_ptr: *const u8, key_len: u32, value_ptr: *const u8, value_len u32);
    fn state_get_at(store_idx: u32, ord: i64, key_ptr: *const u8, key_len: u32, value_ptr: *const u8, value_len u32);

    fn state_get_pairs_at()
}

#[no_mangle]
pub extern "C" fn map_pairs(block_ptr: *mut u8, block_len: usize) {
    register_panic_hook();

    let blk: eth::Block = decode(ptr, len);

	let mut pairs = pcs::Pairs{pairs: vec![]};
	for trx in blk.transaction_traces {
	    if hex::encode(&trx.to) != "ca143ce32fe78f1f7019d7d551a6402fc5350c73" /* PCS Factory address */ {
		continue
	    }

	    for log in trx.receipt.unwrap().logs {
            let sig = hex::encode(&log.topics[0]);
            let msg = format!("trx: 0x{} sig: {}", hex::encode(&trx.hash), &sig);

            unsafe {
                println(msg.as_ptr(), msg.len());
            }

            if sig != "0d3648bd0f6ba80134a33ba9275ac585d9d315f0ad8355cddefde31afa28d0e9" {
                continue
            }

            // topics[0] is the event signature
            let pair_token0 = decode_address(&log.topics[1]);
            let pair_token1 = decode_address(&log.topics[2]);
            let pair_addr = decode_address(&log.data);

            pairs.pairs.push(pcs::Pair{
                address: pair_addr.clone(),
                token0: pair_token0,
                token1: pair_token1,
                creation_transaction_id: hex::encode(&trx.hash),
                block_num: blk.number,
                log_ordinal: log.block_index as u64
                })
            }
	}

    let mut out = Vec::<u8>::new();
    ::prost::Message::encode(&pairs, &mut out).unwrap();

    let out_len = out.len();
    let ptr = out.as_ptr();
    std::mem::forget(out); // to prevent a drop which would crash

    unsafe {
        output(ptr as *const u8, (out_len as i32).try_into().unwrap());
    }

}

#[no_mangle]
pub extern "C" fn build_pairs_state(pairs_ptr: *mut u8, pairs_len: usize) {
    register_panic_hook();

    unsafe {
	let input_data = Vec::from_raw_parts(pairs_ptr, pairs_len, pairs_len);
        let pairs: eth::Pairs = ::prost::Message::decode(&mut Cursor::new(&input_data)).unwrap();
        std::mem::forget(input_data); // otherwise tries to free that memory at the end and crashes

	for pair in pairs {
	    let key = format!("pair:{}", pair.address);
	    let mut val = Vec::<u8>::new();
	    ::prost::Message::encode(&pair, &mut val).unwrap();
	    store_set(pair.log_ordinal, key.as_ptr(), key.len(), val.as_ptr(), val.len());
	}
    }
}

pub extern "C" fn map_reserves(block_ptr: *mut u8, block_len: usize, pairs_store_idx i32) {
    
}

fn decode_address(input: &Vec::<u8>) -> String {
    if input.len() > 40 {
	"larger".to_string()
    } else {
	hex::encode(input)
    }
}


// Ref: https://github.com/infinyon/fluvio/blob/master/crates/fluvio-smartmodule-derive/src/generator/map.rs#L73

// See: https://github.com/Jake-Shadle/wasmer-rust-example/blob/master/wasm-sample-app/src/lib.rs
fn hook(info: &std::panic::PanicInfo<'_>) {
    let error_msg = info
        .payload()
        .downcast_ref::<String>()
        .map(String::as_str)
        .or_else(|| info.payload().downcast_ref::<&'static str>().copied())
        .unwrap_or("");
    let location = info.location();

    unsafe {
        let _ = match location {
            Some(loc) => {
                let file = loc.file();
                let line = loc.line();
                let column = loc.column();

                register_panic(
                    error_msg.as_ptr(),
                    error_msg.len() as u32,
                    file.as_ptr(),
                    file.len() as u32,
                    line,
                    column,
                )
            }
            None => register_panic(
                error_msg.as_ptr(),
                error_msg.len() as u32,
                std::ptr::null(),
                0,
                0,
                0,
            ),
        };
    }
}

fn register_panic_hook() {
    use std::sync::Once;
    static SET_HOOK: Once = Once::new();
    SET_HOOK.call_once(|| {
        std::panic::set_hook(Box::new(hook));
    });
}