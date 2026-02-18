# EAV Products

Product attributes (EAV) are fetched and flattened for API responses.

![EAV Attribute Flattening](images/eav-flattening.png)

## Repository Methods

- `FetchWithAllAttributes(storeID)` — products with EAV preloaded (Varchars, Ints, Decimals, Texts, Datetimes)
- `FetchWithAllAttributesFlat(storeID)` — flattened map, attribute codes as keys

## Flat Product Structure

- Base: `entity_id`, `sku`, `type_id`, `created_at`, `updated_at`
- EAV attributes: keyed by attribute code (from `eav_attribute`)
- `stock_item` — qty, is_in_stock, min_qty, max_sale_qty, manage_stock, website_id
- `index_prices` — price, final_price, customer_group_id, website_id
- `category_ids`, `media_gallery`
