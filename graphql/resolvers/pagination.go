package resolvers

func defaultPageSize(p *int) int {
	if p != nil && *p > 0 {
		return *p
	}
	return 20
}

func defaultCurrentPage(p *int) int {
	if p != nil && *p > 0 {
		return *p
	}
	return 1
}

func paginate(items []map[string]interface{}, currentPage, pageSize int) []map[string]interface{} {
	total := len(items)
	start := (currentPage - 1) * pageSize
	end := start + pageSize
	if start >= total {
		return []map[string]interface{}{}
	}
	if end > total {
		end = total
	}
	return items[start:end]
}
