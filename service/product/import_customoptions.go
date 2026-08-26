package product

import (
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"

	productEntity "magento.GO/model/entity/product"
)

var customOptionColumns = map[string]bool{"custom_options": true}

var validOptionTypes = map[string]bool{
	productEntity.OptionTypeField:       true,
	productEntity.OptionTypeArea:        true,
	productEntity.OptionTypeFile:        true,
	productEntity.OptionTypeDate:        true,
	productEntity.OptionTypeDateTime:    true,
	productEntity.OptionTypeTime:        true,
	productEntity.OptionTypeDropDown:    true,
	productEntity.OptionTypeRadio:       true,
	productEntity.OptionTypeCheckbox:    true,
	productEntity.OptionTypeMultiselect: true,
}

type customOptionValue struct {
	Title     string
	Price     float64
	PriceType string
	SKU       *string
	SortOrder int
}

type customOption struct {
	Type          string
	Title         string
	IsRequire     uint16
	Price         float64
	PriceType     string
	SKU           *string
	MaxCharacters *int
	SortOrder     int
	Values        []customOptionValue // only populated for select-type options
}

type productCustomOptions struct {
	ProductID uint
	Options   []customOption
}

// customOptionsData holds collected custom options ready to flush.
type customOptionsData struct {
	products []productCustomOptions
	warnings []string
}

// collectCustomOptions parses the "custom_options" column:
//
//	type:title:required:price:price_type:sku:max_characters[:values]
//
// entries separated by ";". "values" is only meaningful for the four
// select types (drop_down/radio/checkbox/multiselect): a "|"-separated
// list of "title~price~price_type~sku" choices.
//
// Example: "field:Engraving:1:5.00:fixed:ENGRAVE:50;drop_down:Color:1:0:fixed::0:Red~5~fixed~RED|Blue~10~fixed~BLUE"
func collectCustomOptions(rows [][]string, colIndex map[string]int, skuToID map[string]uint) *customOptionsData {
	d := &customOptionsData{}
	ci, ok := colIndex["custom_options"]
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

		var options []customOption
		for _, entry := range strings.Split(val, ";") {
			entry = strings.TrimSpace(entry)
			if entry == "" {
				continue
			}
			opt, warnings, ok := parseCustomOptionEntry(sku, entry)
			d.warnings = append(d.warnings, warnings...)
			if ok {
				options = append(options, opt)
			}
		}
		if len(options) > 0 {
			d.products = append(d.products, productCustomOptions{ProductID: productID, Options: options})
		}
	}
	return d
}

func parseCustomOptionEntry(sku, entry string) (customOption, []string, bool) {
	var warnings []string
	fields := strings.Split(entry, ":")
	if len(fields) < 2 {
		return customOption{}, append(warnings, fmt.Sprintf("sku=%s: malformed custom_options entry %q, want at least type:title", sku, entry)), false
	}

	optType := strings.TrimSpace(fields[0])
	if !validOptionTypes[optType] {
		return customOption{}, append(warnings, fmt.Sprintf("sku=%s: unknown custom option type %q in entry %q", sku, optType, entry)), false
	}
	title := strings.TrimSpace(fields[1])
	if title == "" {
		return customOption{}, append(warnings, fmt.Sprintf("sku=%s: custom option entry %q has no title", sku, entry)), false
	}

	opt := customOption{Type: optType, Title: title, PriceType: "fixed"}

	if len(fields) > 2 && strings.TrimSpace(fields[2]) != "" {
		v, err := strconv.ParseUint(strings.TrimSpace(fields[2]), 10, 16)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("sku=%s: invalid required flag %q in custom option %q", sku, fields[2], title))
		} else {
			opt.IsRequire = uint16(v)
		}
	}
	if len(fields) > 3 && strings.TrimSpace(fields[3]) != "" {
		v, err := strconv.ParseFloat(strings.TrimSpace(fields[3]), 64)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("sku=%s: invalid price %q in custom option %q", sku, fields[3], title))
		} else {
			opt.Price = v
		}
	}
	if len(fields) > 4 && strings.TrimSpace(fields[4]) != "" {
		pt := strings.TrimSpace(fields[4])
		if pt != "fixed" && pt != "percent" {
			warnings = append(warnings, fmt.Sprintf("sku=%s: invalid price_type %q in custom option %q, want fixed or percent", sku, fields[4], title))
		} else {
			opt.PriceType = pt
		}
	}
	if len(fields) > 5 && strings.TrimSpace(fields[5]) != "" {
		s := strings.TrimSpace(fields[5])
		opt.SKU = &s
	}
	if len(fields) > 6 && strings.TrimSpace(fields[6]) != "" {
		v, err := strconv.Atoi(strings.TrimSpace(fields[6]))
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("sku=%s: invalid max_characters %q in custom option %q", sku, fields[6], title))
		} else {
			opt.MaxCharacters = &v
		}
	}

	if productEntity.SelectOptionTypes[optType] {
		if len(fields) <= 7 || strings.TrimSpace(fields[7]) == "" {
			return customOption{}, append(warnings, fmt.Sprintf("sku=%s: select-type custom option %q has no values", sku, title)), false
		}
		for pos, choice := range strings.Split(fields[7], "|") {
			choice = strings.TrimSpace(choice)
			if choice == "" {
				continue
			}
			v, vWarnings, ok := parseCustomOptionValue(sku, title, choice, pos)
			warnings = append(warnings, vWarnings...)
			if ok {
				opt.Values = append(opt.Values, v)
			}
		}
		if len(opt.Values) == 0 {
			return customOption{}, append(warnings, fmt.Sprintf("sku=%s: select-type custom option %q has no valid values", sku, title)), false
		}
	}

	return opt, warnings, true
}

