# Mongodb loader

### Create a schema for the database changes
For the moment, we only support these types:
- INTEGER
- DOUBLE
- BOOLEAN
- TIMESTAMP
- NULL
- DATE
- STRING (default value for mongodb)

The schema has to be a json file and has to be passed in with the flag `--mongodb-schema` (default value is `./schema.json`).

Here is an example of a schema:
```json
{
  "pair": {
    "timestamp" : "timestamp",
    "block": "integer"
  }
}
```

> Note: any other field which is of type string isn't necessary to be declared 

### Running mongodb cli loader
1. Deploy a mongodb (docker, Robo 3T or any kind of tool). If deployed with docker, make sure the port is exposed.

2. Run command below
```bash
# local deployment of firehose
mongo load ./substreams.yaml db_out --firehose-endpoint localhost:9000 -p -s 6810706 -t 6810806 
# run remotely against bsc
mongo load ./substreams.yaml db_out --firehose-endpoint bsc-dev.streamingfast.io -s 6810706 -t 6810806 
```
