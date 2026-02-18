package servicetest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	entity "magento.GO/model/entity"
	productEntity "magento.GO/model/entity/product"
	productService "magento.GO/service/product"
)

func importDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Use a temp file DB so multiple goroutine connections see the same tables
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("import_test_%s_%d.db", t.Name(), time.Now().UnixNano()))
	t.Cleanup(func() { os.Remove(tmpFile) })
	db, err := gorm.Open(sqlite.Open(tmpFile), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// SQLite performance pragmas (mirrors MySQL's batched-write behavior)
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA synchronous=OFF")
	db.Exec("PRAGMA busy_timeout=5000")

	if err := db.AutoMigrate(
		&productEntity.Product{},
		&productEntity.ProductVarchar{},
		&productEntity.ProductInt{},
		&productEntity.ProductDecimal{},
		&productEntity.ProductText{},
		&productEntity.ProductDatetime{},
		&productEntity.StockItem{},
		&productEntity.ProductMediaGallery{},
		&productEntity.ProductIndexPrice{},
		&entity.EavAttribute{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Create unique indexes matching Magento's real schema for ON CONFLICT upsert
	for _, tbl := range []string{
		"catalog_product_entity_varchar",
		"catalog_product_entity_int",
		"catalog_product_entity_decimal",
		"catalog_product_entity_text",
		"catalog_product_entity_datetime",
	} {
		db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_" + tbl + "_unq ON " + tbl + " (entity_id, attribute_id, store_id)")
	}
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_stock_unq ON cataloginventory_stock_item (product_id, stock_id)")
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_price_unq ON catalog_product_index_price (entity_id, customer_group_id, website_id)")
	return db
}

func seedAttributes(t *testing.T, db *gorm.DB) {
	t.Helper()
	attrs := []entity.EavAttribute{
		{AttributeID: 73, EntityTypeID: 4, AttributeCode: "name", BackendType: "varchar"},
		{AttributeID: 74, EntityTypeID: 4, AttributeCode: "description", BackendType: "text"},
		{AttributeID: 75, EntityTypeID: 4, AttributeCode: "price", BackendType: "decimal"},
		{AttributeID: 76, EntityTypeID: 4, AttributeCode: "status", BackendType: "int"},
		{AttributeID: 77, EntityTypeID: 4, AttributeCode: "special_from_date", BackendType: "datetime"},
		{AttributeID: 78, EntityTypeID: 4, AttributeCode: "url_key", BackendType: "varchar"},
		{AttributeID: 79, EntityTypeID: 4, AttributeCode: "weight", BackendType: "decimal"},
		// A static attribute -- should be skipped during EAV writes
		{AttributeID: 80, EntityTypeID: 4, AttributeCode: "sku", BackendType: "static"},
	}
	if err := db.Create(&attrs).Error; err != nil {
		t.Fatalf("seed attributes: %v", err)
	}
}

func TestImport_NewProducts_AllTypes(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,name,description,price,status,special_from_date,type_id\n" +
		"SKU-001,Widget A,A nice widget,19.99,1,2026-01-15,simple\n" +
		"SKU-002,Widget B,Another widget,29.50,2,2026-06-01 10:00:00,configurable\n"

	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.Created != 2 {
		t.Errorf("Created = %d, want 2", res.Created)
	}
	if res.Updated != 0 {
		t.Errorf("Updated = %d, want 0", res.Updated)
	}
	if res.Skipped != 0 {
		t.Errorf("Skipped = %d, want 0", res.Skipped)
	}

	// Verify entity rows
	var products []productEntity.Product
	db.Order("sku").Find(&products)
	if len(products) != 2 {
		t.Fatalf("product count = %d, want 2", len(products))
	}
	if products[0].SKU != "SKU-001" || products[0].TypeID != "simple" {
		t.Errorf("product[0] = {SKU:%s TypeID:%s}, want {SKU-001, simple}", products[0].SKU, products[0].TypeID)
	}
	if products[1].SKU != "SKU-002" || products[1].TypeID != "configurable" {
		t.Errorf("product[1] = {SKU:%s TypeID:%s}, want {SKU-002, configurable}", products[1].SKU, products[1].TypeID)
	}

	// Verify varchar EAV rows (name column)
	if res.EAVCounts["varchar"] != 2 {
		t.Errorf("varchar count = %d, want 2", res.EAVCounts["varchar"])
	}
	var varchars []productEntity.ProductVarchar
	db.Where("attribute_id = ?", 73).Order("entity_id").Find(&varchars)
	if len(varchars) != 2 {
		t.Fatalf("varchar rows = %d, want 2", len(varchars))
	}
	if varchars[0].Value != "Widget A" {
		t.Errorf("varchar[0].Value = %q, want Widget A", varchars[0].Value)
	}

	// Verify text EAV rows (description)
	if res.EAVCounts["text"] != 2 {
		t.Errorf("text count = %d, want 2", res.EAVCounts["text"])
	}

	// Verify decimal EAV rows (price)
	if res.EAVCounts["decimal"] != 2 {
		t.Errorf("decimal count = %d, want 2", res.EAVCounts["decimal"])
	}
	var decimals []productEntity.ProductDecimal
	db.Where("attribute_id = ?", 75).Order("entity_id").Find(&decimals)
	if len(decimals) != 2 {
		t.Fatalf("decimal rows = %d, want 2", len(decimals))
	}
	if decimals[0].Value != 19.99 {
		t.Errorf("decimal[0].Value = %f, want 19.99", decimals[0].Value)
	}

	// Verify int EAV rows (status)
	if res.EAVCounts["int"] != 2 {
		t.Errorf("int count = %d, want 2", res.EAVCounts["int"])
	}
	var ints []productEntity.ProductInt
	db.Where("attribute_id = ?", 76).Order("entity_id").Find(&ints)
	if len(ints) != 2 {
		t.Fatalf("int rows = %d, want 2", len(ints))
	}
	if ints[0].Value != 1 {
		t.Errorf("int[0].Value = %d, want 1", ints[0].Value)
	}
	if ints[1].Value != 2 {
		t.Errorf("int[1].Value = %d, want 2", ints[1].Value)
	}

	// Verify datetime EAV rows (special_from_date)
	if res.EAVCounts["datetime"] != 2 {
		t.Errorf("datetime count = %d, want 2", res.EAVCounts["datetime"])
	}
	var datetimes []productEntity.ProductDatetime
	db.Where("attribute_id = ?", 77).Order("entity_id").Find(&datetimes)
	if len(datetimes) != 2 {
		t.Fatalf("datetime rows = %d, want 2", len(datetimes))
	}
	if datetimes[0].Value.Day() != 15 || datetimes[0].Value.Month() != 1 {
		t.Errorf("datetime[0] = %v, want 2026-01-15", datetimes[0].Value)
	}

	if len(res.Warnings) != 0 {
		t.Errorf("unexpected warnings: %v", res.Warnings)
	}
}

func TestImport_UpdateExistingProduct(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	// Pre-create a product
	existing := productEntity.Product{SKU: "SKU-EXIST", AttributeSetID: 4, TypeID: "simple"}
	db.Create(&existing)
	// Pre-create a varchar attribute value
	db.Create(&productEntity.ProductVarchar{
		AttributeID: 73, StoreID: 0, EntityID: existing.EntityID, Value: "Old Name",
	})

	csv := "sku,name,price\nSKU-EXIST,New Name,49.99\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.Created != 0 {
		t.Errorf("Created = %d, want 0", res.Created)
	}
	if res.Updated != 1 {
		t.Errorf("Updated = %d, want 1", res.Updated)
	}

	// Verify the name was updated (upsert)
	var v productEntity.ProductVarchar
	db.Where("entity_id = ? AND attribute_id = ? AND store_id = ?", existing.EntityID, 73, 0).First(&v)
	if v.Value != "New Name" {
		t.Errorf("varchar value = %q, want New Name", v.Value)
	}

	// No new entity row should be created
	var count int64
	db.Model(&productEntity.Product{}).Count(&count)
	if count != 1 {
		t.Errorf("product count = %d, want 1", count)
	}
}

func TestImport_MixedCreateAndUpdate(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	existing := productEntity.Product{SKU: "OLD-SKU", AttributeSetID: 4, TypeID: "simple"}
	db.Create(&existing)

	csv := "sku,name,status\n" +
		"OLD-SKU,Updated Product,1\n" +
		"NEW-SKU,Brand New,2\n"

	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.Created != 1 {
		t.Errorf("Created = %d, want 1", res.Created)
	}
	if res.Updated != 1 {
		t.Errorf("Updated = %d, want 1", res.Updated)
	}
	if res.TotalRows != 2 {
		t.Errorf("TotalRows = %d, want 2", res.TotalRows)
	}

	var products []productEntity.Product
	db.Order("sku").Find(&products)
	if len(products) != 2 {
		t.Fatalf("product count = %d, want 2", len(products))
	}
}

func TestImport_SkipsEmptySKU(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,name\n,No SKU\nGOOD-SKU,Has SKU\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", res.Skipped)
	}
	if res.Created != 1 {
		t.Errorf("Created = %d, want 1", res.Created)
	}
}

