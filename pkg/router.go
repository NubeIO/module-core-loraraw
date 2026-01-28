package pkg

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/NubeIO/lib-module-go/nhttp"
	"github.com/NubeIO/lib-module-go/nmodule"
	"github.com/NubeIO/lib-module-go/router"
	"github.com/NubeIO/module-core-loraraw/codecs/rubixDataEncoding"
	"github.com/NubeIO/module-core-loraraw/schema"
	"github.com/NubeIO/nubeio-rubix-lib-helpers-go/pkg/nils"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/dto"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/model"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/nargs"
	log "github.com/sirupsen/logrus"
)

var route *router.Router

const (
	uartPingMaxRetries        = 10
	uartPingMinPointsRequired = 10
	uartPingRetryInterval     = 10 * time.Second
)

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

	if device.Model == schema.DeviceModelUART { // Ping at address 4
		enqueueUartPing(m, dev)
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

	if device.Model == schema.DeviceModelUART { // Ping at address 4
		enqueueUartPing(m, dev)
	}

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

	(*m).(*Module).pointWriteQueueManager.EnqueuePoint(point)

	return json.Marshal(point)
}

func DeletePoint(m *nmodule.Module, r *router.Request) ([]byte, error) {
	err := (*m).(*Module).grpcMarshaller.DeletePoint(r.PathParams["uuid"])
	return nil, err
}

func enqueueUartPing(m *nmodule.Module, device *model.Device) {
	if device.Model == schema.DeviceModelUART { // Ping at address 4
		// Run the retry mechanism in a goroutine to not block other operations
		go enqueueUartPingWithRetry(m, device)
	}
}

func enqueueUartPingWithRetry(m *nmodule.Module, device *model.Device) {
	module := (*m).(*Module)

	for attempt := 1; attempt <= uartPingMaxRetries; attempt++ {
		log.Infof("enqueueUartPing attempt %d/%d for device %s", attempt, uartPingMaxRetries, device.UUID)

		// Create and enqueue the ping point
		point := &model.Point{
			IoNumber:    "UVP-4",
			AddressID:   nils.NewInt(4),
			DataType:    strconv.Itoa(int(rubixDataEncoding.MDK_BOOL)),
			DeviceUUID:  device.UUID,
			AddressUUID: device.AddressUUID,
			WriteValue:  nils.NewFloat64(1),
		}
		module.pointWriteQueueManager.EnqueuePoint(point)

		// Wait for the poll interval before checking
		time.Sleep(uartPingRetryInterval)

		// Check if the device now has the required number of points
		dev, err := module.grpcMarshaller.GetDevice(device.UUID, &nmodule.Opts{Args: &nargs.Args{WithPoints: true}})
		if err != nil {
			log.Errorf("enqueueUartPing error getting device on attempt %d: %s", attempt, err.Error())
			continue
		}

		pointCount := len(dev.Points)
		log.Infof("enqueueUartPing attempt %d: device %s has %d points", attempt, device.UUID, pointCount)

		if pointCount >= uartPingMinPointsRequired {
			log.Infof("enqueueUartPing successful for device %s with %d points", device.UUID, pointCount)
			return
		}

		if attempt < uartPingMaxRetries {
			log.Warnf("enqueueUartPing attempt %d failed: device %s has only %d points (required: %d), retrying...",
				attempt, device.UUID, pointCount, uartPingMinPointsRequired)
		}
	}

	log.Errorf("enqueueUartPing failed after %d attempts for device %s: insufficient points", uartPingMaxRetries, device.UUID)
}
