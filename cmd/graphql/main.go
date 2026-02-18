// Standalone GraphQL server — run with: go run ./cmd/graphql
package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"

	"github.com/common-nighthawk/go-figure"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"magento.GO/api"
	graphqlApi "magento.GO/api/graphql"
	"magento.GO/config"
)

func main() {
	_ = godotenv.Load()

	db, err := config.NewDB()
	if err != nil {
		log.Fatal("db:", err)
	}

	e := echo.New()
	graphqlApi.RegisterGraphQLRoutes(e, db)
	api.ApplyRoutes(e, db)

	// ASCII banner on start (random font each run)
	gqlFonts := []string{"banner", "big", "block", "slant", "standard", "small", "shadow", "speed", "thick", "univers", "doom", "larry3d", "puffy", "rectangles", "bigchief", "cosmic"}
	fig := figure.NewFigure("GoGento GQL ->", gqlFonts[rand.Intn(len(gqlFonts))], true)
	fig.Print()
	fmt.Println("Standalone GraphQL server")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("GraphQL at http://localhost:%s/graphql  Playground at http://localhost:%s/playground", port, port)
	e.Logger.Fatal(e.Start(":" + port))
}
