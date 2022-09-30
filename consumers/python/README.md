Example python Substreams Consumer
----------------------------------


1. Install what is required to build protobufs:

```bash
python3 -m venv env
source env/bin/activate
pip3 install grpcio-tools protobuf==3.20.1
# important note, 3.20.1 works newer updated protobuf seem to cause issues -> https://github.com/protocolbuffers/protobuf/issues/10571
```

2. Grab some [released packages here](https://github.com/streamingfast/substreams-playground/releases). For example:

For example on Linux based systems:

```
wget https://github.com/streamingfast/substreams-uniswap-v3/releases/download/v0.1.0-beta/uniswap-v3-v0.1.0-beta.spkg
```

Or, for macOS

```
curl -L https://github.com/streamingfast/substreams-uniswap-v3/releases/download/v0.1.0-beta/uniswap-v3-v0.1.0-beta.spkg --output uniswap-v3-v0.1.0-beta.spkg
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

4. Request a StreamingFast [authentication token](https://substreams.streamingfast.io/reference-and-specs/authentication). This is a requirement of connecting to and using Substreams.

5. Run `main.py` from the terminal. Give the code, server and internet connection about sixty seconds to work. If everything has been configured and set up correctly output will begin printing to the terminal window.

```bash
python3 main.py
```

6. Output will be printed to the terminal window for the request. The results will be formatted as seen in the following example.

```
progress {
}

data {
  outputs {
    name: "graph_out"
    map_output {
      type_url: "type.googleapis.com/uniswap.types.v1.EntitiesChanges"
      value: "\n \361\025l\267\341\242\254\2524\214f\375\304[\002\362Tp$\336\3263P\251jF\355\022\306+O\364\020\325\375\362\005\032 j;\262\357\n \365P4\225#\216T\376\3626e\237V\361\305~\026\002\260\336+=y\237\341T \324\375\362\005"
    }
  }
  clock {
    id: "f1156cb7e1a2acaa348c66fdc45b02f2547024ded63350a96a46ed12c62b4ff4"
    number: 12369621
    timestamp {
      seconds: 1620156420
    }
  }
  step: STEP_NEW
  cursor: "uP1NIt-G1BoMJkrEevLWC6WwLpc_DFttVg3sLBNAj9338XfDjs-hVWAkakyDwfjz2R24TVz6iozJESx78JZSudXtw7pguXRsQHgvlom_8rK-fPWnbQsZIr1gDe7YNNzRWj7UZAP9eLAKttTmP_WKNUVnY5YkfTXi3jxUqtZSdvVF6ncwkTz_Jc3X1PyW84YVqrYjR-eklXqqAWR5KhoOPM7RYfDNu2p2YQ=="
}

data {
  outputs {
    name: "graph_out"
    map_output {
      type_url: "type.googleapis.com/uniswap.types.v1.EntitiesChanges"
      value: "\n py\372$1\355\247\323\320\271\271f\2740Yw-V\353S\302\263\210Jja\307\035\226\340Z\237\020\326\375\362\005\032 \361\025l\267\341\242\254\2524\214f\375\304[\002\362Tp$\336\3263P\251jF\355\022\306+O\364 \325\375\362\005"
    }
  }
  clock {
    id: "7079fa2431eda7d3d0b9b966bc3059772d56eb53c2b3884a6a61c71d96e05a9f"
    number: 12369622
    timestamp {
      seconds: 1620156455
    }
  }
  step: STEP_NEW
  cursor: "BxHMeHHf1EAQKO-vKB7O0qWwLpc_DFttVg3sLBNDj4z293uTjJ-iA2AgPEzXxKqk3UfiGVOq2I2eF35--8dXvoXvxe026CJrRCkvm4HqqrK-fvKhPgtPeL03X-_fa47RWj7UZAP9eLMK59XgM6WIZUYxY5JyfWHnjGtQ8IwGeaUX6yA2wzn0dMjQhP6QpNBE_LEnFuepnS_yAWR7LRxdPJiLYaefum0pMw=="
}
```
