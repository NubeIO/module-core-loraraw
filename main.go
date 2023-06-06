package main

import (
	"github.com/NubeIO/module-core-lora/logger"
	"github.com/NubeIO/module-core-lora/pkg"
	"github.com/NubeIO/rubix-os/module/shared"
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
