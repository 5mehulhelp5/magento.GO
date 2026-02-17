package resolvers

import (
	"context"

	gqlmodels "magento.GO/graphql/models"
)

// Products returns paginated product list with guest pricing.
func (r *queryResolver) Products(ctx context.Context, pageSize *int, currentPage *int, skus []string, categoryID *string) (*gqlmodels.ProductSearchResult, error) {
	ps := defaultPageSize(pageSize)
	cp := defaultCurrentPage(currentPage)

	ids := r.resolveProductIDsBySKUs(skus)

	flat, err := r.fetchProductsFlat(ids)
	if err != nil {
		return nil, err
	}

	allItems := filterProductsForGuest(flat, r.CustomerGroupID)
	total := len(allItems)
	items := paginate(allItems, cp, ps)
	products := make([]*gqlmodels.Product, len(items))
	for i, p := range items {
		products[i] = flatToProduct(p)
	}
	totalPages := (total + ps - 1) / ps
	if totalPages < 1 {
		totalPages = 1
	}
	return &gqlmodels.ProductSearchResult{
		Items:      products,
		TotalCount: int32(total),
		PageInfo: &gqlmodels.PageInfo{
			PageSize:    int32(ps),
			CurrentPage: int32(cp),
			TotalPages:  int32(totalPages),
		},
	}, nil
}

// Product returns a single product by SKU or url_key.
func (r *queryResolver) Product(ctx context.Context, sku *string, urlKey *string) (*gqlmodels.Product, error) {
	flat, err := r.ProductRepo.FetchWithAllAttributesFlat(r.StoreID)
	if err != nil {
		return nil, err
	}

	for _, p := range flat {
		if sku != nil {
			if s, ok := p["sku"].(string); ok && s == *sku {
				return flatToProduct(filterPriceForGuest(p, r.CustomerGroupID)), nil
			}
		}
		if urlKey != nil {
			if u, ok := p["url_key"].(string); ok && u == *urlKey {
				return flatToProduct(filterPriceForGuest(p, r.CustomerGroupID)), nil
			}
		}
	}
	return nil, nil
}

func (r *queryResolver) resolveProductIDsBySKUs(skus []string) []uint {
	if len(skus) == 0 {
		return nil
	}
	flat, err := r.ProductRepo.FetchWithAllAttributesFlat(r.StoreID)
	if err != nil {
		return nil
	}
	skuSet := make(map[string]bool)
	for _, s := range skus {
		skuSet[s] = true
	}
	var ids []uint
	for _, p := range flat {
		if sku, ok := p["sku"].(string); ok && skuSet[sku] {
			ids = append(ids, uint(toUint(p["entity_id"])))
		}
	}
	return ids
}

func (r *queryResolver) fetchProductsFlat(ids []uint) (map[uint]map[string]interface{}, error) {
	if len(ids) > 0 {
		return r.ProductRepo.FetchWithAllAttributesFlatByIDs(ids, r.StoreID)
	}
	return r.ProductRepo.FetchWithAllAttributesFlat(r.StoreID)
}
