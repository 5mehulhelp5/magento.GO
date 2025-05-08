package html

import (
	"net/http"
	"strconv"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
	productRepo "GO/model/repository/product"
	"html/template"
	"io"
	"log"
	"fmt"
	"GO/config"
)

type Template struct {
	Templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.Templates.ExecuteTemplate(w, name, data)
}

// RegisterProductHTMLRoutes registers HTML routes for product rendering
func RegisterProductHTMLRoutes(e *echo.Echo, db *gorm.DB) {
	repo := productRepo.NewProductRepository(db)
	e.GET("/product/:id", func(c echo.Context) error {
		id := uint(1)
		idStr := c.Param("id")
		if idStr != "" {
			if parsed, err := strconv.Atoi(idStr); err == nil {
				id = uint(parsed)
			}
		}
		flatProducts, err := repo.FetchWithAllAttributesFlatByIDs([]uint{id})
		fmt.Printf("flatProducts: %#v\n", flatProducts)
		if err != nil {
			log.Println("Repo error:", err)
			return c.String(http.StatusInternalServerError, "Error fetching product")
		}
		flatProduct, ok := flatProducts[id]
		if !ok {
			return c.String(http.StatusNotFound, "Product not found")
		}
		return c.Render(http.StatusOK, "products.html", map[string]interface{}{
			"Product": flatProduct,
			"MediaUrl": config.AppConfig.MediaUrl,
		})
	})
} 