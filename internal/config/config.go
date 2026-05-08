package config

type Config struct {
	Profile               string
	Region                string
	EndpointURL           string
	AllowInsecureEndpoint bool
	PathStyle             bool
	Concurrency           int
	PartSize              int64
}
