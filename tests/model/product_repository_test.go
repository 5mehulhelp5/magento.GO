package modeltest

import (
	"os"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	entity "magento.GO/model/entity"
	categoryEntity "magento.GO/model/entity/category"
	productEntity "magento.GO/model/entity/product"
	productRepo "magento.GO/model/repository/product"
)

func productRepoTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// Migrate CategoryProduct first so catalog_category_product has position column
	// (Product's many2many would create it with only product_id, category_id)
	if err := db.AutoMigrate(
		&entity.EavAttribute{},
		&categoryEntity.Category{},
		&categoryEntity.CategoryProduct{},
	); err != nil {
		t.Fatalf("migrate category: %v", err)
	}
	if err := db.AutoMigrate(
		&productEntity.Product{},
		&productEntity.ProductVarchar{},
		&productEntity.ProductInt{},
		&productEntity.ProductDecimal{},
		&productEntity.ProductText{},
		&productEntity.ProductDatetime{},
		&productEntity.ProductMediaGallery{},
		&productEntity.StockItem{},
		&productEntity.ProductIndexPrice{},
	); err != nil {
		t.Fatalf("migrate product: %v", err)
	}
	return db
}

func TestProductRepository_GetProductRepository(t *testing.T) {
	db := productRepoTestDB(t)
	r1 := productRepo.GetProductRepository(db)
	r2 := productRepo.GetProductRepository(db)
	if r1 != r2 {
		t.Error("GetProductRepository should return same instance for same DB")
	}
	if r1 == nil {
		t.Fatal("GetProductRepository returned nil")
	}
}

func TestProductRepository_FindAll(t *testing.T) {
	db := productRepoTestDB(t)
	repo := productRepo.NewProductRepository(db)

	// Empty
	all, err := repo.FindAll()
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("FindAll empty DB: got %d, want 0", len(all))
	}

	// With data
	_ = repo.Create(&productEntity.Product{AttributeSetID: 1, TypeID: "simple", SKU: "FIND-ALL-1"})
	_ = repo.Create(&productEntity.Product{AttributeSetID: 1, TypeID: "simple", SKU: "FIND-ALL-2"})
	all, err = repo.FindAll()
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("FindAll: got %d, want 2", len(all))
	}
}

