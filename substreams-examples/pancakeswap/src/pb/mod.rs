use crate::pb::eth::Block;

#[path = "./dfuse.ethereum.r#type.v1.rs"]
pub mod eth;

#[path = "./pcs.types.v1.rs"]
pub mod pcs;

#[path = "./sf.substreams.tokens.v1.rs"]
pub mod tokens;

#[path = "./pcs.database.v1.rs"]
pub mod database;

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
