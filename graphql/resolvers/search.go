package resolvers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/elastic/go-elasticsearch/v8"
	productRepo "magento.GO/model/repository/product"

	gqlmodels "magento.GO/graphql/models"
)

var (
	searchServiceInstance *SearchService
	searchServiceOnce     sync.Once
)

// GetSearchService returns singleton SearchService.
func GetSearchService() *SearchService {
	searchServiceOnce.Do(func() {
		searchServiceInstance = NewSearchService()
	})
	return searchServiceInstance
}

type SearchService struct {
	client *elasticsearch.Client
	prefix string
}

func NewSearchService() *SearchService {
	host := os.Getenv("ELASTICSEARCH_HOST")
	if host == "" {
		host = "http://localhost:9200"
	}
	prefix := os.Getenv("ELASTICSEARCH_INDEX_PREFIX")
	if prefix == "" {
		prefix = "magento2"
	}

	cfg := elasticsearch.Config{
		Addresses: []string{host},
	}
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return &SearchService{prefix: prefix}
	}

	return &SearchService{
		client: client,
		prefix: prefix,
	}
}

// Search (resolver) delegates to SearchService.
func (r *queryResolver) Search(ctx context.Context, query string, pageSize *int, currentPage *int, categoryID *string) (*gqlmodels.ProductSearchResult, error) {
	return r.SearchService.Search(ctx, r.StoreID, query, pageSize, currentPage, categoryID, r.ProductRepo, r.CustomerGroupID)
}

// Search queries Magento Elasticsearch index: magento2_catalog_product_{storeID}
func (s *SearchService) Search(
	ctx context.Context,
	storeID uint16,
	query string,
	pageSize *int,
	currentPage *int,
	categoryID *string,
	productRepo *productRepo.ProductRepository,
	customerGroupID uint,
) (*gqlmodels.ProductSearchResult, error) {
	if s.client == nil {
		return nil, fmt.Errorf("elasticsearch not configured")
	}

	ps := 20
	if pageSize != nil && *pageSize > 0 {
		ps = *pageSize
	}
	cp := 1
	if currentPage != nil && *currentPage > 0 {
		cp = *currentPage
	}

	indexName := fmt.Sprintf("%s_catalog_product_%d", s.prefix, storeID)
	if storeID == 0 {
		indexName = "magento2_catalog_product_1"
	}

	from := (cp - 1) * ps

	body := map[string]interface{}{
		"from": from,
		"size": ps,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"multi_match": map[string]interface{}{
							"query":  query,
							"fields": []string{"name^3", "sku^2", "description", "short_description"},
						},
					},
				},
			},
		},
	}

	if categoryID != nil && *categoryID != "" {
		body["query"].(map[string]interface{})["bool"].(map[string]interface{})["filter"] = []map[string]interface{}{
			{"term": map[string]interface{}{"category_ids": *categoryID}},
		}
	}

	bodyBytes, _ := json.Marshal(body)

	res, err := s.client.Search(
		s.client.Search.WithContext(ctx),
		s.client.Search.WithIndex(indexName),
		s.client.Search.WithBody(bytes.NewReader(bodyBytes)),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch error: %s", res.String())
	}

	var esResp struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&esResp); err != nil {
		return nil, err
	}

	var ids []uint
	for _, hit := range esResp.Hits.Hits {
		if entityID, ok := hit.Source["entity_id"].(float64); ok {
			ids = append(ids, uint(entityID))
		}
	}

	flat, err := productRepo.FetchWithAllAttributesFlatByIDs(ids, storeID)
	if err != nil {
		return nil, err
	}

	products := make([]*gqlmodels.Product, 0, len(ids))
	for _, id := range ids {
		if p, ok := flat[id]; ok {
			filtered := filterPriceForGuest(p, customerGroupID)
			products = append(products, flatToProduct(filtered))
		}
	}

	total := esResp.Hits.Total.Value
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
