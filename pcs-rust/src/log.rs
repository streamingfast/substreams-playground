use crate::externs;

pub fn log(msg: String) {
    unsafe {
        externs::println(msg.as_ptr(), msg.len());
    }
}

pub fn println(msg: String) {
    log(msg);
}
