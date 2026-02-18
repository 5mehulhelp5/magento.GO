package auth

import (
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"

	"magento.GO/config"
	entity "magento.GO/model/entity"
	authRepo "magento.GO/model/repository/auth"
)

// Middleware returns the auth middleware based on AUTH_TYPE env var.
func Middleware(db *gorm.DB) echo.MiddlewareFunc {
	skipper := buildSkipper()
	authType := os.Getenv("AUTH_TYPE")
	switch authType {
	case "key":
		return keyAuth(skipper)
	case "token":
		return tokenAuth(authRepo.NewAuthRepository(db), skipper)
	default:
		return basicAuth(skipper)
	}
}

func buildSkipper() middleware.Skipper {
	skipPaths := config.GetAuthSkipperPaths()
	return func(c echo.Context) bool {
		path := c.Path()
		for _, skip := range skipPaths {
			if path == skip {
				return true
			}
		}
		return false
	}
}

func basicAuth(skipper middleware.Skipper) echo.MiddlewareFunc {
	return middleware.BasicAuthWithConfig(middleware.BasicAuthConfig{
		Validator: func(username, password string, c echo.Context) (bool, error) {
			return username == os.Getenv("API_USER") && password == os.Getenv("API_PASS"), nil
		},
		Skipper: skipper,
	})
}

func keyAuth(skipper middleware.Skipper) echo.MiddlewareFunc {
	apiKey := os.Getenv("API_KEY")
	return middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		Validator: func(key string, c echo.Context) (bool, error) {
			return key == apiKey, nil
		},
		Skipper: skipper,
	})
}

func tokenAuth(repo *authRepo.AuthRepository, skipper middleware.Skipper) echo.MiddlewareFunc {
	staticKey := os.Getenv("API_KEY")
	return middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		Validator: func(token string, c echo.Context) (bool, error) {
			if staticKey != "" && token == staticKey {
				c.Set("auth_type", "static")
				return true, nil
			}
			oauthToken, err := repo.FindActiveToken(token)
			if err != nil {
				return false, nil
			}
			c.Set("auth_type", "token")
			c.Set("oauth_token", oauthToken)
			loadACL(repo, c, oauthToken)
			return true, nil
		},
		Skipper: skipper,
	})
}

// loadACL resolves the token's role and ACL resources into the request context.
func loadACL(repo *authRepo.AuthRepository, c echo.Context, token *entity.OauthToken) {
	if token.AdminID == nil {
		return
	}
	userRole, err := repo.FindUserRole(*token.AdminID)
	if err != nil {
		return
	}
	groupRole, err := repo.FindGroupRole(userRole.ParentID)
	if err != nil {
		return
	}
	c.Set("role_id", groupRole.RoleID)
	c.Set("role_name", groupRole.RoleName)

	resources, err := repo.FindAllowedResources(groupRole.RoleID)
	if err != nil {
		return
	}
	c.Set("acl_resources", resources)
}