func TestProductRepository_Update(t *testing.T) {
	db := productRepoTestDB(t)
	repo := productRepo.NewProductRepository(db)

	prod := &productEntity.Product{AttributeSetID: 1, TypeID: "simple", SKU: "UPDATE-ORIG"}
	if err := repo.Create(prod); err != nil {
		t.Fatalf("Create: %v", err)
	}
	prod.SKU = "UPDATE-NEW"
	if err := repo.Update(prod); err != nil {
		t.Fatalf("Update: %v", err)
	}
	found, err := repo.FindByID(prod.EntityID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.SKU != "UPDATE-NEW" {
		t.Errorf("Update: SKU = %q, want UPDATE-NEW", found.SKU)
	}
}

func TestProductRepository_Delete(t *testing.T) {
	db := productRepoTestDB(t)
	repo := productRepo.NewProductRepository(db)

	prod := &productEntity.Product{AttributeSetID: 1, TypeID: "simple", SKU: "DELETE-ME"}
	if err := repo.Create(prod); err != nil {
		t.Fatalf("Create: %v", err)
	}
	id := prod.EntityID
	if err := repo.Delete(id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := repo.FindByID(id)
	if err != gorm.ErrRecordNotFound {
		t.Errorf("Delete: FindByID err = %v, want ErrRecordNotFound", err)
	}
}

func TestProductRepository_FetchProductIDsByCategoryWithPosition(t *testing.T) {
	db := productRepoTestDB(t)
	repo := productRepo.NewProductRepository(db)

	// Create category and products
	cat := categoryEntity.Category{AttributeSetID: 1, ParentID: 0, Path: "1/2", Position: 1, Level: 1, ChildrenCount: 0}
	if err := db.Create(&cat).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	p1 := &productEntity.Product{AttributeSetID: 1, TypeID: "simple", SKU: "CAT-P1"}
	p2 := &productEntity.Product{AttributeSetID: 1, TypeID: "simple", SKU: "CAT-P2"}
	if err := repo.Create(p1); err != nil {
		t.Fatalf("Create p1: %v", err)
	}
	if err := repo.Create(p2); err != nil {
		t.Fatalf("Create p2: %v", err)
	}
	if err := db.Create(&categoryEntity.CategoryProduct{CategoryID: cat.EntityID, ProductID: p1.EntityID, Position: 10}).Error; err != nil {
		t.Fatalf("link cat-p1: %v", err)
	}
	if err := db.Create(&categoryEntity.CategoryProduct{CategoryID: cat.EntityID, ProductID: p2.EntityID, Position: 5}).Error; err != nil {
		t.Fatalf("link cat-p2: %v", err)
	}

	ids, err := repo.FetchProductIDsByCategoryWithPosition(cat.EntityID, true)
	if err != nil {
		t.Fatalf("FetchProductIDsByCategoryWithPosition: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("got %d ids, want 2", len(ids))
	}
	if ids[0] != p2.EntityID || ids[1] != p1.EntityID {
		t.Errorf("ASC order: got %v, want [p2, p1] by position", ids)
	}

	idsDesc, _ := repo.FetchProductIDsByCategoryWithPosition(cat.EntityID, false)
	if idsDesc[0] != p1.EntityID || idsDesc[1] != p2.EntityID {
		t.Errorf("DESC order: got %v, want [p1, p2]", idsDesc)
	}
}

func TestProductRepository_FetchWithAllAttributes(t *testing.T) {
	db := productRepoTestDB(t)
	repo := productRepo.NewProductRepository(db)

	// Seed eav_attribute for name (73 is common Magento ID)
	if err := db.Create(&entity.EavAttribute{AttributeID: 73, EntityTypeID: 4, AttributeCode: "name", BackendType: "varchar"}).Error; err != nil {
		t.Fatalf("create attr: %v", err)
	}
	prod := &productEntity.Product{AttributeSetID: 1, TypeID: "simple", SKU: "ATTR-SKU"}
	if err := repo.Create(prod); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := db.Create(&productEntity.ProductVarchar{AttributeID: 73, StoreID: 0, EntityID: prod.EntityID, Value: "Test Name"}).Error; err != nil {
		t.Fatalf("create varchar: %v", err)
	}

	products, err := repo.FetchWithAllAttributes(0)
	if err != nil {
		t.Fatalf("FetchWithAllAttributes: %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("got %d products, want 1", len(products))
	}
	if len(products[0].Varchars) != 1 || products[0].Varchars[0].Value != "Test Name" {
		t.Errorf("Varchars: got %v", products[0].Varchars)
	}
}

func TestProductRepository_FetchWithAllAttributesFlat(t *testing.T) {
	os.Setenv("PRODUCT_FLAT_CACHE", "off")
	defer os.Unsetenv("PRODUCT_FLAT_CACHE")

	db := productRepoTestDB(t)
	repo := productRepo.NewProductRepository(db)
	if err := db.Create(&entity.EavAttribute{AttributeID: 73, EntityTypeID: 4, AttributeCode: "name", BackendType: "varchar"}).Error; err != nil {
		t.Fatalf("create attr: %v", err)
	}
	prod := &productEntity.Product{AttributeSetID: 1, TypeID: "simple", SKU: "FLAT-SKU"}
	if err := repo.Create(prod); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := db.Create(&productEntity.ProductVarchar{AttributeID: 73, StoreID: 0, EntityID: prod.EntityID, Value: "Flat Name"}).Error; err != nil {
		t.Fatalf("create varchar: %v", err)
	}

	flat, err := repo.FetchWithAllAttributesFlat(0)
	if err != nil {
		t.Fatalf("FetchWithAllAttributesFlat: %v", err)
	}
	if len(flat) != 1 {
		t.Fatalf("got %d flat products, want 1", len(flat))
	}
	p, ok := flat[prod.EntityID]
	if !ok {
		t.Fatal("product not in flat map")
	}
	if p["sku"] != "FLAT-SKU" {
		t.Errorf("sku = %v, want FLAT-SKU", p["sku"])
	}
	if p["name"] != "Flat Name" {
		t.Errorf("name = %v, want Flat Name", p["name"])
	}
}

func TestProductRepository_FetchWithAllAttributesFlatByIDs(t *testing.T) {
	os.Setenv("PRODUCT_FLAT_CACHE", "off")
	defer os.Unsetenv("PRODUCT_FLAT_CACHE")

	db := productRepoTestDB(t)
	repo := productRepo.NewProductRepository(db)
	if err := db.Create(&entity.EavAttribute{AttributeID: 73, EntityTypeID: 4, AttributeCode: "name", BackendType: "varchar"}).Error; err != nil {
		t.Fatalf("create attr: %v", err)
	}
	p1 := &productEntity.Product{AttributeSetID: 1, TypeID: "simple", SKU: "BYID-1"}
	p2 := &productEntity.Product{AttributeSetID: 1, TypeID: "simple", SKU: "BYID-2"}
	if err := repo.Create(p1); err != nil {
		t.Fatalf("Create p1: %v", err)
	}
	if err := repo.Create(p2); err != nil {
		t.Fatalf("Create p2: %v", err)
	}

	flat, err := repo.FetchWithAllAttributesFlatByIDs([]uint{p1.EntityID, p2.EntityID}, 0)
	if err != nil {
		t.Fatalf("FetchWithAllAttributesFlatByIDs: %v", err)
	}
	if len(flat) != 2 {
		t.Errorf("got %d, want 2", len(flat))
	}
	if flat[p1.EntityID]["sku"] != "BYID-1" || flat[p2.EntityID]["sku"] != "BYID-2" {
		t.Errorf("flat map: %v", flat)
	}
}

func TestProductRepository_LoadAttributeCodeMap(t *testing.T) {
	db := productRepoTestDB(t)
	if err := db.Create(&entity.EavAttribute{AttributeID: 73, EntityTypeID: 4, AttributeCode: "name", BackendType: "varchar"}).Error; err != nil {
		t.Fatalf("create attr: %v", err)
	}
	if err := db.Create(&entity.EavAttribute{AttributeID: 77, EntityTypeID: 4, AttributeCode: "price", BackendType: "decimal"}).Error; err != nil {
		t.Fatalf("create attr2: %v", err)
	}

	m, err := productRepo.LoadAttributeCodeMap(db)
	if err != nil {
		t.Fatalf("LoadAttributeCodeMap: %v", err)
	}
	if m[73] != "name" || m[77] != "price" {
		t.Errorf("LoadAttributeCodeMap: got %v", m)
	}
}

func TestFlattenProductAttributesWithCodes(t *testing.T) {
	attrMap := map[uint16]string{73: "name", 77: "price"}
	prod := &productEntity.Product{
		EntityID:       42,
		AttributeSetID: 1,
		TypeID:         "simple",
		SKU:            "FLATTEN-SKU",
		Varchars:       []productEntity.ProductVarchar{{AttributeID: 73, Value: "Prod Name"}},
		Decimals:       []productEntity.ProductDecimal{{AttributeID: 77, Value: 99.99}},
	}
	flat := productRepo.FlattenProductAttributesWithCodes(prod, attrMap)

	if flat["entity_id"] != uint(42) {
		t.Errorf("entity_id = %v, want 42", flat["entity_id"])
	}
	if flat["sku"] != "FLATTEN-SKU" {
		t.Errorf("sku = %v, want FLATTEN-SKU", flat["sku"])
	}
	if flat["name"] != "Prod Name" {
		t.Errorf("name = %v, want Prod Name", flat["name"])
	}
	if flat["price"] != 99.99 {
		t.Errorf("price = %v, want 99.99", flat["price"])
	}
	if flat["category_ids"] == nil {
		t.Error("category_ids should be present (nil slice)")
	}
}
