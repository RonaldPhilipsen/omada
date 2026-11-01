//go:build live

// Package livetest probes every read-only (GET) endpoint of the Omada
// northbound API against a live controller.
//
// Log safety: the report never contains response bodies, resource IDs,
// MAC addresses, IP addresses or any other data from the controller. It
// prints only operation names, HTTP status codes and envelope
// errorCode/msg values. Run it with credentials in the environment:
//
//	OMADA_BASE_URL, OMADA_OMADAC_ID, OMADA_CLIENT_ID, OMADA_CLIENT_SECRET
//
// and build with the live tag:
//
//	go test -tags live ./cloud/livetest -v
package livetest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/RonaldPhilipsen/omada/cloud"
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
	switchpkg "github.com/RonaldPhilipsen/omada/cloud/switch"
	"github.com/RonaldPhilipsen/omada/cloud/topology"
	"github.com/RonaldPhilipsen/omada/cloud/user"
	"github.com/RonaldPhilipsen/omada/cloud/voip"
	"github.com/RonaldPhilipsen/omada/cloud/vpn"
	"github.com/RonaldPhilipsen/omada/cloud/wireless"
)

// opEntry is one row of the generated liveOps table.
type opEntry struct {
	name string
	need []string
	call func(ctx context.Context, s *scope) (*http.Response, error)
}

// Result is the outcome of probing one endpoint.
type Result string

const (
	ResultPass Result = "PASS"
	ResultBiz  Result = "BIZ"
	ResultSkip Result = "SKIP"
	ResultFail Result = "FAIL"
)

// Report summarizes a live run. Every string in it is log-safe: operation
// names, status codes and envelope codes/messages only.
type Report struct {
	Total int
	Pass  int
	Biz   int
	Skip  int
	Fail  int
	// Failures and BizLines are ready to print, one line each.
	Failures       []string
	BizLines       []string
	SkippedByParam map[string]int
}

// Config carries the on-prem org-level credentials for a live run.
type Config struct {
	BaseURL      string
	OmadacID     string
	ClientID     string
	ClientSecret string
	// InsecureTLS disables certificate verification (controllers with the
	// stock self-signed CN=localhost certificate).
	InsecureTLS bool
	// PerOpTimeout bounds a single endpoint call. Zero means 30s.
	PerOpTimeout time.Duration
}

// ConfigFromEnv reads the live configuration from the environment.
func ConfigFromEnv() (Config, error) {
	cfg := Config{
		BaseURL:      os.Getenv("OMADA_BASE_URL"),
		OmadacID:     os.Getenv("OMADA_OMADAC_ID"),
		ClientID:     os.Getenv("OMADA_CLIENT_ID"),
		ClientSecret: os.Getenv("OMADA_CLIENT_SECRET"),
		InsecureTLS:  os.Getenv("OMADA_INSECURE_TLS") == "1" || os.Getenv("OMADA_INSECURE_TLS") == "true",
	}
	var missing []string
	for name, v := range map[string]string{
		"OMADA_BASE_URL":      cfg.BaseURL,
		"OMADA_OMADAC_ID":     cfg.OmadacID,
		"OMADA_CLIENT_ID":     cfg.ClientID,
		"OMADA_CLIENT_SECRET": cfg.ClientSecret,
	} {
		if v == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return cfg, fmt.Errorf("missing environment: %v", missing)
	}
	return cfg, nil
}

// scope is the discovered resource context. The ID fields are used to
// resolve path parameters; they are never printed.
type scope struct {
	ap       *ap.Client
	clients  *client.Client
	devices  *device.Client
	gateways *gateway.Client
	insight  *insight.Client
	lans     *lan.Client
	omadalog *omadalog.Client
	msp      *msp.Client
	remote   *remote.Client
	security *security.Client
	sims     *sim.Client
	site     *site.Client
	stacks   *stack.Client
	switches *switchpkg.Client
	topology *topology.Client
	users    *user.Client
	voip     *voip.Client
	vpn      *vpn.Client
	wireless *wireless.Client

	omadacID   string
	siteID     string
	mspID      string
	customerID string
	deviceMac  string
	apID       string
	switchID   string
	gatewayID  string
	networkID  string
	ssidID     string
	vpnID      string
	clientMac  string
	// missing is never set: entries that require a path parameter the
	// scope cannot discover are skipped.
	missing string
}

func (s *scope) value(param string) string {
	switch param {
	case "omadacId":
		return s.omadacID
	case "siteId":
		return s.siteID
	case "mspId":
		return s.mspID
	case "customerId":
		return s.customerID
	case "deviceMac", "mac":
		return s.deviceMac
	case "apMac":
		return s.apID
	case "switchMac":
		return s.switchID
	case "gatewayMac":
		return s.gatewayID
	case "networkId":
		return s.networkID
	case "ssidId":
		return s.ssidID
	case "vpnId":
		return s.vpnID
	case "clientMac":
		return s.clientMac
	}
	return ""
}

