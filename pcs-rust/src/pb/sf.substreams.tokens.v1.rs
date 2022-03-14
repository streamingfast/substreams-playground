#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Tokens {
    #[prost(message, repeated, tag="1")]
    pub tokens: ::std::vec::Vec<Token>,
}
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct Token {
    #[prost(string, tag="1")]
    pub address: std::string::String,
    #[prost(string, tag="2")]
    pub name: std::string::String,
    #[prost(string, tag="3")]
    pub symbol: std::string::String,
    #[prost(uint64, tag="4")]
    pub decimals: u64,
}
