package graphql

import (
	"context"

	gqlmodels "magento.GO/graphql/models"
)

// MagentoFilters for GetCategories-compatible queries
type MagentoCategoryFilters struct {
	CategoryUID *struct {
		In *[]*string
		Eq *string
	}
}

// QueryResolver is the interface for query resolvers (used by resolvers package).
type QueryResolver interface {
	Products(ctx context.Context, pageSize *int, currentPage *int, skus []string, categoryID *string) (*gqlmodels.ProductSearchResult, error)
	Product(ctx context.Context, sku *string, urlKey *string) (*gqlmodels.Product, error)
	Categories(ctx context.Context) ([]*gqlmodels.Category, error)
	Category(ctx context.Context, id string) (*gqlmodels.Category, error)
	CategoryTree(ctx context.Context) ([]*gqlmodels.Category, error)
	Search(ctx context.Context, query string, pageSize *int, currentPage *int, categoryID *string) (*gqlmodels.ProductSearchResult, error)
	MagentoCategories(ctx context.Context, filters *MagentoCategoryFilters) (*gqlmodels.CategoryResult, error)
	MagentoProducts(ctx context.Context, args MagentoProductsArgs) (*gqlmodels.Products, error)
}

type MagentoProductsArgs struct {
	Filter      *struct {
		CategoryUID *struct {
			In *[]*string
			Eq *string
		}
	}
	Sort        *struct{ Position *string }
	PageSize    int32
	CurrentPage int32
}
