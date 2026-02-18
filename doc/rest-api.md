# REST API

All `/api/*` endpoints require authentication. See **[auth.md](auth.md)** for auth modes (basic, key, token), Magento ACL/roles, and configuration.

## Endpoints

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| GET | /api/orders | yes | List orders |
| GET | /api/orders/:id | yes | Get order |
| POST | /api/orders | yes | Create order |
| PUT | /api/orders/:id | yes | Update order |
| DELETE | /api/orders/:id | yes | Delete order |
| GET | /api/products | yes | List products |
| GET | /api/products/:id | yes | Get product by ID |
| POST | /api/products | yes | Create product |
| PUT | /api/products/:id | yes | Update product |
| DELETE | /api/products/:id | yes | Delete product |
| GET | /api/products/flat | yes | All flat products (EAV flattened) |
| GET | /api/products/flat/:ids | yes | Products by comma-separated IDs |
| POST | /api/stock/import | yes | Bulk stock import (JSON) |

---

## Stock Import API

`POST /api/stock/import` — Bulk upsert stock/inventory data by SKU.

Resolves SKUs to product entity IDs, then inserts or updates `cataloginventory_stock_item` rows. Unknown SKUs are skipped with warnings.

### Request

```json
POST /api/stock/import
Content-Type: application/json

{
  "items": [
    {
      "sku": "SKU-001",
      "qty": 100,
      "is_in_stock": 1
    },
    {
      "sku": "SKU-002",
      "qty": 0,
      "is_in_stock": 0,
      "manage_stock": 1,
      "min_qty": 5,
      "min_sale_qty": 1,
      "max_sale_qty": 50
    }
  ],
  "batch_size": 500
}
```

### Item fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `sku` | string | yes | — | Product SKU (must exist in `catalog_product_entity`) |
| `qty` | float | no | 0 | Stock quantity |
| `is_in_stock` | int | no | 1 | 1 = in stock, 0 = out of stock |
| `manage_stock` | int | no | 1 | 1 = manage stock, 0 = don't manage |
| `min_qty` | float | no | 0 | Minimum qty before out-of-stock |
| `min_sale_qty` | float | no | 0 | Minimum qty per order |
| `max_sale_qty` | float | no | 0 | Maximum qty per order |

`batch_size` is optional (default 500). Controls how many rows are upserted per DB batch.

### Response

```json
{
  "imported": 2,
  "skipped": 0,
  "warnings": null,
  "request_duration_ms": 12
}
```

### Error cases

| Status | When |
|--------|------|
| 400 | Missing or empty `items` array, malformed JSON |
| 401 | Missing or invalid auth credentials |
| 500 | Database error |

### curl examples

**Basic Auth:**

```bash
curl -X POST http://localhost:8080/api/stock/import \
  -u "admin:secret" \
  -H "Content-Type: application/json" \
  -d '{
    "items": [
      {"sku": "SKU-001", "qty": 100, "is_in_stock": 1},
      {"sku": "SKU-002", "qty": 0, "is_in_stock": 0}
    ]
  }'
```

**Key Auth:**

```bash
curl -X POST http://localhost:8080/api/stock/import \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"items": [{"sku": "SKU-001", "qty": 50}]}'
```

---

## Product Import CLI

`products:import` — Bulk import products from CSV into Magento EAV tables.

Supports all EAV attribute types (varchar, int, decimal, text, datetime), stock, media gallery, and price index data. New products are created automatically; existing products are updated (upsert).

### Usage

```bash
gogento products:import --file products.csv [flags]
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--file` | `-f` | — | CSV file path (required) |
| `--store` | | 0 | Store ID |
| `--batch-size` | | 500 | Batch size for DB operations |
| `--attribute-set` | | 4 | Default attribute set ID for new products |
| `--raw-sql` | | false | Use raw SQL instead of GORM (faster) |

### CSV format

Flexible — columns auto-detected from header row. The `sku` column is required.

**Entity columns** (optional):

| Column | Description |
|--------|-------------|
| `sku` | Product SKU (required) |
| `type_id` | Product type: `simple`, `configurable`, `bundle`, etc. (default: `simple`) |
| `attribute_set_id` | Attribute set ID (default: from `--attribute-set` flag) |

**EAV columns** — Any column matching an EAV attribute code (e.g. `name`, `description`, `price`, `status`, `weight`, `url_key`, `special_from_date`) is written to the corresponding typed table.

**Stock columns:**

| Column | Description |
|--------|-------------|
| `qty` | Stock quantity |
| `is_in_stock` | 1 or 0 |
| `manage_stock` | 1 or 0 |
| `min_qty`, `min_sale_qty`, `max_sale_qty` | Qty limits |

**Gallery columns:**

| Column | Description |
|--------|-------------|
| `image`, `small_image`, `thumbnail` | Image paths |
| `media_gallery` | Pipe-separated image paths (`/img1.jpg\|/img2.jpg`) |

**Price index columns:**

| Column | Description |
|--------|-------------|
| `price_index` | Base price |
| `final_price`, `min_price`, `max_price`, `tier_price` | Price index values |

### Example CSV

```csv
sku,name,price,status,qty,is_in_stock,image
SKU-001,Widget A,19.99,1,100,1,/m/y/widget-a.jpg
SKU-002,Widget B,29.50,2,0,0,/m/y/widget-b.jpg
```

### Example run

```bash
gogento products:import -f products.csv --batch-size 1000 --raw-sql
```

```
=== Import Report ===
CSV rows:       2
Created:        2
Updated:        0
Skipped:        0
EAV values:     6 (varchar=2 int=2 decimal=2 text=0 datetime=0)
Stock rows:     2
Gallery rows:   2
Price rows:     0
Mode:           Raw SQL
Total time:     15ms
  - Processing: 12ms
  - DB upsert:  8ms
=====================
```

---

## Architecture

### File map

```
api/stock/stock_api.go                 # Stock import API endpoint
cmd/product_import.go                  # Product import CLI command
service/product/import_service.go      # Import orchestrator
service/product/import_eav.go          # EAV attribute import (5 types)
service/product/import_stock.go        # Stock import + JSON API service
service/product/import_gallery.go      # Media gallery import
service/product/import_price.go        # Price index import
tests/service/import_test.go           # All import tests
```

### Data flow

```
CSV file ─► CLI (cmd/product_import.go)
                 │
                 ▼
JSON body ─► API (api/stock/stock_api.go)
                 │
                 ▼
         service/product/
         ┌─── import_service.go (orchestrator, CSV path)
         │    ├── import_eav.go     (varchar/int/decimal/text/datetime)
         │    ├── import_stock.go   (cataloginventory_stock_item)
         │    ├── import_gallery.go (catalog_product_entity_media_gallery)
         │    └── import_price.go   (catalog_product_index_price)
         │
         └─── import_stock.go → ImportStockJSON() (JSON API path)
```

The CSV import runs all modules in parallel (EAV, stock, gallery, price). The JSON stock API calls `ImportStockJSON` directly — a standalone function that resolves SKUs and upserts stock rows.
