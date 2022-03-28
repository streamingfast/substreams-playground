use once_cell::sync::Lazy;

use crate::pb::eth::Log;
use crate::util::keccak256;

pub const ADDRESS: &'static str = "1f98431c8ad98523631ae4a59f267346ea31f984";

static POOL_CREATED_SIGNATURE: Lazy<[u8; 32]> = Lazy::new(|| {
    let signature = "PoolCreated(address,address,uint24,int24,address)";
    keccak256(signature.as_bytes())
});

pub struct PoolCreatedEvent {}

impl PoolCreatedEvent {
    pub fn matches(log: &Log) -> bool {
        &log.topics[0] == &*POOL_CREATED_SIGNATURE
    }
}
