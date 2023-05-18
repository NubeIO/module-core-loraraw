package marshal

import (
	"encoding/json"
	"github.com/NubeIO/flow-framework/module/shared"
	"github.com/NubeIO/nubeio-rubix-lib-models-go/pkg/v1/model"
)

type Marshaller interface {
	GetFlowNetworks(args string) ([]*model.FlowNetwork, error)
	GetNetworksByPluginName(pluginName, args string) ([]*model.Network, error)
	CreateNetwork(body *model.Network) (*model.Network, error)
}

type GrpcMarshaller struct {
	DbHelper shared.DBHelper
}

func (g *GrpcMarshaller) GetFlowNetworks(args string) ([]*model.FlowNetwork, error) {
	res, err := g.DbHelper.GetList("flow_networks", args)
	if err != nil {
		return nil, err
	}
	var fns []*model.FlowNetwork
	err = json.Unmarshal(res, &fns)
	if err != nil {
		return nil, err
	}
	return fns, nil
}

func (g *GrpcMarshaller) GetNetworksByPluginName(pluginName, args string) ([]*model.Network, error) {
	res, err := g.DbHelper.Get("networks_by_plugin_name", pluginName, args)
	if err != nil {
		return nil, err
	}
	var fns []*model.Network
	err = json.Unmarshal(res, &fns)
	if err != nil {
		return nil, err
	}
	return fns, nil
}

func (g *GrpcMarshaller) CreateNetwork(body *model.Network) (*model.Network, error) {
	net, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	res, err := g.DbHelper.Post("networks", net)
	if err != nil {
		return nil, err
	}
	var network *model.Network
	err = json.Unmarshal(res, &network)
	if err != nil {
		return nil, err
	}
	return network, nil
}
