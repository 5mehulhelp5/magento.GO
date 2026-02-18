package realtime

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"

	"magento.GO/api"
	"magento.GO/config"
	inventoryRepo "magento.GO/model/repository/inventory"
	priceRepo "magento.GO/model/repository/price"
)

func init() {
	api.RegisterModule(RegisterRealtimeRoutes)
}

// Response for price+inventory endpoint
type PriceInventoryResponse struct {
	SKU   string  `json:"sku"`
	Price float64 `json:"price"`
	Stock float64 `json:"stock"`
}

// Singleton repositories (created once per DB)
var (
	priceRepoInstance     *priceRepo.PriceRepository
	inventoryRepoInstance *inventoryRepo.InventoryRepository
	repoOnce              sync.Once
	repoErr               error
)

func getRepositories(db *gorm.DB) (*priceRepo.PriceRepository, *inventoryRepo.InventoryRepository, error) {
	repoOnce.Do(func() {
		priceRepoInstance, repoErr = priceRepo.NewPriceRepository(db)
		if repoErr != nil {
			return
		}
		inventoryRepoInstance, repoErr = inventoryRepo.NewInventoryRepository(db)
	})
	return priceRepoInstance, inventoryRepoInstance, repoErr
}

// getCryptKey returns the Magento crypt key from env
func getCryptKey() string {
	return config.GetEnv("MAGENTO_CRYPT_KEY", "")
}

// verifyCustomerSignature validates HMAC-SHA256 signature using constant-time comparison
func verifyCustomerSignature(customerID, signature, cryptKey string) bool {
	if cryptKey == "" || customerID == "" || signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(cryptKey))
	mac.Write([]byte(customerID))
	expected := mac.Sum(nil)
	sig, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	return hmac.Equal(expected, sig)
}

// RegisterRealtimeRoutes sets up the high-performance realtime pricing/inventory API
func RegisterRealtimeRoutes(apiGroup *echo.Group, db *gorm.DB) {
	g := apiGroup.Group("/realtime")

	// GET /api/realtime/price-inventory?sku=XXX&source=default
	g.GET("/price-inventory", func(c echo.Context) error {
		start := time.Now()

		// Extract and verify customer signature
		customerID := c.Request().Header.Get("X-Customer-ID")
		customerSig := c.Request().Header.Get("X-Customer-Sig")
		cryptKey := getCryptKey()

		if cryptKey != "" && !verifyCustomerSignature(customerID, customerSig, cryptKey) {
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid signature"})
		}

		sku := c.QueryParam("sku")
		if sku == "" {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "sku required"})
		}

		sourceCode := c.QueryParam("source")
		if sourceCode == "" {
			sourceCode = "default"
		}

		customerGroupID := 0
		if customerID != "" {
			if id, err := strconv.Atoi(customerID); err == nil {
				customerGroupID = id
			}
		}

		// Get repositories
		priceR, inventoryR, err := getRepositories(db)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "repository init failed"})
		}

		var price float64
		var priceFound bool
		var stock float64
		var stockFound bool

		// Parallel fetch using errgroup
		eg := new(errgroup.Group)

		eg.Go(func() error {
			price, priceFound = priceR.GetLowestPriceBySKU(sku, customerGroupID)
			return nil
		})

		eg.Go(func() error {
			stock, stockFound = inventoryR.GetQuantityBySKU(sku, sourceCode)
			return nil
		})

		_ = eg.Wait()

		duration := time.Since(start).Milliseconds()
		c.Response().Header().Set("X-Request-Duration-ms", strconv.FormatInt(duration, 10))

		if !priceFound && !stockFound {
			return c.JSON(http.StatusNotFound, echo.Map{
				"error":               "product not found",
				"request_duration_ms": duration,
			})
		}

		return c.JSON(http.StatusOK, PriceInventoryResponse{
			SKU:   sku,
			Price: price,
			Stock: stock,
		})
	})

	// GET /api/realtime/price?sku=XXX - price only
	g.GET("/price", func(c echo.Context) error {
		start := time.Now()

		sku := c.QueryParam("sku")
		if sku == "" {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "sku required"})
		}

		customerGroupID := 0
		if cgStr := c.QueryParam("customer_group"); cgStr != "" {
			if id, err := strconv.Atoi(cgStr); err == nil {
				customerGroupID = id
			}
		}

		priceR, _, err := getRepositories(db)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "repository init failed"})
		}

		price, found := priceR.GetLowestPriceBySKU(sku, customerGroupID)
		duration := time.Since(start).Milliseconds()
		c.Response().Header().Set("X-Request-Duration-ms", strconv.FormatInt(duration, 10))

		if !found {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "price not found"})
		}

		return c.JSON(http.StatusOK, echo.Map{"sku": sku, "price": price})
	})

	// GET /api/realtime/stock?sku=XXX&source=default - stock only
	g.GET("/stock", func(c echo.Context) error {
		start := time.Now()

		sku := c.QueryParam("sku")
		if sku == "" {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "sku required"})
		}

		sourceCode := c.QueryParam("source")
		if sourceCode == "" {
			sourceCode = "default"
		}

		_, inventoryR, err := getRepositories(db)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "repository init failed"})
		}

		qty, found := inventoryR.GetQuantityBySKU(sku, sourceCode)
		duration := time.Since(start).Milliseconds()
		c.Response().Header().Set("X-Request-Duration-ms", strconv.FormatInt(duration, 10))

		if !found {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "stock not found"})
		}

		return c.JSON(http.StatusOK, echo.Map{"sku": sku, "stock": qty, "source": sourceCode})
	})

	// GET /api/realtime/tier-prices?sku=XXX - all tier prices
	g.GET("/tier-prices", func(c echo.Context) error {
		start := time.Now()

		sku := c.QueryParam("sku")
		if sku == "" {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "sku required"})
		}

		priceR, _, err := getRepositories(db)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "repository init failed"})
		}

		tiers, err := priceR.GetTierPricesBySKU(sku)
		duration := time.Since(start).Milliseconds()
		c.Response().Header().Set("X-Request-Duration-ms", strconv.FormatInt(duration, 10))

		if err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, echo.Map{"sku": sku, "tier_prices": tiers})
	})
}
