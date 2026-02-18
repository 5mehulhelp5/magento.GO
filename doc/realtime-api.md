# Realtime Pricing & Inventory API

High-performance, stateless API for real-time product pricing and inventory lookups. Designed for sub-30ms response times with direct SQL queries.

## Why Realtime API?

Traditional Magento price and inventory lookups go through multiple PHP layers: models, plugins, indexers, and cache checks. Even with Full Page Cache, dynamic price/stock data requires PHP execution. This creates latency that hurts conversion rates and user experience.

**GoGento's Realtime API solves this:**

- **Sub-30ms responses** — Direct SQL queries bypass all PHP overhead
- **Stateless authentication** — HMAC-SHA256 signatures, no session storage
- **Parallel execution** — Price and inventory fetched concurrently via goroutines
- **Zero allocations** — Raw `database/sql` queries, no ORM overhead
- **Automatic schema detection** — Works with both CE (`entity_id`) and EE (`row_id`)
- **Multi-source inventory** — Native MSI support via `inventory_source_item`
- **Customer-specific pricing** — Tier prices resolved per customer group
- **Frontend-ready** — Works with React, Vue, Hyvä/Alpine.js, vanilla JS

**Use cases:**

- Real-time price updates on PDP/PLP without full page reload
- Live stock availability in cart and checkout
- Dynamic pricing for logged-in customers (tier prices)
- AJAX add-to-cart with instant stock validation
- Price comparison widgets and tools
- B2B portals with customer-specific pricing

## Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/realtime/price-inventory` | GET | Price + stock (parallel fetch) |
| `/api/realtime/price` | GET | Lowest price only |
| `/api/realtime/stock` | GET | Stock quantity only |
| `/api/realtime/tier-prices` | GET | All tier prices for SKU |

## Authentication

Uses HMAC-SHA256 signature verification with Magento's crypt key. No sessions required.

### Headers

| Header | Description |
|--------|-------------|
| `X-Customer-ID` | Customer ID or customer group ID |
| `X-Customer-Sig` | HMAC-SHA256 signature (hex encoded) |

### Configuration

Set the Magento crypt key in your environment:

```bash
# From app/etc/env.php: $config['crypt']['key']
export MAGENTO_CRYPT_KEY="3254cdb1ae5233a336cdec765aeb3bb6"
```

## PHP Integration

### Generating Signed Requests

```php
<?php
/**
 * GoGento Realtime API Client
 * 
 * Signs requests using Magento's crypt key for secure communication
 * with the Go microservice.
 */
class GoGentoRealtimeClient
{
    private string $baseUrl;
    private string $cryptKey;

    public function __construct(string $baseUrl, string $cryptKey = null)
    {
        $this->baseUrl = rtrim($baseUrl, '/');
        $this->cryptKey = $cryptKey ?? $this->getMagentoCryptKey();
    }

    /**
     * Get crypt key from Magento env.php
     */
    private function getMagentoCryptKey(): string
    {
        $envPath = BP . '/app/etc/env.php';
        if (!file_exists($envPath)) {
            throw new \RuntimeException('env.php not found');
        }
        $config = include $envPath;
        return $config['crypt']['key'] ?? '';
    }

    /**
     * Generate HMAC-SHA256 signature for customer ID
     */
    public function generateSignature(string $customerId): string
    {
        return hash_hmac('sha256', $customerId, $this->cryptKey);
    }

    /**
     * Get price and inventory for a SKU
     * 
     * @param string $sku Product SKU
     * @param int|null $customerId Customer ID for tier pricing
     * @param string $source Inventory source code
     * @return array ['sku' => string, 'price' => float, 'stock' => float]
     */
    public function getPriceInventory(string $sku, ?int $customerId = null, string $source = 'default'): array
    {
        $url = sprintf(
            '%s/api/realtime/price-inventory?sku=%s&source=%s',
            $this->baseUrl,
            urlencode($sku),
            urlencode($source)
        );

        $headers = [];
        if ($customerId !== null) {
            $headers['X-Customer-ID'] = (string) $customerId;
            $headers['X-Customer-Sig'] = $this->generateSignature((string) $customerId);
        }

        return $this->request($url, $headers);
    }

    /**
     * Get lowest price for a SKU
     */
    public function getPrice(string $sku, int $customerGroupId = 0): array
    {
        $url = sprintf(
            '%s/api/realtime/price?sku=%s&customer_group=%d',
            $this->baseUrl,
            urlencode($sku),
            $customerGroupId
        );
        return $this->request($url);
    }

    /**
     * Get stock quantity for a SKU
     */
    public function getStock(string $sku, string $source = 'default'): array
    {
        $url = sprintf(
            '%s/api/realtime/stock?sku=%s&source=%s',
            $this->baseUrl,
            urlencode($sku),
            urlencode($source)
        );
        return $this->request($url);
    }

    /**
     * Get all tier prices for a SKU
     */
    public function getTierPrices(string $sku): array
    {
        $url = sprintf('%s/api/realtime/tier-prices?sku=%s', $this->baseUrl, urlencode($sku));
        return $this->request($url);
    }

    /**
     * Batch get prices for multiple SKUs (parallel requests)
     */
    public function batchGetPrices(array $skus, int $customerGroupId = 0): array
    {
        $mh = curl_multi_init();
        $handles = [];

        foreach ($skus as $sku) {
            $url = sprintf(
                '%s/api/realtime/price?sku=%s&customer_group=%d',
                $this->baseUrl,
                urlencode($sku),
                $customerGroupId
            );
            $ch = curl_init($url);
            curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
            curl_setopt($ch, CURLOPT_TIMEOUT, 5);
            curl_multi_add_handle($mh, $ch);
            $handles[$sku] = $ch;
        }

        // Execute all requests in parallel
        do {
            $status = curl_multi_exec($mh, $active);
            if ($active) {
                curl_multi_select($mh);
            }
        } while ($active && $status === CURLM_OK);

        $results = [];
        foreach ($handles as $sku => $ch) {
            $response = curl_multi_getcontent($ch);
            $data = json_decode($response, true);
            $results[$sku] = $data['price'] ?? null;
            curl_multi_remove_handle($mh, $ch);
            curl_close($ch);
        }

        curl_multi_close($mh);
        return $results;
    }

    private function request(string $url, array $headers = []): array
    {
        $ch = curl_init($url);
        curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
        curl_setopt($ch, CURLOPT_TIMEOUT, 5);
        
        if (!empty($headers)) {
            $headerLines = [];
            foreach ($headers as $key => $value) {
                $headerLines[] = "$key: $value";
            }
            curl_setopt($ch, CURLOPT_HTTPHEADER, $headerLines);
        }

        $response = curl_exec($ch);
        $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
        curl_close($ch);

        if ($httpCode !== 200) {
            throw new \RuntimeException("API request failed with status $httpCode: $response");
        }

        return json_decode($response, true) ?? [];
    }
}
```

