//go:build cli
// +build cli

package main

import (
	_ "magento.GO/custom"

	"magento.GO/cmd"
	"magento.GO/config"
)

func main() {
	config.LoadEnv()
	cmd.Execute()
}
