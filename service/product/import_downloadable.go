package product

import (
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"

	productEntity "magento.GO/model/entity/product"
)

var downloadableColumns = map[string]bool{
	"downloadable_links": true, "downloadable_samples": true,
}

// downloadableData holds collected downloadable links/samples ready to
// flush, grouped per product so a reimport can fully replace each
// product's set (see flushDownloadable).
type downloadableData struct {
	links    []productEntity.DownloadableLink
	samples  []productEntity.DownloadableSample
	touched  map[uint]bool // product IDs whose link/sample set this import provides
	warnings []string
}

// collectDownloadable parses the "downloadable_links" and
// "downloadable_samples" columns.
//
//	downloadable_links:   "title:price:number_of_downloads:url" entries, ";"-separated
//	downloadable_samples: "title:url" entries, ";"-separated
//
// Example: "Album MP3:9.99:0:https://example.com/album.zip;Bonus Track:1.99:5:https://example.com/bonus.mp3"
func collectDownloadable(rows [][]string, colIndex map[string]int, skuToID map[string]uint) *downloadableData {
	d := &downloadableData{touched: make(map[uint]bool)}
	linkCol, hasLinks := colIndex["downloadable_links"]
	sampleCol, hasSamples := colIndex["downloadable_samples"]
	if !hasLinks && !hasSamples {
		return d
	}
	skuCol := colIndex["sku"]

	for _, row := range rows {
		sku := ""
		if skuCol < len(row) {
			sku = strings.TrimSpace(row[skuCol])
		}
		if sku == "" {
			continue
		}
		productID, ok := skuToID[sku]
		if !ok {
			continue
		}

		if hasLinks && linkCol < len(row) {
			if val := strings.TrimSpace(row[linkCol]); val != "" {
				d.touched[productID] = true
				for _, entry := range strings.Split(val, ";") {
					entry = strings.TrimSpace(entry)
					if entry == "" {
						continue
					}
					if link, warnings, ok := parseDownloadableLink(sku, productID, entry); ok {
						d.links = append(d.links, link)
						d.warnings = append(d.warnings, warnings...)
					} else {
						d.warnings = append(d.warnings, warnings...)
					}
				}
			}
		}
		if hasSamples && sampleCol < len(row) {
			if val := strings.TrimSpace(row[sampleCol]); val != "" {
				d.touched[productID] = true
				for _, entry := range strings.Split(val, ";") {
					entry = strings.TrimSpace(entry)
					if entry == "" {
						continue
					}
					if sample, warnings, ok := parseDownloadableSample(sku, productID, entry); ok {
						d.samples = append(d.samples, sample)
						d.warnings = append(d.warnings, warnings...)
					} else {
						d.warnings = append(d.warnings, warnings...)
					}
				}
			}
		}
	}
	return d
}

func parseDownloadableLink(sku string, productID uint, entry string) (productEntity.DownloadableLink, []string, bool) {
	var warnings []string
	// SplitN(..., 4): the URL is always last and may itself contain colons
	// (e.g. "https://..."), so it must not be split further.
	fields := strings.SplitN(entry, ":", 4)
	title := strings.TrimSpace(fields[0])
	if title == "" {
		return productEntity.DownloadableLink{}, append(warnings, fmt.Sprintf("sku=%s: downloadable link entry %q has no title", sku, entry)), false
	}

	link := productEntity.DownloadableLink{ProductID: productID, Title: title}
	if len(fields) > 1 && strings.TrimSpace(fields[1]) != "" {
		v, err := strconv.ParseFloat(strings.TrimSpace(fields[1]), 64)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("sku=%s: invalid price %q for downloadable link %q", sku, fields[1], title))
		} else {
			link.Price = v
		}
	}
	if len(fields) > 2 && strings.TrimSpace(fields[2]) != "" {
		v, err := strconv.Atoi(strings.TrimSpace(fields[2]))
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("sku=%s: invalid number_of_downloads %q for downloadable link %q", sku, fields[2], title))
		} else {
			link.NumberOfDownloads = v
		}
	}
	if len(fields) > 3 && strings.TrimSpace(fields[3]) != "" {
		u := strings.TrimSpace(fields[3])
		link.LinkURL = &u
	}
	return link, warnings, true
}

func parseDownloadableSample(sku string, productID uint, entry string) (productEntity.DownloadableSample, []string, bool) {
	var warnings []string
	// SplitN(..., 2): the URL is always last and may itself contain colons.
	fields := strings.SplitN(entry, ":", 2)
	title := strings.TrimSpace(fields[0])
	if title == "" {
		return productEntity.DownloadableSample{}, append(warnings, fmt.Sprintf("sku=%s: downloadable sample entry %q has no title", sku, entry)), false
	}

	sample := productEntity.DownloadableSample{ProductID: productID, Title: title}
	if len(fields) > 1 && strings.TrimSpace(fields[1]) != "" {
		u := strings.TrimSpace(fields[1])
		sample.SampleURL = &u
	}
	return sample, warnings, true
}

// flushDownloadable replaces each touched product's full link/sample set --
// same full-replace-on-reimport approach as custom options, so reimporting
// the same CSV doesn't accumulate duplicates.
func flushDownloadable(db *gorm.DB, d *downloadableData, opts ImportOptions) error {
	if len(d.touched) == 0 {
		return nil
	}
	productIDs := make([]uint, 0, len(d.touched))
	for id := range d.touched {
		productIDs = append(productIDs, id)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("product_id IN ?", productIDs).Delete(&productEntity.DownloadableLink{}).Error; err != nil {
			return err
		}
		if err := tx.Where("product_id IN ?", productIDs).Delete(&productEntity.DownloadableSample{}).Error; err != nil {
			return err
		}
		if len(d.links) > 0 {
			if err := tx.CreateInBatches(d.links, opts.BatchSize).Error; err != nil {
				return err
			}
		}
		if len(d.samples) > 0 {
			if err := tx.CreateInBatches(d.samples, opts.BatchSize).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
