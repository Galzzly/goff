package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const FileName = "goff.config.json"

type Config struct {
	PHPSESSID   string `mapstructure:"phpsessid"`
	CurrentYear int    `mapstructure:"currentyear"`
}

func Load() (Config, error) {
	v, _, err := newViper()
	if err != nil {
		return Config{}, err
	}

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if errors.As(err, &notFound) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if strings.TrimSpace(cfg.PHPSESSID) == "" {
		cfg.PHPSESSID = strings.TrimSpace(v.GetString("phpssid"))
	}

	return cfg, nil
}

func Save(cfg Config) error {
	v, path, err := newViper()
	if err != nil {
		return err
	}

	v.Set("phpsessid", cfg.PHPSESSID)
	v.Set("currentyear", cfg.CurrentYear)

	if err := v.WriteConfigAs(path); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	_ = os.Chmod(path, 0o600)
	return nil
}

func RepoRoot() (string, error) {
	if root := strings.TrimSpace(os.Getenv("GOFF_ROOT")); root != "" {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			return root, nil
		}
		return "", fmt.Errorf("GOFF_ROOT does not contain go.mod: %s", root)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get cwd: %w", err)
	}

	paths := []string{cwd}
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Dir(exe))
	}

	visited := make(map[string]struct{}, len(paths))
	for _, start := range paths {
		dir := start
		for {
			if _, seen := visited[dir]; seen {
				break
			}
			visited[dir] = struct{}{}
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				return dir, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	return "", errors.New("repo root not found (missing go.mod)")
}

func newViper() (*viper.Viper, string, error) {
	root, err := RepoRoot()
	if err != nil {
		return nil, "", err
	}

	path := filepath.Join(root, FileName)
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("json")
	return v, path, nil
}
