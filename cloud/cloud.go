package cloud

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/RonaldPhilipsen/omada/cloud/ap"
	"github.com/RonaldPhilipsen/omada/cloud/client"
	"github.com/RonaldPhilipsen/omada/cloud/device"
	"github.com/RonaldPhilipsen/omada/cloud/gateway"
	"github.com/RonaldPhilipsen/omada/cloud/insight"
	"github.com/RonaldPhilipsen/omada/cloud/lan"
	omadalog "github.com/RonaldPhilipsen/omada/cloud/log"
	"github.com/RonaldPhilipsen/omada/cloud/msp"
	"github.com/RonaldPhilipsen/omada/cloud/remote"
	"github.com/RonaldPhilipsen/omada/cloud/security"
	"github.com/RonaldPhilipsen/omada/cloud/sim"
	"github.com/RonaldPhilipsen/omada/cloud/site"
	"github.com/RonaldPhilipsen/omada/cloud/stack"
	"github.com/RonaldPhilipsen/omada/cloud/switch"
	"github.com/RonaldPhilipsen/omada/cloud/topology"
	"github.com/RonaldPhilipsen/omada/cloud/user"
	"github.com/RonaldPhilipsen/omada/cloud/voip"
	"github.com/RonaldPhilipsen/omada/cloud/vpn"
	"github.com/RonaldPhilipsen/omada/cloud/wireless"
)

const defaultBaseURL = "https://use1-omada-northbound.tplinkcloud.com"

// Options configures a northbound cloud Client. Exactly one credential set
// is required: hosted (Username + Password) or on-prem org-level
// (OmadacID + ClientID + ClientSecret). BaseURL defaults to the US-1
// northbound endpoint only in hosted mode; it is required in on-prem mode.
type Options struct {
	BaseURL      string
	Username     string // hosted mode
	Password     string // hosted mode
	OmadacID     string // on-prem org-level mode
	ClientID     string // on-prem org-level mode
	ClientSecret string // on-prem org-level mode
	// InsecureTLS disables TLS certificate verification for every request,
	// including token requests. Enable it only for controllers that serve
	// the stock self-signed certificate (CN=localhost, no SAN), which
	// fails hostname verification in every modern client. Prefer installing
	// a proper certificate when possible: with verification off the
	// credentials are exposed to any on-path attacker.
	InsecureTLS bool
	HTTPClient  *http.Client
}

// Client is a northbound Omada cloud API client. Every domain accessor
// returns a generated client whose requests flow through the auth transport.
type Client struct {
	baseURL    string
	auth       authenticator
	httpClient *http.Client
}

// New creates a Client. Exactly one credential set must be provided: hosted
// (Username + Password) or on-prem org-level (OmadacID + ClientID +
// ClientSecret); both or neither is an error. BaseURL defaults to the
// hosted US-1 endpoint only in hosted mode; on-prem mode requires an
// explicit BaseURL.
func New(opts Options) (*Client, error) {
	hosted := opts.Username != "" && opts.Password != ""
	org := opts.OmadacID != "" && opts.ClientID != "" && opts.ClientSecret != ""
	if hosted == org {
		return nil, errors.New("cloud: exactly one credential set is required: hosted (Username + Password) or on-prem org-level (OmadacID + ClientID + ClientSecret)")
	}

	baseURL := opts.BaseURL
	if hosted {
		if baseURL == "" {
			baseURL = defaultBaseURL
		}
	} else if baseURL == "" {
		return nil, errors.New("cloud: Options.BaseURL is required in on-prem org-level mode")
	}
	baseURL = strings.TrimRight(baseURL, "/")

	if opts.InsecureTLS {
		if opts.HTTPClient == nil {
			opts.HTTPClient = &http.Client{}
		}
		opts.HTTPClient.Transport = &http.Transport{
			Proxy:             http.ProxyFromEnvironment,
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
			ForceAttemptHTTP2: true,
		}
	}

	var auth authenticator
	if hosted {
		auth = NewTokenProvider(baseURL, opts.Username, opts.Password, opts.HTTPClient)
	} else {
		auth = NewOrgTokenProvider(baseURL, opts.OmadacID, opts.ClientID, opts.ClientSecret, opts.HTTPClient)
	}

	var base http.RoundTripper = http.DefaultTransport
	if opts.HTTPClient != nil && opts.HTTPClient.Transport != nil {
		base = opts.HTTPClient.Transport
	}
	httpClient := &http.Client{Transport: authTransport{base: base, auth: auth}}

	return &Client{baseURL: baseURL, auth: auth, httpClient: httpClient}, nil
}

// HTTPClient returns the client's auth transport wrapped http.Client for
// generic dispatch-style calls through the same token provider.
func (c *Client) HTTPClient() *http.Client { return c.httpClient }

// InvalidateToken drops the cached token so the next request re-authenticates.
func (c *Client) InvalidateToken() {
	c.auth.Invalidate()
}

// PreflightToken forces a token fetch (or serves the cached token) so
// callers can surface authentication problems — bad credentials,
// unreachable or failing token endpoints — before making domain calls.
func (c *Client) PreflightToken(ctx context.Context) error {
	_, err := c.auth.HeaderValue(ctx)
	return err
}

