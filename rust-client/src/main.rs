mod pb;

use crate::pb::substreams::Request;
use prost::DecodeError;
use std::{env, fs};

// use futures::stream::Stream;
use tokio_stream::StreamExt;
use tonic::transport::Channel;

#[tokio::main]
async fn main() {
    let args: Vec<String> = env::args().collect();

    let filename = &args[0].as_str(); // /Users/eduardvoiculescu/git/streamingFast/substreams-playground/pcs-rust/substreams.request
    let grpc_endpoint = &args[1].as_str(); // "http://[::1]:9000"

    let contents = fs::read(filename).expect("Something went wrong reading the file");
    let request: pb::substreams::Request = decode(&contents).unwrap();

    let mut client = pb::substreams::stream_client::StreamClient::connect(grpc_endpoint)
        .await
        .unwrap();

    let request = tonic::Request::new(request);

    let mut stream = client.blocks(request).await.unwrap().into_inner();

    while let Some(resp) = stream.next().await {
        // let blk: pb::substreams::Response = resp.message();
        println!("{:?}", resp.unwrap().message)
    }

    // println!("{:?}", request.start_block_num);
    // println!("{:?}", request.stop_block_num);
}

pub fn decode<T: std::default::Default + prost::Message>(buf: &Vec<u8>) -> Result<T, DecodeError> {
    ::prost::Message::decode(&buf[..])
}
