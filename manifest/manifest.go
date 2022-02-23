package main

import (
	"fmt"
	"strings"
)

type Manifest struct {
	Description string   `yaml:"description"`
	Streams     []Stream `yaml:"streams"`
}

type Stream struct {
	Name   string            `yaml:"name"`
	Kind   string            `yaml:"kind"`
	Code   string            `yaml:"code"`
	Input  []string          `yaml:"input"`
	Output map[string]string `yaml:"output"`
}

func NewManifest(path string) (*Manifest, error) {
	_, manifest, err := DecodeYamlManifestFromFile(path)
	if err != nil {
		return nil, err
	}

	return manifest, nil
}

//wip
func ParseManifestLinks(manifest *Manifest) (*StreamLinks, error) {
	streamLinks := &StreamLinks{
		streams: map[string]Stream{},
		links:   map[string][]Stream{},
	}

	for _, stream := range manifest.Streams {
		streamLinks.streams[stream.Name] = stream
	}

	for _, stream := range manifest.Streams {
		links := []Stream{}
		for _, input := range stream.Input {
			if strings.HasPrefix(input, "stream:") {
				linkName := strings.TrimPrefix(input, "stream:")
				linkedStream, ok := streamLinks.streams[linkName]
				if !ok {
					return nil, fmt.Errorf("stream %s does not exist", linkName)
				}
				links = append(links, linkedStream)
			}
		}
		streamLinks.links[stream.Name] = links
	}

	return streamLinks, nil
}

type StreamLinks struct {
	streams map[string]Stream
	links   map[string][]Stream
}
