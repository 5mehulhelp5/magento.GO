package servicetest

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	inventoryRepo "magento.GO/model/repository/inventory"
	priceRepo "magento.GO/model/repository/price"
)

func TestHMAC_SignatureGeneration(t *testing.T) {
	cryptKey := "3254cdb1ae5233a336cdec765aeb3bb6"
	customerID := "123"

	mac := hmac.New(sha256.New, []byte(cryptKey))
	mac.Write([]byte(customerID))
	sig := hex.EncodeToString(mac.Sum(nil))

	if sig == "" {
		t.Error("signature should not be empty")
	}
	if len(sig) != 64 {
		t.Errorf("signature length = %d, want 64 hex chars", len(sig))
	}
}

func TestHMAC_SignatureVerification(t *testing.T) {
	cryptKey := "3254cdb1ae5233a336cdec765aeb3bb6"
	customerID := "123"

	// Generate signature
	mac := hmac.New(sha256.New, []byte(cryptKey))
	mac.Write([]byte(customerID))
	expected := mac.Sum(nil)
	sigHex := hex.EncodeToString(expected)

	// Verify with same key
	mac2 := hmac.New(sha256.New, []byte(cryptKey))
	mac2.Write([]byte(customerID))
	computed := mac2.Sum(nil)

	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		t.Fatalf("hex decode: %v", err)
	}

	if !hmac.Equal(computed, sig) {
		t.Error("signature verification failed")
	}
}

func TestHMAC_TamperedID_Fails(t *testing.T) {
	cryptKey := "3254cdb1ae5233a336cdec765aeb3bb6"
	customerID := "123"
	tamperedID := "124"

	// Generate signature for original ID
	mac := hmac.New(sha256.New, []byte(cryptKey))
	mac.Write([]byte(customerID))
	sigHex := hex.EncodeToString(mac.Sum(nil))

	// Verify with tampered ID
	mac2 := hmac.New(sha256.New, []byte(cryptKey))
	mac2.Write([]byte(tamperedID))
	computed := mac2.Sum(nil)

	sig, _ := hex.DecodeString(sigHex)

	if hmac.Equal(computed, sig) {
		t.Error("tampered ID should fail verification")
	}
}

func TestInventoryRepository_SQLite(t *testing.T) {
	db := importDB(t)

	// Create inventory_source_item table for SQLite
	db.Exec(`CREATE TABLE IF NOT EXISTS inventory_source_item (
		source_item_id INTEGER PRIMARY KEY AUTOINCREMENT,
		source_code VARCHAR(255) NOT NULL,
		sku VARCHAR(64) NOT NULL,
		quantity DECIMAL(12,4) NOT NULL DEFAULT 0,
		status INTEGER NOT NULL DEFAULT 0
	)`)

	// Insert test data
	db.Exec(`INSERT INTO inventory_source_item (source_code, sku, quantity, status) VALUES (?, ?, ?, ?)`,
		"default", "TEST-SKU-001", 150.5, 1)

	repo, err := inventoryRepo.NewInventoryRepository(db)
	if err != nil {
		t.Fatalf("NewInventoryRepository: %v", err)
	}

	qty, found := repo.GetQuantityBySKU("TEST-SKU-001", "default")
	if !found {
		t.Error("expected to find stock for TEST-SKU-001")
	}
	if qty != 150.5 {
		t.Errorf("quantity = %f, want 150.5", qty)
	}

	// Test not found
	_, found = repo.GetQuantityBySKU("NONEXISTENT", "default")
	if found {
		t.Error("should not find nonexistent SKU")
	}
}