func TestImport_UnknownColumnWarning(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,name,bogus_column\nSKU-1,Test,value\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	found := false
	for _, w := range res.Warnings {
		if strings.Contains(w, "bogus_column") {
			found = true
		}
	}
	if !found {
		t.Error("expected warning about bogus_column")
	}
}

func TestImport_InvalidValueWarnings(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,status,price,special_from_date\nSKU-BAD,not_a_number,abc,not-a-date\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	// Should have 3 warnings: invalid int, invalid decimal, invalid datetime
	if len(res.Warnings) != 3 {
		t.Errorf("warnings = %d, want 3: %v", len(res.Warnings), res.Warnings)
	}
	if res.EAVCounts["int"] != 0 {
		t.Errorf("int count = %d, want 0 (invalid value should be skipped)", res.EAVCounts["int"])
	}
	if res.EAVCounts["decimal"] != 0 {
		t.Errorf("decimal count = %d, want 0", res.EAVCounts["decimal"])
	}
	if res.EAVCounts["datetime"] != 0 {
		t.Errorf("datetime count = %d, want 0", res.EAVCounts["datetime"])
	}
}

func TestImport_NoSKUColumn(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "name,price\nTest,9.99\n"
	_, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err == nil {
		t.Fatal("expected error for missing sku column")
	}
	if !strings.Contains(err.Error(), "sku") {
		t.Errorf("error = %q, want mention of 'sku'", err.Error())
	}
}

