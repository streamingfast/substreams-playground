#!/bin/bash -eu

pushd substreams-examples
  for proj in *; do
    pushd $proj  
      ./build.sh
      substreams manifest package substreams.yaml 
    popd
  done
popd

