package pkg

import (
	log "github.com/sirupsen/logrus"
	"time"
)

var reIterationTime = 5

// LoRa plugin loop
func (m *Module) run() {
	defer m.SerialClose()

	for {
		sc, err := m.SerialOpen()
		select {
		case <-m.interruptChan:
			log.Info("loraraw: interrupt received on run")
			return
		default:
			if err != nil {
				log.Error("loraraw: error opening serial ", err)
				time.Sleep(time.Duration(reIterationTime) * time.Second)
				continue
			}
		}
		serialPayloadChan := make(chan string, 1)
		serialCloseChan := make(chan error, 1)
		go sc.Loop(serialPayloadChan, serialCloseChan)

		for {
			select {
			case <-m.interruptChan:
				log.Info("loraraw: interrupt received on run")
				return
			case err := <-serialCloseChan:
				log.Error("loraraw: serial connection error: ", err)
				log.Info("loraraw: serial connection attempting to reconnect...")
				continue
			case data := <-serialPayloadChan:
				m.handleSerialPayload(data)
			}
		}
	}
}
