package price

import (
	"database/sql"
	"sync"

	"gorm.io/gorm"
)

type PriceRepository struct {
	db           *gorm.DB
	sqlDB        *sql.DB
	isEnterprise bool
	schemaOnce   sync.Once
}

func NewPriceRepository(db *gorm.DB) (*PriceRepository, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	return &PriceRepository{db: db, sqlDB: sqlDB}, nil
}

// detectSchema checks if EE schema (row_id) is in use
func (r *PriceRepository) detectSchema() {
	r.schemaOnce.Do(func() {
		var hasRowID bool
		row := r.sqlDB.QueryRow(`
			SELECT COUNT(*) > 0 
			FROM information_schema.COLUMNS 
			WHERE TABLE_NAME = 'catalog_product_entity_decimal' 
			AND COLUMN_NAME = 'row_id'
		`)
		if err := row.Scan(&hasRowID); err != nil {
			hasRowID = false
		}
		r.isEnterprise = hasRowID
	})
}

// GetLowestPriceBySKU returns the lowest price considering base, special, and tier prices
// Uses raw SQL with LEAST() for optimal performance
func (r *PriceRepository) GetLowestPriceBySKU(sku string, customerGroupID int) (float64, bool) {
	r.detectSchema()
	if r.isEnterprise {
		return r.getLowestPriceEE(sku, customerGroupID)
	}
	return r.getLowestPriceCE(sku, customerGroupID)
}

func (r *PriceRepository) getLowestPriceEE(sku string, customerGroupID int) (float64, bool) {
	const query = `
		SELECT LEAST(
			COALESCE(base.value, 999999999),
			COALESCE(special.value, 999999999),
			COALESCE(tier.value, 999999999)
		) AS lowest_price
		FROM catalog_product_entity cpe
		LEFT JOIN catalog_product_entity_decimal base 
			ON base.row_id = cpe.row_id 
			AND base.attribute_id = (SELECT attribute_id FROM eav_attribute WHERE attribute_code = 'price' AND entity_type_id = 4)
			AND base.store_id = 0
		LEFT JOIN catalog_product_entity_decimal special 
			ON special.row_id = cpe.row_id 
			AND special.attribute_id = (SELECT attribute_id FROM eav_attribute WHERE attribute_code = 'special_price' AND entity_type_id = 4)
			AND special.store_id = 0
		LEFT JOIN catalog_product_entity_tier_price tier 
			ON tier.row_id = cpe.row_id 
			AND (tier.customer_group_id = ? OR tier.all_groups = 1)
			AND tier.qty = 1
		WHERE cpe.sku = ?
		LIMIT 1
	`
	return r.execPriceQuery(query, customerGroupID, sku)
}

func (r *PriceRepository) getLowestPriceCE(sku string, customerGroupID int) (float64, bool) {
	const query = `
		SELECT LEAST(
			COALESCE(base.value, 999999999),
			COALESCE(special.value, 999999999),
			COALESCE(tier.value, 999999999)
		) AS lowest_price
		FROM catalog_product_entity cpe
		LEFT JOIN catalog_product_entity_decimal base 
			ON base.entity_id = cpe.entity_id 
			AND base.attribute_id = (SELECT attribute_id FROM eav_attribute WHERE attribute_code = 'price' AND entity_type_id = 4)
			AND base.store_id = 0
		LEFT JOIN catalog_product_entity_decimal special 
			ON special.entity_id = cpe.entity_id 
			AND special.attribute_id = (SELECT attribute_id FROM eav_attribute WHERE attribute_code = 'special_price' AND entity_type_id = 4)
			AND special.store_id = 0
		LEFT JOIN catalog_product_entity_tier_price tier 
			ON tier.entity_id = cpe.entity_id 
			AND (tier.customer_group_id = ? OR tier.all_groups = 1)
			AND tier.qty = 1
		WHERE cpe.sku = ?
		LIMIT 1
	`
	return r.execPriceQuery(query, customerGroupID, sku)
}

func (r *PriceRepository) execPriceQuery(query string, customerGroupID int, sku string) (float64, bool) {
	var price sql.NullFloat64
	if err := r.sqlDB.QueryRow(query, customerGroupID, sku).Scan(&price); err != nil || !price.Valid {
		return 0, false
	}
	if price.Float64 >= 999999999 {
		return 0, false
	}
	return price.Float64, true
}

// GetBasePriceBySKU returns only the base price
func (r *PriceRepository) GetBasePriceBySKU(sku string) (float64, bool) {
	r.detectSchema()
	linkCol := "entity_id"
	if r.isEnterprise {
		linkCol = "row_id"
	}

	query := `
		SELECT d.value
		FROM catalog_product_entity cpe
		JOIN catalog_product_entity_decimal d ON d.` + linkCol + ` = cpe.` + linkCol + `
		WHERE cpe.sku = ?
		AND d.attribute_id = (SELECT attribute_id FROM eav_attribute WHERE attribute_code = 'price' AND entity_type_id = 4)
		AND d.store_id = 0
		LIMIT 1
	`
	var price sql.NullFloat64
	if err := r.sqlDB.QueryRow(query, sku).Scan(&price); err != nil || !price.Valid {
		return 0, false
	}
	return price.Float64, true
}

// GetTierPricesBySKU returns all tier prices for a SKU
func (r *PriceRepository) GetTierPricesBySKU(sku string) ([]TierPriceResult, error) {
	r.detectSchema()
	linkCol := "entity_id"
	if r.isEnterprise {
		linkCol = "row_id"
	}

	query := `
		SELECT t.customer_group_id, t.qty, t.value, t.all_groups, t.website_id
		FROM catalog_product_entity cpe
		JOIN catalog_product_entity_tier_price t ON t.` + linkCol + ` = cpe.` + linkCol + `
		WHERE cpe.sku = ?
		ORDER BY t.qty ASC
	`
	rows, err := r.sqlDB.Query(query, sku)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TierPriceResult
	for rows.Next() {
		var tp TierPriceResult
		if err := rows.Scan(&tp.CustomerGroupID, &tp.Qty, &tp.Value, &tp.AllGroups, &tp.WebsiteID); err != nil {
			continue
		}
		results = append(results, tp)
	}
	return results, nil
}

// TierPriceResult holds tier price query result
type TierPriceResult struct {
	CustomerGroupID uint16  `json:"customer_group_id"`
	Qty             float64 `json:"qty"`
	Value           float64 `json:"value"`
	AllGroups       uint8   `json:"all_groups"`
	WebsiteID       uint16  `json:"website_id"`
}

// IsEnterprise returns whether EE schema is detected
func (r *PriceRepository) IsEnterprise() bool {
	r.detectSchema()
	return r.isEnterprise
}
