package product

import (
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	priceEntity "magento.GO/model/entity/price"
)

var tierPriceColumns = map[string]bool{"tier_prices": true}

// tierPriceWebsiteID mirrors the single fixed website ID this project's
// simplified price import already uses elsewhere (see collectPrice).
const tierPriceWebsiteID uint16 = 1

// tierPriceData holds collected tier/group price rows ready to flush.
type tierPriceData struct {
	rows     []priceEntity.TierPrice
	warnings []string
}

// collectTierPrices parses the "tier_prices" column: a "|"-separated list of
// "group:qty:price" entries, e.g. "all:5:8.99|1:10:7.50|2:1:9.99". "group" is
// either the literal "all" (applies to every customer group) or a numeric
// customer_group_id. A qty=1 entry for one specific group is what Magento
// calls a "group price" -- there is no separate table for it, tier pricing
// and group pricing are the same mechanism at different qty breaks.
func collectTierPrices(rows [][]string, colIndex map[string]int, skuToID map[string]uint) *tierPriceData {
	d := &tierPriceData{}
	ci, ok := colIndex["tier_prices"]
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
		entityID, ok := skuToID[sku]
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

		for _, entry := range strings.Split(val, "|") {
			entry = strings.TrimSpace(entry)
			if entry == "" {
				continue
			}
			parts := strings.Split(entry, ":")
			if len(parts) != 3 {
				d.warnings = append(d.warnings, fmt.Sprintf("sku=%s: malformed tier_prices entry %q, want group:qty:price", sku, entry))
				continue
			}
			groupRaw := strings.TrimSpace(parts[0])
			qtyRaw := strings.TrimSpace(parts[1])
			priceRaw := strings.TrimSpace(parts[2])

			tp := priceEntity.TierPrice{EntityID: entityID, WebsiteID: tierPriceWebsiteID}
			if strings.EqualFold(groupRaw, "all") {
				tp.AllGroups = 1
				tp.CustomerGroupID = 0
			} else {
				gid, err := strconv.ParseUint(groupRaw, 10, 16)
				if err != nil {
					d.warnings = append(d.warnings, fmt.Sprintf("sku=%s: invalid customer group %q in tier_prices entry %q", sku, groupRaw, entry))
					continue
				}
				tp.AllGroups = 0
				tp.CustomerGroupID = uint16(gid)
			}

			qty, err := strconv.ParseFloat(qtyRaw, 64)
			if err != nil {
				d.warnings = append(d.warnings, fmt.Sprintf("sku=%s: invalid qty %q in tier_prices entry %q", sku, qtyRaw, entry))
				continue
			}
			tp.Qty = qty

			price, err := strconv.ParseFloat(priceRaw, 64)
			if err != nil {
				d.warnings = append(d.warnings, fmt.Sprintf("sku=%s: invalid price %q in tier_prices entry %q", sku, priceRaw, entry))
				continue
			}
			tp.Value = price

			d.rows = append(d.rows, tp)
		}
	}
	return d
}

// flushTierPrices upserts buffered tier/group price rows, keyed on the same
// (entity_id, all_groups, customer_group_id, qty, website_id) tuple Magento
// itself uses to distinguish price breaks.
func flushTierPrices(db *gorm.DB, d *tierPriceData, opts ImportOptions) error {
	if len(d.rows) == 0 {
		return nil
	}
	upsert := clause.OnConflict{
		Columns:   []clause.Column{{Name: "entity_id"}, {Name: "all_groups"}, {Name: "customer_group_id"}, {Name: "qty"}, {Name: "website_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "percentage_value"}),
	}
	return db.Transaction(func(tx *gorm.DB) error {
		return tx.Clauses(upsert).CreateInBatches(d.rows, opts.BatchSize).Error
	})
}
