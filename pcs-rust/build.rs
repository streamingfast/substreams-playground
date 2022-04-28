fn main() {
    println!("Building proto");
    let mut prost_build = prost_build::Config::new();
    prost_build.out_dir("./src/pb");
    prost_build
        .compile_protos(
            &[
                "codec_eth.proto",
                "pcs.proto",
                "tokens.proto",
                "pcs/database/v1/database.proto",
            ],
            &["../proto"],
        )
        .unwrap();
    println!("Done!");
}