### Usage Examples

```php
<?php
// Initialize client
$client = new GoGentoRealtimeClient('http://gogento:8080');

// Get price and inventory (signed request)
$result = $client->getPriceInventory('SKU-001', customerId: 42);
// Returns: ['sku' => 'SKU-001', 'price' => 29.99, 'stock' => 150]

// Get price only (with customer group for tier pricing)
$price = $client->getPrice('SKU-001', customerGroupId: 2);
// Returns: ['sku' => 'SKU-001', 'price' => 24.99]

// Get stock from specific source
$stock = $client->getStock('SKU-001', source: 'warehouse_east');
// Returns: ['sku' => 'SKU-001', 'stock' => 75, 'source' => 'warehouse_east']

// Get all tier prices
$tiers = $client->getTierPrices('SKU-001');
// Returns: ['sku' => 'SKU-001', 'tier_prices' => [...]]

// Batch pricing (parallel requests)
$prices = $client->batchGetPrices(['SKU-001', 'SKU-002', 'SKU-003']);
// Returns: ['SKU-001' => 29.99, 'SKU-002' => 49.99, 'SKU-003' => 19.99]
```

### Magento Observer Integration

```php
<?php
namespace Vendor\Module\Observer;

use Magento\Framework\Event\Observer;
use Magento\Framework\Event\ObserverInterface;

class ProductPriceObserver implements ObserverInterface
{
    private GoGentoRealtimeClient $client;
    
    public function __construct()
    {
        $this->client = new GoGentoRealtimeClient('http://gogento:8080');
    }

    public function execute(Observer $observer)
    {
        $product = $observer->getEvent()->getProduct();
        $sku = $product->getSku();
        
        try {
            $data = $this->client->getPriceInventory($sku);
            // Use realtime price/stock data
            $product->setData('realtime_price', $data['price']);
            $product->setData('realtime_stock', $data['stock']);
        } catch (\Exception $e) {
            // Fallback to cached data
        }
    }
}
```

## Frontend Integration

All frontend examples assume the HMAC signature is generated server-side and passed to the client. Never expose your crypt key in JavaScript.

### Vanilla JavaScript

```javascript
// Signature should be generated server-side and passed to frontend
async function getRealtimePrice(sku, customerId, signature) {
    const response = await fetch(
        `https://gogento.example.com/api/realtime/price-inventory?sku=${encodeURIComponent(sku)}`,
        {
            method: 'GET',
            headers: {
                'X-Customer-ID': customerId,
                'X-Customer-Sig': signature
            }
        }
    );
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    return response.json();
}

// Usage
getRealtimePrice('SKU-001', '42', 'a1b2c3...')
    .then(data => {
        document.getElementById('price').textContent = `$${data.price}`;
        document.getElementById('stock').textContent = data.stock > 0 ? 'In Stock' : 'Out of Stock';
    });
```

### React Integration

```jsx
import { useState, useEffect } from 'react';

