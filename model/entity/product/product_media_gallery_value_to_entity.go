package product

// ProductMediaGalleryValueToEntity links a shared gallery pool row
// (ProductMediaGallery, catalog_product_entity_media_gallery is a global
// value pool with no entity_id of its own) to the product it belongs to.
// Real Magento also has a per-store catalog_product_entity_media_gallery_value
// table for label/position/disabled overrides; this project simplifies that
// by keeping those fields directly on ProductMediaGallery instead.
type ProductMediaGalleryValueToEntity struct {
	EntityID uint `gorm:"column:entity_id;primaryKey"`
	ValueID  uint `gorm:"column:value_id;primaryKey"`
}

func (ProductMediaGalleryValueToEntity) TableName() string {
	return "catalog_product_entity_media_gallery_value_to_entity"
}
