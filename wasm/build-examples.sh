#!/bin/bash -ex


pushd example-block
  PROTOC_INCLUDE=. wasm-pack build --target nodejs
popd
