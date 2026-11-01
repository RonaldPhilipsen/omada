package conn

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

	models "github.com/RonaldPhilipsen/omada/controller/models"
)

// Connection is an authenticated connection to an Omada controller.
// Data requests run concurrently; only login/re-login transitions are
// serialized internally.
type Connection interface {
	// Login authenticates against the controller and returns the controller ID.
	Login(ctx context.Context) (string, error)
	// RequestEndpoint performs an authenticated v2 API request.
	RequestEndpoint(ctx context.Context, method, api, endpoint, site string, query url.Values, body any) (map[string]any, error)
	// OpenapiRequest performs an authenticated OpenAPI-versioned request.
	OpenapiRequest(ctx context.Context, method, version, endpoint, site string, query url.Values, body any) (map[string]any, error)
	// SiteInfo fetches a site's detail object.
	SiteInfo(ctx context.Context, site string) (map[string]any, error)
	// SiteVersion returns the site's reported version.
	SiteVersion(ctx context.Context, site string) (models.Version, error)
	// DoMultipart performs a multipart form-data request against an explicit URL.
	DoMultipart(ctx context.Context, method, u string, query url.Values, fields map[string]string, csrf string) (map[string]any, error)
	// ControllerInfo fetches unauthenticated controller diagnostics.
	ControllerInfo(ctx context.Context) (map[string]any, error)
	// ControllerVersion returns the controller version, fetching it if unknown.
	ControllerVersion(ctx context.Context) (models.Version, error)
	// BuildURL joins the endpoint onto the base under the current controller ID.
	BuildURL(api, endpoint, site string) string
	// CSRFToken returns the current CSRF token snapshot.
	CSRFToken() string
	// CloseIdleConnections releases pooled idle connections.
	CloseIdleConnections()
	// PagesV2 collects all pages of a legacy v2 GET endpoint.
	PagesV2(ctx context.Context, endpoint, site string, params url.Values) ([]map[string]any, error)
	// PagesOpenAPIPost collects all pages of an OpenAPI POST endpoint.
	PagesOpenAPIPost(ctx context.Context, api, endpoint, site string, body map[string]any) ([]map[string]any, error)
	// PagesOpenAPI collects all pages of an OpenAPI versioned POST endpoint.
	PagesOpenAPI(ctx context.Context, version, endpoint, site string, body map[string]any) ([]map[string]any, error)
	// PagesOpenAPIGet collects all pages of an OpenAPI GET endpoint.
	PagesOpenAPIGet(ctx context.Context, api, endpoint, site string, params url.Values) ([]map[string]any, error)
}

// conn is the Connection implementation: HTTP + auth state.
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

// New creates a Connection for the controller at rawURL. The URL is
// normalized: a missing scheme becomes https://, trailing slashes are
// trimmed, and the bare host (no port) is recorded. No login is performed.
func New(rawURL, username, password string, httpClient *http.Client) (Connection, error) {
	return newConn(rawURL, username, password, httpClient)
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
		return nil, fmt.Errorf("%w: %v", models.ErrBadControllerURL, err)
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

var minControllerVersion = models.ParseVersionMust("5.12.0")

// clearLoginState drops csrf state and the session cookies for this host.
// Callers must hold authMu.
func (c *conn) clearLoginState() {
	c.csrf = ""
	c.jar.ClearDomain(c.host)
}

// Login authenticates against the controller and returns the controller ID.
func (c *conn) Login(ctx context.Context) (string, error) {
	c.authMu.Lock()
	defer c.authMu.Unlock()
	return c.login(ctx)
}

// login authenticates against the controller. It must be called with authMu
// held (Login and RequestEndpoint do this); it is not safe to call directly
// concurrently with data requests.
func (c *conn) login(ctx context.Context) (string, error) {
	c.clearLoginState()

	info, err := c.do(ctx, "GET", c.base+"/api/info", nil, nil, "")
	if err != nil {
		return "", err
	}
	version := models.MapStr(info, "controllerVer")
	if v, err := models.ParseVersion(version); err != nil || v.Compare(minControllerVersion) < 0 {
		return "", fmt.Errorf("%w: %q", models.ErrUnsupportedControllerVersion, version)
	}
	c.version = version
	c.controllerID = models.MapStr(info, "omadacId")

	res, err := c.do(ctx, "POST", c.base+"/"+c.controllerID+"/api/v2/login", nil, map[string]any{
		"username": c.username,
		"password": c.password,
	}, "")
	if err != nil {
		return "", err
	}
	c.csrf = models.MapStr(res, "token")
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
		if errors.Is(err, models.ErrLoginFailed) || errors.Is(err, models.ErrLoginSessionClosed) {
			return false, nil
		}
		return false, err
	}
	loggedIn := models.MapBool(res, "login")
	if loggedIn {
		c.lastLogon = c.now()
	}
	return loggedIn, nil
}

