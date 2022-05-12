#!/bin/bash -eu

cargo build --target wasm32-unknown-unknown --release

echo "Compiling Substreams manifest"
pushd substreams-examples > /dev/null
  for proj in *; do
    pushd $proj > /dev/null
      substreams manifest package substreams.yaml
    popd > /dev/null
  done
popd > /dev/null