func TestInventoryRepository_GetAllBySKU(t *testing.T) {
	db := importDB(t)

	db.Exec(`CREATE TABLE IF NOT EXISTS inventory_source_item (
		source_item_id INTEGER PRIMARY KEY AUTOINCREMENT,
		source_code VARCHAR(255) NOT NULL,
		sku VARCHAR(64) NOT NULL,
		quantity DECIMAL(12,4) NOT NULL DEFAULT 0,
		status INTEGER NOT NULL DEFAULT 0
	)`)

	// Multiple sources
	db.Exec(`INSERT INTO inventory_source_item (source_code, sku, quantity, status) VALUES (?, ?, ?, ?)`,
		"warehouse_a", "MULTI-SKU", 100, 1)
	db.Exec(`INSERT INTO inventory_source_item (source_code, sku, quantity, status) VALUES (?, ?, ?, ?)`,
		"warehouse_b", "MULTI-SKU", 50, 1)

	repo, _ := inventoryRepo.NewInventoryRepository(db)

	items, err := repo.GetAllBySKU("MULTI-SKU")
	if err != nil {
		t.Fatalf("GetAllBySKU: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestInventoryRepository_GetTotalQuantity(t *testing.T) {
	db := importDB(t)

	db.Exec(`CREATE TABLE IF NOT EXISTS inventory_source_item (
		source_item_id INTEGER PRIMARY KEY AUTOINCREMENT,
		source_code VARCHAR(255) NOT NULL,
		sku VARCHAR(64) NOT NULL,
		quantity DECIMAL(12,4) NOT NULL DEFAULT 0,
		status INTEGER NOT NULL DEFAULT 0
	)`)

	db.Exec(`INSERT INTO inventory_source_item (source_code, sku, quantity, status) VALUES (?, ?, ?, ?)`,
		"src1", "TOTAL-SKU", 100, 1)
	db.Exec(`INSERT INTO inventory_source_item (source_code, sku, quantity, status) VALUES (?, ?, ?, ?)`,
		"src2", "TOTAL-SKU", 75, 1)

	repo, _ := inventoryRepo.NewInventoryRepository(db)

	total, err := repo.GetTotalQuantityBySKU("TOTAL-SKU")
	if err != nil {
		t.Fatalf("GetTotalQuantityBySKU: %v", err)
	}
	if total != 175 {
		t.Errorf("total = %f, want 175", total)
	}
}

func TestInventoryRepository_BatchGetQuantities(t *testing.T) {
	db := importDB(t)

	db.Exec(`CREATE TABLE IF NOT EXISTS inventory_source_item (
		source_item_id INTEGER PRIMARY KEY AUTOINCREMENT,
		source_code VARCHAR(255) NOT NULL,
		sku VARCHAR(64) NOT NULL,
		quantity DECIMAL(12,4) NOT NULL DEFAULT 0,
		status INTEGER NOT NULL DEFAULT 0
	)`)

	db.Exec(`INSERT INTO inventory_source_item (source_code, sku, quantity, status) VALUES (?, ?, ?, ?)`,
		"default", "BATCH-1", 10, 1)
	db.Exec(`INSERT INTO inventory_source_item (source_code, sku, quantity, status) VALUES (?, ?, ?, ?)`,
		"default", "BATCH-2", 20, 1)
	db.Exec(`INSERT INTO inventory_source_item (source_code, sku, quantity, status) VALUES (?, ?, ?, ?)`,
		"default", "BATCH-3", 30, 1)

	repo, _ := inventoryRepo.NewInventoryRepository(db)

	result, err := repo.BatchGetQuantities([]string{"BATCH-1", "BATCH-2", "BATCH-3", "BATCH-MISSING"}, "default")
	if err != nil {
		t.Fatalf("BatchGetQuantities: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 results, got %d", len(result))
	}
	if result["BATCH-1"] != 10 {
		t.Errorf("BATCH-1 = %f, want 10", result["BATCH-1"])
	}
	if result["BATCH-2"] != 20 {
		t.Errorf("BATCH-2 = %f, want 20", result["BATCH-2"])
	}
}

func TestPriceRepository_SQLite_BasePriceFallback(t *testing.T) {
	db := importDB(t)

	// SQLite doesn't have information_schema, so schema detection returns CE mode
	repo, err := priceRepo.NewPriceRepository(db)
	if err != nil {
		t.Fatalf("NewPriceRepository: %v", err)
	}

	// Schema detection should not crash on SQLite
	isEE := repo.IsEnterprise()
	if isEE {
		t.Error("SQLite should be detected as CE schema")
	}
}
