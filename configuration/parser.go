package configuration

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"strings"
  "time"
)

// ParseRequests takes Reader, and uses yaml library to decode file into Requests struct
func ParseRequests(file io.Reader) (*Requests, error) {
	dec := yaml.NewDecoder(file)
	dec.KnownFields(true)
	r := &Requests{}
	err := dec.Decode(r)
	if err != nil {
		return nil, err
	}
	r.Timeout, err = time.ParseDuration(r.RawTimeout)
	if err != nil {
		return nil, err
	}
	//r.DataFilter.Delay, err = time.ParseDuration(r.DataFilter.RawDelay)
	//if err != nil {
	//	return nil, err
	//}
	return r, nil
}

var (
	ErrUnsupportedAggregationMethod = errors.New("unsupported aggregation method")
	ErrUnknownSource                = errors.New("unknown source")
	ErrInvalidSourceArguments       = errors.New("invalid source arguments")
	ErrNoSources                    = errors.New("no sources")
	ErrInvalidSourceMethod          = errors.New("invalid source method")
	ErrInvalidParserType            = errors.New("invalid parser type")
)

// ParseFeeds takes Reader, and uses yaml library to decode file into Feeds struct
func ParseFeeds(file io.Reader) (*Feeds, error) {
	dec := yaml.NewDecoder(file)
	dec.KnownFields(true)
	f := &Feeds{}
	err := dec.Decode(f)
	if err != nil {
		return nil, err
	}
	err = ValidateFeeds(f)
	return f, err
}

// ValidateFeeds takes Feeds struct and runs a few sanity checks to make sure configuration is valid
func ValidateFeeds(feeds *Feeds) error {
	for _, v := range feeds.Sources {
		// Make sure that HTTP method is supported
		if v.Method != "get" && v.Method != "post" {
			return ErrInvalidSourceMethod
		}
		// TODO: check that URL is valid
		// Make sure that parser type is known
		if v.Parser.Type != "json" {
			return ErrInvalidParserType
		}
		// TODO: make sure that parser path is valid
	}

	for _, v := range feeds.Feeds {
		// Make sure that we understand the aggregation method
		if v.Aggregation.Method != "average" {
			return ErrUnsupportedAggregationMethod
		}
		// Make sure there are sources defined
		if len(v.Aggregation.Sources) == 0 {
			return ErrNoSources
		}
		// Check every source
		for _, src := range v.Aggregation.Sources {
			source, ok := feeds.Sources[src.Source]
			// Make sure that source exists
			if !ok {
				return ErrUnknownSource
			}
			// Make sure that all arguments are provided
			if len(src.Arguments) != len(source.Arguments) {
				return ErrInvalidSourceArguments
			}
			// Make sure that all arguments are specified correctly
			for _, arg := range source.Arguments {
				if _, ok := src.Arguments[arg]; !ok {
					return ErrInvalidSourceArguments
				}
			}
		}
	}
	return nil
}

func ExpandVariables(value string, variables map[string]string) string {
	for k, v := range variables {
		value = strings.ReplaceAll(value, fmt.Sprintf("${%s}", k), v)
	}
	return value
}
