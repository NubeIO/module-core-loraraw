package main

import (
	"github.com/NubeIO/flow-framework/module/shared"
	"github.com/NubeIO/lora-module/logger"
	"github.com/NubeIO/lora-module/pkg"
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
