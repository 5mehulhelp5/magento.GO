package integration

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	productEntity "magento.GO/model/entity/product"
	productService "magento.GO/service/product"
)

func magentoTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	host := envOrDefault("MYSQL_HOST", "db")
	port := envOrDefault("MYSQL_PORT", "3306")
	user := envOrDefault("MYSQL_USER", "magento")
	pass := envOrDefault("MYSQL_PASS", "magento")
	name := envOrDefault("MYSQL_DB", "magento")

	dsn := user + ":" + pass + "@tcp(" + host + ":" + port + ")/" + name + "?charset=utf8mb4&parseTime=True&loc=Local"

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("cannot connect to Magento DB (%s:%s): %v — skipping integration test", host, port, err)
	}
	return db
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ---------- Products ----------

func TestMagentoDB_ProductCount(t *testing.T) {
	db := magentoTestDB(t)
	var count int64
	db.Table("catalog_product_entity").Count(&count)
	t.Logf("catalog_product_entity: %d products", count)
	if count == 0 {
		t.Error("expected at least 1 product in Magento DB")
	}
}

func TestMagentoDB_ProductTypes(t *testing.T) {
	db := magentoTestDB(t)

	type typeRow struct {
		TypeID string `gorm:"column:type_id"`
		Cnt    int64  `gorm:"column:cnt"`
	}
	var rows []typeRow
	db.Raw("SELECT type_id, COUNT(*) as cnt FROM catalog_product_entity GROUP BY type_id ORDER BY cnt DESC").Scan(&rows)

	for _, r := range rows {
		t.Logf("  %-20s %d", r.TypeID, r.Cnt)
	}
	if len(rows) == 0 {
		t.Error("expected at least 1 product type")
	}
}

// ---------- EAV Attributes ----------

func TestMagentoDB_EavAttributes(t *testing.T) {
	db := magentoTestDB(t)

	var count int64
	db.Table("eav_attribute").Where("entity_type_id = 4").Count(&count)
	t.Logf("Product EAV attributes (entity_type_id=4): %d", count)
	if count == 0 {
		t.Error("expected product EAV attributes")
	}
}

func TestMagentoDB_EavValueCounts(t *testing.T) {
	db := magentoTestDB(t)

	tables := []string{
		"catalog_product_entity_varchar",
		"catalog_product_entity_int",
		"catalog_product_entity_decimal",
		"catalog_product_entity_text",
		"catalog_product_entity_datetime",
	}
	for _, tbl := range tables {
		var count int64
		db.Table(tbl).Count(&count)
		t.Logf("  %-45s %d rows", tbl, count)
	}
}

// ---------- Categories ----------

func TestMagentoDB_CategoryCount(t *testing.T) {
	db := magentoTestDB(t)
	var count int64
	db.Table("catalog_category_entity").Count(&count)
	t.Logf("catalog_category_entity: %d categories", count)
	if count == 0 {
		t.Error("expected at least 1 category")
	}
}

// ---------- Stock ----------

func TestMagentoDB_StockItems(t *testing.T) {
	db := magentoTestDB(t)
	var count int64
	db.Table("cataloginventory_stock_item").Count(&count)
	t.Logf("cataloginventory_stock_item: %d rows", count)

	var inStock int64
	db.Table("cataloginventory_stock_item").Where("is_in_stock = 1 AND qty > 0").Count(&inStock)
	t.Logf("  in stock (qty > 0): %d", inStock)
}

// ---------- Orders ----------

func TestMagentoDB_OrderCount(t *testing.T) {
	db := magentoTestDB(t)
	var count int64
	db.Table("sales_order").Count(&count)
	t.Logf("sales_order: %d orders", count)
}

// ---------- Stores ----------

func TestMagentoDB_Stores(t *testing.T) {
	db := magentoTestDB(t)

	type storeRow struct {
		StoreID uint   `gorm:"column:store_id"`
		Code    string `gorm:"column:code"`
		Name    string `gorm:"column:name"`
	}
	var stores []storeRow
	db.Raw("SELECT store_id, code, name FROM store").Scan(&stores)

	for _, s := range stores {
		t.Logf("  store %d: %s (%s)", s.StoreID, s.Name, s.Code)
	}
	if len(stores) == 0 {
		t.Error("expected at least 1 store")
	}
}

// ---------- Admin / Auth ----------

