package product

import (
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"

	productEntity "magento.GO/model/entity/product"
)

var bundleColumns = map[string]bool{"bundle_options": true}

type bundleSelection struct {
	ProductID    uint
	Qty          float64
	PriceValue   float64
	PriceType    string
	IsDefault    uint16
	CanChangeQty uint16
}

type bundleOption struct {
	Type       string
	Title      string
	Required   uint16
	Selections []bundleSelection
}

type parentBundleOptions struct {
	ProductID uint
	Options   []bundleOption
}

// bundleData holds collected bundle option/selection rows ready to flush.
type bundleData struct {
	products []parentBundleOptions
	warnings []string
}

func (d *bundleData) optionCount() int {
	n := 0
	for _, p := range d.products {
		n += len(p.Options)
	}
	return n
}

func (d *bundleData) selectionCount() int {
	n := 0
	for _, p := range d.products {
		for _, opt := range p.Options {
			n += len(opt.Selections)
		}
	}
	return n
}

// bundleSelectionSKUs scans the "bundle_options" column across every row
// and returns every distinct component SKU referenced by a selection, so
// ImportProducts can fold them into the same batch SKU lookup used for
// related/upsell/crosssell links -- a bundle component must already exist,
// it isn't created from this column.
func bundleSelectionSKUs(rows [][]string, colIndex map[string]int) []string {
	ci, ok := colIndex["bundle_options"]
	if !ok {
		return nil
	}
	var skus []string
	for _, row := range rows {
		if ci >= len(row) {
			continue
		}
		for _, entry := range strings.Split(row[ci], ";") {
			fields := strings.SplitN(strings.TrimSpace(entry), ":", 4)
			if len(fields) < 4 {
				continue
			}
			for _, sel := range strings.Split(fields[3], "|") {
				parts := strings.SplitN(strings.TrimSpace(sel), "~", 2)
				if sku := strings.TrimSpace(parts[0]); sku != "" {
					skus = append(skus, sku)
				}
			}
		}
	}
	return skus
}

// collectBundleOptions parses the "bundle_options" column:
//
//	type:title:required:selections
//
// entries separated by ";", where "selections" is a "|"-separated list of
// "sku~qty~price_value~price_type~is_default~can_change_qty" component
// entries.
//
// Example: "select:CPU:1:Intel i5~1~0~fixed~1~0|Intel i7~1~50~fixed~0~1"
func collectBundleOptions(rows [][]string, colIndex map[string]int, skuToID map[string]uint) *bundleData {
	d := &bundleData{}
	ci, ok := colIndex["bundle_options"]
	if !ok {
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
		if ci >= len(row) {
			continue
		}
		val := strings.TrimSpace(row[ci])
		if val == "" {
			continue
		}

		var options []bundleOption
		for _, entry := range strings.Split(val, ";") {
			entry = strings.TrimSpace(entry)
			if entry == "" {
				continue
			}
			opt, warnings, ok := parseBundleOptionEntry(sku, entry, skuToID)
			d.warnings = append(d.warnings, warnings...)
			if ok {
				options = append(options, opt)
			}
		}
		if len(options) > 0 {
			d.products = append(d.products, parentBundleOptions{ProductID: productID, Options: options})
		}
	}
	return d
}

func parseBundleOptionEntry(sku, entry string, skuToID map[string]uint) (bundleOption, []string, bool) {
	var warnings []string
	fields := strings.SplitN(entry, ":", 4)
	if len(fields) < 2 {
		return bundleOption{}, append(warnings, fmt.Sprintf("sku=%s: malformed bundle option entry %q, want at least type:title", sku, entry)), false
	}

	optType := strings.TrimSpace(fields[0])
	if !productEntity.ValidBundleOptionTypes[optType] {
		return bundleOption{}, append(warnings, fmt.Sprintf("sku=%s: unknown bundle option type %q in entry %q", sku, optType, entry)), false
	}
	title := strings.TrimSpace(fields[1])
	if title == "" {
		return bundleOption{}, append(warnings, fmt.Sprintf("sku=%s: bundle option entry %q has no title", sku, entry)), false
	}

	opt := bundleOption{Type: optType, Title: title}
	if len(fields) > 2 && strings.TrimSpace(fields[2]) != "" {
		v, err := strconv.ParseUint(strings.TrimSpace(fields[2]), 10, 16)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("sku=%s: invalid required flag %q in bundle option %q", sku, fields[2], title))
		} else {
			opt.Required = uint16(v)
		}
	}

	if len(fields) <= 3 || strings.TrimSpace(fields[3]) == "" {
		return bundleOption{}, append(warnings, fmt.Sprintf("sku=%s: bundle option %q has no selections", sku, title)), false
	}
	for _, sel := range strings.Split(fields[3], "|") {
		sel = strings.TrimSpace(sel)
		if sel == "" {
			continue
		}
		s, selWarnings, ok := parseBundleSelection(sku, title, sel, skuToID)
		warnings = append(warnings, selWarnings...)
		if ok {
			opt.Selections = append(opt.Selections, s)
		}
	}
	if len(opt.Selections) == 0 {
		return bundleOption{}, append(warnings, fmt.Sprintf("sku=%s: bundle option %q has no valid selections", sku, title)), false
	}

	return opt, warnings, true
}