func TestImport_AttributeSetFromCSV(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,attribute_set_id,type_id\nSKU-CUSTOM,10,bundle\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.Created != 1 {
		t.Fatalf("Created = %d, want 1", res.Created)
	}

	var p productEntity.Product
	db.Where("sku = ?", "SKU-CUSTOM").First(&p)
	if p.AttributeSetID != 10 {
		t.Errorf("AttributeSetID = %d, want 10", p.AttributeSetID)
	}
	if p.TypeID != "bundle" {
		t.Errorf("TypeID = %q, want bundle", p.TypeID)
	}
}

func TestImport_StoreIDOption(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,name\nSKU-STORE,Store Name\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{StoreID: 5})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.Created != 1 {
		t.Fatalf("Created = %d, want 1", res.Created)
	}

	var v productEntity.ProductVarchar
	db.Where("attribute_id = ? AND store_id = ?", 73, 5).First(&v)
	if v.Value != "Store Name" {
		t.Errorf("varchar value = %q, want Store Name", v.Value)
	}
	if v.StoreID != 5 {
		t.Errorf("StoreID = %d, want 5", v.StoreID)
	}
}

func TestImport_MultipleAttributes_SameProduct(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,name,url_key,price,weight,description,status\n" +
		"SKU-FULL,Full Product,full-product,99.95,2.5,Long description text,1\n"

	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	// 2 varchar (name + url_key), 1 text (description), 2 decimal (price + weight), 1 int (status)
	if res.EAVCounts["varchar"] != 2 {
		t.Errorf("varchar = %d, want 2", res.EAVCounts["varchar"])
	}
	if res.EAVCounts["text"] != 1 {
		t.Errorf("text = %d, want 1", res.EAVCounts["text"])
	}
	if res.EAVCounts["decimal"] != 2 {
		t.Errorf("decimal = %d, want 2", res.EAVCounts["decimal"])
	}
	if res.EAVCounts["int"] != 1 {
		t.Errorf("int = %d, want 1", res.EAVCounts["int"])
	}

	// Verify actual DB rows
	var varchars []productEntity.ProductVarchar
	db.Find(&varchars)
	if len(varchars) != 2 {
		t.Errorf("varchar DB rows = %d, want 2", len(varchars))
	}
}

