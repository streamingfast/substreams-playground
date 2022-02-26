use std::convert::TryInto;
use std::io::Cursor;

pub mod eth {
    include!(concat!(env!("OUT_DIR"), "/dfuse.ethereum.codec.v1.rs"));
}

extern "C" {
    fn println(ptr: *const u8, len: usize);
    fn output(ptr: *const u8, len: usize);
    fn register_panic(msg_ptr: *const u8, msg_len: u32, file_ptr: *const u8, file_len: u32, line: u32, column: u32);
}

#[no_mangle]
pub extern "C" fn map(ptr: *mut u8, len: usize) {
    register_panic_hook();

    unsafe {
        let input_data = Vec::from_raw_parts(ptr, len, len);
        let mut buf = Cursor::new(&input_data);
        let blk: eth::Block = ::prost::Message::decode(&mut buf).unwrap();
        std::mem::forget(input_data); // otherwise tries to free that memory at the end and crashes


	// Log comments with:
        //let msg = format!("msg0"); println(msg.as_ptr(), msg.len());


	for trx in blk.transaction_traces {
	    let msg = format!("trx: {:?}", trx.hash); println(msg.as_ptr(), msg.len());
	}


        let mut out = Vec::<u8>::new();
        ::prost::Message::encode(&blk.header.unwrap(), &mut out).unwrap();

        let out_len = out.len();
        let ptr = out.as_ptr();
        std::mem::forget(out); // to prevent a drop which would crash
        output(ptr as *const u8, (out_len as i32).try_into().unwrap());
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
