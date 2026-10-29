package controller

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// conn is the unexported HTTP + auth state shared by Client and SiteClient.
// Data requests run concurrently; only login/re-login transitions are
// serialized via authMu.
type conn struct {
	base     string
	host     string
	username string
	password string
	client   *http.Client

	authMu     sync.Mutex
	controllerID string
	version    string
	csrf       string
	lastLogon  time.Time
	jar        *permissiveJar
	now        func() time.Time
}

// newConn normalizes the controller URL: a missing scheme becomes https://,
// trailing slashes are trimmed, and the bare host (no port) is recorded.
func newConn(rawURL, username, password string, httpClient *http.Client) (*conn, error) {
	if httpClient == nil {
		return nil, fmt.Errorf("nil http client")
	}
	base := rawURL
	if !strings.HasPrefix(strings.ToLower(base), "http://") && !strings.HasPrefix(strings.ToLower(base), "https://") {
		base = "https://" + base
	}
	base = strings.TrimRight(base, "/")
	parsed, err := http.NewRequest("GET", base, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBadControllerURL, err)
	}
	return &conn{
		base:     parsed.URL.String(),
		host:     parsed.URL.Hostname(),
		username: username,
		password: password,
		client:   httpClient,
		jar:      newPermissiveJar().(*permissiveJar),
		now:      time.Now,
	}, nil
}

func (c *conn) url(endpoint, site string) string {
	if site != "" {
		endpoint = "sites/" + site + "/" + endpoint
	}
	return c.base + "/" + c.controllerID + "/api/v2/" + endpoint
}

func (c *conn) openapiURL(version, endpoint, site string) string {
	if site != "" {
		endpoint = "sites/" + site + "/" + endpoint
	}
	return c.base + "/openapi/" + version + "/" + c.controllerID + "/" + endpoint
}
