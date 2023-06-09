package pkg

import (
	"github.com/go-yaml/yaml"
	log "github.com/sirupsen/logrus"
	"time"
)

type Config struct {
	ReIterationTime time.Duration `yaml:"re_iteration_time"`
}

func (m *Module) ValidateAndSetConfig(config []byte) ([]byte, error) {
	newConf := &Config{}
	err := yaml.Unmarshal(config, newConf)
	if err != nil {
		return nil, err
	}
	if newConf.ReIterationTime == 0 {
		newConf.ReIterationTime = time.Duration(5) * time.Second
	}
	newConfValid, err := yaml.Marshal(newConf)
	if err != nil {
		return nil, err
	}
	m.config = newConf
	log.Infof("config is set")
	return newConfValid, nil
}
