use std::env;
use std::path::PathBuf;

fn main() {
    tonic_build::configure()
        .out_dir("src/pb")
        .compile(
            &["sf/substreams/v1/substreams.proto"],
            &["/Users/eduardvoiculescu/git/streamingFast/substreams/proto"],
        )
        .unwrap();
}
