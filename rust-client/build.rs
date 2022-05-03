fn main() {
    tonic_build::configure()
        .out_dir("src/pb")
        .compile(
            &["sf/substreams/v1/substreams.proto", "tokens.proto"],
            &["../../substreams/proto", "../eth-token/proto/"],
        )
        .unwrap();
}
