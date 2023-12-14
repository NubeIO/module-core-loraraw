package pkg

import (
	"github.com/NubeIO/lib-module-go/module"
	"sync"
)

type Module struct {
	dbHelper       module.DBHelper
	moduleName     string
	grpcMarshaller module.Marshaller
	config         *Config
	networkUUID    string
	interruptChan  chan struct{}
	mutex          *sync.RWMutex
}

func (m *Module) Init(dbHelper module.DBHelper, moduleName string) error {
	InitRouter()
	m.mutex = &sync.RWMutex{}
	grpcMarshaller := module.GRPCMarshaller{DbHelper: dbHelper}
	m.dbHelper = dbHelper
	m.moduleName = moduleName
	m.grpcMarshaller = &grpcMarshaller
	return nil
}

func (m *Module) GetInfo() (*module.Info, error) {
	return &module.Info{
		Name:       pluginName,
		Author:     "RaiBnod",
		Website:    "https://nube-io.com",
		License:    "N/A",
		HasNetwork: true,
	}, nil
}
