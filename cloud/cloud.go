package cloud

import (
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

// Options configures a northbound cloud Client. Username and Password are
// required; BaseURL defaults to the US-1 northbound endpoint.
type Options struct {
	BaseURL    string
	Username   string
	Password   string
	HTTPClient *http.Client
}

// Client is a northbound Omada cloud API client. Every domain accessor
// returns a generated client whose requests flow through the bearer-token
// auth transport.
type Client struct {
	baseURL    string
	tp         *TokenProvider
	httpClient *http.Client
}

// New creates a Client. It returns an error when Username or Password is empty.
func New(opts Options) (*Client, error) {
	if opts.Username == "" {
		return nil, errors.New("cloud: Options.Username is required")
	}
	if opts.Password == "" {
		return nil, errors.New("cloud: Options.Password is required")
	}
	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	tp := NewTokenProvider(baseURL, opts.Username, opts.Password, opts.HTTPClient)

	var base http.RoundTripper = http.DefaultTransport
	if opts.HTTPClient != nil && opts.HTTPClient.Transport != nil {
		base = opts.HTTPClient.Transport
	}
	httpClient := &http.Client{Transport: authTransport{base: base, tp: tp}}

	return &Client{baseURL: baseURL, tp: tp, httpClient: httpClient}, nil
}

// authTransport injects a bearer token into every request. On a 401 response
// it force-refreshes the token and retries the request exactly once; a second
// 401 is returned as-is (the token is refreshed again so the next request
// starts with a fresh one).
type authTransport struct {
	base http.RoundTripper
	tp   *TokenProvider
}

func (t authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tok, err := t.tp.Token(req.Context())
	if err != nil {
		return nil, err
	}
	out := req.Clone(req.Context())
	out.Header.Set("Authorization", "Bearer "+tok)

	resp, err := t.base.RoundTrip(out)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		return resp, err
	}

	t.tp.Invalidate()
	tok, err = t.tp.Token(req.Context())
	if err != nil {
		drainAndClose(resp)
		return nil, err
	}
	retry := out.Clone(req.Context())
	retry.Header.Set("Authorization", "Bearer "+tok)
	drainAndClose(resp)

	resp, err = t.base.RoundTrip(retry)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		t.tp.Invalidate()
		_, _ = t.tp.Token(req.Context())
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
