package graphql

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
)

// Context keys for resolver injection (avoids circular imports).
type contextKey string

const CtxKeyStoreID contextKey = "storeID"

// StoreIDFromContext returns the store ID for the current request.
func StoreIDFromContext(ctx context.Context) uint16 {
	if v := ctx.Value(CtxKeyStoreID); v != nil {
		if id, ok := v.(uint16); ok {
			return id
		}
	}
	return 0
}

// WithStoreID attaches storeID to context.
func WithStoreID(ctx context.Context, storeID uint16) context.Context {
	return context.WithValue(ctx, CtxKeyStoreID, storeID)
}

// StoreContext holds store_id for the current request.
// Resolved from: Store header > __Store query param > JSON variables.__Store
const (
	HeaderStore     = "Store"
	QueryParamStore = "__Store"
	VarStore        = "__Store"
)

// GetStoreID extracts store_id from request.
// Priority: 1) Store header, 2) __Store query param, 3) JSON body variables.__Store
func GetStoreID(r *http.Request) uint16 {
	// 1. Header
	if h := r.Header.Get(HeaderStore); h != "" {
		if id, err := strconv.ParseUint(h, 10, 16); err == nil {
			return uint16(id)
		}
	}

	// 2. Query param
	if q := r.URL.Query().Get(QueryParamStore); q != "" {
		if id, err := strconv.ParseUint(q, 10, 16); err == nil {
			return uint16(id)
		}
	}

	// 3. JSON payload variables (for POST /graphql)
	// Body is read in handler; we pass it via context
	return 0
}

// ParseStoreFromVariables parses variables from JSON body for Store
func ParseStoreFromVariables(body []byte) (uint16, bool) {
	var payload struct {
		Variables map[string]interface{} `json:"variables"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.Variables == nil {
		return 0, false
	}
	if v, ok := payload.Variables[VarStore]; ok {
		switch val := v.(type) {
		case string:
			if id, err := strconv.ParseUint(val, 10, 16); err == nil {
				return uint16(id), true
			}
		case float64:
			return uint16(val), true
		}
	}
	return 0, false
}
