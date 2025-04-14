package pkg

import (
	"sync"

	"github.com/NubeIO/lib-module-go/nmodule"
)

type Module struct {
	dbHelper        nmodule.DBHelper
	moduleName      string
	grpcMarshaller  nmodule.Marshaller
	config          *Config
	networkUUID     string
	interruptChan   chan struct{}
	mutex           *sync.RWMutex
	pointWriteQueue *PointWriteQueue
	writeQueue     chan []byte
    writeQueueDone chan struct{}
    writeQueueInit sync.Once
}

func (m *Module) Init(dbHelper nmodule.DBHelper, moduleName string) error {
	InitRouter()
	m.mutex = &sync.RWMutex{}
	grpcMarshaller := nmodule.GRPCMarshaller{DbHelper: dbHelper}
	m.dbHelper = dbHelper
	m.moduleName = moduleName
	m.grpcMarshaller = &grpcMarshaller
	return nil
}

func (m *Module) GetInfo() (*nmodule.Info, error) {
	return &nmodule.Info{
		Name:       m.moduleName,
		Author:     "RaiBnod",
		Website:    "https://nube-io.com",
		License:    "N/A",
		HasNetwork: true,
	}, nil
}
