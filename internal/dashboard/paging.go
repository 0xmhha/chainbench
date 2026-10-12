package dashboard

import (
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

// errPage refuses a malformed page request with 400.
var errPage = errors.New("invalid cursor or limit")

// pageOf cuts one page from a list ordered by id, so a cursor means the same
// position on every request whatever order the store keeps. limit defaults to 25
// and may be 1 to 100; cursor is the id of the last item already read, and
// nextCursor is null on the last page.
func pageOf[T any](r *http.Request, items []T, id func(T) string) (map[string]any, error) {
	limit := 25
	if raw := r.URL.Query().Get("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 || v > 100 {
			return nil, errPage
		}
		limit = v
	}
	items = slices.Clone(items)
	slices.SortStableFunc(items, func(a, b T) int { return strings.Compare(id(a), id(b)) })
	start := 0
	if cursor := r.URL.Query().Get("cursor"); cursor != "" {
		start = -1
		for i, item := range items {
			if id(item) == cursor {
				start = i + 1
				break
			}
		}
		if start < 0 {
			return nil, errPage
		}
	}
	end := min(start+limit, len(items))
	page := append([]T{}, items[start:end]...)
	var next any
	if end < len(items) {
		next = id(items[end-1])
	}
	return map[string]any{"items": page, "nextCursor": next}, nil
}

// writePage answers a list request one page at a time.
func writePage[T any](w http.ResponseWriter, r *http.Request, items []T, id func(T) string) {
	page, err := pageOf(r, items, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	deploymentJSON(w, http.StatusOK, page)
}