func seedPerfAttributes(t *testing.T, db *gorm.DB) {
	t.Helper()
	var attrs []entity.EavAttribute
	id := uint16(100)
	// 20 varchar
	for i := 0; i < 20; i++ {
		attrs = append(attrs, entity.EavAttribute{
			AttributeID: id, EntityTypeID: 4,
			AttributeCode: fmt.Sprintf("varchar_attr_%d", i), BackendType: "varchar",
		})
		id++
	}
	// 10 int
	for i := 0; i < 10; i++ {
		attrs = append(attrs, entity.EavAttribute{
			AttributeID: id, EntityTypeID: 4,
			AttributeCode: fmt.Sprintf("int_attr_%d", i), BackendType: "int",
		})
		id++
	}
	// 10 decimal
	for i := 0; i < 10; i++ {
		attrs = append(attrs, entity.EavAttribute{
			AttributeID: id, EntityTypeID: 4,
			AttributeCode: fmt.Sprintf("decimal_attr_%d", i), BackendType: "decimal",
		})
		id++
	}
	// 5 text
	for i := 0; i < 5; i++ {
		attrs = append(attrs, entity.EavAttribute{
			AttributeID: id, EntityTypeID: 4,
			AttributeCode: fmt.Sprintf("text_attr_%d", i), BackendType: "text",
		})
		id++
	}
	// 5 datetime
	for i := 0; i < 5; i++ {
		attrs = append(attrs, entity.EavAttribute{
			AttributeID: id, EntityTypeID: 4,
			AttributeCode: fmt.Sprintf("datetime_attr_%d", i), BackendType: "datetime",
		})
		id++
	}
	if err := db.Create(&attrs).Error; err != nil {
		t.Fatalf("seed perf attributes: %v", err)
	}
}

func buildPerfCSV(numProducts int) string {
	var header []string
	header = append(header, "sku")
	for i := 0; i < 20; i++ {
		header = append(header, fmt.Sprintf("varchar_attr_%d", i))
	}
	for i := 0; i < 10; i++ {
		header = append(header, fmt.Sprintf("int_attr_%d", i))
	}
	for i := 0; i < 10; i++ {
		header = append(header, fmt.Sprintf("decimal_attr_%d", i))
	}
	for i := 0; i < 5; i++ {
		header = append(header, fmt.Sprintf("text_attr_%d", i))
	}
	for i := 0; i < 5; i++ {
		header = append(header, fmt.Sprintf("datetime_attr_%d", i))
	}

	var b strings.Builder
	b.WriteString(strings.Join(header, ","))
	b.WriteByte('\n')

	for p := 0; p < numProducts; p++ {
		var row []string
		row = append(row, fmt.Sprintf("PERF-SKU-%06d", p))
		for i := 0; i < 20; i++ {
			row = append(row, fmt.Sprintf("varchar value %d for product %d", i, p))
		}
		for i := 0; i < 10; i++ {
			row = append(row, fmt.Sprintf("%d", (p*10)+i))
		}
		for i := 0; i < 10; i++ {
			row = append(row, fmt.Sprintf("%.2f", float64(p)*1.5+float64(i)*0.1))
		}
		for i := 0; i < 5; i++ {
			row = append(row, fmt.Sprintf("Long text content for attribute %d of product %d with extra padding", i, p))
		}
		for i := 0; i < 5; i++ {
			row = append(row, fmt.Sprintf("2026-%02d-%02d 10:00:00", (i%12)+1, (p%28)+1))
		}
		b.WriteString(strings.Join(row, ","))
		b.WriteByte('\n')
	}
	return b.String()
}

