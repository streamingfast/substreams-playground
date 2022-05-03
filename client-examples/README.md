Dump that somewhere like `.bashrc`:
```bash
export STREAMINGFAST_KEY=server_YOUR_KEY_HERE  # Ask us on Discord for a key
function sftoken {
    export FIREHOSE_API_TOKEN=$(curl https://auth.dfuse.io/v1/auth/issue -s --data-binary '{"api_key":"'$STREAMINGFAST_KEY'"}' | jq -r .token)
	export SUBSTREAMS_API_TOKEN=$FIREHOSE_API_TOKEN
    echo Token set on FIREHOSE_API_TOKEN and SUBSTREAMS_API_TOKEN
}
```

Then in your shell, load a key in an env var with:

```bash
sftoken
```