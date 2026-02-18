package product

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	productEntity "magento.GO/model/entity/product"
)

// eavRow holds a single pre-validated EAV value for raw SQL insert.
type eavRow struct {
	EntityID    uint
	AttributeID uint16
	StoreID     uint16
	Value       string
}

type eavColMapping struct {
	Code        string
	AttrID      uint16
	BackendType string
}

var eavTables = map[string]string{
	"varchar":  "catalog_product_entity_varchar",
	"int":      "catalog_product_entity_int",
	"decimal":  "catalog_product_entity_decimal",
	"text":     "catalog_product_entity_text",
	"datetime": "catalog_product_entity_datetime",
}

// eavData holds collected EAV rows ready to flush.
type eavData struct {
	mappings map[string]eavColMapping
	buckets  map[string][]eavRow
	warnings []string

	varcharRows  []productEntity.ProductVarchar
	intRows      []productEntity.ProductInt
	decimalRows  []productEntity.ProductDecimal
	textRows     []productEntity.ProductText
	datetimeRows []productEntity.ProductDatetime
}

func (d *eavData) counts() map[string]int {
	return map[string]int{
		"varchar":  len(d.buckets["varchar"]) + len(d.varcharRows),
		"int":      len(d.buckets["int"]) + len(d.intRows),
		"decimal":  len(d.buckets["decimal"]) + len(d.decimalRows),
		"text":     len(d.buckets["text"]) + len(d.textRows),
		"datetime": len(d.buckets["datetime"]) + len(d.datetimeRows),
	}
}

// collectEAV parses CSV rows and buffers EAV attribute values.
func collectEAV(rows [][]string, colIndex map[string]int, skuToID map[string]uint, attrMap map[string]attrMeta, headers []string, opts ImportOptions) *eavData {
	mappings := make(map[string]eavColMapping)
	for _, h := range headers {
		if staticFields[h] {
			continue
		}
		if meta, ok := attrMap[h]; ok && meta.BackendType != "static" {
			mappings[h] = eavColMapping{Code: h, AttrID: meta.ID, BackendType: meta.BackendType}
		}
	}

	d := &eavData{
		mappings: mappings,
		buckets: map[string][]eavRow{
			"varchar": nil, "int": nil, "decimal": nil, "text": nil, "datetime": nil,
		},
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
		entityID, ok := skuToID[sku]
		if !ok {
			continue
		}
		for col, m := range mappings {
			ci, ok := colIndex[col]
			if !ok || ci >= len(row) {
				continue
			}
			val := strings.TrimSpace(row[ci])
			if val == "" {
				continue
			}

			switch m.BackendType {
			case "varchar":
				if opts.RawSQL {
					d.buckets["varchar"] = append(d.buckets["varchar"], eavRow{entityID, m.AttrID, opts.StoreID, val})
				} else {
					d.varcharRows = append(d.varcharRows, productEntity.ProductVarchar{
						AttributeID: m.AttrID, StoreID: opts.StoreID, EntityID: entityID, Value: val,
					})
				}
			case "int":
				if _, err := strconv.Atoi(val); err != nil {
					d.warnings = append(d.warnings, fmt.Sprintf("sku=%s attr=%s: invalid int %q", sku, m.Code, val))
					continue
				}
				if opts.RawSQL {
					d.buckets["int"] = append(d.buckets["int"], eavRow{entityID, m.AttrID, opts.StoreID, val})
				} else {
					iv, _ := strconv.Atoi(val)
					d.intRows = append(d.intRows, productEntity.ProductInt{
						AttributeID: m.AttrID, StoreID: opts.StoreID, EntityID: entityID, Value: iv,
					})
				}
			case "decimal":
				if _, err := strconv.ParseFloat(val, 64); err != nil {
					d.warnings = append(d.warnings, fmt.Sprintf("sku=%s attr=%s: invalid decimal %q", sku, m.Code, val))
					continue
				}
				if opts.RawSQL {
					d.buckets["decimal"] = append(d.buckets["decimal"], eavRow{entityID, m.AttrID, opts.StoreID, val})
				} else {
					dv, _ := strconv.ParseFloat(val, 64)
					d.decimalRows = append(d.decimalRows, productEntity.ProductDecimal{
						AttributeID: m.AttrID, StoreID: opts.StoreID, EntityID: entityID, Value: dv,
					})
				}
			case "text":
				if opts.RawSQL {
					d.buckets["text"] = append(d.buckets["text"], eavRow{entityID, m.AttrID, opts.StoreID, val})
				} else {
					d.textRows = append(d.textRows, productEntity.ProductText{
						AttributeID: m.AttrID, StoreID: opts.StoreID, EntityID: entityID, Value: val,
					})
				}
			case "datetime":
				if _, err := time.Parse("2006-01-02 15:04:05", val); err != nil {
					if _, err2 := time.Parse("2006-01-02", val); err2 != nil {
						d.warnings = append(d.warnings, fmt.Sprintf("sku=%s attr=%s: invalid datetime %q", sku, m.Code, val))
						continue
					}
				}
				if opts.RawSQL {
					d.buckets["datetime"] = append(d.buckets["datetime"], eavRow{entityID, m.AttrID, opts.StoreID, val})
				} else {
					t, err := time.Parse("2006-01-02 15:04:05", val)
					if err != nil {
						t, _ = time.Parse("2006-01-02", val)
					}
					d.datetimeRows = append(d.datetimeRows, productEntity.ProductDatetime{
						AttributeID: m.AttrID, StoreID: opts.StoreID, EntityID: entityID, Value: t,
					})
				}
			}
		}
	}
	return d
}

