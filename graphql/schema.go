package graphql

import (
	"strings"
	"sync"

	_ "embed"
)

//go:embed schema.graphqls
var schemaBase string

var (
	schemaExtensions []string
	schemaMu         sync.Mutex
)

// RegisterSchemaExtension appends schema to the Query. Call from init() in custom packages.
func RegisterSchemaExtension(schema string) {
	schemaMu.Lock()
	defer schemaMu.Unlock()
	schemaExtensions = append(schemaExtensions, strings.TrimSpace(schema))
}

// Schema returns base schema + registered extensions.
func Schema() string {
	schemaMu.Lock()
	ext := schemaExtensions
	schemaMu.Unlock()
	if len(ext) == 0 {
		return schemaBase
	}
	return schemaBase + "\n\n" + strings.Join(ext, "\n\n")
}

// --- Schema arg types (used by resolvers for graphql-go method matching) ---

type MagentoCategoryFilters struct {
	CategoryUID *struct {
		In *[]*string
		Eq *string
	}
}

type MagentoProductsArgs struct {
	Filter *struct {
		CategoryUID *struct {
			In *[]*string
			Eq *string
		}
	}
	Sort        *struct{ Position *string }
	PageSize    int32
	CurrentPage int32
}
