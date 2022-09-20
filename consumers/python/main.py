#!/usr/bin/env python3

from ast import arg
import http.client
import sys
import os
import grpc
import sys

from sf.substreams.v1 import substreams_pb2_grpc
from sf.substreams.v1.substreams_pb2 import Request, STEP_IRREVERSIBLE
from sf.substreams.v1.package_pb2 import Package

jwt_token = os.getenv("SUBSTREAMS_API_TOKEN")
if not jwt_token: raise Error("set SUBSTREAMS_API_TOKEN")
endpoint = "api.streamingfast.io:443"
package_pb = "uniswap-v3-v0.1.0-beta.spkg"
output_modules = ["graph_out"]
start_block = 12369621
end_block = 12369800

def substreams_service():
    credentials = grpc.composite_channel_credentials(
        grpc.ssl_channel_credentials(),
        grpc.access_token_call_credentials(jwt_token),
    )
    channel = grpc.secure_channel(endpoint, credentials=credentials)
    return substreams_pb2_grpc.StreamStub(channel)

def main():
    with open(package_pb, 'rb') as f:
        pkg = Package()
        pkg.ParseFromString(f.read())

    service = substreams_service()
    stream = service.Blocks(Request(
        start_block_num=start_block,
        stop_block_num=end_block,
        fork_steps=[STEP_IRREVERSIBLE],
        modules=pkg.modules,
        output_modules=output_modules,
    ))

    for response in stream:
        print(response)

main()
