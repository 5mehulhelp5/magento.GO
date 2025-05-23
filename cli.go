package main

import (
	"magento.GO/cmd"
	"magento.GO/config"
)

func main() {
	config.LoadEnv()
	cmd.Execute()
}
