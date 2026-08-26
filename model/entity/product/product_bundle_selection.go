package product

// ProductBundleSelection represents catalog_product_bundle_selection -- one
// selectable component product within a ProductBundleOption group.
type ProductBundleSelection struct {
	SelectionID uint `gorm:"column:selection_id;primaryKey;autoIncrement"`
	OptionID    uint `gorm:"column:option_id;type:int unsigned;not null;default:0"`
	// ParentProductID is the bundle product; ProductID is the selectable
	// component product.
	ParentProductID     uint    `gorm:"column:parent_product_id;type:int unsigned;not null;default:0"`
	ProductID           uint    `gorm:"column:product_id;type:int unsigned;not null;default:0"`
	Position            int     `gorm:"column:position;not null;default:0"`
	IsDefault           uint16  `gorm:"column:is_default;type:smallint unsigned;not null;default:0"`
	SelectionQty        float64 `gorm:"column:selection_qty;type:decimal(12,4);not null;default:1"`
	SelectionPriceValue float64 `gorm:"column:selection_price_value;type:decimal(12,4);not null;default:0"`
	SelectionPriceType  string  `gorm:"column:selection_price_type;type:varchar(10);not null;default:fixed"`
	// No GORM `default` tag: GORM substitutes a field's `default` tag
	// value for ANY zero Go value at bind time (see the same fix applied
	// to TierPrice.AllGroups), so an explicit "can't change qty" (0)
	// would silently become 1 on every insert if this had `default:1`.
	SelectionCanChangeQty uint16 `gorm:"column:selection_can_change_qty;type:smallint unsigned;not null"`
}

func (ProductBundleSelection) TableName() string {
	return "catalog_product_bundle_selection"
}
