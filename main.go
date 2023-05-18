package main

import (
	"github.com/NubeIO/data-processing-module/logger"
	"github.com/NubeIO/data-processing-module/pkg"
	"github.com/NubeIO/flow-framework/module/shared"
	"github.com/hashicorp/go-plugin"
)

func ServePlugin() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins: plugin.PluginSet{
			"nube-module": &shared.NubeModule{Impl: &pkg.Module{}},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}

func main() {
	logger.SetLogger("INFO")
	go pkg.Test()
	ServePlugin()
}
