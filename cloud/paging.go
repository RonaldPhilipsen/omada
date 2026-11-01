package cloud

import (
	"context"
	"errors"
)

// Page is one page of results; TotalRows <= 0 means "unknown".
type Page[T any] struct {
	Rows      []T
	TotalRows int64
}

// CollectAll fetches pages via fetch(ctx, page, pageSize) starting at page 1
// until it has TotalRows rows (when known), a short page, or maxRows rows.
// A first error from fetch stops collection and is returned. The result
// never exceeds maxRows, and never exceeds TotalRows when it is known: a
// page that crosses the total is truncated to it.
func CollectAll[T any](ctx context.Context, fetch func(ctx context.Context, page, pageSize int) (Page[T], error), pageSize int, maxRows int64) ([]T, error) {
	if pageSize < 1 {
		return nil, errors.New("cloud: pageSize must be >= 1")
	}
	if maxRows < 1 {
		return nil, errors.New("cloud: maxRows must be >= 1")
	}

	var out []T
	for page := 1; ; page++ {
		p, err := fetch(ctx, page, pageSize)
		if err != nil {
			return nil, err
		}
		out = append(out, p.Rows...)
		// The total is only known to be reached once a page crosses it, so
		// the crossing page is fetched (and truncated) before stopping.
		if p.TotalRows > 0 && int64(len(out)) > p.TotalRows {
			return out[:p.TotalRows], nil
		}
		if len(p.Rows) < pageSize {
			return out, nil
		}
		if int64(len(out)) >= maxRows {
			return out[:maxRows], nil
		}
	}
}
