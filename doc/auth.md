# Authentication & Authorization

All `/api/*` endpoints require authentication. The auth mode is selected via the `AUTH_TYPE` env var.

## Configuration

| Env var | Description |
|---------|-------------|
| `AUTH_TYPE` | `basic` (default), `key`, or `token` |
| `API_USER` / `API_PASS` | Credentials for Basic Auth |
| `API_KEY` | Static key for `key` mode, or fallback key for `token` mode |

## Modes

### `basic` (default)

Standard HTTP Basic Auth. Set `API_USER` and `API_PASS`.

```bash
curl -u admin:secret http://localhost:8080/api/stock/import ...
```

### `key`

Static API key via `Authorization: Bearer <key>`. Set `API_KEY`.

```bash
curl -H "Authorization: Bearer my-api-key" http://localhost:8080/api/stock/import ...
```

### `token`

Validates against Magento's `oauth_token` table (integration/admin tokens). Also accepts a static fallback key from `API_KEY` if set.

```bash
# Magento integration token
curl -H "Authorization: Bearer 8muvgq6bvtm95ky64fys7hnkjp1v4xmf" ...

# Or static key (if API_KEY is configured)
curl -H "Authorization: Bearer my-static-key" ...
```

**Token validation rules:**
- Token must exist in `oauth_token` table
- `type` must be `access`
- `revoked` must be `0`

---

## ACL & Roles (token mode)

When a valid DB token authenticates, the middleware resolves the user's role and ACL permissions from Magento's authorization tables and stores them in the Echo request context.

### Context values

| Context key | Type | Description |
|-------------|------|-------------|
| `auth_type` | `string` | `"token"` (DB token) or `"static"` (API_KEY fallback) |
| `oauth_token` | `*entity.OauthToken` | The matched token record |
| `role_id` | `uint` | The user's group role ID |
| `role_name` | `string` | The role name (e.g. `"Administrators"`) |
| `acl_resources` | `[]string` | List of allowed Magento ACL resource IDs |

### Resolution chain

```
oauth_token.admin_id
  → authorization_role (role_type='U', user_id=admin_id)
    → authorization_role (role_type='G', role_id=parent_id)
      → authorization_rule (role_id, permission='allow')
```

### Magento authorization tables

| Table | Purpose |
|-------|---------|
| `oauth_token` | Bearer tokens (integration & admin) |
| `admin_user` | Admin user accounts |
| `authorization_role` | Roles (`role_type='G'`) and user-to-role assignments (`role_type='U'`) |
| `authorization_rule` | ACL rules mapping role → resource with `allow`/`deny` permission |

### Role types in `authorization_role`

| `role_type` | Meaning | `user_id` | `parent_id` |
|-------------|---------|-----------|-------------|
| `G` | Group (role definition) | `0` | `0` (top-level) |
| `U` | User assignment | admin `user_id` | parent `role_id` of group |

### ACL resource examples

Magento ACL resources follow the `Module::resource` pattern:

```
Magento_Backend::all
Magento_Backend::admin
Magento_Catalog::products
Magento_Catalog::categories
Magento_Sales::sales
Magento_Sales::sales_order
Magento_Customer::customer
```

### Accessing ACL in handlers

```go
func myHandler(c echo.Context) error {
    authType, _ := c.Get("auth_type").(string) // "token" or "static"

    // Role info (only set for DB token auth)
    roleName, _ := c.Get("role_name").(string)
    resources, _ := c.Get("acl_resources").([]string)

    // Check a specific resource
    for _, r := range resources {
        if r == "Magento_Catalog::products" {
            // user has catalog product access
        }
    }
    // ...
}
```

### Skipped paths

Some paths skip auth entirely (configured in `config/api.go`):

```go
func GetAuthSkipperPaths() []string {
    return []string{"/health", "/api/products", "/api/products/:id", "/graphql"}
}
```

---

## Entity models

| Model | Table | File |
|-------|-------|------|
| `entity.OauthToken` | `oauth_token` | `model/entity/oauth_token.go` |
| `entity.AdminUser` | `admin_user` | `model/entity/admin_user.go` |
| `entity.AuthorizationRole` | `authorization_role` | `model/entity/authorization_role.go` |
| `entity.AuthorizationRule` | `authorization_rule` | `model/entity/authorization_rule.go` |

## Implementation

```
core/auth/auth.go                              # auth.Middleware(db) — middleware logic
model/repository/auth/auth_repository.go       # AuthRepository — DB queries
model/entity/oauth_token.go                    # OauthToken entity
model/entity/admin_user.go                     # AdminUser entity
model/entity/authorization_role.go             # AuthorizationRole entity
model/entity/authorization_rule.go             # AuthorizationRule entity
config/api.go                                  # Auth skipper paths
```

### Repository methods

| Method | Description |
|--------|-------------|
| `FindActiveToken(token)` | Looks up a non-revoked access token by string |
| `FindUserRole(adminID)` | Finds the user's role assignment (`role_type='U'`) |
| `FindGroupRole(roleID)` | Finds the parent group role (`role_type='G'`) |
| `FindAllowedResources(roleID)` | Returns allowed ACL resource IDs for a role |

Usage in `magento.go`:

```go
apiGroup := e.Group("/api")
apiGroup.Use(auth.Middleware(db))
```
