package product

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	productEntity "magento.GO/model/entity/product"
)

// productLinkColumns maps each CSV column to the link type it produces.
// Each column is a comma-separated list of SKUs. "grouped_skus" backs
// grouped products (Magento's LINK_TYPE_GROUPED) -- the "grouped" product
// type itself needs no special handling here since type_id already comes
// straight from the CSV's own type_id column.
var productLinkColumns = map[string]uint16{
	"related_skus":   productEntity.LinkTypeRelated,
	"upsell_skus":    productEntity.LinkTypeUpSell,
	"crosssell_skus": productEntity.LinkTypeCrossSell,
	"grouped_skus":   productEntity.LinkTypeGrouped,
}

// productLinkData holds collected product link rows ready to flush.
type productLinkData struct {
	rows     []productEntity.ProductLink
	warnings []string
}

// linkSKUColumns returns the CSV columns collectProductLinks reads, so
// ImportProducts can fold their SKUs into the same batch SKU lookup used
// for the primary "sku" column -- a related/upsell/crosssell/grouped SKU
// must already resolve to an existing product_id; this project's import
// doesn't create products from a link column, only from "sku" itself.
func linkSKUColumns(colIndex map[string]int) []string {
	cols := make([]string, 0, len(productLinkColumns))
	for col := range productLinkColumns {
		if _, ok := colIndex[col]; ok {
			cols = append(cols, col)
		}
	}
	return cols
}

// collectProductLinks parses the related_skus/upsell_skus/crosssell_skus/
// grouped_skus columns into ProductLink rows. skuToID must already contain
// any SKU referenced by these columns (see linkSKUColumns).
func collectProductLinks(rows [][]string, colIndex map[string]int, skuToID map[string]uint) *productLinkData {
	d := &productLinkData{}
	activeCols := linkSKUColumns(colIndex)
	if len(activeCols) == 0 {
		return d
	}
	skuCol := colIndex["sku"]

	for _, row := range rows {
		sku := ""
		if skuCol < len(row) {
			sku = strings.TrimSpace(row[skuCol])
		}
		if sku == "" {
			continue
		}
		productID, ok := skuToID[sku]
		if !ok {
			continue
		}

		for _, col := range activeCols {
			linkType := productLinkColumns[col]
			ci := colIndex[col]
			if ci >= len(row) {
				continue
			}
			val := strings.TrimSpace(row[ci])
			if val == "" {
				continue
			}

			for pos, linkedSKU := range strings.Split(val, ",") {
				linkedSKU = strings.TrimSpace(linkedSKU)
				if linkedSKU == "" {
					continue
				}
				if linkedSKU == sku {
					d.warnings = append(d.warnings, fmt.Sprintf("sku=%s: %s references itself, skipping", sku, col))
					continue
				}
				linkedID, ok := skuToID[linkedSKU]
				if !ok {
					d.warnings = append(d.warnings, fmt.Sprintf("sku=%s: %s references unknown SKU %q, skipping", sku, col, linkedSKU))
					continue
				}
				d.rows = append(d.rows, productEntity.ProductLink{
					ProductID:       productID,
					LinkedProductID: linkedID,
					LinkTypeID:      linkType,
					Position:        uint(pos),
				})
			}
		}
	}
	return d
}

// flushProductLinks upserts buffered product link rows, keyed on Magento's
// own (product_id, linked_product_id, link_type_id) uniqueness rule.
func flushProductLinks(db *gorm.DB, d *productLinkData, opts ImportOptions) error {
	if len(d.rows) == 0 {
		return nil
	}
	upsert := clause.OnConflict{
		Columns:   []clause.Column{{Name: "product_id"}, {Name: "linked_product_id"}, {Name: "link_type_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"position"}),
	}
	return db.Transaction(func(tx *gorm.DB) error {
		return tx.Clauses(upsert).CreateInBatches(d.rows, opts.BatchSize).Error
	})
}
