package servicetest

import (
	"strings"
	"testing"

	productEntity "magento.GO/model/entity/product"
	productService "magento.GO/service/product"
)

func TestImport_CustomOptions_SimpleAndSelectTypes(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,custom_options\n" +
		"SKU-A,\"field:Engraving:1:5.00:fixed:ENGRAVE:50;drop_down:Color:1:0:fixed::0:Red~5~fixed~RED|Blue~10~fixed~BLUE\"\n"

	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("warnings = %v, want none", res.Warnings)
	}
	if res.EAVCounts["custom_options"] != 2 {
		t.Fatalf("custom_options = %d, want 2", res.EAVCounts["custom_options"])
	}

	var options []productEntity.ProductOption
	db.Order("option_id").Find(&options)
	if len(options) != 2 {
		t.Fatalf("option rows = %d, want 2", len(options))
	}

	field := options[0]
	if field.Type != "field" || field.Title != "Engraving" || field.IsRequire != 1 || field.Price != 5.00 || field.PriceType != "fixed" {
		t.Errorf("field option = %+v, unexpected", field)
	}
	if field.SKU == nil || *field.SKU != "ENGRAVE" {
		t.Errorf("field.SKU = %v, want ENGRAVE", field.SKU)
	}
	if field.MaxCharacters == nil || *field.MaxCharacters != 50 {
		t.Errorf("field.MaxCharacters = %v, want 50", field.MaxCharacters)
	}

	dropdown := options[1]
	if dropdown.Type != "drop_down" || dropdown.Title != "Color" {
		t.Errorf("dropdown option = %+v, unexpected", dropdown)
	}

	var values []productEntity.ProductOptionTypeValue
	db.Where("option_id = ?", dropdown.OptionID).Order("sort_order").Find(&values)
	if len(values) != 2 {
		t.Fatalf("option type values = %d, want 2", len(values))
	}
	if values[0].Title != "Red" || values[0].Price != 5 {
		t.Errorf("values[0] = %+v, want Red/5", values[0])
	}
	if values[1].Title != "Blue" || values[1].Price != 10 {
		t.Errorf("values[1] = %+v, want Blue/10", values[1])
	}

	// The field option should have no type values.
	var fieldValues []productEntity.ProductOptionTypeValue
	db.Where("option_id = ?", field.OptionID).Find(&fieldValues)
	if len(fieldValues) != 0 {
		t.Errorf("field option has %d type values, want 0", len(fieldValues))
	}
}

func TestImport_CustomOptions_ReimportReplacesFully(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv1 := "sku,custom_options\nSKU-A,\"field:Engraving:1:5.00:fixed::0\"\n"
	if _, err := productService.ImportProducts(db, strings.NewReader(csv1), productService.ImportOptions{}); err != nil {
		t.Fatalf("first import: %v", err)
	}
	csv2 := "sku,custom_options\nSKU-A,\"field:Monogram:0:2.00:fixed::0\"\n"
	if _, err := productService.ImportProducts(db, strings.NewReader(csv2), productService.ImportOptions{}); err != nil {
		t.Fatalf("second import: %v", err)
	}

	var options []productEntity.ProductOption
	db.Find(&options)
	if len(options) != 1 {
		t.Fatalf("option rows after reimport = %d, want 1 (full replace, not accumulation)", len(options))
	}
	if options[0].Title != "Monogram" {
		t.Errorf("Title = %q, want Monogram", options[0].Title)
	}
}

func TestImport_CustomOptions_ReimportReplaceCleansUpOrphanedValues(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv1 := "sku,custom_options\nSKU-A,\"drop_down:Color:0:0:fixed::0:Red~0~fixed~|Blue~0~fixed~\"\n"
	if _, err := productService.ImportProducts(db, strings.NewReader(csv1), productService.ImportOptions{}); err != nil {
		t.Fatalf("first import: %v", err)
	}
	csv2 := "sku,custom_options\nSKU-A,\"field:Note:0:0:fixed::0\"\n"
	if _, err := productService.ImportProducts(db, strings.NewReader(csv2), productService.ImportOptions{}); err != nil {
		t.Fatalf("second import: %v", err)
	}

	var valueCount int64
	db.Model(&productEntity.ProductOptionTypeValue{}).Count(&valueCount)
	if valueCount != 0 {
		t.Errorf("option type values after replace = %d, want 0 (old dropdown's values must be cleaned up)", valueCount)
	}
}

func TestImport_CustomOptions_UnknownTypeWarnsAndSkipsEntry(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,custom_options\nSKU-A,not_a_real_type:Title:0:0:fixed::0\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["custom_options"] != 0 {
		t.Errorf("custom_options = %d, want 0", res.EAVCounts["custom_options"])
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "unknown custom option type") {
		t.Fatalf("warnings = %v, want exactly 1 mentioning unknown type", res.Warnings)
	}
}

func TestImport_CustomOptions_MissingTitleWarnsAndSkipsEntry(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,custom_options\nSKU-A,\"field::0:0:fixed::0\"\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["custom_options"] != 0 {
		t.Errorf("custom_options = %d, want 0", res.EAVCounts["custom_options"])
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "no title") {
		t.Fatalf("warnings = %v, want exactly 1 mentioning missing title", res.Warnings)
	}
}

func TestImport_CustomOptions_SelectTypeWithNoValuesWarnsAndSkips(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,custom_options\nSKU-A,drop_down:Color:0:0:fixed::0\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["custom_options"] != 0 {
		t.Errorf("custom_options = %d, want 0", res.EAVCounts["custom_options"])
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "no values") {
		t.Fatalf("warnings = %v, want exactly 1 mentioning missing values", res.Warnings)
	}
}

func TestImport_CustomOptions_InvalidNumericFieldsWarnButOptionStillCreated(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,custom_options\nSKU-A,\"field:Engraving:not-a-number:not-a-number:invalid-type:ENGRAVE:not-a-number\"\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["custom_options"] != 1 {
		t.Fatalf("custom_options = %d, want 1 (option still created despite bad numeric fields)", res.EAVCounts["custom_options"])
	}
	if len(res.Warnings) != 4 {
		t.Fatalf("warnings = %v, want 4 (required, price, price_type, max_characters)", res.Warnings)
	}

	var opt productEntity.ProductOption
	db.First(&opt)
	if opt.IsRequire != 0 || opt.Price != 0 || opt.PriceType != "fixed" || opt.MaxCharacters != nil {
		t.Errorf("opt = %+v, want all defaults since every numeric field was invalid", opt)
	}
}

func TestImport_CustomOptions_NoColumnIsANoOp(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,name\nSKU-A,Widget\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["custom_options"] != 0 {
		t.Errorf("custom_options = %d, want 0", res.EAVCounts["custom_options"])
	}
}

func TestImport_CustomOptions_UnknownSKUIsSkipped(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,custom_options\n,field:Engraving:0:0:fixed::0\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["custom_options"] != 0 {
		t.Errorf("custom_options = %d, want 0", res.EAVCounts["custom_options"])
	}
}
