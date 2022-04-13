use bigdecimal::BigDecimal;
use substreams::state;

pub fn sum_bigfloat_many(ord: u64, keys: &Vec<String>, value: &BigDecimal) {
    for key in keys {
        state::sum_bigfloat(ord as i64, key.to_string(), value);
    }
}

pub fn sum_int64_many(ord: u64, keys: &Vec<String>, value: i64) {
    for key in keys {
        state::sum_int64(ord as i64, key.to_string(), value);
    }
}

pub fn set_many(ord: u64, keys: &Vec<String>, value: &Vec<u8>) {
    for key in keys {
        state::set(ord as i64, key.to_string(), value);
    }
}
