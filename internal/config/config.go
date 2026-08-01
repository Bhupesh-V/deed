package config

type Config struct {
	DSN  string
	Name string
}

func New() *Config {
	return &Config{}
}
