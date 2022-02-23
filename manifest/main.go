package main

import (
	"fmt"
	"io/ioutil"
)

func main() {
	yamlContent, err := ioutil.ReadFile("/Users/colin/code/sf/substream-pancakeswap/sample_substreams.yaml")
	if err != nil {
		panic(err)
	}

	manifest, err := DecodeYamlManifest(string(yamlContent))
	if err != nil {
		panic(err)
	}

	fmt.Println(manifest)
}
