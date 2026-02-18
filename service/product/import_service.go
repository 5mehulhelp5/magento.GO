package product

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	entity "magento.GO/model/entity"
	productEntity "magento.GO/model/entity/product"
)

// ImportOptions configures a product import run.
type ImportOptions struct {
	StoreID      uint16
	BatchSize    int
	AttributeSet uint16
	RawSQL       bool
}

// ImportResult holds counters and timing from an import run.
type ImportResult struct {
	TotalRows   int
	Created     int
	Updated     int
	Skipped     int
	Warnings    []string
	EAVCounts   map[string]int
	ProcessTime time.Duration
	DBTime      time.Duration
	TotalTime   time.Duration
}

type attrMeta struct {
	ID          uint16
	BackendType string
}

var staticFields = map[string]bool{
	"sku": true, "type_id": true, "attribute_set_id": true,
}

// knownColumns returns all column names handled by any module.
func knownColumns(attrMap map[string]attrMeta) map[string]bool {
	known := make(map[string]bool)
	for k := range staticFields {
		known[k] = true
	}
	for code, meta := range attrMap {
		if meta.BackendType != "static" {
			known[code] = true
		}
	}
	for col := range stockColumns {
		known[col] = true
	}
	for col := range galleryColumns {
		known[col] = true
	}
	for col := range priceColumns {
		known[col] = true
	}
	return known
}

// ImportProducts reads CSV data from r and upserts products into Magento tables.
func ImportProducts(db *gorm.DB, r io.Reader, opts ImportOptions) (*ImportResult, error) {
	startTotal := time.Now()

	if opts.BatchSize <= 0 {
		opts.BatchSize = 500
	}
	if opts.AttributeSet == 0 {
		opts.AttributeSet = 4
	}

	// Parse CSV header
	reader := csv.NewReader(r)
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read CSV header: %w", err)
	}
	for i := range headers {
		headers[i] = strings.TrimSpace(headers[i])
	}

	skuCol := -1
	colIndex := make(map[string]int, len(headers))
	for i, h := range headers {
		colIndex[h] = i
		if h == "sku" {
			skuCol = i
		}
	}
	if skuCol < 0 {
		return nil, fmt.Errorf("CSV must contain a 'sku' column")
	}

	// Load EAV attribute metadata
	var attrs []entity.EavAttribute
	if err := db.Where("entity_type_id = ?", 4).Find(&attrs).Error; err != nil {
		return nil, fmt.Errorf("load attributes: %w", err)
	}
	attrMap := make(map[string]attrMeta, len(attrs))
	for _, a := range attrs {
		attrMap[a.AttributeCode] = attrMeta{ID: a.AttributeID, BackendType: a.BackendType}
	}

	result := &ImportResult{EAVCounts: make(map[string]int)}

	// Warn about unknown columns
	known := knownColumns(attrMap)
	for _, h := range headers {
		if !known[h] {
			result.Warnings = append(result.Warnings, fmt.Sprintf("column %q: unknown, skipping", h))
		}
	}

	// Read all rows
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read CSV rows: %w", err)
	}
	result.TotalRows = len(rows)

	// Batch lookup existing SKUs
	skus := make([]string, 0, len(rows))
	for _, row := range rows {
		if skuCol < len(row) && row[skuCol] != "" {
			skus = append(skus, row[skuCol])
		}
	}
	skuToID := lookupSKUs(db, skus, opts.BatchSize)

	startProcess := time.Now()

	// Insert new product entities
	result.Created, result.Skipped = insertNewEntities(db, rows, skuCol, colIndex, skuToID, opts)

	// Collect data for each module
	eavData := collectEAV(rows, colIndex, skuToID, attrMap, headers, opts)
	stockData := collectStock(rows, colIndex, skuToID)
	galleryData := collectGallery(rows, colIndex, skuToID)
	priceData := collectPrice(rows, colIndex, skuToID)

	result.Warnings = append(result.Warnings, eavData.warnings...)
	result.Warnings = append(result.Warnings, stockData.warnings...)
	result.Warnings = append(result.Warnings, priceData.warnings...)

	// Flush all modules to DB in parallel
	startDB := time.Now()
	var wg sync.WaitGroup
	errs := make(chan error, 4)

	wg.Add(4)
	go func() { defer wg.Done(); errs <- flushEAV(db, eavData, opts) }()
	go func() { defer wg.Done(); errs <- flushStock(db, stockData, opts) }()
	go func() { defer wg.Done(); errs <- flushGallery(db, galleryData, opts) }()
	go func() { defer wg.Done(); errs <- flushPrice(db, priceData, opts) }()
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			return nil, err
		}
	}
	result.DBTime = time.Since(startDB)

	// Merge counts
	for k, v := range eavData.counts() {
		result.EAVCounts[k] = v
	}
	result.EAVCounts["stock"] = len(stockData.rows)
	result.EAVCounts["gallery"] = len(galleryData.rows)
	result.EAVCounts["price_index"] = len(priceData.rows)

	result.Updated = result.TotalRows - result.Skipped - result.Created
	result.ProcessTime = time.Since(startProcess)
	result.TotalTime = time.Since(startTotal)

	return result, nil
}

