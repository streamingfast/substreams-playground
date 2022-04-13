use crate::pb::eth::Block;

#[path = "./dfuse.ethereum.codec.v1.rs"]
pub mod eth;

#[path = "./pcs.types.v1.rs"]
pub mod pcs;

#[path = "./sf.substreams.tokens.v1.rs"]
pub mod tokens;

impl Block {
    pub fn timestamp(&self) -> String {
        self.header
            .as_ref()
            .unwrap()
            .timestamp
            .as_ref()
            .unwrap()
            .seconds
            .to_string()
    }
}
