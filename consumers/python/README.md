Example python Substreams Consumer
----------------------------------


1. Install what is required to build protobufs:

```bash
python3 -m venv env
source env/bin/activate
pip3 install grpcio-tools
```

2. Grab some [released packages here](https://github.com/streamingfast/substreams-playground/releases). For example:

For example:

```
wget https://github.com/streamingfast/substreams-playground/releases/download/v0.5.0/pcs-v0.5.0.spkg
```

3. Code gen what is needed to interact with Substreams out of the package:

```
PKG=./pcs-v0.5.0.spkg
CMD="python3 -m grpc_tools.protoc --descriptor_set_in=$PKG --python_out=. --grpc_python_out=."

$CMD sf/substreams/v1/substreams.proto
$CMD sf/substreams/v1/package.proto
$CMD sf/substreams/v1/modules.proto
$CMD sf/substreams/v1/clock.proto
$CMD pcs/v1/pcs.proto
```

4. Get yourself [an access token](https://discord.gg/jZwqxJAvRs)

5. Run `main.py`

6. Profit, or jazz up
