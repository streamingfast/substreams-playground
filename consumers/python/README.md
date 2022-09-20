Example python Substreams Consumer
----------------------------------


1. Install what is required to build protobufs:

```bash
python3 -m venv env
source env/bin/activate
pip3 install grpcio-tools protobuf==3.20.1
```

2. Grab some [released packages here](https://github.com/streamingfast/substreams-playground/releases). For example:

For example:

```
wget https://github.com/streamingfast/substreams-uniswap-v3/releases/download/v0.1.0-beta/uniswap-v3-v0.1.0-beta.spkg
```

3. Code gen what is needed to interact with Substreams out of the package:

```
PKG=./uniswap-v3-v0.1.0-beta.spkg
alias protogen_py="python3 -m grpc_tools.protoc --descriptor_set_in=$PKG --python_out=. --grpc_python_out=."

protogen_py sf/substreams/v1/substreams.proto
protogen_py sf/substreams/v1/package.proto
protogen_py sf/substreams/v1/modules.proto
protogen_py sf/substreams/v1/clock.proto
protogen_py uniswap/v1/uniswap.proto
unalias protogen_py
```

4. Get yourself [an access token](https://discord.gg/jZwqxJAvRs)

5. Run `main.py`

```bash
python3 main.py # wait a couple of seconds to get the reponses
```

6. Profit, or jazz up