func runPerfImport(t *testing.T, label string, rawSQL bool, numProducts int) *productService.ImportResult {
	t.Helper()
	db := importDB(t)
	seedPerfAttributes(t, db)
	csv := buildPerfCSV(numProducts)
	expectedEAV := numProducts * 50

	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{
		BatchSize: 500,
		RawSQL:    rawSQL,
	})
	if err != nil {
		t.Fatalf("%s ImportProducts: %v", label, err)
	}
	if res.Created != numProducts {
		t.Errorf("%s Created = %d, want %d", label, res.Created, numProducts)
	}
	totalEAV := res.EAVCounts["varchar"] + res.EAVCounts["int"] + res.EAVCounts["decimal"] + res.EAVCounts["text"] + res.EAVCounts["datetime"]
	if totalEAV != expectedEAV {
		t.Errorf("%s total EAV = %d, want %d", label, totalEAV, expectedEAV)
	}
	t.Logf(`
=== %s ===
Products:       %d
EAV rows:       %d (varchar=%d int=%d decimal=%d text=%d datetime=%d)
Total time:     %s
  - Processing: %s
  - DB upsert:  %s
Rate:           %.0f products/sec | %.0f products/min | %.0f EAV rows/sec`,
		label, res.Created, totalEAV,
		res.EAVCounts["varchar"], res.EAVCounts["int"], res.EAVCounts["decimal"],
		res.EAVCounts["text"], res.EAVCounts["datetime"],
		res.TotalTime, res.ProcessTime, res.DBTime,
		float64(res.Created)/res.TotalTime.Seconds(),
		float64(res.Created)/res.TotalTime.Minutes(),
		float64(totalEAV)/res.TotalTime.Seconds())
	return res
}

func TestImport_Perf_1000Products_50Attributes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping perf test in short mode")
	}
	gormRes := runPerfImport(t, "GORM ORM", false, 1000)
	rawRes := runPerfImport(t, "Raw SQL", true, 1000)

	speedup := gormRes.DBTime.Seconds() / rawRes.DBTime.Seconds()
	t.Logf(`
=== Comparison ===
GORM DB time:   %s
Raw SQL DB time: %s
Speedup:        %.2fx
==================`, gormRes.DBTime, rawRes.DBTime, speedup)
}

func TestImport_Perf_100kProducts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 100k perf test in short mode")
	}
	rawRes := runPerfImport(t, "Raw SQL 100k", true, 100000)

	t.Logf(`
=== 100k Summary ===
Products:       %d
EAV rows:       %d
Total time:     %s
Rate:           %.0f products/sec | %.0f products/min
====================`,
		rawRes.Created,
		rawRes.EAVCounts["varchar"]+rawRes.EAVCounts["int"]+rawRes.EAVCounts["decimal"]+rawRes.EAVCounts["text"]+rawRes.EAVCounts["datetime"],
		rawRes.TotalTime,
		float64(rawRes.Created)/rawRes.TotalTime.Seconds(),
		float64(rawRes.Created)/rawRes.TotalTime.Minutes())
}

// ---------- Stock sub-importer tests ----------

func TestImport_Stock(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,name,qty,is_in_stock\nSTOCK-1,Widget,100.5,1\nSTOCK-2,Gadget,0,0\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["stock"] != 2 {
		t.Errorf("stock count = %d, want 2", res.EAVCounts["stock"])
	}

	var items []productEntity.StockItem
	db.Order("product_id").Find(&items)
	if len(items) != 2 {
		t.Fatalf("stock rows = %d, want 2", len(items))
	}
	if items[0].Qty != 100.5 || items[0].IsInStock != 1 {
		t.Errorf("stock[0] qty=%f is_in_stock=%d, want 100.5/1", items[0].Qty, items[0].IsInStock)
	}
	if items[1].Qty != 0 || items[1].IsInStock != 0 {
		t.Errorf("stock[1] qty=%f is_in_stock=%d, want 0/0", items[1].Qty, items[1].IsInStock)
	}
}