func TestMagentoDB_AdminUsers(t *testing.T) {
	db := magentoTestDB(t)
	var count int64
	db.Table("admin_user").Count(&count)
	t.Logf("admin_user: %d users", count)
	if count == 0 {
		t.Error("expected at least 1 admin user")
	}
}

func TestMagentoDB_OauthTokens(t *testing.T) {
	db := magentoTestDB(t)
	var count int64
	db.Table("oauth_token").Count(&count)
	t.Logf("oauth_token: %d tokens", count)

	var active int64
	db.Table("oauth_token").Where("revoked = 0").Count(&active)
	t.Logf("  active (not revoked): %d", active)
}

// ---------- Import Performance Test (with teardown) ----------

const perfTestSKUPrefix = "GOGENTO-PERF-TEST-"

type perfAttrRow struct {
	AttributeID   uint   `gorm:"column:attribute_id"`
	AttributeCode string `gorm:"column:attribute_code"`
	BackendType   string `gorm:"column:backend_type"`
}

func TestMagentoDB_ImportPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping import perf test in short mode")
	}

	db := magentoTestDB(t)

	// Log schema info
	schema := productService.DetectSchema(db)
	t.Logf("DB schema: %s, Build: enterprise=%v", schema, productEntity.IsEnterprise)

	// Get real attributes from Magento
	var attrs []perfAttrRow
	db.Raw(`SELECT attribute_id, attribute_code, backend_type 
		FROM eav_attribute 
		WHERE entity_type_id = 4 AND backend_type IN ('varchar','int','decimal','text','datetime')
		LIMIT 50`).Scan(&attrs)

	if len(attrs) < 10 {
		t.Skipf("not enough EAV attributes found (%d), skipping perf test", len(attrs))
	}

	// Build CSV with real attribute codes
	numProducts := 1000
	csv := buildPerfCSVWithAttrs(numProducts, attrs)

	// Clean up before (in case previous run failed)
	cleanupPerfTestProducts(t, db)

	// Register cleanup for after
	t.Cleanup(func() {
		cleanupPerfTestProducts(t, db)
	})

	// Run import
	t.Logf("Importing %d products with %d attributes each...", numProducts, len(attrs))
	start := time.Now()

	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{
		BatchSize: 500,
		RawSQL:    false, // Use GORM mode for MySQL compatibility
	})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}

	elapsed := time.Since(start)

	// Results
	totalEAV := res.EAVCounts["varchar"] + res.EAVCounts["int"] + res.EAVCounts["decimal"] + res.EAVCounts["text"] + res.EAVCounts["datetime"]

	t.Logf(`
=== MySQL Import Performance ===
Products:       %d created, %d updated
EAV rows:       %d (varchar=%d int=%d decimal=%d text=%d datetime=%d)
Total time:     %s
  - Processing: %s
  - DB upsert:  %s
Rate:           %.0f products/sec | %.0f products/min
EAV rate:       %.0f rows/sec
================================`,
		res.Created, res.Updated, totalEAV,
		res.EAVCounts["varchar"], res.EAVCounts["int"], res.EAVCounts["decimal"],
		res.EAVCounts["text"], res.EAVCounts["datetime"],
		elapsed, res.ProcessTime, res.DBTime,
		float64(res.Created)/elapsed.Seconds(),
		float64(res.Created)/elapsed.Minutes(),
		float64(totalEAV)/elapsed.Seconds())

	if res.Created != numProducts {
		t.Errorf("expected %d created, got %d", numProducts, res.Created)
	}
}

