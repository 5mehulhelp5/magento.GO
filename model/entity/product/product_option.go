package product

// Valid catalog_product_option.type values.
const (
	OptionTypeField       = "field"
	OptionTypeArea        = "area"
	OptionTypeFile        = "file"
	OptionTypeDate        = "date"
	OptionTypeDateTime    = "date_time"
	OptionTypeTime        = "time"
	OptionTypeDropDown    = "drop_down"
	OptionTypeRadio       = "radio"
	OptionTypeCheckbox    = "checkbox"
	OptionTypeMultiselect = "multiselect"
)

// SelectOptionTypes are the option types that carry a list of
// ProductOptionTypeValue choices rather than a free-form customer input.
var SelectOptionTypes = map[string]bool{
	OptionTypeDropDown:    true,
	OptionTypeRadio:       true,
	OptionTypeCheckbox:    true,
	OptionTypeMultiselect: true,
}

// ProductOption represents catalog_product_option. Real Magento splits
// title into catalog_product_option_title and price/price_type into
// catalog_product_option_price, both per-store; this project simplifies
// that to plain columns here (store_id=0 / "all stores" only), consistent
// with how other per-store data is simplified elsewhere in this project.
type ProductOption struct {
	OptionID      uint    `gorm:"column:option_id;primaryKey;autoIncrement"`
	ProductID     uint    `gorm:"column:product_id;type:int unsigned;not null;default:0"`
	Type          string  `gorm:"column:type;type:varchar(50);not null"`
	Title         string  `gorm:"column:title;type:varchar(255);not null"`
	IsRequire     uint16  `gorm:"column:is_require;type:smallint unsigned;not null;default:0"`
	Price         float64 `gorm:"column:price;type:decimal(12,4);not null;default:0"`
	PriceType     string  `gorm:"column:price_type;type:varchar(10);not null;default:fixed"`
	SKU           *string `gorm:"column:sku;type:varchar(64)"`
	MaxCharacters *int    `gorm:"column:max_characters"`
	SortOrder     int     `gorm:"column:sort_order;not null;default:0"`
}

func (ProductOption) TableName() string {
	return "catalog_product_option"
}
