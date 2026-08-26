package servicetest

import (
	"strings"
	"testing"

	productEntity "magento.GO/model/entity/product"
	productService "magento.GO/service/product"
)

func TestImport_Configurable_AttributesAndVariations(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,type_id,configurable_attributes,configurable_variations\n" +
		"SIMPLE-RED-S,simple,,\n" +
		"SIMPLE-RED-M,simple,,\n" +
		"CONFIG-1,configurable,\"status,special_from_date\",\"SIMPLE-RED-S,SIMPLE-RED-M\"\n"

	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("warnings = %v, want none", res.Warnings)
	}
	if res.EAVCounts["configurable_attributes"] != 2 {
		t.Errorf("configurable_attributes = %d, want 2", res.EAVCounts["configurable_attributes"])
	}
	if res.EAVCounts["configurable_links"] != 2 {
		t.Errorf("configurable_links = %d, want 2", res.EAVCounts["configurable_links"])
	}

	var configID, redSID, redMID uint
	db.Model(&productEntity.Product{}).Where("sku = ?", "CONFIG-1").Pluck("entity_id", &configID)
	db.Model(&productEntity.Product{}).Where("sku = ?", "SIMPLE-RED-S").Pluck("entity_id", &redSID)
	db.Model(&productEntity.Product{}).Where("sku = ?", "SIMPLE-RED-M").Pluck("entity_id", &redMID)

	var attrs []productEntity.ProductSuperAttribute
	db.Where("product_id = ?", configID).Order("position").Find(&attrs)
	if len(attrs) != 2 {
		t.Fatalf("super attribute rows = %d, want 2", len(attrs))
	}
	// attribute_id 76 = status, 77 = special_from_date (per seedAttributes).
	if attrs[0].AttributeID != 76 || attrs[1].AttributeID != 77 {
		t.Errorf("attrs = %+v, want attribute_id 76 then 77", attrs)
	}

	var links []productEntity.ProductSuperLink
	db.Where("parent_id = ?", configID).Order("product_id").Find(&links)
	if len(links) != 2 {
		t.Fatalf("super link rows = %d, want 2", len(links))
	}
	linkedIDs := map[uint]bool{links[0].ProductID: true, links[1].ProductID: true}
	if !linkedIDs[redSID] || !linkedIDs[redMID] {
		t.Errorf("links = %+v, want children %d and %d", links, redSID, redMID)
	}
}

func TestImport_Configurable_UnknownAttributeCodeWarnsAndSkips(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,type_id,configurable_attributes\nCONFIG-1,configurable,not_a_real_attribute\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["configurable_attributes"] != 0 {
		t.Errorf("configurable_attributes = %d, want 0", res.EAVCounts["configurable_attributes"])
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "unknown attribute") {
		t.Fatalf("warnings = %v, want exactly 1 mentioning unknown attribute", res.Warnings)
	}
}

func TestImport_Configurable_UnknownChildSKUWarnsAndSkips(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,type_id,configurable_variations\nCONFIG-1,configurable,DOES-NOT-EXIST\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["configurable_links"] != 0 {
		t.Errorf("configurable_links = %d, want 0", res.EAVCounts["configurable_links"])
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "unknown SKU") {
		t.Fatalf("warnings = %v, want exactly 1 mentioning unknown SKU", res.Warnings)
	}
}

func TestImport_Configurable_SelfReferenceWarnsAndSkips(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,type_id,configurable_variations\nCONFIG-1,configurable,CONFIG-1\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["configurable_links"] != 0 {
		t.Errorf("configurable_links = %d, want 0", res.EAVCounts["configurable_links"])
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "references itself") {
		t.Fatalf("warnings = %v, want exactly 1 mentioning self-reference", res.Warnings)
	}
}

func TestImport_Configurable_ReimportReassignsChildToNewParent(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv1 := "sku,type_id,configurable_variations\n" +
		"CHILD,simple,\n" +
		"CONFIG-A,configurable,CHILD\n" +
		"CONFIG-B,configurable,\n"
	if _, err := productService.ImportProducts(db, strings.NewReader(csv1), productService.ImportOptions{}); err != nil {
		t.Fatalf("first import: %v", err)
	}
	csv2 := "sku,type_id,configurable_variations\n" +
		"CHILD,simple,\n" +
		"CONFIG-A,configurable,\n" +
		"CONFIG-B,configurable,CHILD\n"
	if _, err := productService.ImportProducts(db, strings.NewReader(csv2), productService.ImportOptions{}); err != nil {
		t.Fatalf("second import: %v", err)
	}

	var configBID uint
	db.Model(&productEntity.Product{}).Where("sku = ?", "CONFIG-B").Pluck("entity_id", &configBID)

	var links []productEntity.ProductSuperLink
	db.Find(&links)
	if len(links) != 1 {
		t.Fatalf("super link rows = %d, want 1 (upsert on product_id, not a second row)", len(links))
	}
	if links[0].ParentID != configBID {
		t.Errorf("ParentID = %d, want CONFIG-B's id %d (reassigned)", links[0].ParentID, configBID)
	}
}

func TestImport_Configurable_NoColumnsIsANoOp(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,name\nSKU-A,Widget\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["configurable_attributes"] != 0 || res.EAVCounts["configurable_links"] != 0 {
		t.Errorf("configurable counts = %d/%d, want 0/0", res.EAVCounts["configurable_attributes"], res.EAVCounts["configurable_links"])
	}
}
