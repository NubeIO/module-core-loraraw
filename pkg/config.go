package pkg

import (
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/NubeIO/module-core-loraraw/logger"
	"github.com/go-yaml/yaml"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	ReIterationTime      time.Duration `yaml:"re_iteration_time"`
	LogLevel             string        `yaml:"log_level"`
	DefaultKey           string        `yaml:"default_key" type:"secret"`
	WriteQueueMaxRetries int           `yaml:"write_queue_max_retries"`
	MQTTEnable           bool          `yaml:"mqtt_enable"`
	MQTTBroker           string        `yaml:"mqtt_broker"`
	MQTTClientID         string        `yaml:"mqtt_client_id"`
	MQTTUsername         string        `yaml:"mqtt_username"`
	MQTTPassword         string        `yaml:"mqtt_password" type:"secret"`
	MQTTTopicPrefix      string        `yaml:"mqtt_topic_prefix"`
	// WriteResponseTimeout is how long the radio is held idle after each write
	// transmission waiting for the device's RESPONSE before the next frame goes.
	WriteResponseTimeout time.Duration `yaml:"write_response_timeout"`
}

const DefaultDeviceKey = "0301021604050f07e6095a0b0c12630f"

func (m *Module) DefaultConfig() *Config {
	return &Config{
		ReIterationTime:      5 * time.Second,
		LogLevel:             "ERROR",
		DefaultKey:           DefaultDeviceKey,
		WriteQueueMaxRetries: 5,
		MQTTEnable:           true,
		MQTTBroker:           "tcp://127.0.0.1:1883",
		MQTTClientID:         "module-core-loraraw",
		MQTTUsername:         "",
		MQTTPassword:         "",
		MQTTTopicPrefix:      MQTTTopicPrefix,
		WriteResponseTimeout: 5 * time.Second,
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

	if newConfig.WriteResponseTimeout <= 0 {
		newConfig.WriteResponseTimeout = 5 * time.Second
	}
	if newConfig.WriteQueueMaxRetries <= 0 {
		newConfig.WriteQueueMaxRetries = 1
	}

	keyBytes, err := hex.DecodeString(newConfig.DefaultKey)
	if err != nil {
		return nil, err
	}
	if len(keyBytes) == 0 {
		newConfig.DefaultKey = DefaultDeviceKey
	} else if len(keyBytes) != 16 {
		return nil, errors.New("invalid default key: must be exactly 16 bytes")
	}

	newConfValid, err := yaml.Marshal(newConfig)
	if err != nil {
		return nil, err
	}
	m.config = newConfig
	log.Info("config is set")
	return newConfValid, nil
}
