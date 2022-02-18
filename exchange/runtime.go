package exchange


/// WASM, Rust, C++, C, AssemblyScript, TypeScript


// grpc endpoint, with protobuf:
/*


   grpc client call: firehose.v1.SubstreamRuntime{
     genesis_block = 2;  // 67000
     repeated Mapper mappers = 1;
     start_block_num = 3;  // 7000000
   }

   message Mapper {
     name string = 1;
     repeated string inputs = 4; // "blocks", or another named Mapper's "name"
     code bytes = 2;
     vm enum = 2.2; // WASM, Lua, CompileMyGoCodeHereV1
     json_params string = 3;
     kind enum = 6; // Map, BuildState
     build_state_store_name string = 5;
   }

*/
