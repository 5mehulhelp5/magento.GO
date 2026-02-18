package graphqltest

import (
	"context"

	gql "github.com/graph-gophers/graphql-go"
	"magento.GO/graphql"
	gqlmodels "magento.GO/graphql/models"
)

// MockRootResolver is the root for graphql-go tests (no DB).
type MockRootResolver struct{}

func (m *MockRootResolver) Query() *MockQueryResolver {
	return &MockQueryResolver{}
}

type MockQueryResolver struct{}

type mockProductsArgs struct {
	PageSize    int32
	CurrentPage int32
	Skus       *[]string
	CategoryID *string
}

func (m *MockQueryResolver) Products(ctx context.Context, args mockProductsArgs) (*gqlmodels.ProductSearchResult, error) {
	name := "Mock Product"
	price := 99.99
	return &gqlmodels.ProductSearchResult{
		Items:      []*gqlmodels.Product{{EntityID: "1", SKU: "MOCK-SKU-1", Name: &name, Price: &price}},
		TotalCount: 1,
		PageInfo:   &gqlmodels.PageInfo{PageSize: 20, CurrentPage: 1, TotalPages: 1},
	}, nil
}

type mockProductArgs struct {
	Sku    *string
	URLKey *string
}

func (m *MockQueryResolver) Product(ctx context.Context, args mockProductArgs) (*gqlmodels.Product, error) {
	name := "Mock Single Product"
	price := 49.99
	return &gqlmodels.Product{EntityID: "42", SKU: "MOCK-SINGLE", Name: &name, Price: &price}, nil
}

func (m *MockQueryResolver) Categories(ctx context.Context) ([]*gqlmodels.Category, error) {
	name := "Mock Category"
	parentID := "0"
	return []*gqlmodels.Category{{EntityID: "1", Name: &name, ParentID: &parentID}}, nil
}

type mockCategoryArgs struct {
	ID string
}

func (m *MockQueryResolver) Category(ctx context.Context, args mockCategoryArgs) (*gqlmodels.Category, error) {
	name := "Mock Category By ID"
	parentID := "0"
	return &gqlmodels.Category{EntityID: "1", Name: &name, ParentID: &parentID}, nil
}

func (m *MockQueryResolver) CategoryTree(ctx context.Context) ([]*gqlmodels.Category, error) {
	name := "Mock Tree Root"
	parentID := "0"
	return []*gqlmodels.Category{{EntityID: "1", Name: &name, ParentID: &parentID}}, nil
}

type mockSearchArgs struct {
	Query       string
	PageSize    int32
	CurrentPage int32
	CategoryID  *string
}

func (m *MockQueryResolver) Search(ctx context.Context, args mockSearchArgs) (*gqlmodels.ProductSearchResult, error) {
	name := "Mock Search Result"
	price := 29.99
	return &gqlmodels.ProductSearchResult{
		Items:      []*gqlmodels.Product{{EntityID: "100", SKU: "SEARCH-1", Name: &name, Price: &price}},
		TotalCount: 1,
		PageInfo:   &gqlmodels.PageInfo{PageSize: 20, CurrentPage: 1, TotalPages: 1},
	}, nil
}

func (m *MockQueryResolver) MagentoCategories(ctx context.Context, args *struct {
	Filters *graphql.MagentoCategoryFilters
}) (*gqlmodels.CategoryResult, error) {
	urlKey := "mock-category"
	return &gqlmodels.CategoryResult{
		Items: []*gqlmodels.CategoryTree{{UID: "MQ==", URLKey: &urlKey}},
	}, nil
}

func (m *MockQueryResolver) MagentoProducts(ctx context.Context, args graphql.MagentoProductsArgs) (*gqlmodels.Products, error) {
	name := "Mock Product"
	urlKey := "mock-product"
	return &gqlmodels.Products{
		Items: []*gqlmodels.MagentoProduct{{
			ID: int32(1), UID: "MQ==", Name: &name, SKU: "MOCK",
			PriceRange: gqlmodels.PriceRange{MaximumPrice: gqlmodels.ProductPrice{
				FinalPrice:   gqlmodels.Money{Currency: "USD", Value: 99.99},
				RegularPrice: gqlmodels.Money{Currency: "USD", Value: 99.99},
			}},
			StockStatus: "IN_STOCK", RatingSummary: 0, URLKey: &urlKey,
		}},
		PageInfo:   gqlmodels.SearchResultPageInfo{TotalPages: int32(1)},
		TotalCount: int32(1),
	}, nil
}

type mockExtensionArgs struct {
	Name string
	Args *string
}

func (m *MockQueryResolver) Extension(ctx context.Context, args mockExtensionArgs) (*string, error) {
	s := `{"mock":"ok"}`
	return &s, nil
}

// NewMockSchema creates a schema with mock resolvers for tests.
func NewMockSchema() *gql.Schema {
	schema, err := gql.ParseSchema(graphql.Schema(), &MockRootResolver{}, gql.UseFieldResolvers())
	if err != nil {
		panic("mock schema: " + err.Error())
	}
	return schema
}
