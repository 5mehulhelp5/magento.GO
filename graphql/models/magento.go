package models

// Magento-compatible types for Venia/PWA GetCategories query

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
	ID           int32        `json:"id"`
	UID          string       `json:"uid"`
	Name         *string      `json:"name,omitempty"`
	PriceRange   PriceRange   `json:"price_range"`
	SKU          string       `json:"sku"`
	SmallImage   *ProductImage `json:"small_image,omitempty"`
	StockStatus  string       `json:"stock_status"`
	RatingSummary float64     `json:"rating_summary"`
	URLKey       *string      `json:"url_key,omitempty"`
}

type SearchResultPageInfo struct {
	TotalPages int32 `json:"total_pages"`
}

type Products struct {
	Items      []*MagentoProduct     `json:"items"`
	PageInfo   SearchResultPageInfo `json:"page_info"`
	TotalCount int32                 `json:"total_count"`
}
