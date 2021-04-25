package configuration

// Feeds contains definition of feeds.yaml configuration file
type Feeds struct {
	// Sources contain instructions on how to obtain data from a source
	Sources map[string]Source `yaml:"sources"`
	// Feeds contain instructions on how to aggregate data from multiple sources
	Feeds map[string]Feed `yaml:"feeds"`
}

// Source describes HTTP data source
type Source struct {
	// URL is an HTTP(s) URL that will be used to make a request
	URL string `yaml:"url"`
	// Arguments contains array of arguments that this source accepts
	Arguments []string `yaml:"arguments"`
	// Headers contains map of headers that will be added to the request
	Headers map[string]string `yaml:"headers"`
	// Method contains HTTP method that will be used to make request.
	// As of now, this value only takes "get" or "post"
	Method string `yaml:"method"`
	// Parser contains instructions on how to extract data from the response
	Parser Parser `yaml:"parser"`
}

// Parser describes how response from HTTP data source should be processed
type Parser struct {
	// Type contains type of parser.
	// Currently, this value only takes "json"
	Type string `yaml:"type"`
	// Path contains a path that will be used to extract data from the response
	Path string `yaml:"path"`
}

// Feed describes how data should be sent to the smart-contract
type Feed struct {
	// Name contains ID of the feed that will be used to commit data
	Name string `yaml:"name"`
	// Aggregation defines how data sources will be combined together
	Aggregation Aggregation `yaml:"aggregation"`
}

// Aggregation describes how data from multiple data sources should be aggregated
type Aggregation struct {
	// Method contains aggregation method that will be used.
	// Currently, this value only takes "average"
	Method string `yaml:"method"`
	// Sources contains a list of sources that will be called in order to aggregate data
	Sources []AggregationSource `yaml:"sources"`
}

// AggregationSource describes data sources that will be used for aggregation
type AggregationSource struct {
	// Source contains source name that is defined in `sources` field
	Source string `yaml:"source"`
	// Arguments contains map of arguments that will be passed to the source
	Arguments map[string]string `yaml:"arguments"`
}
