package product

// ProductOptionTypeValue represents catalog_product_option_type_value, one
// selectable choice belonging to a select-type ProductOption (drop_down,
// radio, checkbox, multiselect). Simplified the same way ProductOption is:
// title/price/price_type are plain columns rather than per-store rows in
// separate catalog_product_option_type_title/_price tables.
type ProductOptionTypeValue struct {
	OptionTypeID uint    `gorm:"column:option_type_id;primaryKey;autoIncrement"`
	OptionID     uint    `gorm:"column:option_id;type:int unsigned;not null;default:0"`
	Title        string  `gorm:"column:title;type:varchar(255);not null"`
	Price        float64 `gorm:"column:price;type:decimal(12,4);not null;default:0"`
	PriceType    string  `gorm:"column:price_type;type:varchar(10);not null;default:fixed"`
	SKU          *string `gorm:"column:sku;type:varchar(64)"`
	SortOrder    int     `gorm:"column:sort_order;not null;default:0"`
}

func (ProductOptionTypeValue) TableName() string {
	return "catalog_product_option_type_value"
}
