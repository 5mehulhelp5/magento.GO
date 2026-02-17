package models

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
