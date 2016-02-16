package config

import (
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// PanicOnError is a helper function to panic on Error
func PanicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

// Config stores all configuration parameters for Go
type Config struct {
	InfoDir              string `toml:"info_dir"`
	DataDir              string `toml:"data_dir"`
	Port                 uint   `toml:"port"`
	SaveThresholdSeconds uint   `toml:"save_threshold_seconds"`
}

var config *Config

// MaxKeySize ...
const MaxKeySize int = 32768 // max key size BoltDB in bytes

func parseConfigTOML() *Config {
	configPath := os.Getenv("SKZ_CONFIG")
	if configPath == "" {
		path, err := os.Getwd()
		PanicOnError(err)
		path, err = filepath.Abs(path)
		PanicOnError(err)
		configPath = filepath.Join(path, "src/config/default.toml")
	}
	_, err := os.Open(configPath)
	PanicOnError(err)
	config = &Config{}
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		PanicOnError(err)
	}
	return config
}

// GetConfig returns a singleton Configuration
func GetConfig() *Config {
	if config == nil {
		config = parseConfigTOML()
		usr, err := user.Current()
		PanicOnError(err)
		dir := usr.HomeDir

		infoDir := strings.TrimSpace(os.Getenv("SKZ_INFO_DIR"))
		if len(infoDir) == 0 {
			if config.InfoDir[:2] == "~/" {
				infoDir = strings.Replace(config.InfoDir, "~", dir, 1)
			}
		}

		dataDir := strings.TrimSpace(os.Getenv("SKZ_DATA_DIR"))
		if len(dataDir) == 0 {
			if config.DataDir[:2] == "~/" {
				dataDir = strings.Replace(config.DataDir, "~", dir, 1)
			}
		}

		portInt, err := strconv.Atoi(strings.TrimSpace(os.Getenv("SKZ_PORT")))
		port := uint(portInt)
		if err != nil {
			port = config.Port
		}

		saveThresholdSecondsInt, err := strconv.Atoi(strings.TrimSpace(os.Getenv("SKZ_SAVE_TRESHOLD_SECS")))
		saveThresholdSeconds := uint(saveThresholdSecondsInt)
		if err != nil {
			saveThresholdSeconds = config.SaveThresholdSeconds
		}

		if saveThresholdSeconds < 3 {
			saveThresholdSeconds = 3
		}

		if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
			panic(err)
		}

		config = &Config{
			infoDir,
			dataDir,
			port,
			saveThresholdSeconds,
		}
	}
	return config
}

// Reset ...
func Reset() {
	config = nil
}
