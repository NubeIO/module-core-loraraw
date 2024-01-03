package pkg

import (
	"fmt"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	log "github.com/sirupsen/logrus"
	"time"
)

// LoRa plugin loop
func (m *Module) run() {
	defer m.SerialClose()

	for {
		sc, err := m.SerialOpen()
		select {
		case <-m.interruptChan:
			log.Info("interrupt received on run")
			return
		default:
			if err != nil {
				errMsg := fmt.Sprintf("error opening serial: %v", err.Error())
				log.Error(errMsg)
				_ = m.updatePluginMessage(model.MessageLevel.Fail, errMsg)
				time.Sleep(m.config.ReIterationTime)
				continue
			}
		}
		serialPayloadChan := make(chan string, 1)
		serialCloseChan := make(chan error, 1)
		go sc.Loop(serialPayloadChan, serialCloseChan)

		for {
			select {
			case <-m.interruptChan:
				log.Info("interrupt received on run")
				return
			case err := <-serialCloseChan:
				errMsg := fmt.Sprintf("serial connection error: %v", err.Error())
				log.Error(errMsg)
				_ = m.updatePluginMessage(model.MessageLevel.Fail, errMsg)
				log.Info("serial connection attempting to reconnect...")
				continue
			case data := <-serialPayloadChan:
				m.handleSerialPayload(data)
			}
		}
	}
}
