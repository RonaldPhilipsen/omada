package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
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

	authMu       sync.Mutex
	controllerID string
	version      string
	csrf         string
	lastLogon    time.Time
	jar          *permissiveJar
	now          func() time.Time
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
	c.authMu.Lock()
	id := c.controllerID
	c.authMu.Unlock()
	if site != "" {
		endpoint = "sites/" + site + "/" + endpoint
	}
	return c.base + "/" + id + "/api/v2/" + endpoint
}

func (c *conn) openapiURL(version, endpoint, site string) string {
	c.authMu.Lock()
	id := c.controllerID
	c.authMu.Unlock()
	if site != "" {
		endpoint = "sites/" + site + "/" + endpoint
	}
	return c.base + "/openapi/" + version + "/" + id + "/" + endpoint
}

var minControllerVersion = ParseVersionMust("5.1.0")

// clearLoginState drops csrf state and the session cookies for this host.
// Callers must hold authMu.
func (c *conn) clearLoginState() {
	c.csrf = ""
	c.jar.ClearDomain(c.host)
}

// login authenticates against the controller. It must be called with authMu
// held (request() and New do this); it is not safe to call directly
// concurrently with data requests.
func (c *conn) login(ctx context.Context) (string, error) {
	c.clearLoginState()

	info, err := c.do(ctx, "GET", c.base+"/api/info", nil, nil, "")
	if err != nil {
		return "", err
	}
	version := mapStr(info, "controllerVer")
	if v, err := ParseVersion(version); err != nil || v.Compare(minControllerVersion) < 0 {
		return "", fmt.Errorf("%w: %q", ErrUnsupportedControllerVersion, version)
	}
	c.version = version
	c.controllerID = mapStr(info, "omadacId")

	res, err := c.do(ctx, "POST", c.base+"/"+c.controllerID+"/api/v2/login", nil, map[string]any{
		"username": c.username,
		"password": c.password,
	}, "")
	if err != nil {
		return "", err
	}
	c.csrf = mapStr(res, "token")
	c.lastLogon = c.now()
	return c.controllerID, nil
}

// checkLogin reports whether the current session is believed to be valid.
// Callers must hold authMu. Transient (non-auth) failures are returned as an
// error so the caller can decide; only auth-class failures count as "not
// logged in".
func (c *conn) checkLogin(ctx context.Context) (bool, error) {
	if c.csrf == "" {
		return false, nil
	}
	if c.now().Sub(c.lastLogon) < time.Hour {
		return true, nil
	}
	res, err := c.do(ctx, "GET", c.base+"/"+c.controllerID+"/api/v2/loginStatus", nil, nil, c.csrf)
	if err != nil {
		if errors.Is(err, ErrLoginFailed) || errors.Is(err, ErrLoginSessionClosed) {
			return false, nil
		}
		return false, err
	}
	loggedIn := mapBool(res, "login")
	if loggedIn {
		c.lastLogon = c.now()
	}
	return loggedIn, nil
}

// buildURL joins the endpoint onto the base under the current controllerID.
// Callers must hold authMu.
func (c *conn) buildURL(api, endpoint, site string) string {
	if site != "" {
		endpoint = "sites/" + site + "/" + endpoint
	}
	switch api {
	case "v2":
		return c.base + "/" + c.controllerID + "/api/v2/" + endpoint
	default:
		return c.base + "/openapi/" + api + "/" + c.controllerID + "/" + endpoint
	}
}

// requestEndpoint performs an authenticated request, logging in (and
// re-logging in once, on auth-class failures) as needed. The URL is built
// under authMu after the login decision so a concurrent first request can
// never race on an empty controllerID.
func (c *conn) requestEndpoint(ctx context.Context, method, api, endpoint, site string, query url.Values, body any) (map[string]any, error) {
	c.authMu.Lock()
	ok, err := c.checkLogin(ctx)
	if err == nil && !ok {
		_, err = c.login(ctx)
	}
	var u string
	csrf := ""
	if err == nil {
		u = c.buildURL(api, endpoint, site)
		csrf = c.csrf
	}
	c.authMu.Unlock()
	if err != nil {
		return nil, err
	}

	res, err := c.do(ctx, method, u, query, body, csrf)
	if err == nil {
		return res, nil
	}
	if !errors.Is(err, ErrLoginFailed) && !errors.Is(err, ErrLoginSessionClosed) {
		return nil, err
	}
	c.authMu.Lock()
	_, loginErr := c.login(ctx)
	if loginErr == nil {
		u = c.buildURL(api, endpoint, site)
		csrf = c.csrf
	}
	c.authMu.Unlock()
	if loginErr != nil {
		return nil, loginErr
	}
	return c.do(ctx, method, u, query, body, csrf)
}

