use std::convert::TryInto;
use std::io::Cursor;

pub mod eth {
    include!(concat!(env!("OUT_DIR"), "/dfuse.ethereum.codec.v1.rs"));
}

extern "C" {
    fn println(ptr: *const u8, len: usize);
}

#[no_mangle]
pub extern "C" fn map(ptr: *mut u8, len: usize) -> i32 {
    unsafe {
        let input_data = Vec::from_raw_parts(ptr, len, len);

	let ptr_info = format!("input ptr 0 {:?} {:?} {:?}", &input_data, ptr, len);
	println(ptr_info.as_ptr(), ptr_info.len());

	let msg = format!("msg0"); println(msg.as_ptr(), msg.len());

	let buf = Cursor::new(input_data);

	let msg = format!("msg0-1"); println(msg.as_ptr(), msg.len());

	let blk: eth::Block = ::prost::Message::decode(buf).unwrap();

	let msg = format!("msg1"); println(msg.as_ptr(), msg.len());
	
	let mut out = Vec::<u8>::new();
	::prost::Message::encode(&blk.header.unwrap(), &mut out).unwrap();

	let msg = format!("msg2"); println(msg.as_ptr(), msg.len());

	let out_len = out.len();
	let ptr = out.as_ptr();
	std::mem::forget(out);
	println(ptr as *const u8, (out_len as i32).try_into().unwrap());

	let msg = format!("msg3"); println(msg.as_ptr(), msg.len());
	
	0
    }

    // println!("input {:?} {:?} {:?}", ptr, len, input_data);
    // unsafe {
    //     let ptr_info = format!("input ptr {:?}", buf);
    //     println(ptr_info.as_ptr(), ptr_info.len());
    // }

    // let slice = unsafe { slice::from_raw_parts(ptr as _, len as _) };

    // let formated = format!("Hello {}, ca marche pontiac", string_from_host);
    // unsafe {
    //     println(formated.as_ptr(), formated.len());
    // }

    // let from_within = format!("This {}, comes from within", string_from_host);

    // ptr = from_within.as_mut_ptr();
    // std::mem::forget(from_within);

    // from_within.len()
    // 0
}

// Ref: https://github.com/infinyon/fluvio/blob/master/crates/fluvio-smartmodule-derive/src/generator/map.rs#L73
