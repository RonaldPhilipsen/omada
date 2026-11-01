package cloud

import (
	"context"
	"testing"
)

func TestCollectAllStopsAtTotalRows(t *testing.T) {
	// 3 pages of 2 rows are available, but the envelope says totalRows=4:
	// collection must stop once the total is exceeded and return exactly 4
	// rows after 3 fetches.
	rows := []string{"r1", "r2", "r3", "r4", "r5", "r6"}
	var fetches int
	fetch := func(ctx context.Context, page, pageSize int) (Page[string], error) {
		fetches++
		start := (page - 1) * pageSize
		if start >= len(rows) {
			return Page[string]{TotalRows: 4}, nil
		}
		end := start + pageSize
		if end > len(rows) {
			end = len(rows)
		}
		return Page[string]{Rows: rows[start:end], TotalRows: 4}, nil
	}

	got, err := CollectAll(context.Background(), fetch, 2, 100)
	if err != nil {
		t.Fatalf("CollectAll() error: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("len(got) = %d, want 4", len(got))
	}
	for i, want := range []string{"r1", "r2", "r3", "r4"} {
		if got[i] != want {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want)
		}
	}
	if fetches != 3 {
		t.Fatalf("fetches = %d, want 3", fetches)
	}
}

func TestCollectAllStopsOnShortPage(t *testing.T) {
	// totalRows unknown (0): collection must stop when a page has fewer
	// than pageSize rows.
	pages := [][]string{{"r1", "r2"}, {"r3"}}
	var fetches int
	fetch := func(ctx context.Context, page, pageSize int) (Page[string], error) {
		fetches++
		if page > len(pages) {
			return Page[string]{}, nil
		}
		return Page[string]{Rows: pages[page-1]}, nil
	}

	got, err := CollectAll(context.Background(), fetch, 2, 100)
	if err != nil {
		t.Fatalf("CollectAll() error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len(got) = %d, want 3", len(got))
	}
	if fetches != 2 {
		t.Fatalf("fetches = %d, want 2", fetches)
	}
}

func TestCollectAllBoundsMaxRows(t *testing.T) {
	// fetch always returns a full page with a huge totalRows: maxRows must
	// bound the result (no infinite loop).
	var fetches int
	fetch := func(ctx context.Context, page, pageSize int) (Page[string], error) {
		fetches++
		return Page[string]{Rows: []string{"a", "b"}, TotalRows: 999999}, nil
	}

	got, err := CollectAll(context.Background(), fetch, 2, 5)
	if err != nil {
		t.Fatalf("CollectAll() error: %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("len(got) = %d, want 5", len(got))
	}
	if fetches > 10 {
		t.Fatalf("fetches = %d, want a small bounded number", fetches)
	}
}

func TestCollectAllInvalidArgs(t *testing.T) {
	fetch := func(ctx context.Context, page, pageSize int) (Page[string], error) {
		t.Fatal("fetch must not be called for invalid args")
		return Page[string]{}, nil
	}
	if _, err := CollectAll(context.Background(), fetch, 0, 10); err == nil {
		t.Fatal("pageSize=0: want error, got nil")
	}
	if _, err := CollectAll(context.Background(), fetch, 10, 0); err == nil {
		t.Fatal("maxRows=0: want error, got nil")
	}
}
