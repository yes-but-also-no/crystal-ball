package configuration

import "time"

// Requests contains definition of requests.yml configuration file
type Requests struct {
	// Filter contains URL filter configuration
	Filter Filter `yaml:"filter"`
	// RawTimeout contains HTTP request timeout in time.Duration format
	RawTimeout string `yaml:"timeout"`
	// Timeout contains parsed RawTimeout
	Timeout time.Duration `yaml:"-"`
	// DataFilter contains configuration for random data prevention filter
	//DataFilter DataFilter `yaml:"data_filter"`
}

// Filter describes URL filter
type Filter struct {
	// Mode contains whether Domains is a whitelist or a blacklist.
	// Can only take "whitelist" or "blacklist"
	Mode string `yaml:"mode"`
	// Domains contains list of domains to filter
	Domains []string `yaml:"domains"`
}

// DataFilter describes random data prevention filter
type DataFilter struct {
	// OnlyNumbers describes that node will only submit values that are numbers
	OnlyNumbers bool `yaml:"only_numbers"`
	// Samples contains amount of samples to derive end result from
	Samples int `yaml:"samples"`
	// RawDelay contains time.Duration encoded delay between sample request
	RawDelay string `yaml:"delay"`
	// Delay contains parsed RawDelay
	Delay time.Duration `yaml:"-"`
}
