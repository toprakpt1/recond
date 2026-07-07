package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	DataDir        string `mapstructure:"data_dir"`
	SocketPath     string `mapstructure:"socket_path"`
	DefaultProfile string `mapstructure:"default_profile"`
	MaxRetries     int    `mapstructure:"max_retries"`
	RetryBackoff   string `mapstructure:"retry_backoff"`
	Wordlist       string `mapstructure:"wordlist"`
}

var v *viper.Viper

func Init() error {
	v = viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("$HOME/.recond")
	v.AddConfigPath("/etc/recond")

	v.SetDefault("data_dir", "~/.recond")
	v.SetDefault("socket_path", "~/.recond/recond.sock")
	v.SetDefault("default_profile", "balanced")
	v.SetDefault("max_retries", 3)
	v.SetDefault("retry_backoff", "2s")
	v.SetDefault("wordlist", "~/.recond/wordlists/common.txt")

	v.SetEnvPrefix("RECOND")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.ReadInConfig()
	v.WatchConfig()

	return nil
}

func LoadConfig() (*Config, error) {
	if v == nil {
		if err := Init(); err != nil {
			return nil, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	cfg.DataDir = expandPath(cfg.DataDir)
	cfg.SocketPath = expandPath(cfg.SocketPath)

	return &cfg, nil
}

func GetViper() *viper.Viper {
	return v
}

func Get(key string) interface{} {
	return v.Get(key)
}

func Set(key string, value interface{}) error {
	v.Set(key, value)
	return v.WriteConfigAs(ConfigPath())
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func EnsureDataDir(dataDir string) error {
	return os.MkdirAll(dataDir, 0755)
}

func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".recond", "config.yaml")
}

func WriteDefaultConfig() error {
	path := ConfigPath()
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	cfg := `data_dir: ~/.recond
socket_path: ~/.recond/recond.sock
default_profile: balanced
max_retries: 3
retry_backoff: 2s
wordlist: ~/.recond/wordlists/common.txt

profiles:
  safe:
    concurrency: 3
    rate_limit: 10
    cpu_max: 20
    ram_max: 1GB
    timeout: 30s
  balanced:
    concurrency: 10
    rate_limit: 50
    cpu_max: 50
    ram_max: 2GB
    timeout: 15s
  aggressive:
    concurrency: 25
    rate_limit: 100
    cpu_max: 80
    ram_max: 4GB
    timeout: 10s
`

	return os.WriteFile(path, []byte(cfg), 0644)
}

func (c *Config) ResolvePath(sub string) string {
	return filepath.Join(c.DataDir, sub)
}

func GetString(key, defaultValue string) string {
	if v == nil {
		return defaultValue
	}
	val := v.GetString(key)
	if val == "" {
		return defaultValue
	}
	return val
}

func GetInt(key string, defaultValue int) int {
	if v == nil {
		return defaultValue
	}
	if v.IsSet(key) {
		return v.GetInt(key)
	}
	return defaultValue
}
