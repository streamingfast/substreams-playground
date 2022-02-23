package main

import (
	"bytes"
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

func DecodeYamlManifestFromFile(yamlFilePath string) (string, *Manifest, error) {
	yamlContent, err := ioutil.ReadFile(yamlFilePath)
	if err != nil {
		return "", nil, fmt.Errorf("reading subgraph file %q: %w", yamlFilePath, err)
	}

	subgraphManifest, err := DecodeYamlManifest(string(yamlContent))
	if err != nil {
		return "", nil, fmt.Errorf("decoding subgraph file %q: %w", yamlFilePath, err)
	}

	return string(yamlContent), subgraphManifest, nil
}

func DecodeYamlManifest(manifestContent string) (*Manifest, error) {
	var subgraphManifest *Manifest
	if err := yaml.NewDecoder(bytes.NewReader([]byte(manifestContent))).Decode(&subgraphManifest); err != nil {
		return nil, fmt.Errorf("decoding manifest content %q: %w", manifestContent, err)
	}

	return subgraphManifest, nil
}
