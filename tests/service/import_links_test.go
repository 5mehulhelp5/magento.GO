package servicetest

import (
	"strings"
	"testing"

	productEntity "magento.GO/model/entity/product"
	productService "magento.GO/service/product"
)

func TestImport_ProductLinks_AllFourTypes(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,related_skus,upsell_skus,crosssell_skus,grouped_skus\n" +
		"SKU-A,\"SKU-B,SKU-C\",SKU-B,SKU-C,SKU-B\n" +
		"SKU-B,,,,\n" +
		"SKU-C,,,,\n"

	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("warnings = %v, want none", res.Warnings)
	}
	// related=[B,C] (2) + upsell=B (1) + crosssell=C (1) + grouped=B (1) = 5.
	if res.EAVCounts["product_links"] != 5 {
		t.Fatalf("product_links = %d, want 5", res.EAVCounts["product_links"])
	}

	var links []productEntity.ProductLink
	db.Order("link_type_id, linked_product_id").Find(&links)
	if len(links) != 5 {
		t.Fatalf("link rows = %d, want 5", len(links))
	}

	var a, b, c uint
	db.Model(&productEntity.Product{}).Where("sku = ?", "SKU-A").Pluck("entity_id", &a)
	db.Model(&productEntity.Product{}).Where("sku = ?", "SKU-B").Pluck("entity_id", &b)
	db.Model(&productEntity.Product{}).Where("sku = ?", "SKU-C").Pluck("entity_id", &c)

	wantByType := map[uint16][]uint{
		productEntity.LinkTypeRelated:   {b, c},
		productEntity.LinkTypeGrouped:   {b},
		productEntity.LinkTypeUpSell:    {b},
		productEntity.LinkTypeCrossSell: {c},
	}
	gotByType := map[uint16][]uint{}
	for _, l := range links {
		if l.ProductID != a {
			t.Errorf("link.ProductID = %d, want SKU-A's id %d", l.ProductID, a)
		}
		gotByType[l.LinkTypeID] = append(gotByType[l.LinkTypeID], l.LinkedProductID)
	}
	for linkType, want := range wantByType {
		got := gotByType[linkType]
		if len(got) != len(want) {
			t.Errorf("link_type_id=%d: got %v, want %v", linkType, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("link_type_id=%d: got %v, want %v", linkType, got, want)
				break
			}
		}
	}
}

func TestImport_ProductLinks_ReferencesPreExistingProductNotInThisCSV(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	existing := productEntity.Product{SKU: "PRE-EXISTING", AttributeSetID: 4, TypeID: "simple"}
	db.Create(&existing)

	csv := "sku,related_skus\nSKU-A,PRE-EXISTING\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("warnings = %v, want none", res.Warnings)
	}
	if res.EAVCounts["product_links"] != 1 {
		t.Fatalf("product_links = %d, want 1", res.EAVCounts["product_links"])
	}

	var link productEntity.ProductLink
	db.First(&link)
	if link.LinkedProductID != existing.EntityID {
		t.Errorf("LinkedProductID = %d, want %d", link.LinkedProductID, existing.EntityID)
	}

	// Only one product should have been created by this import -- the
	// pre-existing one must not be re-created.
	var count int64
	db.Model(&productEntity.Product{}).Count(&count)
	if count != 2 {
		t.Errorf("product count = %d, want 2 (SKU-A + the pre-existing one)", count)
	}
}

func TestImport_ProductLinks_UnknownSKUWarnsAndIsSkipped(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,related_skus\nSKU-A,DOES-NOT-EXIST\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["product_links"] != 0 {
		t.Errorf("product_links = %d, want 0", res.EAVCounts["product_links"])
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "unknown SKU") {
		t.Fatalf("warnings = %v, want exactly 1 mentioning unknown SKU", res.Warnings)
	}
}

func TestImport_ProductLinks_SelfReferenceWarnsAndIsSkipped(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,related_skus\nSKU-A,SKU-A\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["product_links"] != 0 {
		t.Errorf("product_links = %d, want 0", res.EAVCounts["product_links"])
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "references itself") {
		t.Fatalf("warnings = %v, want exactly 1 mentioning self-reference", res.Warnings)
	}
}

func TestImport_ProductLinks_ReimportUpsertsPosition(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,related_skus\nSKU-A,\"SKU-B,SKU-C\"\nSKU-B,\nSKU-C,\n"
	if _, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{}); err != nil {
		t.Fatalf("first import: %v", err)
	}
	// Reverse order on reimport.
	csv2 := "sku,related_skus\nSKU-A,\"SKU-C,SKU-B\"\nSKU-B,\nSKU-C,\n"
	if _, err := productService.ImportProducts(db, strings.NewReader(csv2), productService.ImportOptions{}); err != nil {
		t.Fatalf("second import: %v", err)
	}

	var count int64
	db.Model(&productEntity.ProductLink{}).Count(&count)
	if count != 2 {
		t.Errorf("link rows after reimport = %d, want 2 (upsert, not duplicate)", count)
	}
}

func TestImport_ProductLinks_NoColumnsIsANoOp(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,name\nSKU-A,Widget\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["product_links"] != 0 {
		t.Errorf("product_links = %d, want 0", res.EAVCounts["product_links"])
	}
}