// authTransport injects an auth header into every request. On a 401 response
// it force-refreshes the token and retries the request exactly once,
// rebuilding the request body from GetBody when available so the retry
// carries the same body; a second 401 is returned as-is (the token is
// refreshed again so the next request starts with a fresh one).
type authTransport struct {
	base http.RoundTripper
	auth authenticator
}

func (t authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tok, err := t.auth.HeaderValue(req.Context())
	if err != nil {
		return nil, err
	}
	out := req.Clone(req.Context())
	out.Header.Set("Authorization", tok)

	resp, err := t.base.RoundTrip(out)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		return resp, err
	}

	t.auth.Invalidate()
	tok, err = t.auth.HeaderValue(req.Context())
	if err != nil {
		drainAndClose(resp)
		return nil, err
	}
	retry := out.Clone(req.Context())
	retry.Header.Set("Authorization", tok)
	// The first attempt consumed the body reader (shared with the original
	// request); rebuild it so the retry carries the same body.
	if out.GetBody != nil {
		body, berr := out.GetBody()
		if berr != nil {
			drainAndClose(resp)
			return nil, berr
		}
		retry.Body = body
		retry.ContentLength = out.ContentLength
	}
	drainAndClose(resp)

	resp, err = t.base.RoundTrip(retry)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		t.auth.Invalidate()
		_, _ = t.auth.HeaderValue(req.Context())
	}
	return resp, nil
}

func drainAndClose(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

// Site returns the site domain client.
func (c *Client) Site() *site.Client {
	cl, _ := site.NewClient(c.baseURL, site.WithHTTPClient(c.httpClient))
	return cl
}

// Clients returns the customer domain client.
func (c *Client) Clients() *client.Client {
	cl, _ := client.NewClient(c.baseURL, client.WithHTTPClient(c.httpClient))
	return cl
}

// Devices returns the device domain client.
func (c *Client) Devices() *device.Client {
	cl, _ := device.NewClient(c.baseURL, device.WithHTTPClient(c.httpClient))
	return cl
}

// Ap returns the AP domain client.
func (c *Client) Ap() *ap.Client {
	cl, _ := ap.NewClient(c.baseURL, ap.WithHTTPClient(c.httpClient))
	return cl
}

// Switch returns the switch domain client.
func (c *Client) Switch() *switchpkg.Client {
	cl, _ := switchpkg.NewClient(c.baseURL, switchpkg.WithHTTPClient(c.httpClient))
	return cl
}

// Gateway returns the gateway domain client.
func (c *Client) Gateway() *gateway.Client {
	cl, _ := gateway.NewClient(c.baseURL, gateway.WithHTTPClient(c.httpClient))
	return cl
}

// Lan returns the LAN domain client.
func (c *Client) Lan() *lan.Client {
	cl, _ := lan.NewClient(c.baseURL, lan.WithHTTPClient(c.httpClient))
	return cl
}

// Wireless returns the wireless domain client.
func (c *Client) Wireless() *wireless.Client {
	cl, _ := wireless.NewClient(c.baseURL, wireless.WithHTTPClient(c.httpClient))
	return cl
}

// Vpn returns the VPN domain client.
func (c *Client) Vpn() *vpn.Client {
	cl, _ := vpn.NewClient(c.baseURL, vpn.WithHTTPClient(c.httpClient))
	return cl
}

// Security returns the security domain client.
func (c *Client) Security() *security.Client {
	cl, _ := security.NewClient(c.baseURL, security.WithHTTPClient(c.httpClient))
	return cl
}

// Insight returns the insight domain client.
func (c *Client) Insight() *insight.Client {
	cl, _ := insight.NewClient(c.baseURL, insight.WithHTTPClient(c.httpClient))
	return cl
}

// Users returns the user domain client.
func (c *Client) Users() *user.Client {
	cl, _ := user.NewClient(c.baseURL, user.WithHTTPClient(c.httpClient))
	return cl
}

// Msp returns the MSP domain client.
func (c *Client) Msp() *msp.Client {
	cl, _ := msp.NewClient(c.baseURL, msp.WithHTTPClient(c.httpClient))
	return cl
}

// Stack returns the stack domain client.
func (c *Client) Stack() *stack.Client {
	cl, _ := stack.NewClient(c.baseURL, stack.WithHTTPClient(c.httpClient))
	return cl
}

// Voip returns the VoIP domain client.
func (c *Client) Voip() *voip.Client {
	cl, _ := voip.NewClient(c.baseURL, voip.WithHTTPClient(c.httpClient))
	return cl
}

// Sim returns the SIM domain client.
func (c *Client) Sim() *sim.Client {
	cl, _ := sim.NewClient(c.baseURL, sim.WithHTTPClient(c.httpClient))
	return cl
}

// OmadaLog returns the log domain client.
func (c *Client) OmadaLog() *omadalog.Client {
	cl, _ := omadalog.NewClient(c.baseURL, omadalog.WithHTTPClient(c.httpClient))
	return cl
}

// Topology returns the topology domain client.
func (c *Client) Topology() *topology.Client {
	cl, _ := topology.NewClient(c.baseURL, topology.WithHTTPClient(c.httpClient))
	return cl
}

// Remote returns the remote domain client.
func (c *Client) Remote() *remote.Client {
	cl, _ := remote.NewClient(c.baseURL, remote.WithHTTPClient(c.httpClient))
	return cl
}