// BuildURL joins the endpoint onto the base under the current controller ID.
func (c *conn) BuildURL(api, endpoint, site string) string {
	c.authMu.Lock()
	defer c.authMu.Unlock()
	return c.buildURL(api, endpoint, site)
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

// RequestEndpoint performs an authenticated request, logging in (and
// re-logging in once, on auth-class failures) as needed. The URL is built
// under authMu after the login decision so a concurrent first request can
// never race on an empty controllerID.
func (c *conn) RequestEndpoint(ctx context.Context, method, api, endpoint, site string, query url.Values, body any) (map[string]any, error) {
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
	if !errors.Is(err, models.ErrLoginFailed) && !errors.Is(err, models.ErrLoginSessionClosed) {
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

// OpenapiRequest performs an authenticated OpenAPI-versioned request, logging
// in (and re-logging in once, on auth-class failures) as needed. The URL is
// built under authMu after the login decision so a concurrent first request
// can never race on an empty controllerID.
func (c *conn) OpenapiRequest(ctx context.Context, method, version, endpoint, site string, query url.Values, body any) (map[string]any, error) {
	c.authMu.Lock()
	ok, err := c.checkLogin(ctx)
	if err == nil && !ok {
		_, err = c.login(ctx)
	}
	var u string
	csrf := ""
	if err == nil {
		if site != "" {
			endpoint = "sites/" + site + "/" + endpoint
		}
		u = c.base + "/openapi/" + version + "/" + c.controllerID + "/" + endpoint
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
	if !errors.Is(err, models.ErrLoginFailed) && !errors.Is(err, models.ErrLoginSessionClosed) {
		return nil, err
	}
	c.authMu.Lock()
	_, loginErr := c.login(ctx)
	var retryCSRF string
	if loginErr == nil {
		retryCSRF = c.csrf
	}
	c.authMu.Unlock()
	if loginErr != nil {
		return nil, loginErr
	}
	return c.do(ctx, method, u, query, body, retryCSRF)
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
		return nil, fmt.Errorf("%w: %v", models.ErrBadControllerURL, err)
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

// DoMultipart performs a multipart form-data request and unwraps the Omada
// response envelope. The caller supplies the csrf token to send (a snapshot
// taken under authMu); DoMultipart itself never touches shared auth state.
func (c *conn) DoMultipart(ctx context.Context, method, u string, query url.Values, fields map[string]string, csrf string) (map[string]any, error) {
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
		return nil, fmt.Errorf("%w: %v", models.ErrBadControllerURL, err)
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
		return nil, fmt.Errorf("%w: %v", models.ErrConnectionFailed, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrConnectionFailed, err)
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
		return nil, &models.RequestError{Code: resp.StatusCode, Msg: "HTTP Request Error"}
	}

	if !isJSON {
		return nil, models.ErrLoginSessionClosed
	}

	var content map[string]any
	if err := json.Unmarshal(raw, &content); err != nil {
		return nil, &models.RequestError{Code: -30109, Msg: "Unexpected response: not a JSON object"}
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
		return &models.RequestError{Code: -30109, Msg: "Unexpected response: " + fmt.Sprint(content)}
	}
	code := models.MapInt(content, "errorCode")
	if code == 0 {
		return nil
	}
	msg := models.MapStr(content, "msg")
	if code == -30109 {
		return &models.RequestError{Code: -30109, Msg: msg}
	}
	return &models.RequestError{Code: code, Msg: msg}
}

// ControllerInfo fetches unauthenticated controller diagnostics.
func (c *conn) ControllerInfo(ctx context.Context) (map[string]any, error) {
	return c.do(ctx, "GET", c.base+"/api/info", nil, nil, "")
}

// ControllerVersion returns the controller version, fetching it if unknown.
func (c *conn) ControllerVersion(ctx context.Context) (models.Version, error) {
	c.authMu.Lock()
	cached := c.version
	c.authMu.Unlock()
	if cached != "" {
		return models.ParseVersion(cached)
	}
	info, err := c.ControllerInfo(ctx)
	if err != nil {
		return models.Version{}, err
	}
	return models.ParseVersion(models.MapStr(info, "controllerVer"))
}

// SiteInfo fetches a site's detail object.
func (c *conn) SiteInfo(ctx context.Context, site string) (map[string]any, error) {
	c.authMu.Lock()
	csrf, id := c.csrf, c.controllerID
	c.authMu.Unlock()
	return c.do(ctx, "GET", c.base+"/"+id+"/api/v2/sites/"+site, nil, nil, csrf)
}

// SiteVersion returns the site's reported version.
func (c *conn) SiteVersion(ctx context.Context, site string) (models.Version, error) {
	info, err := c.SiteInfo(ctx, site)
	if err != nil {
		return models.Version{}, err
	}
	return models.ParseVersion(models.MapStr(info, "controllerVer"))
}

// CSRFToken returns the current CSRF token snapshot.
func (c *conn) CSRFToken() string {
	c.authMu.Lock()
	defer c.authMu.Unlock()
	return c.csrf
}

// CloseIdleConnections releases pooled idle connections.
func (c *conn) CloseIdleConnections() {
	c.client.CloseIdleConnections()
}
