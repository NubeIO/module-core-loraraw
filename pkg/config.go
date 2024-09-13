package pkg

import (
	"encoding/hex"
	"errors"
	"github.com/NubeIO/module-core-loraraw/logger"
	"github.com/go-yaml/yaml"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

type Config struct {
	ReIterationTime    time.Duration `yaml:"re_iteration_time"`
	LogLevel           string        `yaml:"log_level"`
	DefaultKey         string        `yaml:"default_key" type:"secret"`
	DecryptionDisabled bool          `yaml:"decryption_disabled"`
}

func (m *Module) DefaultConfig() *Config {
	return &Config{
		ReIterationTime:    5 * time.Second,
		LogLevel:           "ERROR",
		DefaultKey:         "0301021604050f07e6095a0b0c12630f",
		DecryptionDisabled: false,
	}
}

func (m *Module) ValidateAndSetConfig(config []byte) ([]byte, error) {
	newConfig := m.DefaultConfig()
	_ = yaml.Unmarshal(config, newConfig)

	logLevel, err := log.ParseLevel(newConfig.LogLevel)
	if err != nil {
		logLevel = log.ErrorLevel
	}
	logger.SetLogger(logLevel)
	newConfig.LogLevel = strings.ToUpper(logLevel.String())

	keyBytes, err := hex.DecodeString(newConfig.DefaultKey)
	if err != nil {
		return nil, err
	}
	if len(keyBytes) != 16 {
		return nil, errors.New("invalid default key: key must be exactly 16 bytes")
	}

	newConfValid, err := yaml.Marshal(newConfig)
	if err != nil {
		return nil, err
	}
	m.config = newConfig
	log.Info("config is set")
	return newConfValid, nil
}
