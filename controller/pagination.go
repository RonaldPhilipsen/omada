package controller

import (
	"context"
	"net/url"
	"strconv"

	models "github.com/RonaldPhilipsen/omada/controller/models"
)

const (
	pageSize = 100
	// maxPages bounds the collector against controllers that report a
	// totalRows larger than the data they actually return; the upstream
	// library loops forever in that situation.
	maxPages = 1000
)

// pageOutcome is one fetched page plus the facts needed to decide whether a
// next page exists.
type pageOutcome struct {
	entries   []map[string]any
	size      int
	totalRows int64
	hasTotal  bool
}

func (c *conn) collectPages(ctx context.Context, fetch func(page int) (pageOutcome, error)) ([]map[string]any, error) {
	var out []map[string]any
	for page := 1; page <= maxPages; page++ {
		res, err := fetch(page)
		if err != nil {
			return nil, err
		}
		out = append(out, res.entries...)
		if len(res.entries) == 0 {
			break
		}
		if res.hasTotal {
			if int64(page)*int64(res.size) >= res.totalRows {
				break
			}
		} else if len(res.entries) < pageSize {
			break
		}
	}
	return out, nil
}

func outcomeFrom(data map[string]any, requestedSize int) (pageOutcome, error) {
	entries := make([]map[string]any, 0, len(models.MapList(data, "data")))
	for _, item := range models.MapList(data, "data") {
		if m, ok := item.(map[string]any); ok {
			entries = append(entries, m)
		}
	}
	size := models.MapInt(data, "currentSize")
	if size <= 0 {
		size = requestedSize
	}
	var out pageOutcome
	out.entries = entries
	out.size = size
	if _, ok := data["totalRows"]; ok {
		out.totalRows = int64(models.MapInt(data, "totalRows"))
		out.hasTotal = true
	}
	return out, nil
}

// pagesV2 collects all pages of a legacy v2 GET endpoint using
// currentPage/currentPageSize.
func (c *conn) pagesV2(ctx context.Context, endpoint, site string, params url.Values) ([]map[string]any, error) {
	return c.collectPages(ctx, func(page int) (pageOutcome, error) {
		q := make(map[string][]string, len(params))
		for k, v := range params {
			q[k] = append([]string(nil), v...)
		}
		query := url.Values(q)
		query.Set("currentPage", strconv.Itoa(page))
		query.Set("currentPageSize", strconv.Itoa(pageSize))
		res, err := c.requestEndpoint(ctx, "GET", "v2", endpoint, site, query, nil)
		if err != nil {
			return pageOutcome{}, err
		}
		return outcomeFrom(res, pageSize)
	})
}

// pagesOpenAPIPost collects all pages of an OpenAPI POST endpoint using
// page/pageSize in the JSON body.
func (c *conn) pagesOpenAPIPost(ctx context.Context, api, endpoint, site string, body map[string]any) ([]map[string]any, error) {
	return c.collectPages(ctx, func(page int) (pageOutcome, error) {
		b := make(map[string]any, len(body)+2)
		for k, v := range body {
			b[k] = v
		}
		b["page"] = page
		b["pageSize"] = pageSize
		res, err := c.requestEndpoint(ctx, "POST", api, endpoint, site, nil, b)
		if err != nil {
			return pageOutcome{}, err
		}
		return outcomeFrom(res, pageSize)
	})
}

// pagesOpenAPI collects all pages of an OpenAPI versioned POST endpoint using
// page/pageSize in the JSON body.
func (c *conn) pagesOpenAPI(ctx context.Context, version, endpoint, site string, body map[string]any) ([]map[string]any, error) {
	return c.collectPages(ctx, func(page int) (pageOutcome, error) {
		b := make(map[string]any, len(body)+2)
		for k, v := range body {
			b[k] = v
		}
		b["page"] = page
		b["pageSize"] = pageSize
		res, err := c.openapiRequest(ctx, "POST", version, endpoint, site, nil, b)
		if err != nil {
			return pageOutcome{}, err
		}
		return outcomeFrom(res, pageSize)
	})
}

// pagesOpenAPIGet collects all pages of an OpenAPI GET endpoint using
// page/pageSize query parameters.
func (c *conn) pagesOpenAPIGet(ctx context.Context, api, endpoint, site string, params url.Values) ([]map[string]any, error) {
	return c.collectPages(ctx, func(page int) (pageOutcome, error) {
		query := make(map[string][]string, len(params)+2)
		for k, v := range params {
			query[k] = append([]string(nil), v...)
		}
		uv := url.Values(query)
		uv.Set("page", strconv.Itoa(page))
		uv.Set("pageSize", strconv.Itoa(pageSize))
		res, err := c.openapiRequest(ctx, "GET", api, endpoint, site, uv, nil)
		if err != nil {
			return pageOutcome{}, err
		}
		return outcomeFrom(res, pageSize)
	})
}
