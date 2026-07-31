package config

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	EnvAPIKey  = "OB_API_KEY"
	EnvConfig  = "OB_CONFIG"
	EnvNoClear = "OB_NO_CLEAR"

	AppName       = "openbench"
	LocalFileName = ".openbench.yaml"
)

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", AppName), nil
}

// ConfigPath returns the global config file path.
func ConfigPath() (string, error) {
	if env := os.Getenv(EnvConfig); env != "" {
		return env, nil
	}
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// LocalConfigPath returns the per-project config path for the current working directory.
func LocalConfigPath() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return LocalConfigPathIn(wd)
}

// LocalConfigPathIn returns {dir}/.openbench.yaml.
func LocalConfigPathIn(dir string) string {
	if strings.TrimSpace(dir) == "" {
		return ""
	}
	return filepath.Join(dir, LocalFileName)
}

// DataDir returns ~/.config/openbench.
func DataDir() (string, error) {
	return configDir()
}

// APIKeyFromEnv reads OB_API_KEY.
func APIKeyFromEnv() string {
	return os.Getenv(EnvAPIKey)
}

// NoClearFromEnv reports whether screen clear is disabled via env.
func NoClearFromEnv() bool {
	return os.Getenv(EnvNoClear) != ""
}
