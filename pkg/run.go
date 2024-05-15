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
				_ = m.grpcMarshaller.UpdateNetworkFault(m.networkUUID, &model.CommonFault{
					InFault:  true,
					Message:  errMsg,
					LastFail: time.Now().UTC(),
				})
				time.Sleep(m.config.ReIterationTime)
				continue
			} else {
				_ = m.grpcMarshaller.UpdateNetworkFault(m.networkUUID, &model.CommonFault{
					InFault: false,
					Message: "",
					LastOk:  time.Now().UTC(),
				})
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
				_ = m.grpcMarshaller.UpdateNetworkFault(m.networkUUID, &model.CommonFault{
					InFault:  true,
					Message:  errMsg,
					LastFail: time.Now().UTC(),
				})
				log.Info("serial connection attempting to reconnect...")
				continue
			case data := <-serialPayloadChan:
				m.handleSerialPayload(data)
			}
		}
	}
}