// Run probes every GET operation in the generated table against the live
// controller and reports the outcomes.
func Run(ctx context.Context, cfg Config) (*Report, error) {
	if cfg.PerOpTimeout == 0 {
		cfg.PerOpTimeout = 30 * time.Second
	}
	cl, err := cloud.New(cloud.Options{
		BaseURL:      cfg.BaseURL,
		OmadacID:     cfg.OmadacID,
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		InsecureTLS:  cfg.InsecureTLS,
	})
	if err != nil {
		return nil, err
	}

	// Preflight: surface token-endpoint problems (bad credentials, HTTP
	// 500 from the controller) as one clear line instead of a transport
	// error on every single endpoint.
	pctx, pcancel := context.WithTimeout(ctx, cfg.PerOpTimeout)
	if err := cl.PreflightToken(pctx); err != nil {
		pcancel()
		return nil, fmt.Errorf("token preflight: %w", err)
	}
	pcancel()

	s := &scope{
		omadacID: cfg.OmadacID,
		ap:       cl.Ap(),
		clients:  cl.Clients(),
		devices:  cl.Devices(),
		gateways: cl.Gateway(),
		insight:  cl.Insight(),
		lans:     cl.Lan(),
		omadalog: cl.OmadaLog(),
		msp:      cl.Msp(),
		remote:   cl.Remote(),
		security: cl.Security(),
		sims:     cl.Sim(),
		site:     cl.Site(),
		stacks:   cl.Stack(),
		switches: cl.Switch(),
		topology: cl.Topology(),
		users:    cl.Users(),
		voip:     cl.Voip(),
		vpn:      cl.Vpn(),
		wireless: cl.Wireless(),
	}
	discover(ctx, cfg.PerOpTimeout, s)

	rep := &Report{Total: len(liveOps), SkippedByParam: map[string]int{}}
	for _, op := range liveOps {
		missing := ""
		for _, p := range op.need {
			if s.value(p) == "" {
				missing = p
				break
			}
		}
		if missing != "" {
			rep.Skip++
			rep.SkippedByParam[missing]++
			continue
		}
		opCtx, cancel := context.WithTimeout(ctx, cfg.PerOpTimeout)
		resp, callErr := op.call(opCtx, s)
		cancel()
		if callErr != nil {
			rep.Fail++
			rep.Failures = append(rep.Failures, op.name+": "+transportDetail(callErr))
			continue
		}
		res, detail := classifyResponse(resp)
		switch res {
		case ResultPass:
			rep.Pass++
		case ResultBiz:
			rep.Biz++
			rep.BizLines = append(rep.BizLines, op.name+": "+detail)
		default:
			rep.Fail++
			rep.Failures = append(rep.Failures, op.name+": "+detail)
		}
	}
	return rep, nil
}

// transportDetail renders a call error without the request URL (which would
// embed discovered resource IDs).
func transportDetail(err error) string {
	if strings.Contains(err.Error(), "deadline exceeded") {
		return "timeout"
	}
	return "transport error"
}

// classifyResponse drains the response and maps it to a Result. The detail
// string never contains the response body.
func classifyResponse(resp *http.Response) (Result, string) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ResultFail, "read response body"
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return ResultFail, fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	var env struct {
		ErrorCode *int32  `json:"errorCode"`
		Msg       *string `json:"msg"`
	}
	if json.Unmarshal(body, &env) != nil {
		// Bare or binary payload (e.g. file export): a 2xx is a pass.
		return ResultPass, ""
	}
	if env.ErrorCode != nil && *env.ErrorCode != 0 {
		msg := ""
		if env.Msg != nil {
			msg = *env.Msg
		}
		return ResultBiz, fmt.Sprintf("errorCode=%d msg=%s", *env.ErrorCode, msg)
	}
	return ResultPass, ""
}

