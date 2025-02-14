package pkg

import (
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/NubeIO/module-core-loraraw/logger"
	"github.com/go-yaml/yaml"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	ReIterationTime    time.Duration `yaml:"re_iteration_time"`
	LogLevel           string        `yaml:"log_level"`
	DefaultKey         string        `yaml:"default_key" type:"secret"`
	DecryptionDisabled bool          `yaml:"decryption_disabled"`
	LoRaFrequencyPlan  string        `yaml:"lora_freq_plan"`
}

const DefaultDeviceKey = "0301021604050f07e6095a0b0c12630f"

func (m *Module) DefaultConfig() *Config {
	return &Config{
		ReIterationTime:    5 * time.Second,
		LogLevel:           "ERROR",
		DefaultKey:         DefaultDeviceKey,
		DecryptionDisabled: false,
		LoRaFrequencyPlan:  "AU915",
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
	if len(keyBytes) == 0 {
		newConfig.DefaultKey = DefaultDeviceKey
	} else if len(keyBytes) != 16 {
		return nil, errors.New("invalid default key: must be exactly 16 bytes")
	}

	freqConfig := map[string]string{
		"lora_frequency_plan": newConfig.LoRaFrequencyPlan,
	}

	configYaml, err := yaml.Marshal(freqConfig)
	if err != nil {
		return nil, errors.New("error marshaling frequency plan config")
	}

	err = os.WriteFile("/data/lora_frequency_plan.yml", configYaml, 0644)
	if err != nil {
		return nil, errors.New("error writing frequency plan file")
	}

	newConfValid, err := yaml.Marshal(newConfig)
	if err != nil {
		return nil, err
	}
	m.config = newConfig
	log.Info("config is set")
	return newConfValid, nil
}
