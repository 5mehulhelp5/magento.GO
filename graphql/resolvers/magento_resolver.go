package resolvers

import (
	"context"
	"strings"

	"magento.GO/graphql"
	gqlmodels "magento.GO/graphql/models"
)

// MagentoCategories returns categories filtered by uid (Magento GetCategories format).
func (r *queryResolver) MagentoCategories(ctx context.Context, filters *graphql.MagentoCategoryFilters) (*gqlmodels.CategoryResult, error) {
	var ids []uint
	if filters != nil && filters.CategoryUID != nil {
		if filters.CategoryUID.In != nil {
			for _, uid := range *filters.CategoryUID.In {
				if uid != nil && *uid != "" {
					if id, ok := uidDecode(*uid); ok {
						ids = append(ids, id)
					}
				}
			}
		}
		if filters.CategoryUID.Eq != nil {
			if id, ok := uidDecode(*filters.CategoryUID.Eq); ok {
				ids = append(ids, id)
			}
		}
	}
	if len(ids) == 0 {
		return &gqlmodels.CategoryResult{Items: []*gqlmodels.CategoryTree{}}, nil
	}

	cats, flats, err := r.CategoryRepo.GetByIDsWithAttributesAndFlat(ids, r.StoreID)
	if err != nil || len(cats) == 0 {
		return &gqlmodels.CategoryResult{Items: []*gqlmodels.CategoryTree{}}, nil
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

// MagentoProducts returns products with Magento format (filter by category_uid, sort by position).
func (r *queryResolver) MagentoProducts(ctx context.Context, args graphql.MagentoProductsArgs) (*gqlmodels.Products, error) {
	ps := int(args.PageSize)
	cp := int(args.CurrentPage)
	if ps <= 0 {
		ps = 12
	}
	if cp <= 0 {
		cp = 1
	}

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
		ids, err := r.ProductRepo.FetchProductIDsByCategoryWithPosition(categoryID, asc)
		if err != nil {
			return &gqlmodels.Products{Items: []*gqlmodels.MagentoProduct{}, PageInfo: gqlmodels.SearchResultPageInfo{TotalPages: 1}, TotalCount: 0}, nil
		}
		productIDs = ids
	} else {
		flat, err := r.ProductRepo.FetchWithAllAttributesFlat(r.StoreID)
		if err != nil {
			return &gqlmodels.Products{Items: []*gqlmodels.MagentoProduct{}, PageInfo: gqlmodels.SearchResultPageInfo{TotalPages: 1}, TotalCount: 0}, nil
		}
		for id := range flat {
			productIDs = append(productIDs, id)
		}
	}

	flat, err := r.ProductRepo.FetchWithAllAttributesFlatByIDs(productIDs, r.StoreID)
	if err != nil {
		return &gqlmodels.Products{Items: []*gqlmodels.MagentoProduct{}, PageInfo: gqlmodels.SearchResultPageInfo{TotalPages: 1}, TotalCount: 0}, nil
	}
	// Preserve order from productIDs
	allItems := make([]map[string]interface{}, 0, len(productIDs))
	for _, id := range productIDs {
		if p, ok := flat[id]; ok {
			allItems = append(allItems, filterPriceForGuest(p, r.CustomerGroupID))
		}
	}

	total := len(allItems)
	items := paginate(allItems, cp, ps)

	baseURL := "" // TODO: from config if needed
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
