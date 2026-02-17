package registry

// Core keys for GlobalRegistry and RequestRegistry.
const (
	// RequestRegistry keys (per-request)
	KeyRequestStart = "request_start"

	// Extension registries (cmd, cron, api, graphql) — stored in GlobalRegistry
	KeyRegistryCmd    = "registry:cmd"
	KeyRegistryCron   = "registry:cron"
	KeyRegistryAPI    = "registry:api"
	KeyRegistryGraphQL = "registry:graphql"
)
