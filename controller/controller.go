package controller

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"
)

// Controller is the controller-level API facade. Controller-scoped operations
// that do not belong to a site are served through it.
type Controller struct {
	c *conn
}

// Option customizes a Controller.
type Option func(*options)

// options collects the resolved Client options.
type options struct {
	httpClient         *http.Client
	timeout            time.Duration
	insecureSkipVerify bool
}

// defaultRequestTimeout is used when no WithTimeout option is supplied.
const defaultRequestTimeout = 30 * time.Second

// WithHTTPClient sets the *http.Client used for controller requests.
func WithHTTPClient(hc *http.Client) Option { return func(o *options) { o.httpClient = hc } }

// WithTimeout sets the timeout applied to controller requests.
func WithTimeout(d time.Duration) Option { return func(o *options) { o.timeout = d } }

// WithInsecureSkipVerify enables TLS certificate verification skipping.
func WithInsecureSkipVerify() Option { return func(o *options) { o.insecureSkipVerify = true } }

// New creates a Controller for the controller at controllerURL and performs the
// initial login. New has no context; the initial login uses context.Background.
func New(controllerURL, username, password string, opts ...Option) (*Controller, error) {
	o := &options{timeout: defaultRequestTimeout}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(o)
	}
	hc := o.httpClient
	if hc == nil {
		hc = &http.Client{Timeout: o.timeout}
		if o.insecureSkipVerify {
			hc.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		}
	}
	c, err := newConn(controllerURL, username, password, hc)
	if err != nil {
		return nil, err
	}
	cl := &Controller{c: c}
	if err := cl.login(context.Background()); err != nil {
		hc.CloseIdleConnections()
		return nil, err
	}
	return cl, nil
}

// login authenticates the client against the controller.
func (cl *Controller) login(ctx context.Context) error {
	cl.c.authMu.Lock()
	defer cl.c.authMu.Unlock()
	_, err := cl.c.login(ctx)
	return err
}

// url builds the controller-scoped URL for an API endpoint. Callers must hold authMu.
func (cl *Controller) url(api, endpoint, site string) string {
	return cl.c.buildURL(api, endpoint, site)
}

// Sites lists the sites the current user can administer.
func (cl *Controller) Sites(ctx context.Context) ([]Site, error) {
	m, err := cl.c.requestEndpoint(ctx, "GET", "v2", "users/current", "", nil, nil)
	if err != nil {
		return nil, err
	}
	priv, ok := mapMap(m, "privilege")
	if !ok {
		return nil, fmt.Errorf("unexpected sites response: %v", m)
	}
	entries := mapList(priv, "sites")
	sites := make([]Site, 0, len(entries))
	for _, entry := range entries {
		sm, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		sites = append(sites, Site{Name: mapStr(sm, "name"), ID: mapStr(sm, "key")})
	}
	return sites, nil
}

// Site returns the site with the given controller site ID.
func (cl *Controller) Site(ctx context.Context, siteID string) (Site, error) {
	if siteID == "" {
		return Site{}, &ValidationError{Msg: "site ID must not be empty"}
	}
	m, err := cl.c.siteInfo(ctx, siteID)
	if err != nil {
		return Site{}, err
	}
	return Site{Name: mapStr(m, "name"), ID: mapStr(m, "id")}, nil
}

// SiteByName resolves a site by display name.
func (cl *Controller) SiteByName(ctx context.Context, name string) (Site, error) {
	sites, err := cl.Sites(ctx)
	if err != nil {
		return Site{}, err
	}
	for _, s := range sites {
		if s.Name == name {
			return s, nil
		}
	}
	return Site{}, fmt.Errorf("%w: site %q", ErrSiteNotFound, name)
}

// ControllerInfo fetches unauthenticated controller diagnostic information.
func (cl *Controller) ControllerInfo(ctx context.Context) (ControllerInfo, error) {
	m, err := cl.c.controllerInfo(ctx)
	if err != nil {
		return ControllerInfo{}, err
	}
	return NewControllerInfo(m), nil
}

// ControllerVersion returns the controller software version.
func (cl *Controller) ControllerVersion(ctx context.Context) (Version, error) {
	return cl.c.controllerVersion(ctx)
}

