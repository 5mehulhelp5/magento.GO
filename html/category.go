package html

import (
	"net/http"
	"strconv"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
	categoryRepo "magento.GO/model/repository/category"
	"html/template"
	"log"
	parts "magento.GO/html/parts"
	productRepo "magento.GO/model/repository/product"
	"magento.GO/config"
	"time"
	"fmt"
)

type CategoryTemplate struct {
	Templates *template.Template
}

func (t *CategoryTemplate) Render(w http.ResponseWriter, name string, data interface{}, c echo.Context) error {
	return t.Templates.ExecuteTemplate(w, name, data)
}

// RegisterCategoryHTMLRoutes registers HTML routes for category rendering
func RegisterCategoryHTMLRoutes(e *echo.Echo, db *gorm.DB) {
	repo := categoryRepo.GetCategoryRepository(db)
	prodRepo := productRepo.GetProductRepository(db)
	e.GET("/category/:id", func(c echo.Context) error {
		idStr := c.Param("id")
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			return c.String(http.StatusBadRequest, "Invalid category ID")
		}
		start := time.Now()
		cat, flat, err := repo.GetByIDWithAttributesAndFlat(uint(id), 0)
		log.Printf("GetByIDWithAttributesAndFlat took %s", time.Since(start))
		if err != nil || cat == nil {
			return c.String(http.StatusNotFound, "Category not found")
		}
		// Pagination parameters
		limit := 20
		if lStr := c.QueryParam("limit"); lStr != "" {
			if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
				limit = l
			}
		}
		page := 1
		if pStr := c.QueryParam("p"); pStr != "" {
			if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
				page = p
			}
		}
		// Extract product IDs from cat.Products
		var productIDs []uint
		for _, cp := range cat.Products {
			productIDs = append(productIDs, cp.ProductID)
		}
		// Pagination logic
		totalProducts := len(productIDs)
		totalPages := (totalProducts + limit - 1) / limit
		startIdx := (page - 1) * limit
		endIdx := startIdx + limit
		if startIdx > totalProducts {
			startIdx = totalProducts
		}
		if endIdx > totalProducts {
			endIdx = totalProducts
		}
		pagedProductIDs := productIDs[startIdx:endIdx]
		// Fetch product data with attributes
		var products []map[string]interface{}
		if len(pagedProductIDs) > 0 {
			start := time.Now()
			flatProducts, err := prodRepo.FetchWithAllAttributesFlatByIDs(pagedProductIDs)
			log.Printf("FetchWithAllAttributesFlatByIDs took %s", time.Since(start))
			if err == nil {
				for _, id := range pagedProductIDs {
					if prod, ok := flatProducts[id]; ok {
						products = append(products, prod)
					}
				}
			}
		}
		// After calculating totalPages and page
		var pageNumbers []int
		for i := 1; i <= totalPages; i++ {
			pageNumbers = append(pageNumbers, i)
		}
		prevPage := page - 1
		if prevPage < 1 {
			prevPage = 1
		}
		nextPage := page + 1
		if nextPage > totalPages {
			nextPage = totalPages
		}
		tmpl := c.Echo().Renderer.(*Template)
		start = time.Now()
		categoryTree, err := repo.BuildCategoryTree(0, 0)
		log.Printf("BuildCategoryTree took %s", time.Since(start))
		var categoryTreeHTML string
		if err == nil {
			start = time.Now()
			categoryTreeHTML, err = RenderCategoryTreeCached(tmpl.Templates, categoryTree)
			log.Printf("RenderCategoryTreeCached took %s", time.Since(start))
			if err != nil {
				log.Println("Category tree render error:", err)
				categoryTreeHTML = ""
			}
		}
		criticalCSS, err := parts.GetCriticalCSSCached()
		if err != nil {
			criticalCSS = ""
		}
		var title string
		if nameMap, ok := flat["name"]; ok {
			if val, ok := nameMap["Value"].(string); ok {
				title = val
			}
		}
		if title == "" {
			title = fmt.Sprintf("%v", cat.EntityID) // fallback to ID if name is missing
		}
		title = "Category Page - " + title + " - Magento.GO"
		return c.Render(http.StatusOK, "parts/category_layout.html", map[string]interface{}{
			"Category": cat,
			"Attributes": flat,
			"Title": title,
			"Products": products,
			"CriticalCSS": template.CSS(criticalCSS),
			"CategoryTreeHTML": template.HTML(categoryTreeHTML),
			"Page": page,
			"TotalPages": totalPages,
			"Limit": limit,
			"MediaUrl": config.AppConfig.MediaUrl,
			"PageNumbers": pageNumbers,
			"PrevPage": prevPage,
			"NextPage": nextPage,
		})
	})
}

// CategoryTemplateFuncs returns FuncMap with helpers for pagination
func CategoryTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"until": func(count int) []int {
			s := make([]int, count)
			for i := 0; i < count; i++ {
				s[i] = i
			}
			return s
		},
	}
} 