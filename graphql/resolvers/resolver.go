package resolvers

import (
	categoryRepo "magento.GO/model/repository/category"
	productRepo "magento.GO/model/repository/product"
	"gorm.io/gorm"

	"magento.GO/graphql"
)

// Resolver holds shared dependencies for all resolvers.
type Resolver struct {
	DB              *gorm.DB
	ProductRepo     *productRepo.ProductRepository
	CategoryRepo    *categoryRepo.CategoryRepository
	SearchService   *SearchService
	StoreID         uint16
	CustomerGroupID uint // 0 = guest
}

// NewResolver creates a resolver with store context.
func NewResolver(db *gorm.DB, storeID uint16) *Resolver {
	return &Resolver{
		DB:              db,
		ProductRepo:     productRepo.GetProductRepository(db),
		CategoryRepo:    categoryRepo.GetCategoryRepository(db),
		SearchService:   GetSearchService(),
		StoreID:         storeID,
		CustomerGroupID: 0,
	}
}

// Query returns the query resolver.
func (r *Resolver) Query() graphql.QueryResolver {
	return &queryResolver{r}
}

type queryResolver struct{ *Resolver }
