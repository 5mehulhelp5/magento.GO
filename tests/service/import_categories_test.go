package servicetest

import (
	"strings"
	"testing"

	categoryEntity "magento.GO/model/entity/category"
	productService "magento.GO/service/product"
)

func TestImport_Categories_CreatesHierarchyAndAssignsProduct(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,name,categories\n" +
		"SKU-CAT-1,Widget,Default Category/Shoes/Running\n"

	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.Created != 1 {
		t.Fatalf("Created = %d, want 1", res.Created)
	}
	if res.EAVCounts["category_links"] != 1 {
		t.Errorf("category_links = %d, want 1", res.EAVCounts["category_links"])
	}
	if len(res.Warnings) != 0 {
		t.Errorf("warnings = %v, want none", res.Warnings)
	}

	// Four categories should now exist: an implicit unnamed true root
	// (parent_id=0, matching Magento's own entity_id=1 root), then
	// Default Category, Shoes, Running underneath it.
	var cats []categoryEntity.Category
	db.Order("entity_id").Find(&cats)
	if len(cats) != 4 {
		t.Fatalf("category count = %d, want 4: %+v", len(cats), cats)
	}

	// Names should match the path, in parent -> child order (the true root
	// itself has no name row).
	var names []categoryEntity.CategoryVarchar
	db.Order("entity_id").Find(&names)
	if len(names) != 3 {
		t.Fatalf("category name rows = %d, want 3", len(names))
	}
	wantNames := []string{"Default Category", "Shoes", "Running"}
	for i, n := range names {
		if n.Value != wantNames[i] {
			t.Errorf("names[%d] = %q, want %q", i, n.Value, wantNames[i])
		}
	}

	// Hierarchy: root -> Default Category -> Shoes -> Running.
	if cats[0].ParentID != 0 {
		t.Errorf("root parent_id = %d, want 0", cats[0].ParentID)
	}
	if cats[1].ParentID != cats[0].EntityID {
		t.Errorf("Default Category.ParentID = %d, want %d", cats[1].ParentID, cats[0].EntityID)
	}
	if cats[2].ParentID != cats[1].EntityID {
		t.Errorf("Shoes.ParentID = %d, want %d", cats[2].ParentID, cats[1].EntityID)
	}
	if cats[3].ParentID != cats[2].EntityID {
		t.Errorf("Running.ParentID = %d, want %d", cats[3].ParentID, cats[2].EntityID)
	}
	if cats[1].Level != 1 || cats[2].Level != 2 || cats[3].Level != 3 {
		t.Errorf("levels = %d,%d,%d want 1,2,3", cats[1].Level, cats[2].Level, cats[3].Level)
	}
	if cats[0].ChildrenCount != 1 {
		t.Errorf("root ChildrenCount = %d, want 1", cats[0].ChildrenCount)
	}

	// Product should be linked only to the leaf category (Running).
	var links []categoryEntity.CategoryProduct
	db.Find(&links)
	if len(links) != 1 {
		t.Fatalf("category_product rows = %d, want 1", len(links))
	}
	if links[0].CategoryID != cats[3].EntityID {
		t.Errorf("link.CategoryID = %d, want leaf category %d", links[0].CategoryID, cats[3].EntityID)
	}
}

func TestImport_Categories_ReusesExistingPathAcrossProducts(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,categories\n" +
		"SKU-A,Default Category/Shoes\n" +
		"SKU-B,Default Category/Shoes\n"

	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["category_links"] != 2 {
		t.Errorf("category_links = %d, want 2", res.EAVCounts["category_links"])
	}

	// Only 3 categories should exist (root, Default Category, Shoes), not
	// 5 -- both rows share the same path and must resolve to the same
	// category_id rather than creating the path twice.
	var cats []categoryEntity.Category
	db.Find(&cats)
	if len(cats) != 3 {
		t.Fatalf("category count = %d, want 3 (path must be shared, not recreated): %+v", len(cats), cats)
	}

	var links []categoryEntity.CategoryProduct
	db.Find(&links)
	if len(links) != 2 {
		t.Fatalf("category_product rows = %d, want 2", len(links))
	}
	if links[0].CategoryID != links[1].CategoryID {
		t.Errorf("both products should link to the same Shoes category_id, got %d and %d", links[0].CategoryID, links[1].CategoryID)
	}
}

func TestImport_Categories_ReimportIsIdempotent(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,categories\nSKU-A,Default Category/Shoes\n"

	if _, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{}); err != nil {
		t.Fatalf("first ImportProducts: %v", err)
	}
	if _, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{}); err != nil {
		t.Fatalf("second ImportProducts: %v", err)
	}

	var cats []categoryEntity.Category
	db.Find(&cats)
	if len(cats) != 3 {
		t.Errorf("category count after reimport = %d, want 3 (no duplicate creation)", len(cats))
	}
	var links []categoryEntity.CategoryProduct
	db.Find(&links)
	if len(links) != 1 {
		t.Errorf("category_product rows after reimport = %d, want 1 (upsert, not duplicate)", len(links))
	}
}

func TestImport_Categories_MultiplePathsPerProduct(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,categories\n" +
		"SKU-A,\"Default Category/Shoes,Default Category/Sale\"\n"

	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["category_links"] != 2 {
		t.Errorf("category_links = %d, want 2", res.EAVCounts["category_links"])
	}

	var links []categoryEntity.CategoryProduct
	db.Find(&links)
	if len(links) != 2 {
		t.Fatalf("category_product rows = %d, want 2 (one per path)", len(links))
	}
}

func TestImport_Categories_NoColumnIsANoOp(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,name\nSKU-A,Widget\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["category_links"] != 0 {
		t.Errorf("category_links = %d, want 0", res.EAVCounts["category_links"])
	}
	var cats []categoryEntity.Category
	db.Find(&cats)
	if len(cats) != 0 {
		t.Errorf("category count = %d, want 0 (no categories column present)", len(cats))
	}
}

func TestImport_Categories_BlankCellIsSkipped(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,categories\nSKU-A,\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["category_links"] != 0 {
		t.Errorf("category_links = %d, want 0", res.EAVCounts["category_links"])
	}
}

func TestImport_Categories_EmptyPathSegmentWarns(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,categories\nSKU-A,\"//\"\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if len(res.Warnings) != 1 {
		t.Fatalf("warnings = %v, want exactly 1", res.Warnings)
	}
	if !strings.Contains(res.Warnings[0], "empty category path") {
		t.Errorf("warning = %q, want mention of empty category path", res.Warnings[0])
	}
	if res.EAVCounts["category_links"] != 0 {
		t.Errorf("category_links = %d, want 0", res.EAVCounts["category_links"])
	}
}

func TestImport_Categories_UnknownSKUIsSkipped(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	// sku column present but blank -> row is skipped entirely before
	// categories are even considered, so no categories should be created.
	csv := "sku,categories\n,Default Category/Shoes\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["category_links"] != 0 {
		t.Errorf("category_links = %d, want 0", res.EAVCounts["category_links"])
	}
	var cats []categoryEntity.Category
	db.Find(&cats)
	if len(cats) != 0 {
		t.Errorf("category count = %d, want 0", len(cats))
	}
}
