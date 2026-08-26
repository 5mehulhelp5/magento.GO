package product

import (
	"strings"

	"gorm.io/gorm"

	productEntity "magento.GO/model/entity/product"
)

var galleryColumns = map[string]bool{
	"image": true, "small_image": true, "thumbnail": true,
	"media_gallery": true,
}

// galleryData holds collected gallery rows ready to flush. entityIDs is
// parallel to rows (same index = same gallery pool row's owning product) --
// needed because the pool row (rows[i]) has no entity_id column of its own;
// the link is only known once flushGallery inserts it and gets back a
// value_id to pair with entityIDs[i].
type galleryData struct {
	rows      []productEntity.ProductMediaGallery
	entityIDs []uint
}

// collectGallery parses CSV rows and buffers media gallery entries.
func collectGallery(rows [][]string, colIndex map[string]int, skuToID map[string]uint) *galleryData {
	d := &galleryData{}
	skuCol := colIndex["sku"]

	imageCols := []string{"image", "small_image", "thumbnail", "media_gallery"}
	var activeCols []string
	for _, col := range imageCols {
		if _, ok := colIndex[col]; ok {
			activeCols = append(activeCols, col)
		}
	}
	if len(activeCols) == 0 {
		return d
	}

	seen := make(map[string]bool)

	for _, row := range rows {
		sku := ""
		if skuCol < len(row) {
			sku = strings.TrimSpace(row[skuCol])
		}
		if sku == "" {
			continue
		}
		entityID, ok := skuToID[sku]
		if !ok {
			continue
		}

		for _, col := range activeCols {
			ci := colIndex[col]
			if ci >= len(row) {
				continue
			}
			val := strings.TrimSpace(row[ci])
			if val == "" {
				continue
			}
			images := strings.Split(val, "|")
			for _, img := range images {
				img = strings.TrimSpace(img)
				if img == "" {
					continue
				}
				key := sku + ":" + img
				if seen[key] {
					continue
				}
				seen[key] = true

				d.rows = append(d.rows, productEntity.ProductMediaGallery{
					AttributeID: 87,
					Value:       img,
					MediaType:   "image",
					Disabled:    0,
				})
				d.entityIDs = append(d.entityIDs, entityID)
			}
		}
	}
	return d
}

// flushGallery writes buffered gallery rows to DB, then links each one to
// its owning product via catalog_product_entity_media_gallery_value_to_entity
// -- CreateInBatches backfills each row's auto-increment ValueID into d.rows
// in place, so that ID is available immediately afterward to pair with
// entityIDs[i] for the link rows.
func flushGallery(db *gorm.DB, d *galleryData, opts ImportOptions) error {
	if len(d.rows) == 0 {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.CreateInBatches(d.rows, opts.BatchSize).Error; err != nil {
			return err
		}

		links := make([]productEntity.ProductMediaGalleryValueToEntity, len(d.rows))
		for i, row := range d.rows {
			links[i] = productEntity.ProductMediaGalleryValueToEntity{
				ValueID:  row.ValueID,
				EntityID: d.entityIDs[i],
			}
		}
		return tx.CreateInBatches(links, opts.BatchSize).Error
	})
}
