package product

import (
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	productEntity "magento.GO/model/entity/product"
)

var stockColumns = map[string]bool{
	"qty": true, "is_in_stock": true, "manage_stock": true,
	"min_qty": true, "max_sale_qty": true, "min_sale_qty": true,
}

// stockData holds collected stock rows ready to flush.
type stockData struct {
	rows     []productEntity.StockItem
	warnings []string
}

// collectStock parses CSV rows and buffers stock item data.
func collectStock(rows [][]string, colIndex map[string]int, skuToID map[string]uint) *stockData {
	d := &stockData{}
	skuCol := colIndex["sku"]

	hasAny := false
	for col := range stockColumns {
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

		item := productEntity.StockItem{
			ProductID:   entityID,
			StockID:     1,
			IsInStock:   1,
			ManageStock: 1,
		}
		populated := false

		if ci, ok := colIndex["qty"]; ok && ci < len(row) {
			if v := strings.TrimSpace(row[ci]); v != "" {
				q, err := strconv.ParseFloat(v, 64)
				if err != nil {
					d.warnings = append(d.warnings, fmt.Sprintf("sku=%s: invalid qty %q", sku, v))
					continue
				}
				item.Qty = q
				populated = true
			}
		}
		if ci, ok := colIndex["is_in_stock"]; ok && ci < len(row) {
			if v := strings.TrimSpace(row[ci]); v != "" {
				iv, err := strconv.ParseUint(v, 10, 16)
				if err != nil {
					d.warnings = append(d.warnings, fmt.Sprintf("sku=%s: invalid is_in_stock %q", sku, v))
					continue
				}
				item.IsInStock = uint16(iv)
				populated = true
			}
		}
		if ci, ok := colIndex["manage_stock"]; ok && ci < len(row) {
			if v := strings.TrimSpace(row[ci]); v != "" {
				iv, _ := strconv.ParseUint(v, 10, 16)
				item.ManageStock = uint16(iv)
			}
		}
		if ci, ok := colIndex["min_qty"]; ok && ci < len(row) {
			if v := strings.TrimSpace(row[ci]); v != "" {
				fv, _ := strconv.ParseFloat(v, 64)
				item.MinQty = fv
			}
		}
		if ci, ok := colIndex["min_sale_qty"]; ok && ci < len(row) {
			if v := strings.TrimSpace(row[ci]); v != "" {
				fv, _ := strconv.ParseFloat(v, 64)
				item.MinSaleQty = fv
			}
		}
		if ci, ok := colIndex["max_sale_qty"]; ok && ci < len(row) {
			if v := strings.TrimSpace(row[ci]); v != "" {
				fv, _ := strconv.ParseFloat(v, 64)
				item.MaxSaleQty = fv
			}
		}

		if populated {
			d.rows = append(d.rows, item)
		}
	}
	return d
}

// flushStock writes buffered stock items to DB.
func flushStock(db *gorm.DB, d *stockData, opts ImportOptions) error {
	if len(d.rows) == 0 {
		return nil
	}
	upsert := clause.OnConflict{
		Columns:   []clause.Column{{Name: "product_id"}, {Name: "stock_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"qty", "is_in_stock", "manage_stock", "min_qty", "min_sale_qty", "max_sale_qty"}),
	}
	return db.Clauses(upsert).CreateInBatches(d.rows, opts.BatchSize).Error
}

// StockItemInput is the JSON input for the stock import API.
type StockItemInput struct {
	SKU         string   `json:"sku" validate:"required"`
	Qty         *float64 `json:"qty"`
	IsInStock   *uint16  `json:"is_in_stock"`
	ManageStock *uint16  `json:"manage_stock"`
	MinQty      *float64 `json:"min_qty"`
	MinSaleQty  *float64 `json:"min_sale_qty"`
	MaxSaleQty  *float64 `json:"max_sale_qty"`
}

// StockImportResult holds the result of a stock import run.
type StockImportResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Warnings []string `json:"warnings,omitempty"`
}

// ImportStockJSON resolves SKUs to entity IDs and upserts stock items.
func ImportStockJSON(db *gorm.DB, items []StockItemInput, batchSize int) (*StockImportResult, error) {
	if batchSize <= 0 {
		batchSize = 500
	}

	result := &StockImportResult{}

	// Collect unique SKUs
	skus := make([]string, 0, len(items))
	for _, it := range items {
		if it.SKU != "" {
			skus = append(skus, it.SKU)
		}
	}

	// Batch lookup SKU -> entity_id
	type skuRow struct {
		EntityID uint   `gorm:"column:entity_id"`
		SKU      string `gorm:"column:sku"`
	}
	skuToID := make(map[string]uint)
	for i := 0; i < len(skus); i += batchSize {
		end := i + batchSize
		if end > len(skus) {
			end = len(skus)
		}
		var chunk []skuRow
		db.Table("catalog_product_entity").Select("entity_id, sku").Where("sku IN ?", skus[i:end]).Find(&chunk)
		for _, r := range chunk {
			skuToID[r.SKU] = r.EntityID
		}
	}

	// Build stock item rows
	rows := make([]productEntity.StockItem, 0, len(items))
	for _, it := range items {
		if it.SKU == "" {
			result.Skipped++
			result.Warnings = append(result.Warnings, "empty sku, skipping")
			continue
		}
		entityID, ok := skuToID[it.SKU]
		if !ok {
			result.Skipped++
			result.Warnings = append(result.Warnings, fmt.Sprintf("sku=%s: product not found", it.SKU))
			continue
		}

		item := productEntity.StockItem{
			ProductID:   entityID,
			StockID:     1,
			IsInStock:   1,
			ManageStock: 1,
		}
		if it.Qty != nil {
			item.Qty = *it.Qty
		}
		if it.IsInStock != nil {
			item.IsInStock = *it.IsInStock
		}
		if it.ManageStock != nil {
			item.ManageStock = *it.ManageStock
		}
		if it.MinQty != nil {
			item.MinQty = *it.MinQty
		}
		if it.MinSaleQty != nil {
			item.MinSaleQty = *it.MinSaleQty
		}
		if it.MaxSaleQty != nil {
			item.MaxSaleQty = *it.MaxSaleQty
		}
		rows = append(rows, item)
	}

	if len(rows) > 0 {
		upsert := clause.OnConflict{
			Columns:   []clause.Column{{Name: "product_id"}, {Name: "stock_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"qty", "is_in_stock", "manage_stock", "min_qty", "min_sale_qty", "max_sale_qty"}),
		}
		if err := db.Clauses(upsert).CreateInBatches(rows, batchSize).Error; err != nil {
			return nil, fmt.Errorf("stock upsert: %w", err)
		}
	}

	result.Imported = len(rows)
	return result, nil
}
