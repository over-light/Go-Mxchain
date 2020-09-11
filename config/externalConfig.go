package config

// ExternalConfig will hold the configurations for external tools, such as Explorer or Elastic Search
type ExternalConfig struct {
	ElasticSearchConnector ElasticSearchConfig
}

// ElasticSearchConfig will hold the configuration for the elastic search
type ElasticSearchConfig struct {
	Enabled          bool
	IndexerCacheSize int
	URL              string
	UseKibana        bool
	Username         string
	Password         string
}
