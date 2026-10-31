package site_test

import (
	"context"
	"testing"

	"github.com/RonaldPhilipsen/omada/controller"
	"github.com/RonaldPhilipsen/omada/controller/site"
)

// mustSiteClient returns a site-scoped handle for the mock site-1, failing the
// test when the site lookup does not validate.
func mustSiteClient(t *testing.T, cl *controller.Controller) *site.SiteClient {
	t.Helper()
	sc, err := cl.SiteClient(context.Background(), "site-1")
	if err != nil {
		t.Fatalf("SiteClient() error = %v", err)
	}
	return sc
}
