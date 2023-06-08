package pkg

import (
	"github.com/NubeIO/rubix-os/module/shared"
	"github.com/NubeIO/rubix-os/utils/nstring"
	"sync"
)

type Module struct {
	dbHelper       shared.DBHelper
	moduleName     string
	grpcMarshaller shared.Marshaller
	config         *Config
	// enabled        bool
	// running        bool
	// fault          bool
	// basePath       string
	// store          cachestore.Handler
	// bus            eventbus.BusService
	// pluginUUID    string
	networkUUID   string
	interruptChan chan struct{}
	mutex         *sync.RWMutex
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
		Name:       name,
		Author:     "RaiBnod",
		Website:    "https://nube-io.com",
		License:    "N/A",
		HasNetwork: true,
	}, nil
}

func (m *Module) GetUrlPrefix() (*string, error) {
	return nstring.New(urlPrefix), nil
}