// TestMagentoDB_StockImportPerformance tests stock import on real Magento DB
func TestMagentoDB_StockImportPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stock import perf test in short mode")
	}

	db := magentoTestDB(t)

	// Get real SKUs from database
	type skuRow struct {
		SKU string `gorm:"column:sku"`
	}
	var skus []skuRow
	db.Raw("SELECT sku FROM catalog_product_entity WHERE type_id = 'simple' LIMIT 1000").Scan(&skus)

	if len(skus) < 100 {
		t.Skipf("not enough products found (%d), skipping stock perf test", len(skus))
	}

	// Build stock import items
	var items []productService.StockItemInput
	for i, s := range skus {
		qty := float64(i*10 + 100)
		inStock := uint16(1)
		items = append(items, productService.StockItemInput{
			SKU:       s.SKU,
			Qty:       &qty,
			IsInStock: &inStock,
		})
	}

	t.Logf("Importing stock for %d products...", len(items))
	start := time.Now()

	result, err := productService.ImportStockJSON(db, items, 500)
	if err != nil {
		t.Fatalf("ImportStockJSON: %v", err)
	}

	elapsed := time.Since(start)

	t.Logf(`
=== MySQL Stock Import Performance ===
Products:       %d imported, %d skipped
Warnings:       %d
Total time:     %s
Rate:           %.0f products/sec | %.0f products/min
=======================================`,
		result.Imported, result.Skipped,
		len(result.Warnings),
		elapsed,
		float64(result.Imported)/elapsed.Seconds(),
		float64(result.Imported)/elapsed.Minutes())

	if result.Imported != len(items) {
		t.Errorf("expected %d imported, got %d", len(items), result.Imported)
	}
}

func buildPerfCSVWithAttrs(numProducts int, attrs []perfAttrRow) string {
	var header []string
	header = append(header, "sku", "type_id", "attribute_set_id")

	for _, a := range attrs {
		header = append(header, a.AttributeCode)
	}

	var b strings.Builder
	b.WriteString(strings.Join(header, ","))
	b.WriteByte('\n')

	for p := 0; p < numProducts; p++ {
		var row []string
		row = append(row, fmt.Sprintf("%s%06d", perfTestSKUPrefix, p))
		row = append(row, "simple")
		row = append(row, "4") // Default attribute set

		for i, a := range attrs {
			switch a.BackendType {
			case "varchar":
				row = append(row, fmt.Sprintf("perf test varchar %d-%d", p, i))
			case "int":
				row = append(row, fmt.Sprintf("%d", p*10+i))
			case "decimal":
				row = append(row, fmt.Sprintf("%.2f", float64(p)+float64(i)*0.01))
			case "text":
				row = append(row, fmt.Sprintf("perf test text content for product %d attr %d", p, i))
			case "datetime":
				row = append(row, fmt.Sprintf("2026-%02d-%02d 10:00:00", (i%12)+1, (p%28)+1))
			default:
				row = append(row, "")
			}
		}
		b.WriteString(strings.Join(row, ","))
		b.WriteByte('\n')
	}
	return b.String()
}

func cleanupPerfTestProducts(t *testing.T, db *gorm.DB) {
	t.Helper()

	// Get entity IDs for test products
	var entityIDs []uint
	db.Raw("SELECT entity_id FROM catalog_product_entity WHERE sku LIKE ?", perfTestSKUPrefix+"%").Scan(&entityIDs)

	if len(entityIDs) == 0 {
		t.Log("No perf test products to clean up")
		return
	}

	t.Logf("Cleaning up %d perf test products...", len(entityIDs))

	// Delete from EAV tables
	eavTables := []string{
		"catalog_product_entity_varchar",
		"catalog_product_entity_int",
		"catalog_product_entity_decimal",
		"catalog_product_entity_text",
		"catalog_product_entity_datetime",
	}
	for _, tbl := range eavTables {
		db.Exec(fmt.Sprintf("DELETE FROM %s WHERE entity_id IN ?", tbl), entityIDs)
	}

	// Delete from stock
	db.Exec("DELETE FROM cataloginventory_stock_item WHERE product_id IN ?", entityIDs)

	// Delete from gallery
	db.Exec("DELETE FROM catalog_product_entity_media_gallery_value WHERE entity_id IN ?", entityIDs)
	db.Exec("DELETE FROM catalog_product_entity_media_gallery WHERE value_id NOT IN (SELECT value_id FROM catalog_product_entity_media_gallery_value)")

	// Delete from price index
	db.Exec("DELETE FROM catalog_product_index_price WHERE entity_id IN ?", entityIDs)

	// Delete from category links
	db.Exec("DELETE FROM catalog_category_product WHERE product_id IN ?", entityIDs)

	// Delete from URL rewrites
	db.Exec("DELETE FROM url_rewrite WHERE entity_type = 'product' AND entity_id IN ?", entityIDs)

	// Finally delete products
	result := db.Exec("DELETE FROM catalog_product_entity WHERE sku LIKE ?", perfTestSKUPrefix+"%")
	t.Logf("Deleted %d products from catalog_product_entity", result.RowsAffected)
}

