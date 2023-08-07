package config

import (
	"encoding/json"
	"os"
)

type APIInfo struct {
	Addr  string `json:"addr"`
	Token string `json:"token"`
}

type RecordInterval struct {
	Lotus  string `json:"lotus"`
	Miner  string `json:"miner"`
	FilFox string `json:"filFox"`
	Blocks string `json:"blocks"`
}

type Config struct {
	Listen         string               `json:"listen"`
	Lotus          APIInfo              `json:"lotus"`
	Miners         map[string]APIInfo   `json:"miners"`
	Running        map[string][2]string `json:"running"`
	RecordInterval RecordInterval       `json:"recordInterval"`
	FilFoxURL      string               `json:"filFoxURL"`
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
		Addr:  "10.122.1.29:1234",
		Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJBbGxvdyI6WyJyZWFkIl19.l04qKWmgyDRqeT3kjMfxxhQpKwLmYk8eeDIW-NcaX_c",
	}
	miner := APIInfo{
		Addr:  "10.122.1.29:2345",
		Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJBbGxvdyI6WyJyZWFkIiwid3JpdGUiLCJzaWduIiwiYWRtaW4iXX0.tlJ8d4RIudknLHrKDSjyKzfbh8hGp9Ez1FZszblQLAI",
	}
	miners := make(map[string]APIInfo)
	miners["t017387"] = miner

	running := map[string][2]string{
		"AP":  {"1m", "2m"},
		"PC1": {"3h", "5h"},
		"PC2": {"5m", "10m"},
		"GET": {"1m", "2m"},
	}

	interval := RecordInterval{
		Lotus:  "30s",
		Miner:  "30s",
		FilFox: "3m",
		Blocks: "1m",
	}

	return &Config{
		Listen:         "0.0.0.0:6789",
		Lotus:          lotus,
		Miners:         miners,
		Running:        running,
		RecordInterval: interval,
		FilFoxURL:      "https://calibration.filfox.info/api/v1",
		//FilFoxURL:      "https://filfox.info/api/v1",
	}
}
