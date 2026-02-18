package inventory

// InventorySourceItem represents inventory_source_item table (MSI)
type InventorySourceItem struct {
	SourceItemID uint    `gorm:"column:source_item_id;primaryKey;autoIncrement" json:"source_item_id,omitempty"`
	SourceCode   string  `gorm:"column:source_code;type:varchar(255);not null" json:"source_code"`
	SKU          string  `gorm:"column:sku;type:varchar(64);not null" json:"sku"`
	Quantity     float64 `gorm:"column:quantity;type:decimal(12,4);not null;default:0" json:"quantity"`
	Status       uint8   `gorm:"column:status;type:smallint unsigned;not null;default:0" json:"status"`
}

func (InventorySourceItem) TableName() string {
	return "inventory_source_item"
}
