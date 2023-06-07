package pkg

import (
	log "github.com/sirupsen/logrus"
)

var pluginName = "module-core-lora"

func (m *Module) Enable() error {
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
	log.Info("plugin is disabling...")
	m.interruptChan <- struct{}{}
	log.Info("plugin is disabled")
	return nil
}
