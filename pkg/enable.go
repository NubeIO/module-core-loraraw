package pkg

import (
	"github.com/NubeIO/lib-module-go/nmodule"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/datatype"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/dto"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/nargs"
	log "github.com/sirupsen/logrus"
	"time"
)

func (m *Module) Enable() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	log.Info("plugin is enabling...")
	networks, err := m.grpcMarshaller.GetNetworksByPluginName(m.moduleName)
	if err != nil {
		log.Error(err)
		_ = m.updatePluginMessage(dto.MessageLevel.Fail, err.Error())
	}

	m.pointWriteQueue = NewPointWriteQueue(m.config.WriteQueueMaxRetries, m.config.WriteQueueTimeout)

	if len(networks) == 0 {
		warnMsg := "no LoRaRAW networks exist"
		log.Warn(warnMsg)
		_ = m.updatePluginMessage(dto.MessageLevel.Warning, warnMsg)
	} else {
		network := networks[0]
		net, err := m.grpcMarshaller.GetNetwork(network.UUID, &nmodule.Opts{Args: &nargs.Args{WithDevices: true, WithPoints: true}})
		if err != nil {
			log.Errorf("error getting network: %s", err.Error())
		} else {
			for _, device := range net.Devices {
				for _, point := range device.Points {
					if point.PointState == datatype.PointStateApiUpdatePending {
						m.pointWriteQueue.LoadWriteQueue(point)
					}
				}
			}
		}
	}
	_ = m.updatePluginMessage(dto.MessageLevel.Info, "")

	go m.pointWriteQueue.ProcessPointWriteQueue(m.getEncryptionKey, m.WriteToLoRaRaw)

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