// Reboot reboots the controller.
func (cl *Controller) Reboot(ctx context.Context) error {
	_, err := cl.c.requestEndpoint(ctx, "POST", "v2", "cmd/reboot", "", nil, nil)
	return err
}

// Shutdown shuts the controller down.
func (cl *Controller) Shutdown(ctx context.Context) error {
	_, err := cl.c.requestEndpoint(ctx, "POST", "v2", "cmd/shutdown", "", nil, nil)
	return err
}

// InstallControllerFirmware installs controller firmware to the given target version.
func (cl *Controller) InstallControllerFirmware(ctx context.Context, targetVersion string) error {
	if targetVersion == "" {
		return &ValidationError{Msg: "target version must not be empty"}
	}
	body := map[string]any{"targetVersion": targetVersion}
	_, err := cl.c.requestEndpoint(ctx, "POST", "v2", "cmd/upgradeFirmware", "", nil, body)
	return err
}

// GetControllerUpgradeStatus returns the status of an ongoing hardware controller firmware upgrade.
func (cl *Controller) GetControllerUpgradeStatus(ctx context.Context) (*HardwareUpgradeStatus, error) {
	m, err := cl.c.requestEndpoint(ctx, "GET", "v2", "maintenance/hardware/upgradeStatus", "", nil, nil)
	if err != nil {
		return nil, err
	}
	status := NewHardwareUpgradeStatus(m)
	return &status, nil
}

// SiteClient validates the site and returns a site-scoped client handle.
func (cl *Controller) SiteClient(ctx context.Context, siteID string) (*SiteClient, error) {
	if _, err := cl.Site(ctx, siteID); err != nil {
		return nil, err
	}
	return &SiteClient{c: cl.c, site: siteID}, nil
}

// FirmwareUpdates checks for controller firmware updates.
func (cl *Controller) FirmwareUpdates(ctx context.Context) (ControllerUpdateInfo, error) {
	version, err := cl.ControllerVersion(ctx)
	if err != nil {
		return ControllerUpdateInfo{}, err
	}
	endpoint := "controller/notification/updateInfo"
	if version.Compare(ParseVersionMust("6.3")) >= 0 {
		endpoint = "maintenance/software/channelUpdate"
	}
	m, err := cl.c.requestEndpoint(ctx, "GET", "v2", endpoint, "", nil, nil)
	if err != nil {
		return ControllerUpdateInfo{}, err
	}
	return NewControllerUpdateInfo(m), nil
}

// UpdateFirmwareChannel selects the firmware release channel to check.
func (cl *Controller) UpdateFirmwareChannel(ctx context.Context, section UpdateSection, channel string) error {
	body := map[string]any{
		"section": string(section),
		"channel": channel,
	}
	_, err := cl.c.requestEndpoint(ctx, "POST", "v2", "upgrade", "", nil, body)
	return err
}

// SetCertificate uploads a new certificate and enables it.
func (cl *Controller) SetCertificate(ctx context.Context, keyPEM, certPEM string) error {
	m, err := cl.c.doMultipart(ctx, "POST", cl.url("v2", "cmd/uploadCert", ""), nil, map[string]string{
		"key":  keyPEM,
		"cert": certPEM,
	}, cl.csrfToken())
	if err != nil {
		return err
	}
	payload := map[string]any{
		"certificate": map[string]any{
			"cerId":       mapStr(m, "cerId"),
			"cerName":     mapStr(m, "cerName"),
			"cerType":     "PFX",
			"enable":      true,
			"keyPassword": "",
		},
	}
	_, err = cl.c.requestEndpoint(ctx, "PATCH", "v2", "controller/setting", "", nil, payload)
	return err
}

// Close releases pooled idle connections.
func (cl *Controller) Close() error {
	cl.c.client.CloseIdleConnections()
	return nil
}

// csrfToken returns the current CSRF token snapshot.
func (cl *Controller) csrfToken() string {
	cl.c.authMu.Lock()
	defer cl.c.authMu.Unlock()
	return cl.c.csrf
}

// UpdateSection identifies the hardware or software half of a firmware update.
type UpdateSection string

const (
	// UpdateSectionHardware targets hardware (firmware) controller updates.
	UpdateSectionHardware UpdateSection = "hardware"
	// UpdateSectionSoftware targets software controller updates.
	UpdateSectionSoftware UpdateSection = "software"
)
