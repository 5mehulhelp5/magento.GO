package resolvers

import (
	"context"
	"encoding/base64"
	"strconv"
	"strings"

	categoryEntity "magento.GO/model/entity/category"

	"magento.GO/graphql"
	gqlmodels "magento.GO/graphql/models"
)

// --- Magento UID helpers ---

func uidEncode(entityID uint) string {
	return base64.StdEncoding.EncodeToString([]byte(strconv.FormatUint(uint64(entityID), 10)))
}

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
		for _, v := range c.Varchars {
			if v.Value != "" {
				switch v.AttributeID {
				case 119:
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

func (r *QueryResolver) MagentoCategories(ctx context.Context, args *struct {
	Filters *graphql.MagentoCategoryFilters
}) (*gqlmodels.CategoryResult, error) {
	empty := &gqlmodels.CategoryResult{Items: []*gqlmodels.CategoryTree{}}

	if args == nil || args.Filters == nil || args.Filters.CategoryUID == nil {
		return empty, nil
	}

	var ids []uint
	f := args.Filters.CategoryUID
	if f.In != nil {
		for _, uid := range *f.In {
			if uid != nil && *uid != "" {
				if id, ok := uidDecode(*uid); ok {
					ids = append(ids, id)
				}
			}
		}
	}
	if f.Eq != nil {
		if id, ok := uidDecode(*f.Eq); ok {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return empty, nil
	}

	cats, flats, err := r.categoryRepo().GetByIDsWithAttributesAndFlat(ids, r.storeID(ctx))
	if err != nil || len(cats) == 0 {
		return empty, nil
	}

	items := make([]*gqlmodels.CategoryTree, len(cats))
	for i := range cats {
		attrs := map[string]map[string]interface{}{}
		if i < len(flats) {
			attrs = flats[i]
		}
		items[i] = categoryToCategoryTree(&cats[i], attrs)
	}
	return &gqlmodels.CategoryResult{Items: items}, nil
}

func (r *QueryResolver) MagentoProducts(ctx context.Context, args graphql.MagentoProductsArgs) (*gqlmodels.Products, error) {
	emptyResult := &gqlmodels.Products{Items: []*gqlmodels.MagentoProduct{}, PageInfo: gqlmodels.SearchResultPageInfo{TotalPages: 1}, TotalCount: 0}

	ps := int(args.PageSize)
	if ps <= 0 {
		ps = 12
	}
	cp := int(args.CurrentPage)
	if cp <= 0 {
		cp = 1
	}

	repo := r.productRepo()
	storeID := r.storeID(ctx)

	var productIDs []uint
	var categoryID uint
	if args.Filter != nil && args.Filter.CategoryUID != nil {
		if args.Filter.CategoryUID.Eq != nil {
			if id, ok := uidDecode(*args.Filter.CategoryUID.Eq); ok {
				categoryID = id
			}
		}
		if args.Filter.CategoryUID.In != nil && len(*args.Filter.CategoryUID.In) > 0 {
			if u := (*args.Filter.CategoryUID.In)[0]; u != nil && *u != "" {
				if id, ok := uidDecode(*u); ok {
					categoryID = id
				}
			}
		}
	}

	asc := true
	if args.Sort != nil && args.Sort.Position != nil {
		asc = strings.ToUpper(*args.Sort.Position) != "DESC"
	}

	if categoryID > 0 {
		ids, err := repo.FetchProductIDsByCategoryWithPosition(categoryID, asc)
		if err != nil {
			return emptyResult, nil
		}
		productIDs = ids
	} else {
		flat, err := repo.FetchWithAllAttributesFlat(storeID)
		if err != nil {
			return emptyResult, nil
		}
		for id := range flat {
			productIDs = append(productIDs, id)
		}
	}

	flat, err := repo.FetchWithAllAttributesFlatByIDs(productIDs, storeID)
	if err != nil {
		return emptyResult, nil
	}

	allItems := make([]map[string]interface{}, 0, len(productIDs))
	for _, id := range productIDs {
		if p, ok := flat[id]; ok {
			allItems = append(allItems, filterPriceForGuest(p, guestGroupID))
		}
	}

	total := len(allItems)
	items := paginate(allItems, cp, ps)

	baseURL := ""
	magentoItems := make([]*gqlmodels.MagentoProduct, len(items))
	for i, p := range items {
		magentoItems[i] = flatToMagentoProduct(p, baseURL)
	}

	totalPages := (total + ps - 1) / ps
	if totalPages < 1 {
		totalPages = 1
	}
	return &gqlmodels.Products{
		Items:      magentoItems,
		PageInfo:   gqlmodels.SearchResultPageInfo{TotalPages: int32(totalPages)},
		TotalCount: int32(total),
	}, nil
}
