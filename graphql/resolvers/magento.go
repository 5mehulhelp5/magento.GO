package resolvers

import (
	"encoding/base64"
	"strconv"

	categoryEntity "magento.GO/model/entity/category"
	gqlmodels "magento.GO/graphql/models"
)

// uidEncode returns base64(entity_id) - Magento's uid format
func uidEncode(entityID uint) string {
	return base64.StdEncoding.EncodeToString([]byte(strconv.FormatUint(uint64(entityID), 10)))
}

// uidDecode parses Magento uid (base64) to entity_id
func uidDecode(uid string) (uint, bool) {
	b, err := base64.StdEncoding.DecodeString(uid)
	if err != nil {
		return 0, false
	}
	n, err := strconv.ParseUint(string(b), 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(n), true
}

func flatToMagentoProduct(p map[string]interface{}, baseURL string) *gqlmodels.MagentoProduct {
	entityID := uint(toUint(p["entity_id"]))
	sku := ""
	if s, ok := p["sku"].(string); ok {
		sku = s
	}
	name := ""
	if n, ok := p["name"].(string); ok {
		name = n
	}
	urlKey := ""
	if u, ok := p["url_key"].(string); ok {
		urlKey = u
	}

	price := 0.0
	if pr, ok := p["price"].(float64); ok {
		price = pr
	}
	finalPrice := price
	if fp, ok := p["final_price"].(float64); ok {
		finalPrice = fp
	}
	discount := price - finalPrice
	if discount < 0 {
		discount = 0
	}

	stockStatus := "OUT_OF_STOCK"
	if si, ok := p["stock_item"].(map[string]interface{}); ok {
		if inStock, ok := si["is_in_stock"]; ok && toUint(inStock) != 0 {
			stockStatus = "IN_STOCK"
		}
	}

	imgURL := ""
	if img, ok := p["image"].(string); ok && img != "" {
		imgURL = img
	} else if mg, ok := p["media_gallery"].([]map[string]interface{}); ok && len(mg) > 0 {
		if v, ok := mg[0]["value"].(string); ok && v != "" {
			if baseURL != "" {
				imgURL = baseURL + "/media/catalog/product" + v
			} else {
				imgURL = v
			}
		}
	}

	mp := &gqlmodels.MagentoProduct{
		ID:            int32(entityID),
		UID:           uidEncode(entityID),
		SKU:           sku,
		StockStatus:   stockStatus,
		RatingSummary: 0,
		PriceRange: gqlmodels.PriceRange{
			MaximumPrice: gqlmodels.ProductPrice{
				FinalPrice:   gqlmodels.Money{Currency: "USD", Value: finalPrice},
				RegularPrice: gqlmodels.Money{Currency: "USD", Value: price},
			},
		},
	}
	if name != "" {
		mp.Name = &name
	}
	if urlKey != "" {
		mp.URLKey = &urlKey
	}
	if imgURL != "" {
		mp.SmallImage = &gqlmodels.ProductImage{URL: imgURL}
	}
	if discount > 0 {
		mp.PriceRange.MaximumPrice.Discount = &gqlmodels.ProductDiscount{AmountOff: &discount}
	}
	return mp
}

func categoryToCategoryTree(c *categoryEntity.Category, attrs map[string]map[string]interface{}) *gqlmodels.CategoryTree {
	ct := &gqlmodels.CategoryTree{
		UID: uidEncode(c.EntityID),
	}
	getStr := func(code string) *string {
		if a, ok := attrs[code]; ok {
			if v, ok := a["value"].(string); ok && v != "" {
				return &v
			}
		}
		return nil
	}
	ct.MetaTitle = getStr("meta_title")
	ct.MetaKeywords = getStr("meta_keywords")
	ct.MetaDescription = getStr("meta_description")
	ct.URLPath = getStr("url_path")
	ct.URLKey = getStr("url_key")
	if ct.URLKey == nil && ct.URLPath == nil {
		// Fallback from category entity varchars
		for _, v := range c.Varchars {
			if v.Value != "" {
				switch v.AttributeID {
				case 119: // url_key
					ct.URLKey = &v.Value
				}
			}
		}
	}
	if ct.URLPath == nil && ct.URLKey != nil {
		ct.URLPath = ct.URLKey
	}
	return ct
}
