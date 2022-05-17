use std::rc::Rc;

use once_cell::sync::Lazy;

use crate::pb::eth::{Block, Log, TransactionTrace};
use crate::util::keccak256;

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

pub struct FactoryContract {
    address: String,
    block: Rc<Block>,
}

impl FactoryContract {
    pub fn bind(block: Rc<Block>, address: &str) -> Self {
        Self {
            address: String::from(address),
            block,
        }
    }

    pub fn traces(&self) -> impl Iterator<Item = &TransactionTrace> {
        self.block
            .transaction_traces
            .iter()
            .filter(|tx| hex::encode(&tx.to) == self.address)
    }

    pub fn pool_created_events(&self) -> impl Iterator<Item = &Log> {
        self.traces()
            .flat_map(|tx| tx.receipt.as_ref().unwrap().logs.iter())
            .filter(|log| PoolCreatedEvent::matches(log))
    }
}
