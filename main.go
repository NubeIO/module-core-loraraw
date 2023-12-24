package main

import (
	"github.com/NubeIO/lib-module-go/nmodule"
	"github.com/NubeIO/module-core-loraraw/logger"
	"github.com/NubeIO/module-core-loraraw/pkg"
	"github.com/hashicorp/go-plugin"
)

func ServePlugin() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: nmodule.HandshakeConfig,
		Plugins: plugin.PluginSet{
			"nube-module": &nmodule.NubeModule{Impl: &pkg.Module{}},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}

func main() {
	logger.SetLogger("INFO")
	ServePlugin()
}