func parseBundleSelection(sku, optionTitle, entry string, skuToID map[string]uint) (bundleSelection, []string, bool) {
	var warnings []string
	parts := strings.Split(entry, "~")
	componentSKU := strings.TrimSpace(parts[0])
	if componentSKU == "" {
		return bundleSelection{}, append(warnings, fmt.Sprintf("sku=%s: bundle option %q has a selection with no SKU, skipping", sku, optionTitle)), false
	}
	componentID, ok := skuToID[componentSKU]
	if !ok {
		return bundleSelection{}, append(warnings, fmt.Sprintf("sku=%s: bundle option %q references unknown SKU %q, skipping", sku, optionTitle, componentSKU)), false
	}

	s := bundleSelection{ProductID: componentID, Qty: 1, PriceType: "fixed", CanChangeQty: 1}
	if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
		v, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("sku=%s: invalid qty %q for %q in bundle option %q", sku, parts[1], componentSKU, optionTitle))
		} else {
			s.Qty = v
		}
	}
	if len(parts) > 2 && strings.TrimSpace(parts[2]) != "" {
		v, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("sku=%s: invalid price_value %q for %q in bundle option %q", sku, parts[2], componentSKU, optionTitle))
		} else {
			s.PriceValue = v
		}
	}
	if len(parts) > 3 && strings.TrimSpace(parts[3]) != "" {
		pt := strings.TrimSpace(parts[3])
		if pt != "fixed" && pt != "percent" {
			warnings = append(warnings, fmt.Sprintf("sku=%s: invalid price_type %q for %q in bundle option %q, want fixed or percent", sku, parts[3], componentSKU, optionTitle))
		} else {
			s.PriceType = pt
		}
	}
	if len(parts) > 4 && strings.TrimSpace(parts[4]) != "" {
		v, err := strconv.ParseUint(strings.TrimSpace(parts[4]), 10, 16)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("sku=%s: invalid is_default %q for %q in bundle option %q", sku, parts[4], componentSKU, optionTitle))
		} else {
			s.IsDefault = uint16(v)
		}
	}
	if len(parts) > 5 && strings.TrimSpace(parts[5]) != "" {
		v, err := strconv.ParseUint(strings.TrimSpace(parts[5]), 10, 16)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("sku=%s: invalid can_change_qty %q for %q in bundle option %q", sku, parts[5], componentSKU, optionTitle))
		} else {
			s.CanChangeQty = uint16(v)
		}
	}
	return s, warnings, true
}

// flushBundleOptions replaces each affected bundle product's full option/
// selection tree -- the same full-replace-on-reimport approach as custom
// options and downloadable links.
func flushBundleOptions(db *gorm.DB, d *bundleData, opts ImportOptions) error {
	if len(d.products) == 0 {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		parentIDs := make([]uint, len(d.products))
		for i, p := range d.products {
			parentIDs[i] = p.ProductID
		}

		var existingOptionIDs []uint
		if err := tx.Model(&productEntity.ProductBundleOption{}).Where("parent_id IN ?", parentIDs).
			Pluck("option_id", &existingOptionIDs).Error; err != nil {
			return err
		}
		if len(existingOptionIDs) > 0 {
			if err := tx.Where("option_id IN ?", existingOptionIDs).Delete(&productEntity.ProductBundleSelection{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("parent_id IN ?", parentIDs).Delete(&productEntity.ProductBundleOption{}).Error; err != nil {
			return err
		}

		// Flatten every option across every product into one slice, same
		// batched-insert-then-backfilled-ID approach as flushCustomOptions:
		// one multi-row INSERT instead of one per option.
		optionRows := make([]productEntity.ProductBundleOption, 0, d.optionCount())
		optionSelections := make([][]bundleSelection, 0, d.optionCount())
		for _, p := range d.products {
			for pos, opt := range p.Options {
				optionRows = append(optionRows, productEntity.ProductBundleOption{
					ParentID: p.ProductID, Required: opt.Required, Position: pos,
					Type: opt.Type, Title: opt.Title,
				})
				optionSelections = append(optionSelections, opt.Selections)
			}
		}
		if len(optionRows) == 0 {
			return nil
		}
		if err := tx.CreateInBatches(&optionRows, opts.BatchSize).Error; err != nil {
			return err
		}

		var selectionRows []productEntity.ProductBundleSelection
		for i, row := range optionRows {
			for pos, sel := range optionSelections[i] {
				selectionRows = append(selectionRows, productEntity.ProductBundleSelection{
					OptionID: row.OptionID, ParentProductID: row.ParentID,
					ProductID: sel.ProductID, Position: pos,
					IsDefault: sel.IsDefault, SelectionQty: sel.Qty,
					SelectionPriceValue: sel.PriceValue, SelectionPriceType: sel.PriceType,
					SelectionCanChangeQty: sel.CanChangeQty,
				})
			}
		}
		if len(selectionRows) > 0 {
			if err := tx.CreateInBatches(&selectionRows, opts.BatchSize).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
