package models

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
