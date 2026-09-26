package app

const (
	defaultReadPageLimit = 50
	maxReadPageLimit     = 100
)

func readPageBounds(total, offset, limit int) (start, end, pageLimit int) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = defaultReadPageLimit
	}
	if limit > maxReadPageLimit {
		limit = maxReadPageLimit
	}
	if offset > total {
		offset = total
	}
	end = offset + limit
	if end > total {
		end = total
	}
	return offset, end, limit
}

func pageHasMore(end, total int) bool { return end < total }