func parseCustomOptionValue(sku, optionTitle, choice string, pos int) (customOptionValue, []string, bool) {
	var warnings []string
	parts := strings.Split(choice, "~")
	title := strings.TrimSpace(parts[0])
	if title == "" {
		return customOptionValue{}, append(warnings, fmt.Sprintf("sku=%s: custom option %q has a value with no title, skipping", sku, optionTitle)), false
	}

	v := customOptionValue{Title: title, PriceType: "fixed", SortOrder: pos}
	if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
		p, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("sku=%s: invalid price %q for value %q of custom option %q", sku, parts[1], title, optionTitle))
		} else {
			v.Price = p
		}
	}
	if len(parts) > 2 && strings.TrimSpace(parts[2]) != "" {
		pt := strings.TrimSpace(parts[2])
		if pt != "fixed" && pt != "percent" {
			warnings = append(warnings, fmt.Sprintf("sku=%s: invalid price_type %q for value %q of custom option %q", sku, parts[2], title, optionTitle))
		} else {
			v.PriceType = pt
		}
	}
	if len(parts) > 3 && strings.TrimSpace(parts[3]) != "" {
		s := strings.TrimSpace(parts[3])
		v.SKU = &s
	}
	return v, warnings, true
}

// flushCustomOptions replaces each affected product's full custom option
// set -- matching how Magento's own admin option editor behaves (a save
// always rewrites the whole set, not a merge), so reimporting the same CSV
// is idempotent rather than accumulating duplicate options every run.
func flushCustomOptions(db *gorm.DB, d *customOptionsData, opts ImportOptions) error {
	if len(d.products) == 0 {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		productIDs := make([]uint, len(d.products))
		for i, p := range d.products {
			productIDs[i] = p.ProductID
		}

		var existingOptionIDs []uint
		if err := tx.Model(&productEntity.ProductOption{}).Where("product_id IN ?", productIDs).
			Pluck("option_id", &existingOptionIDs).Error; err != nil {
			return err
		}
		if len(existingOptionIDs) > 0 {
			if err := tx.Where("option_id IN ?", existingOptionIDs).Delete(&productEntity.ProductOptionTypeValue{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("product_id IN ?", productIDs).Delete(&productEntity.ProductOption{}).Error; err != nil {
			return err
		}

		// Flatten every option across every product into one slice so it
		// can be bulk-inserted in a single batched multi-row INSERT
		// instead of one INSERT per option -- GORM backfills each
		// element's OptionID from LAST_INSERT_ID() plus its offset in the
		// batch, the same consecutive-auto-increment trick
		// insertNewEntities already relies on for catalog_product_entity.
		optionRows := make([]productEntity.ProductOption, 0, d.customOptionCount())
		optionValues := make([][]customOptionValue, 0, d.customOptionCount())
		for _, p := range d.products {
			for _, opt := range p.Options {
				optionRows = append(optionRows, productEntity.ProductOption{
					ProductID: p.ProductID, Type: opt.Type, Title: opt.Title,
					IsRequire: opt.IsRequire, Price: opt.Price, PriceType: opt.PriceType,
					SKU: opt.SKU, MaxCharacters: opt.MaxCharacters, SortOrder: opt.SortOrder,
				})
				optionValues = append(optionValues, opt.Values)
			}
		}
		if len(optionRows) == 0 {
			return nil
		}
		if err := tx.CreateInBatches(&optionRows, opts.BatchSize).Error; err != nil {
			return err
		}

		var valueRows []productEntity.ProductOptionTypeValue
		for i, row := range optionRows {
			for _, v := range optionValues[i] {
				valueRows = append(valueRows, productEntity.ProductOptionTypeValue{
					OptionID: row.OptionID, Title: v.Title, Price: v.Price,
					PriceType: v.PriceType, SKU: v.SKU, SortOrder: v.SortOrder,
				})
			}
		}
		if len(valueRows) > 0 {
			if err := tx.CreateInBatches(&valueRows, opts.BatchSize).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// customOptionCount returns the total number of options across all
// products, for reporting in ImportResult.
func (d *customOptionsData) customOptionCount() int {
	n := 0
	for _, p := range d.products {
		n += len(p.Options)
	}
	return n
}