// lookupSKUs batch-queries existing SKUs and returns sku->entity_id map.
func lookupSKUs(db *gorm.DB, skus []string, batchSize int) map[string]uint {
	type skuRow struct {
		EntityID uint   `gorm:"column:entity_id"`
		SKU      string `gorm:"column:sku"`
	}
	var existing []skuRow
	for i := 0; i < len(skus); i += batchSize {
		end := i + batchSize
		if end > len(skus) {
			end = len(skus)
		}
		var chunk []skuRow
		db.Table("catalog_product_entity").Select("entity_id, sku").Where("sku IN ?", skus[i:end]).Find(&chunk)
		existing = append(existing, chunk...)
	}
	m := make(map[string]uint, len(existing))
	for _, e := range existing {
		m[e.SKU] = e.EntityID
	}
	return m
}

// insertNewEntities creates entity rows for new SKUs and updates skuToID in place.
func insertNewEntities(db *gorm.DB, rows [][]string, skuCol int, colIndex map[string]int, skuToID map[string]uint, opts ImportOptions) (created, skipped int) {
	typeCol, hasType := colIndex["type_id"]
	attrSetCol, hasAttrSet := colIndex["attribute_set_id"]

	newProducts := make([]productEntity.Product, 0, len(rows))
	newIndices := make([]int, 0, len(rows))

	for ri, row := range rows {
		sku := ""
		if skuCol < len(row) {
			sku = strings.TrimSpace(row[skuCol])
		}
		if sku == "" {
			skipped++
			continue
		}
		if _, exists := skuToID[sku]; exists {
			continue
		}
		p := productEntity.Product{SKU: sku, AttributeSetID: opts.AttributeSet, TypeID: "simple"}
		if hasType && typeCol < len(row) {
			if v := strings.TrimSpace(row[typeCol]); v != "" {
				p.TypeID = v
			}
		}
		if hasAttrSet && attrSetCol < len(row) {
			if v, err := strconv.ParseUint(strings.TrimSpace(row[attrSetCol]), 10, 16); err == nil && v > 0 {
				p.AttributeSetID = uint16(v)
			}
		}
		newProducts = append(newProducts, p)
		newIndices = append(newIndices, ri)
	}

	if len(newProducts) > 0 {
		db.Session(&gorm.Session{SkipHooks: true, CreateBatchSize: opts.BatchSize}).Create(&newProducts)
		for i, p := range newProducts {
			sku := strings.TrimSpace(rows[newIndices[i]][skuCol])
			skuToID[sku] = p.EntityID
		}
	}
	return len(newProducts), skipped
}
