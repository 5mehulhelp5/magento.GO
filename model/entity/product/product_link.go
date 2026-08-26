package product

// Link type IDs, matching real Magento's Magento\Catalog\Model\Product\Link
// constants.
const (
	LinkTypeRelated   uint16 = 1
	LinkTypeGrouped   uint16 = 3
	LinkTypeUpSell    uint16 = 4
	LinkTypeCrossSell uint16 = 5
)

type ProductLink struct {
    LinkID         uint `gorm:"column:link_id;primaryKey;autoIncrement"`
    ProductID      uint `gorm:"column:product_id;type:int unsigned;not null;default:0;uniqueIndex:idx_product_link_unq"`
    LinkedProductID uint `gorm:"column:linked_product_id;type:int unsigned;not null;default:0;uniqueIndex:idx_product_link_unq"`
    LinkTypeID     uint16 `gorm:"column:link_type_id;type:smallint unsigned;not null;default:0;uniqueIndex:idx_product_link_unq"`
    // Position orders grouped/related/up-sell/cross-sell links for display.
    // Real Magento stores this in a separate EAV-style
    // catalog_product_link_attribute_int table keyed by link_type_id; this
    // project simplifies that to a plain column, consistent with how
    // pricing and stock are already simplified elsewhere here.
    Position uint `gorm:"column:position;type:int unsigned;not null;default:0"`
}

// TableName specifies the table name
func (ProductLink) TableName() string {
    return "catalog_product_link"
}

/* Usage Examples:

1. Create:
   ```go
   prodLink := &ProductLink{
       ProductID: 1,
       LinkedProductID: 2,
       LinkTypeID: 1,
   }
   db.Create(prodLink)
   ```

2. Read:
   ```go
   var prodLink ProductLink
   db.First(&prodLink, linkID)
   ```

3. Update:
   ```go
   db.Model(&prodLink).Update("LinkTypeID", 2)
   ```

4. Delete:
   ```go
   db.Delete(&prodLink)
   ```
*/ 