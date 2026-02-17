package resolvers

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/mitchellh/mapstructure"
	gqlmodels "magento.GO/graphql/models"
)

func filterProductsForGuest(flat map[uint]map[string]interface{}, customerGroupID uint) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(flat))
	for _, p := range flat {
		result = append(result, filterPriceForGuest(p, customerGroupID))
	}
	return result
}

func filterPriceForGuest(p map[string]interface{}, customerGroupID uint) map[string]interface{} {
	if prices, ok := p["index_prices"].([]map[string]interface{}); ok {
		for _, ip := range prices {
			cg := toUint(ip["customer_group_id"])
			if cg == customerGroupID {
				p["price"] = ip["price"]
				p["final_price"] = ip["final_price"]
				break
			}
		}
	}
	return p
}

func numberToStringHook() mapstructure.DecodeHookFunc {
	return func(f, t reflect.Type, data interface{}) (interface{}, error) {
		if t.Kind() != reflect.String {
			return data, nil
		}
		switch f.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			return fmt.Sprint(data), nil
		}
		return data, nil
	}
}

func intToBoolHook() mapstructure.DecodeHookFunc {
	return func(f, t reflect.Type, data interface{}) (interface{}, error) {
		if t.Kind() != reflect.Bool {
			return data, nil
		}
		switch v := data.(type) {
		case int:
			return v != 0, nil
		case int64:
			return v != 0, nil
		case float64:
			return int(v) != 0, nil
		}
		return data, nil
	}
}

func uintSliceToStringSliceHook() mapstructure.DecodeHookFunc {
	stringSliceType := reflect.TypeOf([]string(nil))
	return func(f, t reflect.Type, data interface{}) (interface{}, error) {
		if t != stringSliceType {
			return data, nil
		}
		switch v := data.(type) {
		case []uint:
			s := make([]string, len(v))
			for i, u := range v {
				s[i] = strconv.FormatUint(uint64(u), 10)
			}
			return s, nil
		case []interface{}:
			s := make([]string, 0, len(v))
			for _, x := range v {
				s = append(s, strconv.FormatUint(uint64(toUint(x)), 10))
			}
			return s, nil
		}
		return data, nil
	}
}

var flatToProductDecodeHook = mapstructure.ComposeDecodeHookFunc(
	numberToStringHook(),
	intToBoolHook(),
	uintSliceToStringSliceHook(),
)

func flatToProduct(p map[string]interface{}) *gqlmodels.Product {
	// Prepare map for mapstructure: flatten stock_item for is_in_stock, qty
	if si, ok := p["stock_item"].(map[string]interface{}); ok {
		if v, ok := si["is_in_stock"]; ok {
			p["is_in_stock"] = v
		}
		if v, ok := si["qty"]; ok {
			p["qty"] = v
		}
	}

	// Build media_gallery as []map for mapstructure
	if mg, ok := p["media_gallery"].([]map[string]interface{}); ok {
		items := make([]*gqlmodels.MediaGalleryItem, len(mg))
		for i, m := range mg {
			items[i] = mediaItemFromMap(m)
		}
		p["media_gallery"] = items
	}

	var prod gqlmodels.Product
	cfg := &mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		DecodeHook:       flatToProductDecodeHook,
		Result:           &prod,
		TagName:          "mapstructure",
		ZeroFields:       true,
	}
	dec, _ := mapstructure.NewDecoder(cfg)
	if err := dec.Decode(p); err != nil {
		return &gqlmodels.Product{}
	}
	prod.Attributes = p
	return &prod
}

func mediaItemFromMap(m map[string]interface{}) *gqlmodels.MediaGalleryItem {
	item := &gqlmodels.MediaGalleryItem{}
	if v, ok := m["value_id"]; ok {
		item.ValueID = strconv.FormatUint(uint64(toUint(v)), 10)
	}
	if v, ok := m["value"].(string); ok {
		item.Value = v
	}
	if v, ok := m["media_type"].(string); ok {
		item.MediaType = &v
	}
	return item
}

func toUint(v interface{}) uint {
	switch val := v.(type) {
	case uint:
		return val
	case int:
		return uint(val)
	case float64:
		return uint(val)
	case int64:
		return uint(val)
	default:
		return 0
	}
}
