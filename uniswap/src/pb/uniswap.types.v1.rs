#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Pools {
    #[prost(message, repeated, tag="1")]
    pub pools: ::prost::alloc::vec::Vec<Pool>,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Pool {
    #[prost(uint64, tag="1")]
    pub created_at_timestamp: u64,
    #[prost(uint64, tag="2")]
    pub created_at_block_number: u64,
    #[prost(string, tag="3")]
    pub token0: ::prost::alloc::string::String,
    #[prost(string, tag="4")]
    pub token1: ::prost::alloc::string::String,
}