func TestImport_StockUpdate(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	// Create product + stock
	csv1 := "sku,qty,is_in_stock\nSTK-UPD,10,1\n"
	_, err := productService.ImportProducts(db, strings.NewReader(csv1), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("first import: %v", err)
	}

	// Update stock via second import
	csv2 := "sku,qty,is_in_stock\nSTK-UPD,0,0\n"
	_, err = productService.ImportProducts(db, strings.NewReader(csv2), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("second import: %v", err)
	}

	var item productEntity.StockItem
	db.Where("product_id = (SELECT entity_id FROM catalog_product_entity WHERE sku = ?)", "STK-UPD").First(&item)
	if item.Qty != 0 || item.IsInStock != 0 {
		t.Errorf("updated stock qty=%f is_in_stock=%d, want 0/0", item.Qty, item.IsInStock)
	}
}

// ---------- Gallery sub-importer tests ----------

func TestImport_Gallery(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,name,image\nGAL-1,Product A,/m/y/image1.jpg\nGAL-2,Product B,/m/y/image2.jpg|/m/y/image3.jpg\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	// GAL-1: 1 image, GAL-2: 2 images (pipe-separated)
	if res.EAVCounts["gallery"] != 3 {
		t.Errorf("gallery count = %d, want 3", res.EAVCounts["gallery"])
	}

	var galleries []productEntity.ProductMediaGallery
	db.Order("value").Find(&galleries)
	if len(galleries) != 3 {
		t.Fatalf("gallery rows = %d, want 3", len(galleries))
	}
	if galleries[0].Value != "/m/y/image1.jpg" {
		t.Errorf("gallery[0] = %q, want /m/y/image1.jpg", galleries[0].Value)
	}
	if galleries[0].MediaType != "image" {
		t.Errorf("gallery[0].MediaType = %q, want image", galleries[0].MediaType)
	}
}

func TestImport_GalleryDedup(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	// Same image in image and small_image columns should only create one entry
	csv := "sku,image,small_image\nDEDUP-1,/m/y/same.jpg,/m/y/same.jpg\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["gallery"] != 1 {
		t.Errorf("gallery count = %d, want 1 (deduped)", res.EAVCounts["gallery"])
	}
}

// ---------- Price index sub-importer tests ----------

func TestImport_PriceIndex(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,name,price_index,final_price\nPRICE-1,Product,99.99,89.99\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["price_index"] != 1 {
		t.Errorf("price_index count = %d, want 1", res.EAVCounts["price_index"])
	}

	var prices []productEntity.ProductIndexPrice
	db.Find(&prices)
	if len(prices) != 1 {
		t.Fatalf("price rows = %d, want 1", len(prices))
	}
	if prices[0].Price != 99.99 {
		t.Errorf("price = %f, want 99.99", prices[0].Price)
	}
	if prices[0].FinalPrice != 89.99 {
		t.Errorf("final_price = %f, want 89.99", prices[0].FinalPrice)
	}
}

func TestImport_PriceIndexUpdate(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv1 := "sku,price_index\nPRICE-UPD,50.00\n"
	_, err := productService.ImportProducts(db, strings.NewReader(csv1), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("first import: %v", err)
	}

	csv2 := "sku,price_index,final_price\nPRICE-UPD,75.00,69.00\n"
	_, err = productService.ImportProducts(db, strings.NewReader(csv2), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("second import: %v", err)
	}

	var price productEntity.ProductIndexPrice
	db.First(&price)
	if price.Price != 75.00 {
		t.Errorf("updated price = %f, want 75.00", price.Price)
	}
	if price.FinalPrice != 69.00 {
		t.Errorf("updated final_price = %f, want 69.00", price.FinalPrice)
	}
}

// ---------- ImportStockJSON (API service) tests ----------

func ptrF64(v float64) *float64 { return &v }
func ptrU16(v uint16) *uint16   { return &v }

func seedProduct(t *testing.T, db *gorm.DB, sku string) uint {
	t.Helper()
	p := productEntity.Product{SKU: sku, AttributeSetID: 4, TypeID: "simple"}
	if err := db.Create(&p).Error; err != nil {
		t.Fatalf("seed product %s: %v", sku, err)
	}
	return p.EntityID
}

