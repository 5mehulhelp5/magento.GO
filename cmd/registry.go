package cmd

import (
	"github.com/spf13/cobra"

	"magento.GO/core/registry"
)

// Register adds a command. Call from init() in custom packages. Panics if registry is locked.
func Register(c *cobra.Command) {
	if registry.GlobalRegistry.IsLocked(registry.KeyRegistryCmd) {
		panic("cmd/registry: locked (register only during init before Apply)")
	}
	var list []*cobra.Command
	if v, ok := registry.GlobalRegistry.GetGlobal(registry.KeyRegistryCmd); ok && v != nil {
		list = v.([]*cobra.Command)
	}
	list = append(list, c)
	registry.GlobalRegistry.SetGlobal(registry.KeyRegistryCmd, list)
}

// Apply adds all registered commands to root. Locks the cmd registry (immutable after).
func Apply() {
	var list []*cobra.Command
	if v, ok := registry.GlobalRegistry.GetGlobal(registry.KeyRegistryCmd); ok && v != nil {
		list = v.([]*cobra.Command)
	}
	for _, c := range list {
		rootCmd.AddCommand(c)
	}
	registry.GlobalRegistry.Lock(registry.KeyRegistryCmd)
}