// do performs a single HTTP request and unwraps the Omada response envelope.
// The caller supplies the csrf token to send (a snapshot taken under authMu);
// do itself never touches shared auth state.
func (c *conn) do(ctx context.Context, method, u string, query url.Values, body any, csrf string) (map[string]any, error) {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reader = bytes.NewReader(payload)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, reader)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBadControllerURL, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if len(query) > 0 {
		req.URL.RawQuery = query.Encode()
	}
	if csrf != "" {
		req.Header.Set("Csrf-Token", csrf)
		req.Header.Set("Omada-Request-Source", "web-local")
		req.Header.Set("Referer", c.base+"/")
		req.Header.Set("Origin", c.base)
	}

	return c.perform(ctx, req)
}

// doMultipart performs a multipart form-data request and unwraps the Omada
// response envelope. The caller supplies the csrf token to send (a snapshot
// taken under authMu); doMultipart itself never touches shared auth state.
func (c *conn) doMultipart(ctx context.Context, method, u string, query url.Values, fields map[string]string, csrf string) (map[string]any, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for name, value := range fields {
		w, err := mw.CreateFormField(name)
		if err != nil {
			return nil, fmt.Errorf("build multipart form: %w", err)
		}
		if _, err := io.WriteString(w, value); err != nil {
			return nil, fmt.Errorf("build multipart form: %w", err)
		}
	}
	mw.Close()
	req, err := http.NewRequestWithContext(ctx, method, u, &buf)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBadControllerURL, err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if len(query) > 0 {
		req.URL.RawQuery = query.Encode()
	}
	if csrf != "" {
		req.Header.Set("Csrf-Token", csrf)
		req.Header.Set("Omada-Request-Source", "web-local")
		req.Header.Set("Referer", c.base+"/")
		req.Header.Set("Origin", c.base)
	}
	return c.perform(ctx, req)
}

// perform performs a single HTTP request and unwraps the Omada response envelope.
func (c *conn) perform(ctx context.Context, req *http.Request) (map[string]any, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	isJSON := strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json")

	if resp.StatusCode != http.StatusOK {
		if isJSON {
			var content map[string]any
			if json.Unmarshal(raw, &content) == nil {
				if appErr := c.checkAppErrors(content); appErr != nil {
					return nil, appErr
				}
			}
		}
		return nil, &RequestError{Code: resp.StatusCode, Msg: "HTTP Request Error"}
	}

	if !isJSON {
		return nil, ErrLoginSessionClosed
	}

	var content map[string]any
	if err := json.Unmarshal(raw, &content); err != nil {
		return nil, &RequestError{Code: -30109, Msg: "Unexpected response: not a JSON object"}
	}
	if appErr := c.checkAppErrors(content); appErr != nil {
		return nil, appErr
	}
	if result, ok := content["result"]; ok {
		if m, ok := result.(map[string]any); ok {
			return m, nil
		}
	}
	return content, nil
}

// checkAppErrors maps a response envelope onto an error, if any.
func (c *conn) checkAppErrors(content map[string]any) error {
	_, ok := content["errorCode"]
	if !ok {
		return &RequestError{Code: -30109, Msg: "Unexpected response: " + fmt.Sprint(content)}
	}
	code := mapInt(content, "errorCode")
	if code == 0 {
		return nil
	}
	msg := mapStr(content, "msg")
	if code == -30109 {
		return &RequestError{Code: -30109, Msg: msg}
	}
	return &RequestError{Code: code, Msg: msg}
}

// controllerInfo fetches unauthenticated controller diagnostics.
func (c *conn) controllerInfo(ctx context.Context) (map[string]any, error) {
	return c.do(ctx, "GET", c.base+"/api/info", nil, nil, "")
}

// controllerVersion returns the controller version, fetching it if unknown.
func (c *conn) controllerVersion(ctx context.Context) (Version, error) {
	c.authMu.Lock()
	cached := c.version
	c.authMu.Unlock()
	if cached != "" {
		return ParseVersion(cached)
	}
	info, err := c.controllerInfo(ctx)
	if err != nil {
		return Version{}, err
	}
	return ParseVersion(mapStr(info, "controllerVer"))
}

// siteInfo fetches a site's detail object.
func (c *conn) siteInfo(ctx context.Context, site string) (map[string]any, error) {
	c.authMu.Lock()
	csrf, id := c.csrf, c.controllerID
	c.authMu.Unlock()
	return c.do(ctx, "GET", c.base+"/"+id+"/api/v2/sites/"+site, nil, nil, csrf)
}

// siteVersion returns the site's reported version.
func (c *conn) siteVersion(ctx context.Context, site string) (Version, error) {
	info, err := c.siteInfo(ctx, site)
	if err != nil {
		return Version{}, err
	}
	return ParseVersion(mapStr(info, "controllerVer"))
}
