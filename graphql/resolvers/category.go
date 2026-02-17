package resolvers

import (
	"context"
	"strconv"

	gqlmodels "magento.GO/graphql/models"
)

// Categories returns all categories with attributes for the store.
func (r *queryResolver) Categories(ctx context.Context) ([]*gqlmodels.Category, error) {
	cats, err := r.CategoryRepo.FetchAllWithAttributes(r.StoreID)
	if err != nil {
		return nil, err
	}
	result := make([]*gqlmodels.Category, len(cats))
	for i, c := range cats {
		result[i] = categoryToGraphQL(&c)
	}
	return result, nil
}

// Category returns a single category by ID.
func (r *queryResolver) Category(ctx context.Context, id string) (*gqlmodels.Category, error) {
	idUint, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return nil, nil
	}
	cats, err := r.CategoryRepo.GetByIDsWithAttributes([]uint{uint(idUint)}, r.StoreID)
	if err != nil || len(cats) == 0 {
		return nil, nil
	}
	return categoryToGraphQL(&cats[0]), nil
}

// CategoryTree returns the category tree for the store.
func (r *queryResolver) CategoryTree(ctx context.Context) ([]*gqlmodels.Category, error) {
	tree, err := r.CategoryRepo.BuildCategoryTree(r.StoreID, 0)
	if err != nil {
		return nil, err
	}
	return categoryTreeToGraphQL(tree), nil
}
