package main

import (
	"github.com/NubeIO/lib-module-go/module"
	"github.com/NubeIO/module-core-loraraw/logger"
	"github.com/NubeIO/module-core-loraraw/pkg"
	"github.com/hashicorp/go-plugin"
)

func ServePlugin() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: module.HandshakeConfig,
		Plugins: plugin.PluginSet{
			"nube-module": &module.NubeModule{Impl: &pkg.Module{}},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}

func main() {
	logger.SetLogger("INFO")
	ServePlugin()
}
