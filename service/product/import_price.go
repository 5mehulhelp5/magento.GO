package product

import (
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	productEntity "magento.GO/model/entity/product"
)

var priceColumns = map[string]bool{
	"price_index": true, "final_price": true, "min_price": true,
	"max_price": true, "tier_price": true,
}

// priceData holds collected price index rows ready to flush.
type priceData struct {
	rows     []productEntity.ProductIndexPrice
	warnings []string
}

// collectPrice parses CSV rows and buffers price index data.
func collectPrice(rows [][]string, colIndex map[string]int, skuToID map[string]uint) *priceData {
	d := &priceData{}
	skuCol := colIndex["sku"]

	hasAny := false
	for col := range priceColumns {
		if _, ok := colIndex[col]; ok {
			hasAny = true
			break
		}
	}
	if !hasAny {
		return d
	}

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

		item := productEntity.ProductIndexPrice{
			EntityID:        entityID,
			CustomerGroupID: 0,
			WebsiteID:       1,
		}
		populated := false

		if ci, ok := colIndex["price_index"]; ok && ci < len(row) {
			if v := strings.TrimSpace(row[ci]); v != "" {
				fv, err := strconv.ParseFloat(v, 64)
				if err != nil {
					d.warnings = append(d.warnings, fmt.Sprintf("sku=%s: invalid price_index %q", sku, v))
					continue
				}
				item.Price = fv
				item.FinalPrice = fv
				item.MinPrice = fv
				item.MaxPrice = fv
				populated = true
			}
		}
		if ci, ok := colIndex["final_price"]; ok && ci < len(row) {
			if v := strings.TrimSpace(row[ci]); v != "" {
				fv, _ := strconv.ParseFloat(v, 64)
				item.FinalPrice = fv
				populated = true
			}
		}
		if ci, ok := colIndex["min_price"]; ok && ci < len(row) {
			if v := strings.TrimSpace(row[ci]); v != "" {
				fv, _ := strconv.ParseFloat(v, 64)
				item.MinPrice = fv
			}
		}
		if ci, ok := colIndex["max_price"]; ok && ci < len(row) {
			if v := strings.TrimSpace(row[ci]); v != "" {
				fv, _ := strconv.ParseFloat(v, 64)
				item.MaxPrice = fv
			}
		}
		if ci, ok := colIndex["tier_price"]; ok && ci < len(row) {
			if v := strings.TrimSpace(row[ci]); v != "" {
				fv, _ := strconv.ParseFloat(v, 64)
				item.TierPrice = fv
			}
		}

		if populated {
			d.rows = append(d.rows, item)
		}
	}
	return d
}

// flushPrice writes buffered price index rows to DB.
func flushPrice(db *gorm.DB, d *priceData, opts ImportOptions) error {
	if len(d.rows) == 0 {
		return nil
	}
	upsert := clause.OnConflict{
		Columns:   []clause.Column{{Name: "entity_id"}, {Name: "customer_group_id"}, {Name: "website_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"price", "final_price", "min_price", "max_price", "tier_price"}),
	}
	return db.Clauses(upsert).CreateInBatches(d.rows, opts.BatchSize).Error
}
