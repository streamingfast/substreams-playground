#[derive(Clone, PartialEq, ::prost::Message)]
pub struct TokenTransfers {
    #[prost(message, repeated, tag="1")]
    pub transfers: ::prost::alloc::vec::Vec<TokenTransfer>,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct TokenTransfer {
    #[prost(string, tag="4")]
    pub transaction_id: ::prost::alloc::string::String,
    #[prost(uint64, tag="5")]
    pub ordinal: u64,
    #[prost(bytes="vec", tag="1")]
    pub from: ::prost::alloc::vec::Vec<u8>,
    #[prost(bytes="vec", tag="2")]
    pub to: ::prost::alloc::vec::Vec<u8>,
    #[prost(string, tag="3")]
    pub amount: ::prost::alloc::string::String,
}
