package modeltest

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	productEntity "magento.GO/model/entity/product"
	salesEntity "magento.GO/model/entity/sales"
	productRepo "magento.GO/model/repository/product"
	categoryRepo "magento.GO/model/repository/category"
	salesRepo "magento.GO/model/repository/sales"
)

func testDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&productEntity.Product{}, &salesEntity.SalesOrderGrid{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestNewProductRepository(t *testing.T) {
	db := testDB(t)
	repo := productRepo.NewProductRepository(db)
	if repo == nil {
		t.Fatal("NewProductRepository returned nil")
	}
}

func TestProductRepository_CreateAndFindByID(t *testing.T) {
	db := testDB(t)
	repo := productRepo.NewProductRepository(db)

	prod := &productEntity.Product{
		AttributeSetID: 1,
		TypeID:         "simple",
		SKU:            "TEST-SKU-001",
	}
	if err := repo.Create(prod); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if prod.EntityID == 0 {
		t.Error("EntityID not set after Create")
	}

	found, err := repo.FindByID(prod.EntityID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.SKU != "TEST-SKU-001" {
		t.Errorf("SKU = %q, want TEST-SKU-001", found.SKU)
	}
}

func TestNewCategoryRepository(t *testing.T) {
	db := testDB(t)
	repo := categoryRepo.NewCategoryRepository(db)
	if repo == nil {
		t.Fatal("NewCategoryRepository returned nil")
	}
}

func TestNewSalesOrderGridRepository(t *testing.T) {
	db := testDB(t)
	repo := salesRepo.NewSalesOrderGridRepository(db)
	if repo == nil {
		t.Fatal("NewSalesOrderGridRepository returned nil")
	}
}

func TestSalesOrderGridRepository_CreateAndFindByID(t *testing.T) {
	db := testDB(t)
	repo := salesRepo.NewSalesOrderGridRepository(db)

	order := &salesEntity.SalesOrderGrid{
		EntityID: 1,
		Status:   "pending",
		IncrementID: "000000001",
	}
	if err := repo.Create(order); err != nil {
		t.Fatalf("Create: %v", err)
	}

	found, err := repo.FindByID(order.EntityID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.Status != "pending" {
		t.Errorf("Status = %q, want pending", found.Status)
	}
	if found.IncrementID != "000000001" {
		t.Errorf("IncrementID = %q, want 000000001", found.IncrementID)
	}
}

func TestSalesOrderGridRepository_FindAll(t *testing.T) {
	db := testDB(t)
	repo := salesRepo.NewSalesOrderGridRepository(db)

	all, err := repo.FindAll()
	if err != nil {
		t.Fatalf("FindAll empty: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected 0, got %d", len(all))
	}

	_ = repo.Create(&salesEntity.SalesOrderGrid{EntityID: 1, Status: "pending", IncrementID: "100000001"})
	_ = repo.Create(&salesEntity.SalesOrderGrid{EntityID: 2, Status: "complete", IncrementID: "100000002"})

	all, err = repo.FindAll()
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2, got %d", len(all))
	}
}

func TestSalesOrderGridRepository_Update(t *testing.T) {
	db := testDB(t)
	repo := salesRepo.NewSalesOrderGridRepository(db)

	order := &salesEntity.SalesOrderGrid{EntityID: 1, Status: "pending", IncrementID: "100000001"}
	if err := repo.Create(order); err != nil {
		t.Fatalf("Create: %v", err)
	}

	order.Status = "complete"
	if err := repo.Update(order); err != nil {
		t.Fatalf("Update: %v", err)
	}

	found, err := repo.FindByID(order.EntityID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.Status != "complete" {
		t.Errorf("Status = %q, want complete", found.Status)
	}
}

func TestSalesOrderGridRepository_Delete(t *testing.T) {
	db := testDB(t)
	repo := salesRepo.NewSalesOrderGridRepository(db)

	order := &salesEntity.SalesOrderGrid{EntityID: 1, Status: "pending", IncrementID: "100000001"}
	if err := repo.Create(order); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(order.EntityID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := repo.FindByID(order.EntityID)
	if err != gorm.ErrRecordNotFound {
		t.Errorf("after Delete: err = %v, want ErrRecordNotFound", err)
	}
}
