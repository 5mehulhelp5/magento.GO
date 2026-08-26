package servicetest

import (
	"strings"
	"testing"

	priceEntity "magento.GO/model/entity/price"
	productService "magento.GO/service/product"
)

func TestImport_TierPrices_AllGroupsAndPerGroupEntries(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,tier_prices\nSKU-A,\"all:5:8.99|1:10:7.50|2:1:9.99\"\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["tier_prices"] != 3 {
		t.Fatalf("tier_prices = %d, want 3", res.EAVCounts["tier_prices"])
	}
	if len(res.Warnings) != 0 {
		t.Errorf("warnings = %v, want none", res.Warnings)
	}

	var rows []priceEntity.TierPrice
	db.Order("qty").Find(&rows)
	if len(rows) != 3 {
		t.Fatalf("tier price rows = %d, want 3", len(rows))
	}

	// qty=1 -> the group=2 entry (a "group price").
	if rows[0].AllGroups != 0 || rows[0].CustomerGroupID != 2 || rows[0].Value != 9.99 {
		t.Errorf("rows[0] = %+v, want AllGroups=0 CustomerGroupID=2 Value=9.99", rows[0])
	}
	// qty=5 -> the all-groups entry.
	if rows[1].AllGroups != 1 || rows[1].Value != 8.99 {
		t.Errorf("rows[1] = %+v, want AllGroups=1 Value=8.99", rows[1])
	}
	// qty=10 -> group=1.
	if rows[2].AllGroups != 0 || rows[2].CustomerGroupID != 1 || rows[2].Value != 7.50 {
		t.Errorf("rows[2] = %+v, want AllGroups=0 CustomerGroupID=1 Value=7.50", rows[2])
	}
}

func TestImport_TierPrices_ReimportUpserts(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv1 := "sku,tier_prices\nSKU-A,all:5:8.99\n"
	if _, err := productService.ImportProducts(db, strings.NewReader(csv1), productService.ImportOptions{}); err != nil {
		t.Fatalf("first import: %v", err)
	}
	csv2 := "sku,tier_prices\nSKU-A,all:5:6.99\n"
	if _, err := productService.ImportProducts(db, strings.NewReader(csv2), productService.ImportOptions{}); err != nil {
		t.Fatalf("second import: %v", err)
	}

	var rows []priceEntity.TierPrice
	db.Find(&rows)
	if len(rows) != 1 {
		t.Fatalf("tier price rows = %d, want 1 (upsert, not duplicate)", len(rows))
	}
	if rows[0].Value != 6.99 {
		t.Errorf("Value = %v, want 6.99 (updated)", rows[0].Value)
	}
}

func TestImport_TierPrices_MalformedEntryWarnsAndIsSkipped(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,tier_prices\nSKU-A,\"all:5:8.99|not-enough-parts|all:abc:1.00|all:5:not-a-number\"\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["tier_prices"] != 1 {
		t.Errorf("tier_prices = %d, want 1 (only the valid entry)", res.EAVCounts["tier_prices"])
	}
	if len(res.Warnings) != 3 {
		t.Fatalf("warnings = %v, want 3", res.Warnings)
	}
}

func TestImport_TierPrices_NoColumnIsANoOp(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,name\nSKU-A,Widget\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["tier_prices"] != 0 {
		t.Errorf("tier_prices = %d, want 0", res.EAVCounts["tier_prices"])
	}
}

func TestImport_TierPrices_BlankCellIsSkipped(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,tier_prices\nSKU-A,\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["tier_prices"] != 0 {
		t.Errorf("tier_prices = %d, want 0", res.EAVCounts["tier_prices"])
	}
}

func TestImport_TierPrices_UnknownSKUIsSkipped(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,tier_prices\n,all:5:8.99\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["tier_prices"] != 0 {
		t.Errorf("tier_prices = %d, want 0", res.EAVCounts["tier_prices"])
	}
}
