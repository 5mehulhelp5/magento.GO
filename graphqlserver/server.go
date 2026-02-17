package graphqlserver

import (
	"context"
	"encoding/json"

	gql "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
	"gorm.io/gorm"

	"magento.GO/graphql"
	gqlmodels "magento.GO/graphql/models"
	"magento.GO/graphql/registry"
	"magento.GO/graphql/resolvers"
)

// RootResolver is the root for graphql-go. Query resolvers are created dynamically
// per request with store context from headers/variables.
type RootResolver struct {
	DB *gorm.DB
}

// Query returns the query resolver.
func (r *RootResolver) Query() *QueryResolver {
	return &QueryResolver{db: r.DB}
}

// QueryResolver implements Query fields. Delegates to resolvers package.
type QueryResolver struct {
	db *gorm.DB
}

// ProductsArgs matches the products query arguments (defaults in schema: pageSize=20, currentPage=1).
type ProductsArgs struct {
	PageSize    int32
	CurrentPage int32
	Skus       *[]string
	CategoryID *string
}

func (r *QueryResolver) Products(ctx context.Context, args ProductsArgs) (*gqlmodels.ProductSearchResult, error) {
	storeID := graphql.StoreIDFromContext(ctx)
	res := resolvers.NewResolver(r.db, storeID)
	var skus []string
	if args.Skus != nil {
		skus = *args.Skus
	}
	ps, cp := int(args.PageSize), int(args.CurrentPage)
	if ps <= 0 {
		ps = 20
	}
	if cp <= 0 {
		cp = 1
	}
	return res.Query().Products(ctx, &ps, &cp, skus, args.CategoryID)
}

// ProductArgs matches the product query arguments.
type ProductArgs struct {
	Sku    *string
	URLKey *string
}

func (r *QueryResolver) Product(ctx context.Context, args ProductArgs) (*gqlmodels.Product, error) {
	storeID := graphql.StoreIDFromContext(ctx)
	res := resolvers.NewResolver(r.db, storeID)
	return res.Query().Product(ctx, args.Sku, args.URLKey)
}

func (r *QueryResolver) Categories(ctx context.Context) ([]*gqlmodels.Category, error) {
	storeID := graphql.StoreIDFromContext(ctx)
	res := resolvers.NewResolver(r.db, storeID)
	return res.Query().Categories(ctx)
}

// CategoryArgs matches the category query arguments.
type CategoryArgs struct {
	ID string
}

func (r *QueryResolver) Category(ctx context.Context, args CategoryArgs) (*gqlmodels.Category, error) {
	storeID := graphql.StoreIDFromContext(ctx)
	res := resolvers.NewResolver(r.db, storeID)
	return res.Query().Category(ctx, args.ID)
}

func (r *QueryResolver) CategoryTree(ctx context.Context) ([]*gqlmodels.Category, error) {
	storeID := graphql.StoreIDFromContext(ctx)
	res := resolvers.NewResolver(r.db, storeID)
	return res.Query().CategoryTree(ctx)
}

// SearchArgs matches the search query arguments (defaults in schema: pageSize=20, currentPage=1).
type SearchArgs struct {
	Query       string
	PageSize    int32
	CurrentPage int32
	CategoryID  *string
}

func (r *QueryResolver) Search(ctx context.Context, args SearchArgs) (*gqlmodels.ProductSearchResult, error) {
	storeID := graphql.StoreIDFromContext(ctx)
	res := resolvers.NewResolver(r.db, storeID)
	ps, cp := int(args.PageSize), int(args.CurrentPage)
	if ps <= 0 {
		ps = 20
	}
	if cp <= 0 {
		cp = 1
	}
	return res.Query().Search(ctx, args.Query, &ps, &cp, args.CategoryID)
}

func (r *QueryResolver) MagentoCategories(ctx context.Context, args *struct {
	Filters *graphql.MagentoCategoryFilters
}) (*gqlmodels.CategoryResult, error) {
	storeID := graphql.StoreIDFromContext(ctx)
	res := resolvers.NewResolver(r.db, storeID)
	if args == nil || args.Filters == nil {
		return res.Query().MagentoCategories(ctx, nil)
	}
	return res.Query().MagentoCategories(ctx, args.Filters)
}

func (r *QueryResolver) MagentoProducts(ctx context.Context, args graphql.MagentoProductsArgs) (*gqlmodels.Products, error) {
	storeID := graphql.StoreIDFromContext(ctx)
	res := resolvers.NewResolver(r.db, storeID)
	return res.Query().MagentoProducts(ctx, args)
}

// ExtensionArgs for _extension(name, args).
type ExtensionArgs struct {
	Name string
	Args *string
}

func (r *QueryResolver) Extension(ctx context.Context, args ExtensionArgs) (*string, error) {
	var m map[string]interface{}
	if args.Args != nil && *args.Args != "" {
		_ = json.Unmarshal([]byte(*args.Args), &m)
	}
	if m == nil {
		m = make(map[string]interface{})
	}
	out, err := registry.Resolve(ctx, args.Name, m)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	s := string(b)
	return &s, nil
}

// NewSchema parses the schema and returns a graphql-go Schema.
func NewSchema(db *gorm.DB) (*gql.Schema, error) {
	return gql.ParseSchema(graphql.Schema, &RootResolver{DB: db}, gql.UseFieldResolvers())
}

// Handler returns an http.Handler for GraphQL (relay format).
func Handler(schema *gql.Schema) *relay.Handler {
	return &relay.Handler{Schema: schema}
}
