// Seeds a fresh MySQL database with a minimal, self-contained Magento-like
// schema and sample data, so GoGento's integration tests can run without a
// real Magento installation. Not part of the upstream project.
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	entity "magento.GO/model/entity"
	"magento.GO/model/entity/category"
	"magento.GO/model/entity/price"
	"magento.GO/model/entity/product"
)

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	host := envOrDefault("MYSQL_HOST", "127.0.0.1")
	port := envOrDefault("MYSQL_PORT", "3306")
	user := envOrDefault("MYSQL_USER", "magento")
	pass := envOrDefault("MYSQL_PASS", "magento")
	name := envOrDefault("MYSQL_DB", "magento")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, pass, host, port, name)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Warn),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		log.Fatalf("connect: %v", err)
	}

	log.Println("migrating schema...")
	if err := db.AutoMigrate(
		&entity.EavAttribute{},
		&entity.AdminUser{},
		&entity.OauthToken{},
		&product.Product{},
		&product.ProductVarchar{},
		&product.ProductInt{},
		&product.ProductDecimal{},
		&product.ProductText{},
		&product.ProductDatetime{},
		&product.StockItem{},
		&product.ProductMediaGallery{},
		&product.ProductMediaGalleryValueToEntity{},
		&product.ProductIndexPrice{},
		&product.ProductLink{},
		&product.ProductOption{},
		&product.ProductOptionTypeValue{},
		&product.DownloadableLink{},
		&product.DownloadableSample{},
		&product.ProductBundleOption{},
		&product.ProductBundleSelection{},
		&product.ProductSuperAttribute{},
		&product.ProductSuperLink{},
		&price.TierPrice{},
		&category.Category{},
		&category.CategoryProduct{},
		&category.CategoryInt{},
		&category.CategoryVarchar{},
		&category.CategoryText{},
	); err != nil {
		log.Fatalf("automigrate: %v", err)
	}

	// Tables with no corresponding GORM entity in the app.
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS store (
		store_id SMALLINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
		code VARCHAR(32) NOT NULL UNIQUE,
		website_id SMALLINT UNSIGNED NOT NULL DEFAULT 0,
		group_id SMALLINT UNSIGNED NOT NULL DEFAULT 0,
		name VARCHAR(255) NOT NULL,
		sort_order SMALLINT UNSIGNED NOT NULL DEFAULT 0,
		is_active SMALLINT UNSIGNED NOT NULL DEFAULT 0
	)`).Error; err != nil {
		log.Fatalf("create store: %v", err)
	}
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS sales_order (
		entity_id INT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
		increment_id VARCHAR(50),
		status VARCHAR(32),
		grand_total DECIMAL(20,4),
		created_at DATETIME
	)`).Error; err != nil {
		log.Fatalf("create sales_order: %v", err)
	}

	log.Println("seeding sample data...")

	// eav_attribute — one attribute per backend type, entity_type_id=4 (catalog_product)
	attrs := []entity.EavAttribute{
		{EntityTypeID: 4, AttributeCode: "name", BackendType: "varchar"},
		{EntityTypeID: 4, AttributeCode: "description", BackendType: "text"},
		{EntityTypeID: 4, AttributeCode: "short_description", BackendType: "text"},
		{EntityTypeID: 4, AttributeCode: "meta_title", BackendType: "varchar"},
		{EntityTypeID: 4, AttributeCode: "url_key", BackendType: "varchar"},
		{EntityTypeID: 4, AttributeCode: "color", BackendType: "int"},
		{EntityTypeID: 4, AttributeCode: "size", BackendType: "int"},
		{EntityTypeID: 4, AttributeCode: "status", BackendType: "int"},
		{EntityTypeID: 4, AttributeCode: "price", BackendType: "decimal"},
		{EntityTypeID: 4, AttributeCode: "weight", BackendType: "decimal"},
		{EntityTypeID: 4, AttributeCode: "special_price", BackendType: "decimal"},
		{EntityTypeID: 4, AttributeCode: "special_from_date", BackendType: "datetime"},
		{EntityTypeID: 4, AttributeCode: "special_to_date", BackendType: "datetime"},
	}
	for i := range attrs {
		if err := db.Where("attribute_code = ? AND entity_type_id = ?", attrs[i].AttributeCode, attrs[i].EntityTypeID).
			FirstOrCreate(&attrs[i]).Error; err != nil {
			log.Fatalf("seed eav_attribute %s: %v", attrs[i].AttributeCode, err)
		}
	}

	// categories
	root := category.Category{AttributeSetID: 3, ParentID: 0, Path: "1", Position: 0, Level: 0, ChildrenCount: 1}
	if err := db.Where("path = ?", root.Path).FirstOrCreate(&root).Error; err != nil {
		log.Fatalf("seed root category: %v", err)
	}
	cat := category.Category{AttributeSetID: 3, ParentID: root.EntityID, Path: fmt.Sprintf("1/%d", root.EntityID+1), Position: 1, Level: 1, ChildrenCount: 0}
	if err := db.Where("path = ?", cat.Path).FirstOrCreate(&cat).Error; err != nil {
		log.Fatalf("seed category: %v", err)
	}

	// products: enough "simple" products for the stock-import perf test (needs >=100)
	const numProducts = 150
	var count int64
	db.Model(&product.Product{}).Count(&count)
	if count < numProducts {
		for i := int(count); i < numProducts; i++ {
			p := product.Product{
				AttributeSetID: 4,
				TypeID:         "simple",
				SKU:            fmt.Sprintf("SAMPLE-SKU-%04d", i),
			}
			if err := db.Create(&p).Error; err != nil {
				log.Fatalf("seed product %d: %v", i, err)
			}
			db.Create(&product.ProductVarchar{AttributeID: uint16(attrs[0].AttributeID), StoreID: 0, EntityID: p.EntityID, Value: fmt.Sprintf("Sample Product %d", i)})
			db.Create(&product.ProductDecimal{AttributeID: uint16(attrs[8].AttributeID), StoreID: 0, EntityID: p.EntityID, Value: 9.99 + float64(i)})
			db.Create(&product.StockItem{ProductID: p.EntityID, StockID: 1, Qty: 100, IsInStock: 1, WebsiteID: 0})
			price := 9.99 + float64(i)
			db.Create(&product.ProductIndexPrice{EntityID: p.EntityID, CustomerGroupID: 0, WebsiteID: 0, Price: price, FinalPrice: price, MinPrice: price, MaxPrice: price})
			db.Create(&category.CategoryProduct{CategoryID: cat.EntityID, ProductID: p.EntityID, Position: i})
		}
	}

	// store
	if err := db.Exec(`INSERT IGNORE INTO store (store_id, code, website_id, group_id, name, sort_order, is_active) VALUES
		(0, 'admin', 0, 0, 'Admin', 0, 1),
		(1, 'default', 1, 1, 'Default Store View', 0, 1)`).Error; err != nil {
		log.Fatalf("seed store: %v", err)
	}

	// admin_user
	admin := entity.AdminUser{Username: strPtr("admin"), Email: strPtr("admin@example.com"), Firstname: strPtr("Admin"), Lastname: strPtr("User"), IsActive: 1}
	if err := db.Where("username = ?", "admin").FirstOrCreate(&admin).Error; err != nil {
		log.Fatalf("seed admin_user: %v", err)
	}

	// a couple of sample orders (created_at populated for completeness)
	now := time.Now().Format("2006-01-02 15:04:05")
	db.Exec(`INSERT INTO sales_order (increment_id, status, grand_total, created_at) VALUES (?, ?, ?, ?)`, "100000001", "complete", 49.98, now)

	var pc, cc, sc int64
	db.Model(&product.Product{}).Count(&pc)
	db.Model(&category.Category{}).Count(&cc)
	db.Table("store").Count(&sc)
	log.Printf("done: %d products, %d categories, %d stores", pc, cc, sc)
}

func strPtr(s string) *string { return &s }
