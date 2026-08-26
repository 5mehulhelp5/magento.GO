package product

// Valid catalog_product_bundle_option.type values.
const (
	BundleOptionSelect   = "select"
	BundleOptionRadio    = "radio"
	BundleOptionCheckbox = "checkbox"
	BundleOptionMulti    = "multi"
)

var ValidBundleOptionTypes = map[string]bool{
	BundleOptionSelect:   true,
	BundleOptionRadio:    true,
	BundleOptionCheckbox: true,
	BundleOptionMulti:    true,
}

// ProductBundleOption represents catalog_product_bundle_option -- one
// choice group (e.g. "CPU", "Accessories") on a bundle-type product.
// Real Magento splits title into a per-store
// catalog_product_bundle_option_value table; this project simplifies that
// to a plain column here, the same simplification used for custom options.
type ProductBundleOption struct {
	OptionID uint `gorm:"column:option_id;primaryKey;autoIncrement"`
	// ParentID is the bundle product this option group belongs to.
	ParentID uint   `gorm:"column:parent_id;type:int unsigned;not null;default:0"`
	Required uint16 `gorm:"column:required;type:smallint unsigned;not null;default:0"`
	Position int    `gorm:"column:position;not null;default:0"`
	Type     string `gorm:"column:type;type:varchar(20);not null;default:select"`
	Title    string `gorm:"column:title;type:varchar(255);not null"`
}

func (ProductBundleOption) TableName() string {
	return "catalog_product_bundle_option"
}
