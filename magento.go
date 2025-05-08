package main

import (
	"log"
	"os"
	"time"
	"strconv"
	"html/template"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"GO/config"
	salesApi "GO/api/sales"
	productApi "GO/api/product"
	html "GO/html"
)

func getAuthMiddleware() echo.MiddlewareFunc {
	skipPaths := config.GetAuthSkipperPaths()
	skipper := func(c echo.Context) bool {
		path := c.Path()
		for _, skip := range skipPaths {
			if path == skip {
				return true
			}
		}
		return false
	}
	authType := os.Getenv("AUTH_TYPE")
	switch authType {
	case "key":
		apiKey := os.Getenv("API_KEY")
		return middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
			Validator: func(key string, c echo.Context) (bool, error) {
				return key == apiKey, nil
			},
			Skipper: skipper,
		})
	default:
		return middleware.BasicAuthWithConfig(middleware.BasicAuthConfig{
			Validator: func(username, password string, c echo.Context) (bool, error) {
				return username == os.Getenv("API_USER") && password == os.Getenv("API_PASS"), nil
			},
			Skipper: skipper,
		})
	}
}

func main() {
	config.LoadEnv()
	config.LoadAppConfig()
	// Initialize Redis
	config.InitRedis()
	redisStatus := "Redis not configured or not reachable, caching disabled."
	if config.RedisClient != nil {
		err := config.RedisClient.Ping(config.RedisCtx()).Err()
		if err == nil {
			redisStatus = "Redis connection successful."
		} else {
			config.RedisClient = nil // Disable Redis if not reachable
			redisStatus = "Redis configured but not reachable, caching disabled."
		}
	}
	log.Println(redisStatus)

	db, err := config.NewDB()
	if err != nil {
		log.Fatalf("failed to connect to DB: %v", err)
	}

	// Check DB connection
	sqldb, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get DB instance: %v", err)
	}
	if err := sqldb.Ping(); err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	log.Println("Database connection successful.")

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.Gzip())
	e.Use(middleware.Decompress())

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			duration := time.Since(start).Milliseconds()
			c.Response().Header().Set("X-Request-Duration-ms", strconv.FormatInt(duration, 10))
			log.Printf("Request duration: %d ms", duration)
			return err
		}
	})

	// Register the template renderer
	t := &html.Template{
		Templates: template.Must(template.ParseGlob("html/**/*.html")),
	}
	e.Renderer = t

	for _, tmpl := range t.Templates.Templates() {
		log.Println("Loaded template:", tmpl.Name())
	}

	apiGroup := e.Group("/api")
	apiGroup.Use(getAuthMiddleware())

	salesApi.RegisterSalesOrderGridRoutes(apiGroup, db)
	productApi.RegisterProductRoutes(apiGroup, db)
	html.RegisterProductHTMLRoutes(e, db)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server running on :%s", port)
	e.Logger.Fatal(e.Start(":" + port))
} 