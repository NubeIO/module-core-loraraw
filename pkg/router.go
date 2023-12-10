package pkg

import (
	"encoding/json"
	"github.com/NubeIO/lib-module-go/http"
	"github.com/NubeIO/lib-module-go/router"
	"github.com/NubeIO/lib-module-go/shared"
	"github.com/NubeIO/module-core-loraraw/schema"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/nargs"
)

var route *router.Router

func InitRouter() {
	route = router.NewRouter()

	route.Handle(http.GET, "/api/networks/schema", GetNetworkSchema)
	route.Handle(http.GET, "/api/devices/schema", GetDeviceSchema)
	route.Handle(http.GET, "/api/points/schema", GetPointSchema)

	route.Handle(http.POST, "/api/networks", CreateNetwork)
	route.Handle(http.PATCH, "/api/networks/:uuid", UpdateNetwork)
	route.Handle(http.DELETE, "/api/networks/:uuid", DeleteNetwork)

	route.Handle(http.POST, "/api/devices", CreateDevice)
	route.Handle(http.PATCH, "/api/devices/:uuid", UpdateDevice)
	route.Handle(http.DELETE, "/api/devices/:uuid", DeleteDevice)

	route.Handle(http.POST, "/api/points", CreatePoint)
	route.Handle(http.PATCH, "/api/points/:uuid", UpdatePoint)
	route.Handle(http.PATCH, "/api/points/:uuid/write", PointWrite)
	route.Handle(http.DELETE, "/api/points/:uuid", DeletePoint)
}

func (m *Module) CallModule(method http.Method, api string, args nargs.Args, body []byte) ([]byte, error) {
	module := (shared.Module)(m)
	return route.CallHandler(&module, method, api, args, body)
}

func GetNetworkSchema(m *shared.Module, params map[string]string, args nargs.Args, body []byte) ([]byte, error) {
	return json.Marshal(schema.GetNetworkSchema())
}

func GetDeviceSchema(m *shared.Module, params map[string]string, args nargs.Args, body []byte) ([]byte, error) {
	return json.Marshal(schema.GetDeviceSchema())
}

func GetPointSchema(m *shared.Module, params map[string]string, args nargs.Args, body []byte) ([]byte, error) {
	return json.Marshal(schema.GetPointSchema())
}

func CreateNetwork(m *shared.Module, params map[string]string, args nargs.Args, body []byte) ([]byte, error) {
	var network *model.Network
	err := json.Unmarshal(body, &network)
	if err != nil {
		return nil, err
	}
	net, err := (*m).(*Module).addNetwork(network)
	if err != nil {
		return nil, err
	}
	return json.Marshal(net)
}

func UpdateNetwork(m *shared.Module, params map[string]string, args nargs.Args, body []byte) ([]byte, error) {
	var network *model.Network
	err := json.Unmarshal(body, &network)
	if err != nil {
		return nil, err
	}
	net, err := (*m).(*Module).grpcMarshaller.UpdateNetwork(params["uuid"], network)
	if err != nil {
		return nil, err
	}
	return json.Marshal(net)
}

func DeleteNetwork(m *shared.Module, params map[string]string, args nargs.Args, body []byte) ([]byte, error) {
	err := (*m).(*Module).grpcMarshaller.DeleteNetwork(params["uuid"])
	return nil, err
}

func CreateDevice(m *shared.Module, params map[string]string, args nargs.Args, body []byte) ([]byte, error) {
	var device *model.Device
	err := json.Unmarshal(body, &device)
	if err != nil {
		return nil, err
	}
	dev, err := (*m).(*Module).addDevice(device)
	if err != nil {
		return nil, err
	}
	return json.Marshal(dev)
}

func UpdateDevice(m *shared.Module, params map[string]string, args nargs.Args, body []byte) ([]byte, error) {
	var device *model.Device
	err := json.Unmarshal(body, &device)
	if err != nil {
		return nil, err
	}
	dev, err := (*m).(*Module).grpcMarshaller.UpdateDevice(params["uuid"], device)
	if err != nil {
		return nil, err
	}
	return json.Marshal(dev)
}

func DeleteDevice(m *shared.Module, params map[string]string, args nargs.Args, body []byte) ([]byte, error) {
	err := (*m).(*Module).grpcMarshaller.DeleteDevice(params["uuid"])
	return nil, err
}

func CreatePoint(m *shared.Module, params map[string]string, args nargs.Args, body []byte) ([]byte, error) {
	var point *model.Point
	err := json.Unmarshal(body, &point)
	if err != nil {
		return nil, err
	}
	pnt, err := (*m).(*Module).addPoint(point)
	if err != nil {
		return nil, err
	}
	return json.Marshal(pnt)
}

func UpdatePoint(m *shared.Module, params map[string]string, args nargs.Args, body []byte) ([]byte, error) {
	var point *model.Point
	err := json.Unmarshal(body, &point)
	if err != nil {
		return nil, err
	}
	pnt, err := (*m).(*Module).grpcMarshaller.UpdatePoint(params["uuid"], point, nargs.Args{})
	if err != nil {
		return nil, err
	}
	return json.Marshal(pnt)
}

func PointWrite(m *shared.Module, params map[string]string, args nargs.Args, body []byte) ([]byte, error) {
	var pw *model.PointWriter
	err := json.Unmarshal(body, &pw)
	if err != nil {
		return nil, err
	}
	pnt, err := (*m).(*Module).grpcMarshaller.PointWrite(params["uuid"], pw)
	if err != nil {
		return nil, err
	}
	return json.Marshal(pnt.Point)
}

func DeletePoint(m *shared.Module, params map[string]string, args nargs.Args, body []byte) ([]byte, error) {
	err := (*m).(*Module).grpcMarshaller.DeletePoint(params["uuid"])
	return nil, err
}
