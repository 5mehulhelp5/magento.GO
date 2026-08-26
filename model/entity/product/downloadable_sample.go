package product

// DownloadableSample represents downloadable_sample, a free preview
// attached to a downloadable-type product (as opposed to DownloadableLink,
// which is the paid content itself). Title is a plain column rather than
// a separate per-store downloadable_sample_title table, same
// simplification as DownloadableLink.
type DownloadableSample struct {
	SampleID  uint    `gorm:"column:sample_id;primaryKey;autoIncrement"`
	ProductID uint    `gorm:"column:product_id;type:int unsigned;not null;default:0"`
	Title     string  `gorm:"column:title;type:varchar(255);not null"`
	SampleURL *string `gorm:"column:sample_url;type:varchar(255)"`
	SortOrder int     `gorm:"column:sort_order;not null;default:0"`
}

func (DownloadableSample) TableName() string {
	return "downloadable_sample"
}
