package servicetest

import (
	"strings"
	"testing"

	productEntity "magento.GO/model/entity/product"
	productService "magento.GO/service/product"
)

func TestImport_Downloadable_LinksAndSamples(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,downloadable_links,downloadable_samples\n" +
		"SKU-A,\"Album MP3:9.99:0:https://example.com/album.zip;Bonus Track:1.99:5:https://example.com/bonus.mp3\",Preview Clip:https://example.com/preview.mp3\n"

	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("warnings = %v, want none", res.Warnings)
	}
	if res.EAVCounts["downloadable_links"] != 2 {
		t.Errorf("downloadable_links = %d, want 2", res.EAVCounts["downloadable_links"])
	}
	if res.EAVCounts["downloadable_samples"] != 1 {
		t.Errorf("downloadable_samples = %d, want 1", res.EAVCounts["downloadable_samples"])
	}

	var links []productEntity.DownloadableLink
	db.Order("link_id").Find(&links)
	if len(links) != 2 {
		t.Fatalf("link rows = %d, want 2", len(links))
	}
	if links[0].Title != "Album MP3" || links[0].Price != 9.99 || links[0].NumberOfDownloads != 0 {
		t.Errorf("links[0] = %+v, unexpected", links[0])
	}
	if links[0].LinkURL == nil || *links[0].LinkURL != "https://example.com/album.zip" {
		t.Errorf("links[0].LinkURL = %v, want album.zip URL", links[0].LinkURL)
	}
	if links[1].Title != "Bonus Track" || links[1].Price != 1.99 || links[1].NumberOfDownloads != 5 {
		t.Errorf("links[1] = %+v, unexpected", links[1])
	}

	var samples []productEntity.DownloadableSample
	db.Find(&samples)
	if len(samples) != 1 {
		t.Fatalf("sample rows = %d, want 1", len(samples))
	}
	if samples[0].Title != "Preview Clip" {
		t.Errorf("samples[0].Title = %q, want Preview Clip", samples[0].Title)
	}
	if samples[0].SampleURL == nil || *samples[0].SampleURL != "https://example.com/preview.mp3" {
		t.Errorf("samples[0].SampleURL = %v, unexpected", samples[0].SampleURL)
	}
}

func TestImport_Downloadable_ReimportReplaces(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv1 := "sku,downloadable_links\nSKU-A,Old Track:5:0:https://example.com/old.mp3\n"
	if _, err := productService.ImportProducts(db, strings.NewReader(csv1), productService.ImportOptions{}); err != nil {
		t.Fatalf("first import: %v", err)
	}
	csv2 := "sku,downloadable_links\nSKU-A,New Track:5:0:https://example.com/new.mp3\n"
	if _, err := productService.ImportProducts(db, strings.NewReader(csv2), productService.ImportOptions{}); err != nil {
		t.Fatalf("second import: %v", err)
	}

	var links []productEntity.DownloadableLink
	db.Find(&links)
	if len(links) != 1 {
		t.Fatalf("link rows after reimport = %d, want 1 (full replace)", len(links))
	}
	if links[0].Title != "New Track" {
		t.Errorf("Title = %q, want New Track", links[0].Title)
	}
}

func TestImport_Downloadable_MissingTitleWarnsAndSkips(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,downloadable_links\nSKU-A,\":5:0:https://example.com/x.mp3\"\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["downloadable_links"] != 0 {
		t.Errorf("downloadable_links = %d, want 0", res.EAVCounts["downloadable_links"])
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "no title") {
		t.Fatalf("warnings = %v, want exactly 1 mentioning missing title", res.Warnings)
	}
}

func TestImport_Downloadable_InvalidNumericFieldsWarnButLinkStillCreated(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,downloadable_links\nSKU-A,Track:not-a-number:not-a-number\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["downloadable_links"] != 1 {
		t.Fatalf("downloadable_links = %d, want 1", res.EAVCounts["downloadable_links"])
	}
	if len(res.Warnings) != 2 {
		t.Fatalf("warnings = %v, want 2 (price, number_of_downloads)", res.Warnings)
	}

	var link productEntity.DownloadableLink
	db.First(&link)
	if link.Price != 0 || link.NumberOfDownloads != 0 {
		t.Errorf("link = %+v, want defaults for both bad numeric fields", link)
	}
}

func TestImport_Downloadable_NoColumnsIsANoOp(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,name\nSKU-A,Widget\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["downloadable_links"] != 0 || res.EAVCounts["downloadable_samples"] != 0 {
		t.Errorf("downloadable counts = %d/%d, want 0/0", res.EAVCounts["downloadable_links"], res.EAVCounts["downloadable_samples"])
	}
}

func TestImport_Downloadable_UnknownSKUIsSkipped(t *testing.T) {
	db := importDB(t)
	seedAttributes(t, db)

	csv := "sku,downloadable_links\n,Track:5:0\n"
	res, err := productService.ImportProducts(db, strings.NewReader(csv), productService.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportProducts: %v", err)
	}
	if res.EAVCounts["downloadable_links"] != 0 {
		t.Errorf("downloadable_links = %d, want 0", res.EAVCounts["downloadable_links"])
	}
}