function useRealtimePrice(sku, customerId, signature) {
    const [data, setData] = useState({ price: null, stock: null, loading: true });

    useEffect(() => {
        if (!sku) return;
        
        fetch(`/api/realtime/price-inventory?sku=${encodeURIComponent(sku)}`, {
            headers: {
                'X-Customer-ID': customerId,
                'X-Customer-Sig': signature
            }
        })
        .then(res => res.json())
        .then(json => setData({ price: json.price, stock: json.stock, loading: false }))
        .catch(() => setData(prev => ({ ...prev, loading: false })));
    }, [sku, customerId, signature]);

    return data;
}

// Usage in component
function ProductPrice({ sku, customerId, signature }) {
    const { price, stock, loading } = useRealtimePrice(sku, customerId, signature);

    if (loading) return <span>Loading...</span>;

    return (
        <div>
            <span className="price">${price?.toFixed(2)}</span>
            <span className={stock > 0 ? 'in-stock' : 'out-of-stock'}>
                {stock > 0 ? `${stock} in stock` : 'Out of stock'}
            </span>
        </div>
    );
}
```

### Hyvä Integration (Alpine.js)

```html
<!-- Hyvä theme component using Alpine.js -->
<div x-data="realtimePrice('<?= $escaper->escapeJs($block->getSku()) ?>')" 
     x-init="fetchPrice()">
    
    <span x-show="loading">Loading...</span>
    
    <template x-if="!loading">
        <div>
            <span class="price" x-text="'$' + price?.toFixed(2)"></span>
            <span :class="stock > 0 ? 'text-green-600' : 'text-red-600'"
                  x-text="stock > 0 ? stock + ' in stock' : 'Out of stock'">
            </span>
        </div>
    </template>
</div>

<script>
function realtimePrice(sku) {
    return {
        sku: sku,
        price: null,
        stock: null,
        loading: true,
        customerId: '<?= $escaper->escapeJs($block->getCustomerId()) ?>',
        signature: '<?= $escaper->escapeJs($block->getCustomerSignature()) ?>',
        
        async fetchPrice() {
            try {
                const response = await fetch(
                    `<?= $escaper->escapeUrl($block->getGoGentoUrl()) ?>/api/realtime/price-inventory?sku=${encodeURIComponent(this.sku)}`,
                    {
                        headers: {
                            'X-Customer-ID': this.customerId,
                            'X-Customer-Sig': this.signature
                        }
                    }
                );
                const data = await response.json();
                this.price = data.price;
                this.stock = data.stock;
            } catch (e) {
                console.error('Realtime price fetch failed:', e);
            } finally {
                this.loading = false;
            }
        }
    };
}
</script>
```

### Hyvä Block Class

```php
<?php
namespace Vendor\Module\Block;

use Magento\Framework\View\Element\Template;
use Magento\Customer\Model\Session as CustomerSession;

class RealtimePrice extends Template
{
    private CustomerSession $customerSession;

    public function __construct(
        Template\Context $context,
        CustomerSession $customerSession,
        array $data = []
    ) {
        parent::__construct($context, $data);
        $this->customerSession = $customerSession;
    }

    public function getCustomerId(): string
    {
        return (string) ($this->customerSession->getCustomerId() ?? 0);
    }

    public function getCustomerSignature(): string
    {
        $customerId = $this->getCustomerId();
        $cryptKey = $this->_scopeConfig->getValue('crypt/key');
        return hash_hmac('sha256', $customerId, $cryptKey);
    }

    public function getGoGentoUrl(): string
    {
        return $this->_scopeConfig->getValue('gogento/general/api_url') 
            ?? 'http://gogento:8080';
    }
}
```

## Response Format

### Success Response

```json
{
    "sku": "SKU-001",
    "price": 29.99,
    "stock": 150
}
```

### Error Responses

```json
// 400 Bad Request
{"error": "sku required"}

// 401 Unauthorized  
{"error": "invalid signature"}

// 404 Not Found
{"error": "product not found", "request_duration_ms": 5}
```

## Response Headers

| Header | Description |
|--------|-------------|
| `X-Request-Duration-ms` | Request processing time in milliseconds |

## Price Calculation

The API returns the **lowest price** using SQL `LEAST()` function:

1. **Base Price** - `catalog_product_entity_decimal` (attribute: `price`)
2. **Special Price** - `catalog_product_entity_decimal` (attribute: `special_price`)
3. **Tier Price** - `catalog_product_entity_tier_price` (for customer group or `all_groups=1`)

## Schema Compatibility

Automatically detects and supports both schemas:

| Edition | EAV Link Column |
|---------|-----------------|
| Community Edition (CE) | `entity_id` |
| Enterprise Edition (EE) | `row_id` |

## Performance

| Metric | Target |
|--------|--------|
| Response time | < 30ms |
| Parallel queries | Price + Stock fetched concurrently |
| Connection pooling | Uses `database/sql` pool |
| Zero allocations | Raw SQL, no ORM overhead |
