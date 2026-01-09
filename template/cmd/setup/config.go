package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type configData struct {
	PHPSESSID string `json:"phpsessid"`
	Year      int    `json:"currentyear"`
}

func loadConfig() (configData, error) {
	path, err := configPath()
	if err != nil {
		return configData{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return configData{}, nil
		}
		return configData{}, fmt.Errorf("read config: %w", err)
	}

	var cfg configData
	if err := json.Unmarshal(data, &cfg); err != nil {
		return configData{}, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}

func saveConfig(cfg configData) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

func configPath() (string, error) {
	root, err := repoRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, configFileName), nil
}
