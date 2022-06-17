# Mongodb loader

### Running mongodb cli loader
1. Deploy a mongodb (docker, Robo 3T or any kind of tool). If deployed with docker, make sure the port is exposed.

2. Run command below
```bash
mongo load ./substreams.yaml db_out --firehose-endpoint localhost:9000 -p -s 6810706 -t 6810806 # local deployment of firehose
mongo load ./substreams.yaml db_out --firehose-endpoint bsc-dev.streamingfast.io -s -s 6810706 -t 6810806 # run remotely against bsc
```
