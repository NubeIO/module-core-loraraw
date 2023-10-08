package pkg

import (
	"encoding/json"
	"errors"
	"github.com/NubeIO/lib-schema/loraschema"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/pkg/v1/model"
	"github.com/NubeIO/rubix-os/module/common"
)

const (
	help          = "/help"
	restartSerial = "/serial/restart"
	listSerial    = "/serial/list"
	wizardSerial  = "/wizard/serial"
)

const (
	jsonSchemaNetwork = "/schema/json/network"
	jsonSchemaDevice  = "/schema/json/device"
	jsonSchemaPoint   = "/schema/json/point"
)

func (m *Module) Get(path string) ([]byte, error) {
	if path == jsonSchemaNetwork {
		return json.Marshal(loraschema.GetNetworkSchema())
	} else if path == jsonSchemaDevice {
		return json.Marshal(loraschema.GetDeviceSchema())
	} else if path == jsonSchemaPoint {
		return json.Marshal(loraschema.GetPointSchema())
	}
	return nil, errors.New("not found")
}

func (m *Module) Post(path string, body []byte) ([]byte, error) {
	if path == common.NetworksURL {
		var network *model.Network
		err := json.Unmarshal(body, &network)
		if err != nil {
			return nil, err
		}
		net, err := m.addNetwork(network)
		if err != nil {
			return nil, err
		}
		return json.Marshal(net)
	} else if path == common.DevicesURL {
		var device *model.Device
		err := json.Unmarshal(body, &device)
		if err != nil {
			return nil, err
		}
		dev, err := m.addDevice(device)
		if err != nil {
			return nil, err
		}
		return json.Marshal(dev)
	} else if path == common.PointsURL {
		var point *model.Point
		err := json.Unmarshal(body, &point)
		if err != nil {
			return nil, err
		}
		pnt, err := m.addPoint(point)
		if err != nil {
			return nil, err
		}
		return json.Marshal(pnt)
	}
	return nil, errors.New("not found")
}

func (m *Module) Put(path, uuid string, body []byte) ([]byte, error) {
	return nil, errors.New("not found")
}

func (m *Module) Patch(path, uuid string, body []byte) ([]byte, error) {
	if path == common.NetworksURL {
		var network *model.Network
		err := json.Unmarshal(body, &network)
		if err != nil {
			return nil, err
		}
		net, err := m.grpcMarshaller.UpdateNetwork(uuid, network)
		if err != nil {
			return nil, err
		}
		return json.Marshal(net)
	} else if path == common.DevicesURL {
		var device *model.Device
		err := json.Unmarshal(body, &device)
		if err != nil {
			return nil, err
		}
		dev, err := m.grpcMarshaller.UpdateDevice(uuid, device)
		if err != nil {
			return nil, err
		}
		return json.Marshal(dev)
	} else if path == common.PointsURL {
		var point *model.Point
		err := json.Unmarshal(body, &point)
		if err != nil {
			return nil, err
		}
		pnt, err := m.grpcMarshaller.UpdatePoint(uuid, point)
		if err != nil {
			return nil, err
		}
		return json.Marshal(pnt)
	} else if path == common.PointsWriteURL {
		var pw *model.PointWriter
		err := json.Unmarshal(body, &pw)
		if err != nil {
			return nil, err
		}
		pnt, err := m.grpcMarshaller.PointWrite(uuid, pw)
		if err != nil {
			return nil, err
		}
		return json.Marshal(pnt.Point)
	}
	return nil, errors.New("not found")
}
func (m *Module) Delete(path, uuid string) ([]byte, error) {
	if path == common.NetworksURL {
		err := m.grpcMarshaller.DeleteNetwork(uuid)
		return nil, err
	} else if path == common.DevicesURL {
		err := m.grpcMarshaller.DeleteDevice(uuid)
		return nil, err
	} else if path == common.PointsURL {
		err := m.grpcMarshaller.DeletePoint(uuid)
		return nil, err
	}
	return nil, errors.New("not found")
}
