package resolvers

import (
	"strconv"

	categoryEntity "magento.GO/model/entity/category"
	categoryRepo "magento.GO/model/repository/category"

	gqlmodels "magento.GO/graphql/models"
)

func categoryToGraphQL(c *categoryEntity.Category) *gqlmodels.Category {
	return categoryToGraphQLWithAttrs(c, nil)
}

func categoryToGraphQLWithAttrs(c *categoryEntity.Category, attrs map[string]map[string]interface{}) *gqlmodels.Category {
	lvl := int32(c.Level)
	parentID := strconv.FormatUint(uint64(c.ParentID), 10)
	cat := &gqlmodels.Category{
		EntityID: strconv.FormatUint(uint64(c.EntityID), 10),
		Path:     &c.Path,
		Level:    &lvl,
		ParentID: &parentID,
	}
	if attrs != nil {
		if a, ok := attrs["name"]; ok {
			if v, ok := a["value"].(string); ok {
				cat.Name = &v
			}
		}
		if a, ok := attrs["url_key"]; ok {
			if v, ok := a["value"].(string); ok {
				cat.URLKey = &v
			}
		}
	}
	if cat.Name == nil || cat.URLKey == nil {
		for _, v := range c.Varchars {
			if v.Value != "" {
				switch v.AttributeID {
				case 41:
					cat.Name = &v.Value
				case 119:
					cat.URLKey = &v.Value
				}
			}
		}
	}
	pc := int32(len(c.Products))
	cat.ProductCount = &pc
	return cat
}

func categoryTreeToGraphQL(nodes []*categoryRepo.CategoryTreeNode) []*gqlmodels.Category {
	result := make([]*gqlmodels.Category, len(nodes))
	for i, n := range nodes {
		result[i] = categoryToGraphQLWithAttrs(&n.Category, n.Attributes)
		if len(n.Children) > 0 {
			children := categoryTreeToGraphQL(n.Children)
			result[i].Children = &children
		}
	}
	return result
}
