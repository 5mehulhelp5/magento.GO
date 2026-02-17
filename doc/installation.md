# Installation

## Install Go
```bash
sudo apt update && sudo apt install golang-go
# or: sudo snap install go
go version
```

## Environment Variables
```
MYSQL_USER=magento
MYSQL_PASS=magento
MYSQL_HOST=localhost
MYSQL_PORT=3306
MYSQL_DB=magento
API_USER=admin
API_PASS=secret
REDIS_ADDR=""
REDIS_PASS=""
PORT=8080
GORM_LOG=off             # Disable SQL logging
PRODUCT_FLAT_CACHE=off   # Disable product cache
```

## Dependencies
```bash
cd gogento-catalog
go mod tidy
```

## Running

**Full API** (REST + GraphQL):
```bash
go run magento.go
```

**Standalone GraphQL** (GraphQL only):
```bash
go run ./cmd/graphql
```

Server: `http://localhost:8080`
- GraphQL: `POST /graphql`
- Playground: `GET /playground`
