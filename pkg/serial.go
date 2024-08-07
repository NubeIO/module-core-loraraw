package pkg

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	log "github.com/sirupsen/logrus"
	"go.bug.st/serial"
)

type SerialSetting struct {
	SerialPort     string
	Enable         bool
	BaudRate       int
	StopBits       serial.StopBits
	Parity         serial.Parity
	DataBits       int
	Timeout        int
	ActivePortList []string
	Connected      bool
	Error          bool
}

var Port serial.Port

func (m *Module) SerialOpen() (*SerialSetting, error) {
	s := &SerialSetting{}
	networks, err := m.grpcMarshaller.GetNetworksByPluginName(m.moduleName)
	if err != nil {
		return nil, err
	}
	totalNetworks := len(networks)
	if totalNetworks == 0 {
		return nil, errors.New(fmt.Sprintf("we don't have network of module %s", m.moduleName))
	} else if totalNetworks > 1 {
		return nil, errors.New(fmt.Sprintf("we have %d networks of module %s", totalNetworks, m.moduleName))
	}
	net := networks[0]
	m.networkUUID = net.UUID
	if net.SerialPort == nil || net.SerialBaudRate == nil {
		return s, errors.New("lora-serial: serial_port & serial_baud_rate required to open")
	}
	s.SerialPort = *net.SerialPort
	s.BaudRate = int(*net.SerialBaudRate)

	_, err = s.open()
	if err != nil {
		_ = m.grpcMarshaller.UpdateNetworkFault(net.UUID, &model.CommonFault{
			InFault: true,
			Message: fmt.Sprintf("port: %s, message: %s", s.SerialPort, err.Error()),
		})
	} else {
		_ = m.grpcMarshaller.UpdateNetworkFault(net.UUID, &model.CommonFault{
			InFault: false,
			Message: "",
		})
	}
	return s, err
}

func (m *Module) SerialClose() error {
	return disconnect()
}

func (m *Module) WriteToLoRaRaw(data []byte) error {
	if Port == nil {
		return errors.New("serial connection error: port not set")
	}
	_, err := Port.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func (s *SerialSetting) Loop(plChan chan<- string, errChan chan<- error) {
	scanner := bufio.NewScanner(Port)
	for scanner.Scan() {
		plChan <- scanner.Text()
	}
	errChan <- scanner.Err()
}

func (s *SerialSetting) open() (connected bool, err error) {
	portName := s.SerialPort
	baudRate := s.BaudRate
	parity := s.Parity
	stopBits := s.StopBits
	dataBits := s.DataBits
	if s.Connected {
		log.Info("existing serial port connection by this app is open, so close the existing connection")
		err := disconnect()
		if err != nil {
			log.Info(err)
			s.Error = true
			return false, err
		}
	}
	log.Infof("connecting to port: %s", portName)
	m := &serial.Mode{
		BaudRate: baudRate,
		Parity:   parity,
		DataBits: dataBits,
		StopBits: stopBits,
	}

	ports, _ := serial.GetPortsList()
	s.ActivePortList = ports

	port, err := serial.Open(portName, m)
	if err != nil {
		s.Error = true
		return false, err
	}
	Port = port
	s.Connected = true
	log.Infof("connected to serial port: %s connected: %t", portName, s.Connected)
	return s.Connected, nil
}

func disconnect() error {
	if Port != nil {
		err := Port.Close()
		if err != nil {
			log.Error("err on trying to close the port")
			return err
		}
	}
	return nil
}
