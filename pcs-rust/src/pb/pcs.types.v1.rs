#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Pairs {
    #[prost(message, repeated, tag="1")]
    pub pairs: ::std::vec::Vec<Pair>,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Pair {
    #[prost(string, tag="1")]
    pub address: std::string::String,
    #[prost(message, optional, tag="2")]
    pub erc20_token0: ::std::option::Option<Erc20Token>,
    #[prost(message, optional, tag="3")]
    pub erc20_token1: ::std::option::Option<Erc20Token>,
    #[prost(string, tag="4")]
    pub creation_transaction_id: std::string::String,
    #[prost(uint64, tag="5")]
    pub block_num: u64,
    #[prost(uint64, tag="6")]
    pub log_ordinal: u64,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Erc20Token {
    #[prost(string, tag="1")]
    pub address: std::string::String,
    #[prost(string, tag="2")]
    pub name: std::string::String,
    #[prost(string, tag="3")]
    pub symbol: std::string::String,
    #[prost(uint64, tag="4")]
    pub decimals: u64,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Reserves {
    #[prost(message, repeated, tag="1")]
    pub reserves: ::std::vec::Vec<Reserve>,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Reserve {
    #[prost(uint64, tag="1")]
    pub log_ordinal: u64,
    #[prost(string, tag="2")]
    pub pair_address: std::string::String,
    #[prost(string, tag="3")]
    pub reserve0: std::string::String,
    #[prost(string, tag="4")]
    pub reserve1: std::string::String,
    #[prost(string, tag="5")]
    pub token0_price: std::string::String,
    #[prost(string, tag="6")]
    pub token1_price: std::string::String,
}
