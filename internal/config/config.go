package config

import (
	"flag"
)

type Config struct {
	Port int
}

func ParseFlags() *Config {
	cfg := &Config{}

	flag.IntVar(&cfg.Port, "port", 50051, "Port to listen on")

	flag.Parse()

	return cfg
}