// discover resolves the resource IDs that path parameters need. Every step
// is best effort: a failed lookup just leaves the ID empty and the
// dependent endpoints are skipped. Nothing discovered is printed.
func discover(ctx context.Context, perOp time.Duration, s *scope) {
	call := func(f func(context.Context) (*http.Response, error)) *http.Response {
		cctx, cancel := context.WithTimeout(ctx, perOp)
		defer cancel()
		resp, err := f(cctx)
		if err != nil {
			return nil
		}
		return resp
	}

	// Site list: the root of site-scoped discovery.
	if resp := call(func(c context.Context) (*http.Response, error) {
		return s.site.GetSiteList(c, s.omadacID, &site.GetSiteListParams{Page: 1, PageSize: 10})
	}); resp != nil {
		s.siteID = firstIDFrom(resp, "siteId")
	}
	if s.siteID == "" {
		return
	}

	// Device list: first device overall plus first per type.
	if resp := call(func(c context.Context) (*http.Response, error) {
		return s.devices.GetDeviceList(c, s.omadacID, s.siteID, &device.GetDeviceListParams{Page: 1, PageSize: 100})
	}); resp != nil {
		rows, _ := listRows(resp)
		for _, row := range rows {
			mac, _ := row["mac"].(string)
			typ, _ := row["type"].(string)
			if mac == "" {
				continue
			}
			if s.deviceMac == "" {
				s.deviceMac = mac
			}
			switch typ {
			case "ap":
				if s.apID == "" {
					s.apID = mac
				}
			case "switch":
				if s.switchID == "" {
					s.switchID = mac
				}
			case "gateway":
				if s.gatewayID == "" {
					s.gatewayID = mac
				}
			}
		}
	}

	// MSP probe: an org-level API serves MSP endpoints with mspId =
	// omadacId when the org is an MSP; a business error means it is not.
	if resp := call(func(c context.Context) (*http.Response, error) {
		return s.msp.GetMspMailServerStatusGlobal(c, s.omadacID)
	}); resp != nil {
		if res, _ := classifyResponse(resp); res == ResultPass {
			s.mspID = s.omadacID
		}
	}

	if s.mspID != "" {
		if resp := call(func(c context.Context) (*http.Response, error) {
			return s.site.GetCustomerList(c, s.mspID, &site.GetCustomerListParams{Page: 1, PageSize: 10})
		}); resp != nil {
			s.customerID = firstIDFrom(resp, "customerId")
		}
	}

	if resp := call(func(c context.Context) (*http.Response, error) {
		return s.lans.GetLanNetworkList(c, s.omadacID, s.siteID, &lan.GetLanNetworkListParams{Page: 1, PageSize: 10})
	}); resp != nil {
		s.networkID = firstIDFrom(resp, "lanId", "networkId", "id")
	}

	if resp := call(func(c context.Context) (*http.Response, error) {
		// Type 3 = ap and wireless router.
		return s.wireless.GetSsidsBySite(c, s.omadacID, s.siteID, &wireless.GetSsidsBySiteParams{Type: 3})
	}); resp != nil {
		s.ssidID = firstIDFrom(resp, "ssidId", "id")
	}

	if resp := call(func(c context.Context) (*http.Response, error) {
		return s.vpn.GetAllVpnList(c, s.omadacID, s.siteID)
	}); resp != nil {
		s.vpnID = firstIDFrom(resp, "id", "vpnId")
	}

	if resp := call(func(c context.Context) (*http.Response, error) {
		return s.clients.GetGridActiveClients(c, s.omadacID, s.siteID, &client.GetGridActiveClientsParams{Page: 1, PageSize: 10})
	}); resp != nil {
		s.clientMac = firstIDFrom(resp, "mac")
	}
}

// listRows extracts the data rows of a list response: result.data when the
// result is a grid object, result itself when it is a bare array.
func listRows(resp *http.Response) ([]map[string]interface{}, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var env struct {
		Result json.RawMessage `json:"result"`
	}
	if json.Unmarshal(body, &env) != nil || len(env.Result) == 0 {
		return nil, nil
	}
	var grid struct {
		Data []map[string]interface{} `json:"data"`
	}
	if json.Unmarshal(env.Result, &grid) == nil && grid.Data != nil {
		return grid.Data, nil
	}
	var bare []map[string]interface{}
	if json.Unmarshal(env.Result, &bare) == nil {
		return bare, nil
	}
	return nil, nil
}

// firstIDFrom reads a list response and returns the first row's value for
// the first key that is present. The value is used internally only.
func firstIDFrom(resp *http.Response, keys ...string) string {
	rows, _ := listRows(resp)
	for _, row := range rows {
		for _, k := range keys {
			if v, ok := row[k].(string); ok && v != "" {
				return v
			}
		}
	}
	return ""
}

// Summary renders the log-safe report summary lines.
func (r *Report) Summary() []string {
	lines := []string{
		fmt.Sprintf("live read-only: total=%d pass=%d biz=%d skip=%d fail=%d",
			r.Total, r.Pass, r.Biz, r.Skip, r.Fail),
	}
	if len(r.SkippedByParam) > 0 {
		params := make([]string, 0, len(r.SkippedByParam))
		for p, n := range r.SkippedByParam {
			params = append(params, fmt.Sprintf("%s=%d", p, n))
		}
		sort.Strings(params)
		lines = append(lines, "skipped by missing param: "+join(params))
	}
	lines = append(lines, r.BizLines...)
	lines = append(lines, r.Failures...)
	return lines
}

func join(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}
