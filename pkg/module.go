package pkg

import (
	"github.com/NubeIO/lib-module-go/shared"
	"sync"
)

type Module struct {
	dbHelper       shared.DBHelper
	moduleName     string
	grpcMarshaller shared.Marshaller
	config         *Config
	networkUUID    string
	interruptChan  chan struct{}
	mutex          *sync.RWMutex
}

func (m *Module) Init(dbHelper shared.DBHelper, moduleName string) error {
	m.mutex = &sync.RWMutex{}
	grpcMarshaller := shared.GRPCMarshaller{DbHelper: dbHelper}
	m.dbHelper = dbHelper
	m.moduleName = moduleName
	m.grpcMarshaller = &grpcMarshaller
	return nil
}

func (m *Module) GetInfo() (*shared.Info, error) {
	return &shared.Info{
		Name:       pluginName,
		Author:     "RaiBnod",
		Website:    "https://nube-io.com",
		License:    "N/A",
		HasNetwork: true,
	}, nil
}
