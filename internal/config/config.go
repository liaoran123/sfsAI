package config

import "time"

type Config struct {
	Sidecar SidecarConfig
	Memory  MemoryConfig
	API     APIConfig
}

type SidecarConfig struct {
	DBPath           string
	GracefulShutdown time.Duration
}

type MemoryConfig struct {
	EnableEncryption bool
	EncryptionKey    []byte
	AutoCompress     bool
	DefaultTopK      int
}

type APIConfig struct {
	HTTPAddr     string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func DefaultConfig() *Config {
	return &Config{
		Sidecar: SidecarConfig{
			DBPath:           "./sfsai_data",
			GracefulShutdown: 10 * time.Second,
		},
		Memory: MemoryConfig{
			EnableEncryption: true,
			AutoCompress:     true,
			DefaultTopK:      10,
		},
		API: APIConfig{
			HTTPAddr:     ":8630",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
	}
}