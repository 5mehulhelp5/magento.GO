package models

// --- Product ---

type Product struct {
	EntityID         string                 `json:"entity_id" mapstructure:"entity_id"`
	SKU              string                 `json:"sku" mapstructure:"sku"`
	Name             *string                `json:"name,omitempty" mapstructure:"name"`
	Price            *float64               `json:"price,omitempty" mapstructure:"price"`
	FinalPrice       *float64               `json:"final_price,omitempty" mapstructure:"final_price"`
	URLKey           *string                `json:"url_key,omitempty" mapstructure:"url_key"`
	Image            *string                `json:"image,omitempty" mapstructure:"image"`
	ShortDescription *string                `json:"short_description,omitempty" mapstructure:"short_description"`
	Description      *string                `json:"description,omitempty" mapstructure:"description"`
	IsInStock        *bool                  `json:"is_in_stock,omitempty" mapstructure:"is_in_stock"`
	Qty              *float64               `json:"qty,omitempty" mapstructure:"qty"`
	TypeID           *string                `json:"type_id,omitempty" mapstructure:"type_id"`
	CategoryIDs      *[]string              `json:"category_ids,omitempty" mapstructure:"category_ids"`
	MediaGallery     *[]*MediaGalleryItem   `json:"media_gallery,omitempty" mapstructure:"media_gallery"`
	Attributes       map[string]interface{} `json:"attributes,omitempty" mapstructure:"-"`
}

type MediaGalleryItem struct {
	ValueID   string  `json:"value_id" mapstructure:"value_id"`
	Value     string  `json:"value" mapstructure:"value"`
	MediaType *string `json:"media_type,omitempty" mapstructure:"media_type"`
	Disabled  *bool   `json:"disabled,omitempty" mapstructure:"disabled"`
}

// --- Category ---

type Category struct {
	EntityID     string       `json:"entity_id"`
	Name         *string      `json:"name,omitempty"`
	URLKey       *string      `json:"url_key,omitempty"`
	Path         *string      `json:"path,omitempty"`
	Level        *int32       `json:"level,omitempty"`
	ParentID     *string      `json:"parent_id"`
	Children     *[]*Category `json:"children,omitempty"`
	ProductCount *int32       `json:"product_count,omitempty"`
}

// --- Search ---

type ProductSearchResult struct {
	Items      []*Product `json:"items"`
	TotalCount int32      `json:"total_count"`
	PageInfo   *PageInfo  `json:"page_info"`
}

type PageInfo struct {
	PageSize    int32 `json:"page_size"`
	CurrentPage int32 `json:"current_page"`
	TotalPages  int32 `json:"total_pages"`
}

// --- Magento-compatible types (Venia/PWA) ---

type CategoryTree struct {
	UID             string  `json:"uid"`
	MetaTitle       *string `json:"meta_title,omitempty"`
	MetaKeywords    *string `json:"meta_keywords,omitempty"`
	MetaDescription *string `json:"meta_description,omitempty"`
	URLPath         *string `json:"url_path,omitempty"`
	URLKey          *string `json:"url_key,omitempty"`
}

type CategoryResult struct {
	Items []*CategoryTree `json:"items"`
}

type Money struct {
	Currency string  `json:"currency"`
	Value    float64 `json:"value"`
}

type ProductDiscount struct {
	AmountOff *float64 `json:"amount_off,omitempty"`
}

type ProductPrice struct {
	FinalPrice   Money            `json:"final_price"`
	RegularPrice Money            `json:"regular_price"`
	Discount     *ProductDiscount `json:"discount,omitempty"`
}

type PriceRange struct {
	MaximumPrice ProductPrice `json:"maximum_price"`
}

type ProductImage struct {
	URL string `json:"url"`
}

type MagentoProduct struct {
	ID            int32         `json:"id"`
	UID           string        `json:"uid"`
	Name          *string       `json:"name,omitempty"`
	PriceRange    PriceRange    `json:"price_range"`
	SKU           string        `json:"sku"`
	SmallImage    *ProductImage `json:"small_image,omitempty"`
	StockStatus   string        `json:"stock_status"`
	RatingSummary float64       `json:"rating_summary"`
	URLKey        *string       `json:"url_key,omitempty"`
}

type SearchResultPageInfo struct {
	TotalPages int32 `json:"total_pages"`
}

type Products struct {
	Items      []*MagentoProduct    `json:"items"`
	PageInfo   SearchResultPageInfo `json:"page_info"`
	TotalCount int32                `json:"total_count"`
}
