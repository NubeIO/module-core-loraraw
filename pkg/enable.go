package pkg

import (
	"github.com/NubeIO/nubeio-rubix-lib-models-go/dto"
	log "github.com/sirupsen/logrus"
	"time"
)

var pluginName = "module-core-loraraw"

func (m *Module) Enable() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	log.Info("plugin is enabling...")
	networks, err := m.grpcMarshaller.GetNetworksByPluginName(pluginName)
	if err != nil {
		log.Error(err)
		_ = m.updatePluginMessage(dto.MessageLevel.Fail, err.Error())
	}
	if len(networks) == 0 {
		warnMsg := "no LoRaRAW networks exist"
		log.Warn(warnMsg)
		_ = m.updatePluginMessage(dto.MessageLevel.Warning, warnMsg)
	}
	_ = m.updatePluginMessage(dto.MessageLevel.Info, "")
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
	time.Sleep(m.config.ReIterationTime + 1*time.Second) // we need to do this because, before disable it could possibly be restarted
	log.Info("plugin is disabled")
	return nil
}
