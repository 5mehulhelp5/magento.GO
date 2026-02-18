package inventory

import (
	"database/sql"

	"gorm.io/gorm"

	inventoryEntity "magento.GO/model/entity/inventory"
)

type InventoryRepository struct {
	db    *gorm.DB
	sqlDB *sql.DB
}

func NewInventoryRepository(db *gorm.DB) (*InventoryRepository, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	return &InventoryRepository{db: db, sqlDB: sqlDB}, nil
}

// GetQuantityBySKU returns stock quantity for SKU from specific source
// Uses raw SQL for minimal overhead
func (r *InventoryRepository) GetQuantityBySKU(sku, sourceCode string) (float64, bool) {
	const query = `SELECT quantity FROM inventory_source_item WHERE sku = ? AND source_code = ? LIMIT 1`
	var qty sql.NullFloat64
	if err := r.sqlDB.QueryRow(query, sku, sourceCode).Scan(&qty); err != nil || !qty.Valid {
		return 0, false
	}
	return qty.Float64, true
}

// GetBySourceAndSKU returns full entity using GORM
func (r *InventoryRepository) GetBySourceAndSKU(sourceCode, sku string) (*inventoryEntity.InventorySourceItem, error) {
	var item inventoryEntity.InventorySourceItem
	err := r.db.Where("source_code = ? AND sku = ?", sourceCode, sku).First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// GetAllBySKU returns inventory for a SKU across all sources
func (r *InventoryRepository) GetAllBySKU(sku string) ([]inventoryEntity.InventorySourceItem, error) {
	var items []inventoryEntity.InventorySourceItem
	err := r.db.Where("sku = ?", sku).Find(&items).Error
	return items, err
}

// GetTotalQuantityBySKU sums quantity across all sources for a SKU
func (r *InventoryRepository) GetTotalQuantityBySKU(sku string) (float64, error) {
	const query = `SELECT COALESCE(SUM(quantity), 0) FROM inventory_source_item WHERE sku = ?`
	var total float64
	err := r.sqlDB.QueryRow(query, sku).Scan(&total)
	return total, err
}

// BatchGetQuantities fetches quantities for multiple SKUs in one query
func (r *InventoryRepository) BatchGetQuantities(skus []string, sourceCode string) (map[string]float64, error) {
	if len(skus) == 0 {
		return nil, nil
	}

	result := make(map[string]float64, len(skus))
	rows, err := r.db.Table("inventory_source_item").
		Select("sku, quantity").
		Where("source_code = ? AND sku IN ?", sourceCode, skus).
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var sku string
		var qty float64
		if err := rows.Scan(&sku, &qty); err != nil {
			continue
		}
		result[sku] = qty
	}
	return result, nil
}
