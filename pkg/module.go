package pkg

import (
	"fmt"
	"github.com/NubeIO/data-processing-module/marshal"
	"github.com/NubeIO/flow-framework/module/shared"
	"github.com/hashicorp/go-hclog"
	"time"
)

var module *Module

type Module struct {
	dbHelper       shared.DBHelper
	grpcMarshaller marshal.Marshaller
	config         *Config
	// enabled        bool
	// running        bool
	// fault          bool
	// basePath       string
	// store          cachestore.Handler
	// bus            eventbus.BusService
	pluginUUID    string
	networkUUID   string
	interruptChan chan struct{}
}

func (m *Module) Init(dbHelper shared.DBHelper) error {
	grpcMarshaller := marshal.GrpcMarshaller{DbHelper: dbHelper}
	m.dbHelper = dbHelper
	m.grpcMarshaller = &grpcMarshaller
	module = &Module{dbHelper: m.dbHelper, grpcMarshaller: &grpcMarshaller}
	return nil
}

func (m *Module) GetUrlPrefix() (string, error) {
	return "lora", nil
}

func Test() {
	hclog.Default().Info(fmt.Sprintf("module %v", module))
	if module != nil {

	} else {
		time.Sleep(1 * time.Second)
		Test()
	}
}
