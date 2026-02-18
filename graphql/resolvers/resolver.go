package resolvers

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"

	categoryRepo "magento.GO/model/repository/category"
	productRepo "magento.GO/model/repository/product"

	"magento.GO/graphql"
	gqlregistry "magento.GO/graphql/registry"
)

func init() {
	gqlregistry.RegisterQueryResolverFactory(func(db interface{}) interface{} {
		return &QueryResolver{db: db.(*gorm.DB)}
	})
}

// QueryResolver is the single resolver for all Query fields.
// Methods live in product.go, category.go, search.go, magento_resolver.go.
// New Query fields: use RegisterSchemaExtension + add method on QueryResolver,
// or use _extension for fully dynamic resolvers.
type QueryResolver struct {
	db *gorm.DB
}

const guestGroupID uint = 0

func (r *QueryResolver) storeID(ctx context.Context) uint16 {
	return graphql.StoreIDFromContext(ctx)
}

func (r *QueryResolver) productRepo() *productRepo.ProductRepository {
	return productRepo.GetProductRepository(r.db)
}

func (r *QueryResolver) categoryRepo() *categoryRepo.CategoryRepository {
	return categoryRepo.GetCategoryRepository(r.db)
}

func (r *QueryResolver) searchService() *SearchService {
	return GetSearchService()
}

func paginate(items []map[string]interface{}, currentPage, pageSize int) []map[string]interface{} {
	start := (currentPage - 1) * pageSize
	end := start + pageSize
	if start >= len(items) {
		return []map[string]interface{}{}
	}
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

// Extension dispatches to registered custom resolvers.
func (r *QueryResolver) Extension(ctx context.Context, args struct {
	Name string
	Args *string
}) (*string, error) {
	m := make(map[string]interface{})
	if args.Args != nil && *args.Args != "" {
		_ = json.Unmarshal([]byte(*args.Args), &m)
	}
	out, err := gqlregistry.Resolve(ctx, args.Name, m)
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(out)
	s := string(b)
	return &s, nil
}
