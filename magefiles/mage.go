package main

import (
	//mage:import
	"github.com/rancher/system-agent/magefiles/targets"
)

var Default = targets.Default

var Aliases = map[string]any{
	"test":  targets.Test.All,
	"build": targets.Build.All,
}
