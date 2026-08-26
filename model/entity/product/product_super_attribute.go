package product

// ProductSuperAttribute represents catalog_product_super_attribute -- one
// EAV attribute (e.g. "color", "size") that varies across a configurable
// product's child (simple) products. Real Magento has a separate
// catalog_product_super_attribute_label table for a per-store label
// override; this project omits it and uses the attribute's own
// eav_attribute.frontend_label, the same simplification used elsewhere for
// per-store data.
type ProductSuperAttribute struct {
	ProductSuperAttributeID uint `gorm:"column:product_super_attribute_id;primaryKey;autoIncrement"`
	// ProductID is the configurable parent product.
	ProductID   uint   `gorm:"column:product_id;type:int unsigned;not null;default:0;uniqueIndex:idx_super_attribute_unq"`
	AttributeID uint16 `gorm:"column:attribute_id;type:smallint unsigned;not null;default:0;uniqueIndex:idx_super_attribute_unq"`
	Position    int    `gorm:"column:position;not null;default:0"`
}

func (ProductSuperAttribute) TableName() string {
	return "catalog_product_super_attribute"
}
