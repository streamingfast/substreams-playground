fn main() {
    let mut prost_build = prost_build::Config::new();
    prost_build.out_dir("./src/pb");
    prost_build
        .compile_protos(
            &[
                "codec_eth.proto",
                "pcs/v1/pcs.proto",
                "tokens.proto",
                "pcs/v1/database.proto",
            ],
            &["./proto/", "../../external-proto/", "../eth-token/proto/"],
        )
        .unwrap();
}
