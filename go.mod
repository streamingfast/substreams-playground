module github.com/streamingfast/sparkle-pancakeswap

go 1.15

require (
	github.com/ghodss/yaml v1.0.0
	github.com/streamingfast/bstream v0.0.2-0.20220210135451-43aa5bfb9274
	github.com/streamingfast/dstore v0.1.1-0.20211012134319-16e840827e38
	github.com/streamingfast/eth-go v0.0.0-20210831180555-8d52c827993b
	github.com/streamingfast/logging v0.0.0-20210908162127-bdc5856d5341
	github.com/streamingfast/sparkle v0.0.0-20220128165829-a0218de3831c
	github.com/stretchr/testify v1.7.1-0.20210427113832-6241f9ab9942
	github.com/test-go/testify v1.1.4
	go.uber.org/zap v1.19.1
)

replace github.com/streamingfast/bstream => /Users/abourget/sf/bstream
replace github.com/streamingfast/sparkle => /Users/abourget/sf/sparkle
replace github.com/streamingfast/firehose => /Users/abourget/sf/firehose
