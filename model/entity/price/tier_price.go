package price

// TierPrice represents catalog_product_entity_tier_price table
type TierPrice struct {
	ValueID         uint    `gorm:"column:value_id;primaryKey;autoIncrement" json:"value_id,omitempty"`
	EntityID        uint    `gorm:"column:entity_id;index" json:"entity_id,omitempty"`
	RowID           uint    `gorm:"column:row_id;index" json:"row_id,omitempty"`
	AllGroups       uint8   `gorm:"column:all_groups;type:smallint unsigned;not null;default:1" json:"all_groups"`
	CustomerGroupID uint16  `gorm:"column:customer_group_id;type:smallint unsigned;not null;default:0" json:"customer_group_id"`
	Qty             float64 `gorm:"column:qty;type:decimal(12,4);not null;default:1" json:"qty"`
	Value           float64 `gorm:"column:value;type:decimal(20,6);not null;default:0" json:"value"`
	WebsiteID       uint16  `gorm:"column:website_id;type:smallint unsigned;not null" json:"website_id"`
	PercentageValue float64 `gorm:"column:percentage_value;type:decimal(5,2)" json:"percentage_value,omitempty"`
}

func (TierPrice) TableName() string {
	return "catalog_product_entity_tier_price"
}

// LinkID returns entity_id or row_id based on schema
func (t *TierPrice) LinkID(isEnterprise bool) uint {
	if isEnterprise {
		return t.RowID
	}
	return t.EntityID
}
