package config

import (
	"encoding/json"
	"os"
)

type APIInfo struct {
	Addr  string `json:"addr"`
	Token string `json:"token"`
}

type Config struct {
	Listen string             `json:"listen"`
	Lotus  APIInfo            `json:"lotus"`
	Miners map[string]APIInfo `json:"miners"`
}

func LoadConfig(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var c Config
	err = json.Unmarshal(raw, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func DefaultConfig() *Config {
	lotus := APIInfo{
		Addr:  "",
		Token: "",
	}
	miners := make(map[string]APIInfo)
	miners["f01155"] = lotus
	miners["f010202"] = lotus

	return &Config{
		Listen: "0.0.0.0:6789",
		Lotus:  lotus,
		Miners: miners,
	}
}
