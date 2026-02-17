package custom

import (
	"context"
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/spf13/cobra"

	"magento.GO/api"
	gqlregistry "magento.GO/graphql/registry"
	"magento.GO/cmd"
	"magento.GO/cron"
)

func init() {
	// GraphQL extension
	gqlregistry.Register("ping", func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return map[string]string{"pong": "ok"}, nil
	})

	// CLI command
	cmd.Register(&cobra.Command{
		Use:   "custom:hello",
		Short: "Custom command example",
		Run: func(c *cobra.Command, args []string) {
			fmt.Println("Hello from custom command")
		},
	})

	// Cron job
	cron.Register("customping", "@every 1m", func(args ...string) {
		fmt.Println("Custom cron: ping at", args)
	})

	// HTTP route
	api.RegisterGET("/custom/ping", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"pong": "ok"})
	})
}
