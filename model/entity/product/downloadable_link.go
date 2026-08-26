package product

// DownloadableLink represents downloadable_link. Real Magento splits title
// into downloadable_link_title and price into downloadable_link_price,
// both per-store; this project simplifies that to plain columns here
// (store_id=0 / "all stores" only), the same simplification already used
// for custom options and the media gallery.
type DownloadableLink struct {
	LinkID uint `gorm:"column:link_id;primaryKey;autoIncrement"`
	// ProductID is the *parent* (downloadable-type) product this link
	// belongs to -- not a link to another catalog product.
	ProductID uint    `gorm:"column:product_id;type:int unsigned;not null;default:0"`
	Title     string  `gorm:"column:title;type:varchar(255);not null"`
	Price     float64 `gorm:"column:price;type:decimal(12,4);not null;default:0"`
	// NumberOfDownloads is the purchase's allowed download count; 0 means
	// unlimited, matching Magento's own convention.
	NumberOfDownloads int     `gorm:"column:number_of_downloads;not null;default:0"`
	LinkURL           *string `gorm:"column:link_url;type:varchar(255)"`
	SortOrder         int     `gorm:"column:sort_order;not null;default:0"`
}

func (DownloadableLink) TableName() string {
	return "downloadable_link"
}
