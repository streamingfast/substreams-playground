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

	Graph *StreamsGraph `yaml:"-"`
}

type Stream struct {
	Name   string            `yaml:"name"`
	Kind   string            `yaml:"kind"`
	Code   string            `yaml:"code"`
	Inputs []string          `yaml:"inputs"`
	Output map[string]string `yaml:"output"`
}

func (s *Stream) Signature(graph *StreamsGraph) ([]byte, error) {
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

	ancestors, err := graph.AncestorsOf(s.Name)
	if err != nil {
		return nil, err
	}
	for _, ancestor := range ancestors {
		sig, err := ancestor.Signature(graph)
		if err != nil {
			return nil, err
		}
		buf.Write(sig)
	}

	h := sha1.New()
	h.Write(buf.Bytes())

	return h.Sum(nil), nil
}

func New(path string) (*Manifest, error) {
	_, manif, err := DecodeYamlManifestFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("decoding yaml: %w", err)
	}

	graph, err := NewStreamsGraph(manif.Streams)
	if err != nil {
		return nil, fmt.Errorf("computing streams graph: %w", err)
	}

	manif.Graph = graph

	return manif, nil
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

func (g *StreamsGraph) StreamsFor(streamName string) ([]Stream, error) {
	thisStream, found := g.streams[streamName]
	if !found {
		return nil, fmt.Errorf("stream %q not found", streamName)
	}

	parents := g.ancestorsOf(streamName)
	return append(parents, thisStream), nil
}

func (g *StreamsGraph) AncestorsOf(streamName string) ([]Stream, error) {
	parents := g.ancestorsOf(streamName)
	return parents, nil
}

func (g *StreamsGraph) ancestorsOf(streamName string) []Stream {
	type streamWithTreeDepth struct {
		stream Stream
		depth  int
	}

	var dfs func(rootName string, depth int, alreadyVisited map[string]struct{}) []streamWithTreeDepth
	dfs = func(rootName string, depth int, alreadyVisited map[string]struct{}) []streamWithTreeDepth {
		var result []streamWithTreeDepth
		for _, link := range g.links[rootName] {
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

	//sort by depth in reverse order
	sort.Slice(parentsWithDepth, func(i, j int) bool {
		return parentsWithDepth[i].depth > parentsWithDepth[j].depth
	})

	var result []Stream
	for _, parent := range parentsWithDepth {
		result = append(result, parent.stream)
	}

	return result
}
