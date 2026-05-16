package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds all configuration for the headliner CLI.
type Config struct {
	GoogleClientID     string `mapstructure:"google_client_id"`
	GoogleClientSecret string `mapstructure:"google_client_secret"`
	OpenAIAPIKey       string `mapstructure:"openai_api_key"`
	OpenAIModel        string `mapstructure:"openai_model"`
	CacheDir           string `mapstructure:"cache_dir"`
}

// dotEnvFiles lists .env file names loaded in order (later files do NOT
// override values already set by earlier ones or by real env vars).
var dotEnvFiles = []string{".env", ".env.local"}

// Load reads configuration with the following priority (highest → lowest):
//  1. Real environment variables (already exported in the shell)
//  2. .env.local  (machine/personal overrides, not committed)
//  3. .env        (shared project defaults, can be committed)
//  4. .headliner.yaml config file
//  5. Built-in defaults
//
// .env files are searched in the current working directory and then $HOME.
func Load(cfgFile string) (*Config, error) {
	loadDotEnvFiles()

	v := viper.New()

	// Defaults
	v.SetDefault("openai_model", "gpt-4o")
	v.SetDefault("cache_dir", defaultCacheDir())

	// Config file (.headliner.yaml)
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		v.SetConfigName(".headliner")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath(homeDir())
	}

	// Env vars: HEADLINER_GOOGLE_CLIENT_ID etc.
	v.SetEnvPrefix("HEADLINER")
	v.AutomaticEnv()

	// Also support bare env var names shared with the Next.js client
	bindEnv(v, "google_client_id", "GOOGLE_CLIENT_ID")
	bindEnv(v, "google_client_secret", "GOOGLE_CLIENT_SECRET")
	bindEnv(v, "openai_api_key", "OPENAI_API_KEY")

	if err := v.ReadInConfig(); err != nil {
		// Config file is optional; only fail on real parse errors
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	return cfg, nil
}

// loadDotEnvFiles loads .env files without overwriting variables that are
// already present in the environment. It checks cwd first, then $HOME.
func loadDotEnvFiles() {
	searchDirs := []string{"."}
	if h := homeDir(); h != "." {
		searchDirs = append(searchDirs, h)
	}

	for _, dir := range searchDirs {
		for _, name := range dotEnvFiles {
			path := filepath.Join(dir, name)
			// godotenv.Overload would override real env vars; Load does not.
			_ = godotenv.Load(path) // silently skip missing files
		}
	}
}

func bindEnv(v *viper.Viper, key, envVar string) {
	if val := os.Getenv(envVar); val != "" {
		v.Set(key, val)
	}
}

func homeDir() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return "."
}

// defaultCacheDir returns ~/.headliner
func defaultCacheDir() string {
	return filepath.Join(homeDir(), ".headliner")
}
