package servicetest

import (
	"strings"
	"testing"

	productEntity "magento.GO/model/entity/product"
	productService "magento.GO/service/product"
)

func TestImport_Bundle_OptionsAndSelections(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,type_id,bundle_options\n" +
		"CPU-A,simple,\n" +
		"CPU-B,simple,\n" +
		"MOUSE,simple,\n" +
		"BUNDLE-1,bundle,\"select:CPU:1:CPU-A~1~0~fixed~1~0|CPU-B~1~50~fixed~0~1;checkbox:Extras:0:MOUSE~2~10~fixed~0~1\"\n"

	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("warnings = %v, want none", res.Warnings)
	}
	if res.EAVCounts["bundle_options"] != 2 {
		t.Errorf("bundle_options = %d, want 2", res.EAVCounts["bundle_options"])
	}
	if res.EAVCounts["bundle_selections"] != 3 {
		t.Errorf("bundle_selections = %d, want 3", res.EAVCounts["bundle_selections"])
	}

	var options []productEntity.ProductBundleOption
	db.Order("position").Find(&options)
	if len(options) != 2 {
		t.Fatalf("option rows = %d, want 2", len(options))
	}
	if options[0].Type != "select" || options[0].Title != "CPU" || options[0].Required != 1 {
		t.Errorf("options[0] = %+v, unexpected", options[0])
	}
	if options[1].Type != "checkbox" || options[1].Title != "Extras" || options[1].Required != 0 {
		t.Errorf("options[1] = %+v, unexpected", options[1])
	}

	var cpuID, mouseID uint
	db.Model(&productEntity.Product{}).Where("sku = ?", "CPU-A").Pluck("entity_id", &cpuID)
	db.Model(&productEntity.Product{}).Where("sku = ?", "MOUSE").Pluck("entity_id", &mouseID)

	var cpuSelections []productEntity.ProductBundleSelection
	db.Where("option_id = ?", options[0].OptionID).Order("position").Find(&cpuSelections)
	if len(cpuSelections) != 2 {
		t.Fatalf("CPU option selections = %d, want 2", len(cpuSelections))
	}
	if cpuSelections[0].ProductID != cpuID || cpuSelections[0].IsDefault != 1 || cpuSelections[0].SelectionCanChangeQty != 0 {
		t.Errorf("cpuSelections[0] = %+v, unexpected", cpuSelections[0])
	}
	if cpuSelections[1].SelectionPriceValue != 50 {
		t.Errorf("cpuSelections[1].SelectionPriceValue = %v, want 50", cpuSelections[1].SelectionPriceValue)
	}

	var extraSelections []productEntity.ProductBundleSelection
	db.Where("option_id = ?", options[1].OptionID).Find(&extraSelections)
	if len(extraSelections) != 1 || extraSelections[0].ProductID != mouseID || extraSelections[0].SelectionQty != 2 {
		t.Errorf("extraSelections = %+v, unexpected", extraSelections)
	}
}

func TestImport_Bundle_ReimportReplacesFully(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv1 := "sku,bundle_options\n" +
		"COMPONENT,\n" +
		"BUNDLE-1,select:Choice:1:COMPONENT~1~0~fixed~1~0\n"
	if _, err := productService.ImportProducts(db, strings.NewReader(csv1), productService.ImportOptions{}); err != nil {
		t.Fatalf("first import: %v", err)
	}
	csv2 := "sku,bundle_options\n" +
		"COMPONENT,\n" +
		"BUNDLE-1,checkbox:NewChoice:0:COMPONENT~1~5~fixed~0~1\n"
	if _, err := productService.ImportProducts(db, strings.NewReader(csv2), productService.ImportOptions{}); err != nil {
		t.Fatalf("second import: %v", err)
	}

	var options []productEntity.ProductBundleOption
	db.Find(&options)
	if len(options) != 1 {
		t.Fatalf("option rows after reimport = %d, want 1 (full replace)", len(options))
	}
	if options[0].Title != "NewChoice" {
		t.Errorf("Title = %q, want NewChoice", options[0].Title)
	}

	var selCount int64
	db.Model(&productEntity.ProductBundleSelection{}).Count(&selCount)
	if selCount != 1 {
		t.Errorf("selection rows after reimport = %d, want 1 (old option's selections cleaned up)", selCount)
	}
}

func TestImport_Bundle_UnknownComponentSKUWarnsAndSkipsSelection(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,bundle_options\nBUNDLE-1,select:CPU:1:DOES-NOT-EXIST~1~0~fixed~1~0\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	// The one selection was invalid, so the whole option has no valid
	// selections and is dropped too.
	if res.EAVCounts["bundle_options"] != 0 || res.EAVCounts["bundle_selections"] != 0 {
		t.Errorf("bundle_options/selections = %d/%d, want 0/0", res.EAVCounts["bundle_options"], res.EAVCounts["bundle_selections"])
	}
	if len(res.Warnings) != 2 || !strings.Contains(res.Warnings[0], "unknown SKU") {
		t.Fatalf("warnings = %v, want 2 (unknown SKU + no valid selections)", res.Warnings)
	}
}

func TestImport_Bundle_UnknownTypeWarnsAndSkips(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,bundle_options\nBUNDLE-1,not_a_type:CPU:1:X~1~0~fixed~1~0\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["bundle_options"] != 0 {
		t.Errorf("bundle_options = %d, want 0", res.EAVCounts["bundle_options"])
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "unknown bundle option type") {
		t.Fatalf("warnings = %v, want exactly 1 mentioning unknown type", res.Warnings)
	}
}

func TestImport_Bundle_NoSelectionsWarnsAndSkips(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,bundle_options\nBUNDLE-1,select:CPU:1:\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["bundle_options"] != 0 {
		t.Errorf("bundle_options = %d, want 0", res.EAVCounts["bundle_options"])
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "no selections") {
		t.Fatalf("warnings = %v, want exactly 1 mentioning no selections", res.Warnings)
	}
}

func TestImport_Bundle_NoColumnIsANoOp(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,name\nSKU-A,Widget\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["bundle_options"] != 0 {
		t.Errorf("bundle_options = %d, want 0", res.EAVCounts["bundle_options"])
	}
}
