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

type StreamsGraph struct {
	streams map[string]Stream
	links   map[string][]Stream
}

func NewStreamsGraph(streams []Stream) (*StreamsGraph, error) {
	sg := &StreamsGraph{
		streams: map[string]Stream{},
		links:   map[string][]Stream{},
	}

	for _, stream := range streams {
		sg.streams[stream.Name] = stream
	}

	for _, stream := range streams {
		var links []Stream
		for _, input := range stream.Inputs {
			for _, streamPrefix := range []string{"stream:", "store:"} {
				if strings.HasPrefix(input, streamPrefix) {
					linkName := strings.TrimPrefix(input, streamPrefix)
					linkedStream, ok := sg.streams[linkName]
					if !ok {
						return nil, fmt.Errorf("stream %s does not exist", linkName)
					}
					links = append(links, linkedStream)
				}
			}

		}
		sg.links[stream.Name] = links
	}

	return sg, nil
}

func (g *StreamsGraph) ParentsOf(streamName string) ([]Stream, error) {
	if _, ok := g.streams[streamName]; !ok {
		return nil, fmt.Errorf("stream not found")
	}

	return g.parentsOf(streamName), nil
}

func (m *StreamsGraph) parentsOf(streamName string) []Stream {
	type streamWithTreeDepth struct {
		stream Stream
		depth  int
	}

	var dfs func(rootName string, depth int, alreadyVisited map[string]struct{}) []streamWithTreeDepth
	dfs = func(rootName string, depth int, alreadyVisited map[string]struct{}) []streamWithTreeDepth {
		var result []streamWithTreeDepth
		for _, link := range m.links[rootName] {
			if _, ok := alreadyVisited[link.Name]; ok {
				continue
			}

			result = append(result, streamWithTreeDepth{
				stream: link,
				depth:  depth,
			})
			alreadyVisited[link.Name] = struct{}{}

			result = append(result, dfs(link.Name, depth+1, alreadyVisited)...)
		}

		return result
	}

	parentsWithDepth := dfs(streamName, 0, map[string]struct{}{})

	//sort by depth
	sort.Slice(parentsWithDepth, func(i, j int) bool {
		return parentsWithDepth[i].depth < parentsWithDepth[j].depth
	})

	var result []Stream
	for _, parent := range parentsWithDepth {
		result = append(result, parent.stream)
	}

	return result
}
