package product

import (
	"sync"

	"gorm.io/gorm"
	productEntity "magento.GO/model/entity/product"
)

// SchemaType represents the Magento EAV schema variant
type SchemaType int

const (
	SchemaUnknown SchemaType = iota
	SchemaEntityID           // Standard CE: EAV uses entity_id
	SchemaRowID              // Staging/EE: EAV uses row_id
)

func (s SchemaType) String() string {
	switch s {
	case SchemaEntityID:
		return "entity_id"
	case SchemaRowID:
		return "row_id"
	default:
		return "unknown"
	}
}

// EAVLinkColumn returns the column name used in EAV tables
func (s SchemaType) EAVLinkColumn() string {
	if s == SchemaRowID {
		return "row_id"
	}
	return "entity_id"
}

var (
	detectedSchema     SchemaType
	schemaDetectOnce   sync.Once
	schemaDetectResult SchemaType
)

// DetectSchema checks EAV table structure to determine schema type.
// Also sets productEntity.IsEnterprise for runtime access.
func DetectSchema(db *gorm.DB) SchemaType {
	schemaDetectOnce.Do(func() {
		schemaDetectResult = detectSchemaImpl(db)
		productEntity.IsEnterprise = (schemaDetectResult == SchemaRowID)
	})
	return schemaDetectResult
}

// ResetSchemaDetection clears cached schema (for testing)
func ResetSchemaDetection() {
	schemaDetectOnce = sync.Once{}
	schemaDetectResult = SchemaUnknown
}

func detectSchemaImpl(db *gorm.DB) SchemaType {
	dialect := db.Dialector.Name()

	// SQLite doesn't have row_id schema, always use entity_id
	if dialect == "sqlite" {
		return SchemaEntityID
	}

	// MySQL/MariaDB: use DESCRIBE
	type colInfo struct {
		Field string `gorm:"column:Field"`
	}
	var cols []colInfo
	if err := db.Raw("DESCRIBE catalog_product_entity_varchar").Scan(&cols).Error; err != nil {
		return SchemaUnknown
	}

	hasRowID := false
	hasEntityID := false
	for _, c := range cols {
		switch c.Field {
		case "row_id":
			hasRowID = true
		case "entity_id":
			hasEntityID = true
		}
	}

	if hasEntityID {
		return SchemaEntityID
	}
	if hasRowID {
		return SchemaRowID
	}
	return SchemaUnknown
}
