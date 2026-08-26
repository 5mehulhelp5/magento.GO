package product

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	entity "magento.GO/model/entity"
	categoryEntity "magento.GO/model/entity/category"
)

// categoryEntityTypeID is Magento's eav_entity_type ID for catalog_category.
// There is no eav_entity_type table in this project (entity types are used
// as bare integer constants throughout, e.g. product's entity_type_id=4),
// so this mirrors that convention using the real Magento value.
const categoryEntityTypeID uint16 = 3

var categoryColumns = map[string]bool{"categories": true}

// categoryAssignment is one raw (unresolved) product -> category-path pair
// collected from a CSV row, before the path is turned into a category_id.
type categoryAssignment struct {
	ProductID uint
	Path      []string // e.g. ["Default Category", "Shoes", "Running"]
}

// categoryData holds collected category assignments ready to flush.
type categoryData struct {
	assignments []categoryAssignment
	warnings    []string
}

// collectCategories parses the "categories" column: a comma-separated list
// of "/"-delimited category paths (e.g.
// "Default Category/Shoes,Default Category/Sale"), matching Magento/Magmi's
// own CSV convention for on-the-fly category assignment.
func collectCategories(rows [][]string, colIndex map[string]int, skuToID map[string]uint) *categoryData {
	d := &categoryData{}
	ci, ok := colIndex["categories"]
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

		for _, rawPath := range strings.Split(val, ",") {
			rawPath = strings.TrimSpace(rawPath)
			if rawPath == "" {
				continue
			}
			segments := make([]string, 0, 4)
			for _, seg := range strings.Split(rawPath, "/") {
				seg = strings.TrimSpace(seg)
				if seg != "" {
					segments = append(segments, seg)
				}
			}
			if len(segments) == 0 {
				d.warnings = append(d.warnings, fmt.Sprintf("sku=%s: empty category path %q, skipping", sku, rawPath))
				continue
			}
			d.assignments = append(d.assignments, categoryAssignment{ProductID: productID, Path: segments})
		}
	}
	return d
}

// flushCategories resolves each unique category path to a category_id --
// creating any missing category in the path along the way (Magmi's
// "on the fly category creator/importer") -- then upserts the resulting
// product/category assignments into catalog_category_product.
func flushCategories(db *gorm.DB, d *categoryData, opts ImportOptions) error {
	if len(d.assignments) == 0 {
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		nameAttrID, err := findOrCreateCategoryNameAttribute(tx)
		if err != nil {
			return fmt.Errorf("category name attribute: %w", err)
		}

		root, err := findOrCreateRootCategory(tx)
		if err != nil {
			return fmt.Errorf("root category: %w", err)
		}

		// Memoize path -> category_id for this run: many rows commonly
		// share the same category path, and each resolution costs a few
		// queries, so re-walking it per row would multiply that cost by
		// the number of assigned products instead of the number of
		// distinct paths.
		pathCache := make(map[string]uint)
		links := make([]categoryEntity.CategoryProduct, 0, len(d.assignments))

		for _, a := range d.assignments {
			key := strings.Join(a.Path, "\x00")
			categoryID, ok := pathCache[key]
			if !ok {
				categoryID, err = resolveOrCreateCategoryPath(tx, root, a.Path, nameAttrID)
				if err != nil {
					return fmt.Errorf("category path %q: %w", strings.Join(a.Path, "/"), err)
				}
				pathCache[key] = categoryID
			}
			links = append(links, categoryEntity.CategoryProduct{CategoryID: categoryID, ProductID: a.ProductID})
		}

		upsert := clause.OnConflict{
			Columns:   []clause.Column{{Name: "category_id"}, {Name: "product_id"}},
			DoNothing: true,
		}
		return tx.Clauses(upsert).CreateInBatches(links, opts.BatchSize).Error
	})
}

// findOrCreateCategoryNameAttribute returns the attribute_id backing
// categories' "name" attribute, creating the eav_attribute row the first
// time a category import runs against a database that has none yet.
func findOrCreateCategoryNameAttribute(tx *gorm.DB) (uint16, error) {
	attr := entity.EavAttribute{EntityTypeID: categoryEntityTypeID, AttributeCode: "name", BackendType: "varchar"}
	if err := tx.Where("entity_type_id = ? AND attribute_code = ?", categoryEntityTypeID, "name").
		FirstOrCreate(&attr).Error; err != nil {
		return 0, err
	}
	return attr.AttributeID, nil
}

// findOrCreateRootCategory returns the top-level category (parent_id=0),
// creating one if the database has no categories at all yet.
func findOrCreateRootCategory(tx *gorm.DB) (categoryEntity.Category, error) {
	var root categoryEntity.Category
	err := tx.Where("parent_id = ?", 0).Order("entity_id").First(&root).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		root = categoryEntity.Category{AttributeSetID: 3, ParentID: 0, Path: "1", Level: 0, Position: 0}
		if err := tx.Create(&root).Error; err != nil {
			return categoryEntity.Category{}, err
		}
		return root, nil
	}
	if err != nil {
		return categoryEntity.Category{}, err
	}
	return root, nil
}

// resolveOrCreateCategoryPath walks path under root, creating any missing
// category segment, and returns the leaf category's entity_id. An existing
// category is matched by (parent_id, name) via the category name attribute
// -- there's no uniqueness constraint enforcing distinct sibling names, so
// the first match wins, same as Magento's own CSV importer.
func resolveOrCreateCategoryPath(tx *gorm.DB, root categoryEntity.Category, path []string, nameAttrID uint16) (uint, error) {
	parent := root

	for _, name := range path {
		var existingID uint
		if err := tx.Raw(`
			SELECT c.entity_id FROM catalog_category_entity c
			JOIN catalog_category_entity_varchar v ON v.entity_id = c.entity_id
			WHERE c.parent_id = ? AND v.attribute_id = ? AND v.store_id = 0 AND v.value = ?
			LIMIT 1`, parent.EntityID, nameAttrID, name).Scan(&existingID).Error; err != nil {
			return 0, err
		}

		if existingID != 0 {
			var next categoryEntity.Category
			if err := tx.First(&next, existingID).Error; err != nil {
				return 0, err
			}
			parent = next
			continue
		}

		newCat := categoryEntity.Category{
			AttributeSetID: 3,
			ParentID:       parent.EntityID,
			Level:          parent.Level + 1,
			Position:       0,
		}
		if err := tx.Create(&newCat).Error; err != nil {
			return 0, err
		}
		newCat.Path = fmt.Sprintf("%s/%d", parent.Path, newCat.EntityID)
		if err := tx.Model(&newCat).Update("path", newCat.Path).Error; err != nil {
			return 0, err
		}
		if err := tx.Model(&categoryEntity.Category{}).Where("entity_id = ?", parent.EntityID).
			UpdateColumn("children_count", gorm.Expr("children_count + 1")).Error; err != nil {
			return 0, err
		}
		if err := tx.Create(&categoryEntity.CategoryVarchar{
			AttributeID: nameAttrID,
			StoreID:     0,
			EntityID:    newCat.EntityID,
			Value:       name,
		}).Error; err != nil {
			return 0, err
		}

		parent = newCat
	}

	return parent.EntityID, nil
}
