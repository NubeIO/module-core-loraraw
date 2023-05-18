package pkg

import (
	log "github.com/sirupsen/logrus"
	"time"
)

// LoRa plugin loop
func (m *Module) run() {
	defer m.SerialClose()

	for {
		sc, err := m.SerialOpen()
		if err != nil {
			log.Error("loraraw: error opening serial ", err)
			time.Sleep(5 * time.Second)
			continue
		}
		serialPayloadChan := make(chan string, 1)
		serialCloseChan := make(chan error, 1)
		go sc.Loop(serialPayloadChan, serialCloseChan)

	readLoop:
		for {
			select {
			case <-m.interruptChan:
				log.Info("loraraw: interrupt received on run")
				return
			case err := <-serialCloseChan:
				log.Error("loraraw: serial connection error: ", err)
				log.Info("loraraw: serial connection attempting to reconnect...")
				break readLoop
			case data := <-serialPayloadChan:
				m.handleSerialPayload(data)
			}
		}
	}
}
