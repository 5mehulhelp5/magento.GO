package product

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	productEntity "magento.GO/model/entity/product"
)

var configurableColumns = map[string]bool{
	"configurable_attributes": true, "configurable_variations": true,
}

// configurableData holds collected super-attribute/super-link rows ready
// to flush.
type configurableData struct {
	attributes []productEntity.ProductSuperAttribute
	links      []productEntity.ProductSuperLink
	warnings   []string
}

// configurableChildSKUs scans "configurable_variations" across every row
// and returns every distinct child SKU referenced, so ImportProducts can
// fold them into the same batch SKU lookup used for product links and
// bundle selections -- a configurable's child must already exist as its
// own row in this (or an earlier) import, it isn't created from this
// column.
func configurableChildSKUs(rows [][]string, colIndex map[string]int) []string {
	ci, ok := colIndex["configurable_variations"]
	if !ok {
		return nil
	}
	var skus []string
	for _, row := range rows {
		if ci >= len(row) {
			continue
		}
		for _, sku := range strings.Split(row[ci], ",") {
			if sku = strings.TrimSpace(sku); sku != "" {
				skus = append(skus, sku)
			}
		}
	}
	return skus
}

// collectConfigurable parses "configurable_attributes" (a comma-separated
// list of attribute codes that vary across this configurable's children,
// e.g. "color,size") and "configurable_variations" (a comma-separated list
// of child SKUs) on any row that has at least one of the two columns
// populated -- that row's own SKU becomes the configurable parent.
//
// Unlike Magento's own CSV import, variations here are just child SKUs,
// not "sku=X,color=Y,size=Z" pairs: each child is expected to already be
// its own CSV row (or pre-existing product) carrying its own EAV attribute
// values normally, so there's nothing to duplicate here.
func collectConfigurable(rows [][]string, colIndex map[string]int, skuToID map[string]uint, attrMap map[string]attrMeta) *configurableData {
	d := &configurableData{}
	attrCol, hasAttrs := colIndex["configurable_attributes"]
	varCol, hasVars := colIndex["configurable_variations"]
	if !hasAttrs && !hasVars {
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
		parentID, ok := skuToID[sku]
		if !ok {
			continue
		}

		attrsVal := ""
		if hasAttrs && attrCol < len(row) {
			attrsVal = strings.TrimSpace(row[attrCol])
		}
		varsVal := ""
		if hasVars && varCol < len(row) {
			varsVal = strings.TrimSpace(row[varCol])
		}
		if attrsVal == "" && varsVal == "" {
			continue
		}

		for pos, code := range strings.Split(attrsVal, ",") {
			code = strings.TrimSpace(code)
			if code == "" {
				continue
			}
			meta, ok := attrMap[code]
			if !ok {
				d.warnings = append(d.warnings, fmt.Sprintf("sku=%s: configurable_attributes references unknown attribute %q, skipping", sku, code))
				continue
			}
			d.attributes = append(d.attributes, productEntity.ProductSuperAttribute{
				ProductID: parentID, AttributeID: meta.ID, Position: pos,
			})
		}

		for _, childSKU := range strings.Split(varsVal, ",") {
			childSKU = strings.TrimSpace(childSKU)
			if childSKU == "" {
				continue
			}
			if childSKU == sku {
				d.warnings = append(d.warnings, fmt.Sprintf("sku=%s: configurable_variations references itself, skipping", sku))
				continue
			}
			childID, ok := skuToID[childSKU]
			if !ok {
				d.warnings = append(d.warnings, fmt.Sprintf("sku=%s: configurable_variations references unknown SKU %q, skipping", sku, childSKU))
				continue
			}
			d.links = append(d.links, productEntity.ProductSuperLink{ProductID: childID, ParentID: parentID})
		}
	}
	return d
}

// flushConfigurable upserts buffered super-attribute/super-link rows.
// Links upsert on product_id alone (a child belongs to at most one
// configurable parent, matching real Magento); attributes upsert on
// (product_id, attribute_id).
func flushConfigurable(db *gorm.DB, d *configurableData, opts ImportOptions) error {
	if len(d.attributes) == 0 && len(d.links) == 0 {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if len(d.attributes) > 0 {
			upsert := clause.OnConflict{
				Columns:   []clause.Column{{Name: "product_id"}, {Name: "attribute_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"position"}),
			}
			if err := tx.Clauses(upsert).CreateInBatches(d.attributes, opts.BatchSize).Error; err != nil {
				return err
			}
		}
		if len(d.links) > 0 {
			upsert := clause.OnConflict{
				Columns:   []clause.Column{{Name: "product_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"parent_id"}),
			}
			if err := tx.Clauses(upsert).CreateInBatches(d.links, opts.BatchSize).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
