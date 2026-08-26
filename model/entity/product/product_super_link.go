package product

// ProductSuperLink represents catalog_product_super_link -- links a simple
// (child) product to its owning configurable (parent) product. ProductID
// is unique alone, matching real Magento: a simple product can be a child
// of at most one configurable product.
type ProductSuperLink struct {
	LinkID    uint `gorm:"column:link_id;primaryKey;autoIncrement"`
	ProductID uint `gorm:"column:product_id;type:int unsigned;not null;default:0;uniqueIndex:idx_super_link_unq"`
	ParentID  uint `gorm:"column:parent_id;type:int unsigned;not null;default:0"`
}

func (ProductSuperLink) TableName() string {
	return "catalog_product_super_link"
}
