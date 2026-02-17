# REST API

REST endpoints require Basic Auth (`API_USER`, `API_PASS`).

## Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/orders | List orders |
| GET | /api/orders/:id | Get order |
| POST | /api/orders | Create order |
| PUT | /api/orders/:id | Update order |
| DELETE | /api/orders/:id | Delete order |
| GET | /api/products/flat | All flat products |
| GET | /api/products/flat/:ids | Products by comma-separated IDs |

## Product Flat API

- `GET /api/products/flat` — all products with EAV attributes flattened
- `GET /api/products/flat/1,2,3` — products by IDs
- Optional store ID via query/header
