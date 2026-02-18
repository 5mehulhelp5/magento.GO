package stock

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"magento.GO/api"
	productService "magento.GO/service/product"
)

func init() {
	api.RegisterModule(RegisterStockRoutes)
}

func RegisterStockRoutes(apiGroup *echo.Group, db *gorm.DB) {
	g := apiGroup.Group("/stock")

	// POST /api/stock/import – bulk stock upsert (auth required via /api middleware)
	g.POST("/import", func(c echo.Context) error {
		start := time.Now()

		var body struct {
			Items     []productService.StockItemInput `json:"items"`
			BatchSize int                             `json:"batch_size"`
		}
		if err := c.Bind(&body); err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		if len(body.Items) == 0 {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "items array is required and must not be empty"})
		}

		res, err := productService.ImportStockJSON(db, body.Items, body.BatchSize)
		duration := time.Since(start).Milliseconds()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error(), "request_duration_ms": duration})
		}

		c.Response().Header().Set("X-Request-Duration-ms", strconv.FormatInt(duration, 10))
		return c.JSON(http.StatusOK, echo.Map{
			"imported":            res.Imported,
			"skipped":             res.Skipped,
			"warnings":            res.Warnings,
			"request_duration_ms": duration,
		})
	})
}