func TestImportStockJSON_Basic(t *testing.T) {
	db := importDB(t)
	seedProduct(t, db, "JSON-STK-1")
	seedProduct(t, db, "JSON-STK-2")

	items := []productService.StockItemInput{
		{SKU: "JSON-STK-1", Qty: ptrF64(50), IsInStock: ptrU16(1)},
		{SKU: "JSON-STK-2", Qty: ptrF64(0), IsInStock: ptrU16(0)},
	}

	res, err := productService.ImportStockJSON(db, items, 0)
	if err != nil {
		t.Fatalf("ImportStockJSON: %v", err)
	}
	if res.Imported != 2 {
		t.Errorf("Imported = %d, want 2", res.Imported)
	}
	if res.Skipped != 0 {
		t.Errorf("Skipped = %d, want 0", res.Skipped)
	}

	var rows []productEntity.StockItem
	db.Order("product_id").Find(&rows)
	if len(rows) != 2 {
		t.Fatalf("stock rows = %d, want 2", len(rows))
	}
	if rows[0].Qty != 50 || rows[0].IsInStock != 1 {
		t.Errorf("row[0] qty=%f is_in_stock=%d, want 50/1", rows[0].Qty, rows[0].IsInStock)
	}
	if rows[1].Qty != 0 || rows[1].IsInStock != 0 {
		t.Errorf("row[1] qty=%f is_in_stock=%d, want 0/0", rows[1].Qty, rows[1].IsInStock)
	}
}

func TestImportStockJSON_Upsert(t *testing.T) {
	db := importDB(t)
	seedProduct(t, db, "JSON-UPD")

	// First import
	_, err := productService.ImportStockJSON(db, []productService.StockItemInput{
		{SKU: "JSON-UPD", Qty: ptrF64(100), IsInStock: ptrU16(1)},
	}, 0)
	if err != nil {
		t.Fatalf("first import: %v", err)
	}

	// Second import updates the same product
	res, err := productService.ImportStockJSON(db, []productService.StockItemInput{
		{SKU: "JSON-UPD", Qty: ptrF64(0), IsInStock: ptrU16(0)},
	}, 0)
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if res.Imported != 1 {
		t.Errorf("Imported = %d, want 1", res.Imported)
	}

	var item productEntity.StockItem
	db.First(&item)
	if item.Qty != 0 {
		t.Errorf("qty = %f, want 0", item.Qty)
	}
	if item.IsInStock != 0 {
		t.Errorf("is_in_stock = %d, want 0", item.IsInStock)
	}
}

func TestImportStockJSON_UnknownSKU(t *testing.T) {
	db := importDB(t)
	seedProduct(t, db, "JSON-EXISTS")

	items := []productService.StockItemInput{
		{SKU: "JSON-EXISTS", Qty: ptrF64(10), IsInStock: ptrU16(1)},
		{SKU: "JSON-GHOST", Qty: ptrF64(5), IsInStock: ptrU16(1)},
	}
	res, err := productService.ImportStockJSON(db, items, 0)
	if err != nil {
		t.Fatalf("ImportStockJSON: %v", err)
	}
	if res.Imported != 1 {
		t.Errorf("Imported = %d, want 1", res.Imported)
	}
	if res.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", res.Skipped)
	}
	found := false
	for _, w := range res.Warnings {
		if strings.Contains(w, "JSON-GHOST") && strings.Contains(w, "not found") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning about JSON-GHOST not found, got: %v", res.Warnings)
	}
}

func TestImportStockJSON_EmptySKU(t *testing.T) {
	db := importDB(t)

	res, err := productService.ImportStockJSON(db, []productService.StockItemInput{
		{SKU: "", Qty: ptrF64(10)},
	}, 0)
	if err != nil {
		t.Fatalf("ImportStockJSON: %v", err)
	}
	if res.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", res.Skipped)
	}
	if res.Imported != 0 {
		t.Errorf("Imported = %d, want 0", res.Imported)
	}
}

