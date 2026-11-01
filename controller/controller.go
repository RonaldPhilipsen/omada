package controller

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/RonaldPhilipsen/omada/controller/conn"
	models "github.com/RonaldPhilipsen/omada/controller/models"
	"github.com/RonaldPhilipsen/omada/controller/site"
)

// SiteClient is the site-scoped API handle, defined in the site package.
type SiteClient = site.SiteClient

// UpdateSection is the firmware update section, defined in the models package.
type UpdateSection = models.UpdateSection

const (
	// UpdateSectionHardware targets hardware (firmware) controller updates.
	UpdateSectionHardware = models.UpdateSectionHardware
	// UpdateSectionSoftware targets software controller updates.
	UpdateSectionSoftware = models.UpdateSectionSoftware
)

// Controller is the controller-level API facade. Controller-scoped operations
// that do not belong to a site are served through it.
type Controller struct {
	c conn.Connection
}

// Option customizes a Controller.
type Option func(*options)

// options collects the resolved models.Client options.
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
	c, err := conn.New(controllerURL, username, password, hc)
	if err != nil {
		return nil, err
	}
	cl := &Controller{c: c}
	if _, err := cl.c.Login(context.Background()); err != nil {
		hc.CloseIdleConnections()
		return nil, err
	}
	return cl, nil
}

// OmadacID returns the controller's Omada ID (omadacId). It is available
// after New succeeds and feeds cloud.Options.OmadacID for the on-prem
// handoff to the cloud client.
func (cl *Controller) OmadacID() string { return cl.c.ControllerID() }

// Sites lists the sites the current user can administer.
func (cl *Controller) Sites(ctx context.Context) ([]models.Site, error) {
	m, err := cl.c.RequestEndpoint(ctx, "GET", "v2", "users/current", "", nil, nil)
	if err != nil {
		return nil, err
	}
	priv, ok := models.MapMap(m, "privilege")
	if !ok {
		return nil, fmt.Errorf("unexpected sites response: %v", m)
	}
	entries := models.MapList(priv, "sites")
	sites := make([]models.Site, 0, len(entries))
	for _, entry := range entries {
		sm, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		sites = append(sites, models.Site{Name: models.MapStr(sm, "name"), ID: models.MapStr(sm, "key")})
	}
	return sites, nil
}

// Site returns the site with the given controller site ID.
func (cl *Controller) Site(ctx context.Context, siteID string) (models.Site, error) {
	if siteID == "" {
		return models.Site{}, &models.ValidationError{Msg: "site ID must not be empty"}
	}
	m, err := cl.c.SiteInfo(ctx, siteID)
	if err != nil {
		return models.Site{}, err
	}
	return models.Site{Name: models.MapStr(m, "name"), ID: models.MapStr(m, "id")}, nil
}

// SiteByName resolves a site by display name.
func (cl *Controller) SiteByName(ctx context.Context, name string) (models.Site, error) {
	sites, err := cl.Sites(ctx)
	if err != nil {
		return models.Site{}, err
	}
	for _, s := range sites {
		if s.Name == name {
			return s, nil
		}
	}
	return models.Site{}, fmt.Errorf("%w: site %q", models.ErrSiteNotFound, name)
}

// models.ControllerInfo fetches unauthenticated controller diagnostic information.
func (cl *Controller) ControllerInfo(ctx context.Context) (models.ControllerInfo, error) {
	m, err := cl.c.ControllerInfo(ctx)
	if err != nil {
		return models.ControllerInfo{}, err
	}
	return models.NewControllerInfo(m), nil
}

// ControllerVersion returns the controller software version.
func (cl *Controller) ControllerVersion(ctx context.Context) (models.Version, error) {
	return cl.c.ControllerVersion(ctx)
}

// Reboot reboots the controller.
func (cl *Controller) Reboot(ctx context.Context) error {
	_, err := cl.c.RequestEndpoint(ctx, "POST", "v2", "cmd/reboot", "", nil, nil)
	return err
}

// Shutdown shuts the controller down.
func (cl *Controller) Shutdown(ctx context.Context) error {
	_, err := cl.c.RequestEndpoint(ctx, "POST", "v2", "cmd/shutdown", "", nil, nil)
	return err
}

// InstallControllerFirmware installs controller firmware to the given target version.
func (cl *Controller) InstallControllerFirmware(ctx context.Context, targetVersion string) error {
	if targetVersion == "" {
		return &models.ValidationError{Msg: "target version must not be empty"}
	}
	body := map[string]any{"targetVersion": targetVersion}
	_, err := cl.c.RequestEndpoint(ctx, "POST", "v2", "cmd/upgradeFirmware", "", nil, body)
	return err
}

// GetControllerUpgradeStatus returns the status of an ongoing hardware controller firmware upgrade.
func (cl *Controller) GetControllerUpgradeStatus(ctx context.Context) (*models.HardwareUpgradeStatus, error) {
	m, err := cl.c.RequestEndpoint(ctx, "GET", "v2", "maintenance/hardware/upgradeStatus", "", nil, nil)
	if err != nil {
		return nil, err
	}
	status := models.NewHardwareUpgradeStatus(m)
	return &status, nil
}

// SiteClient validates the site and returns a site-scoped client handle.
func (cl *Controller) SiteClient(ctx context.Context, siteID string) (*site.SiteClient, error) {
	if _, err := cl.Site(ctx, siteID); err != nil {
		return nil, err
	}
	return site.NewSiteClient(cl.c, siteID), nil
}

// FirmwareUpdates checks for controller firmware updates.
func (cl *Controller) FirmwareUpdates(ctx context.Context) (models.ControllerUpdateInfo, error) {
	version, err := cl.ControllerVersion(ctx)
	if err != nil {
		return models.ControllerUpdateInfo{}, err
	}
	endpoint := "controller/notification/updateInfo"
	if version.Compare(models.ParseVersionMust("6.3")) >= 0 {
		endpoint = "maintenance/software/channelUpdate"
	}
	m, err := cl.c.RequestEndpoint(ctx, "GET", "v2", endpoint, "", nil, nil)
	if err != nil {
		return models.ControllerUpdateInfo{}, err
	}
	return models.NewControllerUpdateInfo(m), nil
}

// UpdateFirmwareChannel selects the firmware release channel to check.
func (cl *Controller) UpdateFirmwareChannel(ctx context.Context, section UpdateSection, channel string) error {
	body := map[string]any{
		"section": string(section),
		"channel": channel,
	}
	_, err := cl.c.RequestEndpoint(ctx, "POST", "v2", "upgrade", "", nil, body)
	return err
}

// SetCertificate uploads a new certificate and enables it.
func (cl *Controller) SetCertificate(ctx context.Context, keyPEM, certPEM string) error {
	m, err := cl.c.DoMultipart(ctx, "POST", cl.c.BuildURL("v2", "cmd/uploadCert", ""), nil, map[string]string{
		"key":  keyPEM,
		"cert": certPEM,
	}, cl.c.CSRFToken())
	if err != nil {
		return err
	}
	payload := map[string]any{
		"certificate": map[string]any{
			"cerId":       models.MapStr(m, "cerId"),
			"cerName":     models.MapStr(m, "cerName"),
			"cerType":     "PFX",
			"enable":      true,
			"keyPassword": "",
		},
	}
	_, err = cl.c.RequestEndpoint(ctx, "PATCH", "v2", "controller/setting", "", nil, payload)
	return err
}

// Close releases pooled idle connections.
func (cl *Controller) Close() error {
	cl.c.CloseIdleConnections()
	return nil
}