// flushEAV writes buffered EAV rows to DB.
func flushEAV(db *gorm.DB, d *eavData, opts ImportOptions) error {
	if opts.RawSQL {
		return flushEAVRaw(db, d, opts.BatchSize)
	}
	return flushEAVGorm(db, d, opts.BatchSize)
}

func flushEAVRaw(db *gorm.DB, d *eavData, batchSize int) error {
	var wg sync.WaitGroup
	errs := make(chan error, len(eavTables))

	for bt, table := range eavTables {
		bucket := d.buckets[bt]
		if len(bucket) == 0 {
			continue
		}
		wg.Add(1)
		go func(table string, bucket []eavRow) {
			defer wg.Done()
			if err := rawBatchUpsert(db, table, bucket, batchSize); err != nil {
				errs <- err
			}
		}(table, bucket)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		return err
	}
	return nil
}

func flushEAVGorm(db *gorm.DB, d *eavData, batchSize int) error {
	upsertCols := clause.OnConflict{
		Columns:   []clause.Column{{Name: "entity_id"}, {Name: "attribute_id"}, {Name: "store_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"value"}),
	}

	type job struct {
		name string
		fn   func() error
	}
	jobs := []job{
		{"varchar", func() error {
			if len(d.varcharRows) == 0 {
				return nil
			}
			return db.Clauses(upsertCols).CreateInBatches(d.varcharRows, batchSize).Error
		}},
		{"int", func() error {
			if len(d.intRows) == 0 {
				return nil
			}
			return db.Clauses(upsertCols).CreateInBatches(d.intRows, batchSize).Error
		}},
		{"decimal", func() error {
			if len(d.decimalRows) == 0 {
				return nil
			}
			return db.Clauses(upsertCols).CreateInBatches(d.decimalRows, batchSize).Error
		}},
		{"text", func() error {
			if len(d.textRows) == 0 {
				return nil
			}
			return db.Clauses(upsertCols).CreateInBatches(d.textRows, batchSize).Error
		}},
		{"datetime", func() error {
			if len(d.datetimeRows) == 0 {
				return nil
			}
			return db.Clauses(upsertCols).CreateInBatches(d.datetimeRows, batchSize).Error
		}},
	}

	var wg sync.WaitGroup
	errs := make(chan error, len(jobs))
	for _, j := range jobs {
		wg.Add(1)
		go func(j job) {
			defer wg.Done()
			if err := j.fn(); err != nil {
				errs <- fmt.Errorf("%s: %w", j.name, err)
			}
		}(j)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		return err
	}
	return nil
}

func rawBatchUpsert(db *gorm.DB, table string, rows []eavRow, batchSize int) error {
	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		chunk := rows[i:end]

		var b strings.Builder
		b.Grow(len(chunk) * 60)
		b.WriteString("INSERT INTO ")
		b.WriteString(table)
		b.WriteString(" (entity_id, attribute_id, store_id, value) VALUES ")

		args := make([]interface{}, 0, len(chunk)*4)
		for j, r := range chunk {
			if j > 0 {
				b.WriteByte(',')
			}
			b.WriteString("(?,?,?,?)")
			args = append(args, r.EntityID, r.AttributeID, r.StoreID, r.Value)
		}
		b.WriteString(" ON CONFLICT(entity_id, attribute_id, store_id) DO UPDATE SET value=excluded.value")

		if err := db.Exec(b.String(), args...).Error; err != nil {
			return err
		}
	}
	return nil
}