func TestImportStockJSON_EmptyItems(t *testing.T) {
	db := importDB(t)

	res, err := productService.ImportStockJSON(db, []productService.StockItemInput{}, 0)
	if err != nil {
		t.Fatalf("ImportStockJSON: %v", err)
	}
	if res.Imported != 0 || res.Skipped != 0 {
		t.Errorf("expected 0/0, got imported=%d skipped=%d", res.Imported, res.Skipped)
	}
}

func TestImportStockJSON_OptionalFields(t *testing.T) {
	db := importDB(t)
	seedProduct(t, db, "JSON-OPT")

	// Only set qty, leave other fields nil to get defaults
	res, err := productService.ImportStockJSON(db, []productService.StockItemInput{
		{SKU: "JSON-OPT", Qty: ptrF64(25)},
	}, 0)
	if err != nil {
		t.Fatalf("ImportStockJSON: %v", err)
	}
	if res.Imported != 1 {
		t.Fatalf("Imported = %d, want 1", res.Imported)
	}

	var item productEntity.StockItem
	db.First(&item)
	if item.Qty != 25 {
		t.Errorf("qty = %f, want 25", item.Qty)
	}
	if item.IsInStock != 1 {
		t.Errorf("is_in_stock = %d, want 1 (default)", item.IsInStock)
	}
	if item.ManageStock != 1 {
		t.Errorf("manage_stock = %d, want 1 (default)", item.ManageStock)
	}
}

func TestImportStockJSON_AllFields(t *testing.T) {
	db := importDB(t)
	seedProduct(t, db, "JSON-ALL")

	res, err := productService.ImportStockJSON(db, []productService.StockItemInput{
		{
			SKU:         "JSON-ALL",
			Qty:         ptrF64(200),
			IsInStock:   ptrU16(1),
			ManageStock: ptrU16(0),
			MinQty:      ptrF64(5),
			MinSaleQty:  ptrF64(2),
			MaxSaleQty:  ptrF64(50),
		},
	}, 0)
	if err != nil {
		t.Fatalf("ImportStockJSON: %v", err)
	}
	if res.Imported != 1 {
		t.Fatalf("Imported = %d, want 1", res.Imported)
	}

	var item productEntity.StockItem
	db.First(&item)
	if item.Qty != 200 {
		t.Errorf("qty = %f, want 200", item.Qty)
	}
	if item.ManageStock != 0 {
		t.Errorf("manage_stock = %d, want 0", item.ManageStock)
	}
	if item.MinQty != 5 {
		t.Errorf("min_qty = %f, want 5", item.MinQty)
	}
	if item.MinSaleQty != 2 {
		t.Errorf("min_sale_qty = %f, want 2", item.MinSaleQty)
	}
	if item.MaxSaleQty != 50 {
		t.Errorf("max_sale_qty = %f, want 50", item.MaxSaleQty)
	}
}

// ---------- Combined import test ----------

func TestImport_AllSubImporters(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,name,price,status,qty,is_in_stock,image,price_index,final_price\n" +
		"FULL-1,Full Product,29.99,1,50,1,/m/y/full.jpg,29.99,24.99\n"

	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}

	if res.Created != 1 {
		t.Errorf("Created = %d, want 1", res.Created)
	}
	if res.EAVCounts["varchar"] != 1 {
		t.Errorf("varchar = %d, want 1 (name)", res.EAVCounts["varchar"])
	}
	if res.EAVCounts["decimal"] != 1 {
		t.Errorf("decimal = %d, want 1 (price)", res.EAVCounts["decimal"])
	}
	if res.EAVCounts["int"] != 1 {
		t.Errorf("int = %d, want 1 (status)", res.EAVCounts["int"])
	}
	if res.EAVCounts["stock"] != 1 {
		t.Errorf("stock = %d, want 1", res.EAVCounts["stock"])
	}
	if res.EAVCounts["gallery"] != 1 {
		t.Errorf("gallery = %d, want 1", res.EAVCounts["gallery"])
	}
	if res.EAVCounts["price_index"] != 1 {
		t.Errorf("price_index = %d, want 1", res.EAVCounts["price_index"])
	}
}
