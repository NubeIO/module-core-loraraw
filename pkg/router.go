package pkg

import (
	"encoding/json"
	"github.com/NubeIO/lib-module-go/http"
	"github.com/NubeIO/lib-module-go/module"
	"github.com/NubeIO/lib-module-go/router"
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
	mo := (module.Module)(m)
	return route.CallHandler(&mo, method, api, args, body)
}

func GetNetworkSchema(m *module.Module, r *router.Request) ([]byte, error) {
	return json.Marshal(schema.GetNetworkSchema())
}

func GetDeviceSchema(m *module.Module, r *router.Request) ([]byte, error) {
	return json.Marshal(schema.GetDeviceSchema())
}

func GetPointSchema(m *module.Module, r *router.Request) ([]byte, error) {
	return json.Marshal(schema.GetPointSchema())
}

func CreateNetwork(m *module.Module, r *router.Request) ([]byte, error) {
	var network *model.Network
	err := json.Unmarshal(r.Body, &network)
	if err != nil {
		return nil, err
	}
	net, err := (*m).(*Module).addNetwork(network)
	if err != nil {
		return nil, err
	}
	return json.Marshal(net)
}

func UpdateNetwork(m *module.Module, r *router.Request) ([]byte, error) {
	var network *model.Network
	err := json.Unmarshal(r.Body, &network)
	if err != nil {
		return nil, err
	}
	net, err := (*m).(*Module).grpcMarshaller.UpdateNetwork(r.Params["uuid"], network)
	if err != nil {
		return nil, err
	}
	return json.Marshal(net)
}

func DeleteNetwork(m *module.Module, r *router.Request) ([]byte, error) {
	err := (*m).(*Module).grpcMarshaller.DeleteNetwork(r.Params["uuid"])
	return nil, err
}

func CreateDevice(m *module.Module, r *router.Request) ([]byte, error) {
	var device *model.Device
	err := json.Unmarshal(r.Body, &device)
	if err != nil {
		return nil, err
	}
	dev, err := (*m).(*Module).addDevice(device)
	if err != nil {
		return nil, err
	}
	return json.Marshal(dev)
}

func UpdateDevice(m *module.Module, r *router.Request) ([]byte, error) {
	var device *model.Device
	err := json.Unmarshal(r.Body, &device)
	if err != nil {
		return nil, err
	}
	dev, err := (*m).(*Module).grpcMarshaller.UpdateDevice(r.Params["uuid"], device)
	if err != nil {
		return nil, err
	}
	return json.Marshal(dev)
}

func DeleteDevice(m *module.Module, r *router.Request) ([]byte, error) {
	err := (*m).(*Module).grpcMarshaller.DeleteDevice(r.Params["uuid"])
	return nil, err
}

func CreatePoint(m *module.Module, r *router.Request) ([]byte, error) {
	var point *model.Point
	err := json.Unmarshal(r.Body, &point)
	if err != nil {
		return nil, err
	}
	pnt, err := (*m).(*Module).addPoint(point)
	if err != nil {
		return nil, err
	}
	return json.Marshal(pnt)
}

func UpdatePoint(m *module.Module, r *router.Request) ([]byte, error) {
	var point *model.Point
	err := json.Unmarshal(r.Body, &point)
	if err != nil {
		return nil, err
	}
	pnt, err := (*m).(*Module).grpcMarshaller.UpdatePoint(r.Params["uuid"], point, nargs.Args{})
	if err != nil {
		return nil, err
	}
	return json.Marshal(pnt)
}

func PointWrite(m *module.Module, r *router.Request) ([]byte, error) {
	var pw *model.PointWriter
	err := json.Unmarshal(r.Body, &pw)
	if err != nil {
		return nil, err
	}
	pnt, err := (*m).(*Module).grpcMarshaller.PointWrite(r.Params["uuid"], pw)
	if err != nil {
		return nil, err
	}
	return json.Marshal(pnt.Point)
}

func DeletePoint(m *module.Module, r *router.Request) ([]byte, error) {
	err := (*m).(*Module).grpcMarshaller.DeletePoint(r.Params["uuid"])
	return nil, err
}
