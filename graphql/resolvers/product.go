package resolvers

import (
	"context"

	gqlmodels "magento.GO/graphql/models"
)

func (r *QueryResolver) Products(ctx context.Context, args struct {
	PageSize    int32
	CurrentPage int32
	Skus       *[]string
	CategoryID *string
}) (*gqlmodels.ProductSearchResult, error) {
	ps := int(args.PageSize)
	if ps <= 0 {
		ps = 20
	}
	cp := int(args.CurrentPage)
	if cp <= 0 {
		cp = 1
	}

	repo := r.productRepo()
	storeID := r.storeID(ctx)

	var skus []string
	if args.Skus != nil {
		skus = *args.Skus
	}

	// Resolve IDs by SKU filter
	var ids []uint
	if len(skus) > 0 {
		flat, err := repo.FetchWithAllAttributesFlat(storeID)
		if err == nil {
			skuSet := make(map[string]bool, len(skus))
			for _, s := range skus {
				skuSet[s] = true
			}
			for _, p := range flat {
				if sku, ok := p["sku"].(string); ok && skuSet[sku] {
					ids = append(ids, uint(toUint(p["entity_id"])))
				}
			}
		}
	}

	var flat map[uint]map[string]interface{}
	var err error
	if len(ids) > 0 {
		flat, err = repo.FetchWithAllAttributesFlatByIDs(ids, storeID)
	} else {
		flat, err = repo.FetchWithAllAttributesFlat(storeID)
	}
	if err != nil {
		return nil, err
	}

	allItems := filterProductsForGuest(flat, guestGroupID)
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

func (r *QueryResolver) Product(ctx context.Context, args struct {
	Sku    *string
	URLKey *string
}) (*gqlmodels.Product, error) {
	flat, err := r.productRepo().FetchWithAllAttributesFlat(r.storeID(ctx))
	if err != nil {
		return nil, err
	}
	for _, p := range flat {
		if args.Sku != nil {
			if s, ok := p["sku"].(string); ok && s == *args.Sku {
				return flatToProduct(filterPriceForGuest(p, guestGroupID)), nil
			}
		}
		if args.URLKey != nil {
			if u, ok := p["url_key"].(string); ok && u == *args.URLKey {
				return flatToProduct(filterPriceForGuest(p, guestGroupID)), nil
			}
		}
	}
	return nil, nil
}
