package config

type Config struct {
	Profile     string
	Region      string
	EndpointURL string
	PathStyle   bool
	Concurrency int
	PartSize    int64
}
