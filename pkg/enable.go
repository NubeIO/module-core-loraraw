package pkg

import (
	log "github.com/sirupsen/logrus"
	"time"
)

var pluginName = "module-core-lora"

func (m *Module) Enable() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	log.Info("plugin is enabling...")
	networks, err := m.grpcMarshaller.GetNetworksByPluginName(pluginName, "")
	if err != nil {
		log.Error(err)
	}
	if len(networks) == 0 {
		log.Warn("we don't have networks")
	}
	network := networks[0]
	m.networkUUID = network.UUID
	m.interruptChan = make(chan struct{}, 1)
	go m.run()
	log.Info("plugin is enabled")
	return nil
}

func (m *Module) Disable() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	log.Info("plugin is disabling...")
	m.interruptChan <- struct{}{}
	time.Sleep(time.Duration(reIterationTime+1) * time.Second) // we need to do this because, before disable it could possibly be restarted
	log.Info("plugin is disabled")
	return nil
}
