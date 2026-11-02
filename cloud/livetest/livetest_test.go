//go:build live

package livetest

import (
	"context"
	"testing"
	"time"
)

// TestLiveReadOnly probes every read-only (GET) endpoint of the Omada
// northbound API against a live controller. It skips when the OMADA_*
// environment is not set, so it is safe in plain `go test ./...` runs with
// the live tag and in CI without credentials.
//
// The report is log-safe by construction: no response bodies, resource IDs,
// MAC or IP addresses ever reach the output (see livetest.go).
func TestLiveReadOnly(t *testing.T) {
	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Skipf("live read-only test skipped: %v", err)
	}
	cfg.PerOpTimeout = 30 * time.Second

	rep, err := Run(context.Background(), cfg)
	if err != nil {
		t.Fatalf("live run: %v", err)
	}
	for _, line := range rep.Summary() {
		t.Logf("%s", line)
	}
	if rep.Fail > 0 {
		t.Fatalf("%d of %d read-only endpoints failed", rep.Fail, rep.Total)
	}
}
