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
	ReIterationTime time.Duration `yaml:"re_iteration_time"`
	LogLevel        string        `yaml:"log_level"`
	Secret          string        `yaml:"secret" type:"secret"`
}

func (m *Module) DefaultConfig() *Config {
	return &Config{
		ReIterationTime: 5 * time.Second,
		LogLevel:        "ERROR",
		Secret:          "5f5f5f544f505f5345435245545f5f5f",
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

	keyBytes, err := hex.DecodeString(newConfig.Secret)
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
