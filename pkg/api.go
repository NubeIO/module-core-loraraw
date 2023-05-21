package pkg

import (
	"encoding/json"
	"errors"
	"github.com/NubeIO/flow-framework/module/common"
	"github.com/NubeIO/lib-schema/loraschema"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/pkg/v1/model"
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
	if path == common.NetworksURL {

	} else if path == jsonSchemaNetwork {
		fns, err := m.grpcMarshaller.GetFlowNetworks("")
		if err != nil {
			return nil, err
		}
		networkSchema := loraschema.GetNetworkSchema()
		networkSchema.AutoMappingFlowNetworkName.Options = common.GetFlowNetworkNames(fns)
		return json.Marshal(networkSchema)
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
	}
	return nil, errors.New("not found")
}

func (m *Module) Put(path string, body []byte) ([]byte, error) {
	return nil, errors.New("not found")
}
func (m *Module) Patch(path string, body []byte) ([]byte, error) {
	return nil, errors.New("not found")
}
func (m *Module) Delete(path string) ([]byte, error) {
	return nil, errors.New("not found")
}
