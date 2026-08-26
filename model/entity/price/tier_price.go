package price

// TierPrice represents catalog_product_entity_tier_price table
type TierPrice struct {
	ValueID         uint    `gorm:"column:value_id;primaryKey;autoIncrement" json:"value_id,omitempty"`
	EntityID        uint    `gorm:"column:entity_id;index;uniqueIndex:idx_tier_price_unq" json:"entity_id,omitempty"`
	RowID           uint    `gorm:"column:row_id;index" json:"row_id,omitempty"`
	// AllGroups and Qty deliberately have no GORM `default` tag: GORM
	// substitutes a field's `default` tag value for ANY zero Go value at
	// bind time (not just when the column is omitted), so a legitimate
	// AllGroups=0 ("this is a specific group, not all groups") would
	// silently become 1 on every insert if the tag were present here.
	AllGroups       uint8   `gorm:"column:all_groups;type:smallint unsigned;not null;uniqueIndex:idx_tier_price_unq" json:"all_groups"`
	CustomerGroupID uint16  `gorm:"column:customer_group_id;type:smallint unsigned;not null;default:0;uniqueIndex:idx_tier_price_unq" json:"customer_group_id"`
	Qty             float64 `gorm:"column:qty;type:decimal(12,4);not null;uniqueIndex:idx_tier_price_unq" json:"qty"`
	Value           float64 `gorm:"column:value;type:decimal(20,6);not null;default:0" json:"value"`
	WebsiteID       uint16  `gorm:"column:website_id;type:smallint unsigned;not null;uniqueIndex:idx_tier_price_unq" json:"website_id"`
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
