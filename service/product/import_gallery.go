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

// galleryData holds collected gallery rows ready to flush.
type galleryData struct {
	rows []productEntity.ProductMediaGallery
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
		if _, ok := skuToID[sku]; !ok {
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
			}
		}
	}
	return d
}

// flushGallery writes buffered gallery rows to DB.
func flushGallery(db *gorm.DB, d *galleryData, opts ImportOptions) error {
	if len(d.rows) == 0 {
		return nil
	}
	return db.CreateInBatches(d.rows, opts.BatchSize).Error
}
