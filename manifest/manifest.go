package manifest

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"
)

type Manifest struct {
	Description  string   `yaml:"description"`
	GenesisBlock int      `yaml:"genesisBlock"`
	Streams      []Stream `yaml:"streams"`
}

type Stream struct {
	Name   string            `yaml:"name"`
	Kind   string            `yaml:"kind"`
	Code   string            `yaml:"code"`
	Inputs []string          `yaml:"inputs"`
	Output map[string]string `yaml:"output"`
}

func (s *Stream) Signature() ([]byte, error) {
	codeData, err := ioutil.ReadFile(s.Code)
	if err != nil {
		return nil, fmt.Errorf("could not read code %s: %w", s.Code, err)
	}

	buf := bytes.NewBuffer(nil)
	buf.WriteString(s.Name)
	buf.WriteString(s.Kind)

	sort.Strings(s.Inputs)
	for _, input := range s.Inputs {
		buf.WriteString(input)
	}

	buf.Write(codeData)

	h := sha1.New()
	h.Write(buf.Bytes())

	return h.Sum(nil), nil
}

func NewManifest(path string) (*Manifest, error) {
	_, manifest, err := DecodeYamlManifestFromFile(path)
	if err != nil {
		return nil, err
	}

	return manifest, nil
}

//wip

func (m *Manifest) ParseLinks() (*StreamLinks, error) {
	streamLinks := &StreamLinks{
		streams: map[string]Stream{},
		links:   map[string][]Stream{},
	}

	for _, stream := range m.Streams {
		streamLinks.streams[stream.Name] = stream
	}

	for _, stream := range m.Streams {
		links := []Stream{}
		for _, input := range stream.Inputs {
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

type linkedStreamItem struct {
	stream Stream
	depth  int
}

func (m *StreamLinks) Parents(rootName string) []Stream {
	parents := m.parents(rootName, 0)

	//sort by depth
	sort.Slice(parents, func(i, j int) bool {
		return parents[i].depth < parents[j].depth
	})

	result := []Stream{}

	for _, parent := range parents {
		result = append(result, parent.stream)
	}

	return result
}

func (m *StreamLinks) parents(rootName string, depth int) []linkedStreamItem {
	result := []linkedStreamItem{}

	for _, link := range m.links[rootName] {
		result = append(result, linkedStreamItem{
			stream: link,
			depth:  depth,
		})
		result = append(result, m.parents(link.Name, depth+1)...)
	}

	return result
}
