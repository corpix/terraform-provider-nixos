package main

import (
	"flag"

	"github.com/corpix/terraform-provider-nixos/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	var debugMode bool

	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	plugin.Serve(&plugin.ServeOpts{
		Debug:        debugMode,
		ProviderAddr: "registry.terraform.io/corpix/nixos",
		ProviderFunc: func() *schema.Provider { return provider.New() },
	})
}
