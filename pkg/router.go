package pkg

import (
	"encoding/json"
	"github.com/NubeIO/lib-module-go/nhttp"
	"github.com/NubeIO/lib-module-go/nmodule"
	"github.com/NubeIO/lib-module-go/router"
	"github.com/NubeIO/module-core-loraraw/schema"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/dto"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/nargs"
	"net/http"
)

var route *router.Router

func InitRouter() {
	route = router.NewRouter()

	route.Handle(nhttp.GET, "/api/networks/schema", GetNetworkSchema)
	route.Handle(nhttp.GET, "/api/devices/schema", GetDeviceSchema)
	route.Handle(nhttp.GET, "/api/points/schema", GetPointSchema)

	route.Handle(nhttp.POST, "/api/networks", CreateNetwork)
	route.Handle(nhttp.PATCH, "/api/networks/:uuid", UpdateNetwork)
	route.Handle(nhttp.DELETE, "/api/networks/:uuid", DeleteNetwork)

	route.Handle(nhttp.POST, "/api/devices", CreateDevice)
	route.Handle(nhttp.PATCH, "/api/devices/:uuid", UpdateDevice)
	route.Handle(nhttp.DELETE, "/api/devices/:uuid", DeleteDevice)

	route.Handle(nhttp.POST, "/api/points", CreatePoint)
	route.Handle(nhttp.PATCH, "/api/points/:uuid", UpdatePoint)
	route.Handle(nhttp.PATCH, "/api/points/:uuid/write", PointWrite)
	route.Handle(nhttp.DELETE, "/api/points/:uuid", DeletePoint)
}

func (m *Module) CallModule(method nhttp.Method, urlString string, headers http.Header, body []byte) ([]byte, error) {
	mo := (nmodule.Module)(m)
	return route.CallHandler(&mo, method, urlString, headers, body)
}

func GetNetworkSchema(m *nmodule.Module, r *router.Request) ([]byte, error) {
	return json.Marshal(schema.GetNetworkSchema())
}

func GetDeviceSchema(m *nmodule.Module, r *router.Request) ([]byte, error) {
	return json.Marshal(schema.GetDeviceSchema())
}

func GetPointSchema(m *nmodule.Module, r *router.Request) ([]byte, error) {
	return json.Marshal(schema.GetPointSchema())
}

func CreateNetwork(m *nmodule.Module, r *router.Request) ([]byte, error) {
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

func UpdateNetwork(m *nmodule.Module, r *router.Request) ([]byte, error) {
	var network *model.Network
	err := json.Unmarshal(r.Body, &network)
	if err != nil {
		return nil, err
	}
	net, err := (*m).(*Module).grpcMarshaller.UpdateNetwork(r.PathParams["uuid"], network)
	if err != nil {
		return nil, err
	}
	return json.Marshal(net)
}

func DeleteNetwork(m *nmodule.Module, r *router.Request) ([]byte, error) {
	err := (*m).(*Module).grpcMarshaller.DeleteNetwork(r.PathParams["uuid"])
	return nil, err
}

func CreateDevice(m *nmodule.Module, r *router.Request) ([]byte, error) {
	var device *model.Device
	err := json.Unmarshal(r.Body, &device)
	if err != nil {
		return nil, err
	}
	v := r.QueryParams.Get(nargs.WithPoints)
	dev, err := (*m).(*Module).addDevice(device, v == "true")
	if err != nil {
		return nil, err
	}
	return json.Marshal(dev)
}

func UpdateDevice(m *nmodule.Module, r *router.Request) ([]byte, error) {
	var device *model.Device
	err := json.Unmarshal(r.Body, &device)
	if err != nil {
		return nil, err
	}
	dev, err := (*m).(*Module).grpcMarshaller.UpdateDevice(r.PathParams["uuid"], device)
	if err != nil {
		return nil, err
	}

	_ = (*m).(*Module).updateDevicePointsAddress(dev)

	return json.Marshal(dev)
}

func DeleteDevice(m *nmodule.Module, r *router.Request) ([]byte, error) {
	err := (*m).(*Module).grpcMarshaller.DeleteDevice(r.PathParams["uuid"])
	return nil, err
}

func CreatePoint(m *nmodule.Module, r *router.Request) ([]byte, error) {
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

func UpdatePoint(m *nmodule.Module, r *router.Request) ([]byte, error) {
	var point *model.Point
	err := json.Unmarshal(r.Body, &point)
	if err != nil {
		return nil, err
	}
	pnt, err := (*m).(*Module).updatePoint(r.PathParams["uuid"], point)
	if err != nil {
		return nil, err
	}
	return json.Marshal(pnt)
}

func PointWrite(m *nmodule.Module, r *router.Request) ([]byte, error) {
	var pw *dto.PointWriter
	err := json.Unmarshal(r.Body, &pw)
	if err != nil {
		return nil, err
	}

	point, err := (*m).(*Module).writePoint(r.PathParams["uuid"], pw)
	if err != nil {
		return nil, err
	}

	pendingPointWrite := &PendingPointWrite{Point: point, PointWriteStatus: PointWritePending}
	(*m).(*Module).pointWriteQueue.EnqueueWriteQueue(pendingPointWrite)

	return json.Marshal(point)
}

func DeletePoint(m *nmodule.Module, r *router.Request) ([]byte, error) {
	err := (*m).(*Module).grpcMarshaller.DeletePoint(r.PathParams["uuid"])
	return nil, err
}
