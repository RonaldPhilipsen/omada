package controller

import (
	"context"
	"net/url"
)

// SiteClient is the site-scoped API handle. Connection lifecycle belongs to the Controller.
type SiteClient struct {
	c    *conn
	site string
}

// SiteVersion returns the Omada version reported for the site.
func (s *SiteClient) SiteVersion(ctx context.Context) (Version, error) {
	return s.c.siteVersion(ctx, s.site)
}

// Clients lists the clients visible to the site.
func (s *SiteClient) Clients(ctx context.Context) ([]Client, error) {
	entries, err := s.c.pagesV2(ctx, "clients", s.site, url.Values{"limit": {"1000"}})
	if err != nil {
		return nil, err
	}
	clients := make([]Client, 0, len(entries))
	for _, entry := range entries {
		clients = append(clients, NewClient(entry))
	}
	return clients, nil
}

// Client returns the client details for the given MAC.
func (s *SiteClient) Client(ctx context.Context, mac string) (Client, error) {
	if mac == "" {
		return Client{}, &ValidationError{Msg: "client MAC must not be empty"}
	}
	m, err := s.c.requestEndpoint(ctx, "GET", "v2", "clients/"+mac, s.site, nil, nil)
	if err != nil {
		return Client{}, err
	}
	if len(m) == 0 {
		return Client{}, &ValidationError{Msg: "client not found"}
	}
	return NewClient(m), nil
}

// ClientByID returns the client at the given device index.
func (s *SiteClient) ClientByID(ctx context.Context, deviceIndex string) (Client, error) {
	if deviceIndex == "" {
		return Client{}, &ValidationError{Msg: "client device index must not be empty"}
	}
	m, err := s.c.requestEndpoint(ctx, "GET", "v2", "clients/"+deviceIndex, s.site, nil, nil)
	if err != nil {
		return Client{}, err
	}
	if len(m) == 0 {
		return Client{}, &ValidationError{Msg: "client not found"}
	}
	return NewClient(m), nil
}

// BlockClient blocks the client with the given MAC.
func (s *SiteClient) BlockClient(ctx context.Context, mac string) error {
	if mac == "" {
		return &ValidationError{Msg: "client MAC must not be empty"}
	}
	_, err := s.c.requestEndpoint(ctx, "POST", "v2", "cmd/clients/"+mac+"/block", s.site, nil, nil)
	return err
}

// UnblockClient unblocks the client with the given MAC.
func (s *SiteClient) UnblockClient(ctx context.Context, mac string) error {
	if mac == "" {
		return &ValidationError{Msg: "client MAC must not be empty"}
	}
	_, err := s.c.requestEndpoint(ctx, "POST", "v2", "cmd/clients/"+mac+"/unblock", s.site, nil, nil)
	return err
}

// ReconnectClient reconnects the client with the given MAC.
func (s *SiteClient) ReconnectClient(ctx context.Context, mac string) error {
	if mac == "" {
		return &ValidationError{Msg: "client MAC must not be empty"}
	}
	_, err := s.c.requestEndpoint(ctx, "POST", "v2", "cmd/clients/"+mac+"/reconnect", s.site, nil, nil)
	return err
}

// UpdateClientSettings updates the settings of a client. An empty name with
// nil IP setting and rate limit is a validation error.
func (s *SiteClient) UpdateClientSettings(ctx context.Context, mac string, name string, ipSetting *IpSetting, rateLimit *RateLimit) error {
	if mac == "" {
		return &ValidationError{Msg: "client MAC must not be empty"}
	}
	if name == "" && ipSetting == nil && rateLimit == nil {
		return &ValidationError{Msg: "client settings must not be empty"}
	}
	payload := map[string]any{}
	if name != "" {
		payload["clientName"] = name
	}
	if ipSetting != nil {
		useFixedAddr := ipSetting.UsedFixedAddr()
		ip := map[string]any{"useFixedAddr": useFixedAddr}
		if useFixedAddr {
			ip["netId"] = mapStr(ipSetting.data, "netId")
			ip["ip"] = mapStr(ipSetting.data, "ip")
		}
		payload["ipSetting"] = ip
	}
	if rateLimit != nil {
		payload["rateLimit"] = map[string]any{
			"enable":     rateLimit.Enabled(),
			"downEnable": rateLimit.DownEnabled(),
			"downLimit":  rateLimit.DownLimit(),
			"downUnit":   rateLimit.DownUnit(),
			"upEnable":   rateLimit.UpEnabled(),
			"upLimit":    rateLimit.UpLimit(),
			"upUnit":     rateLimit.UpUnit(),
		}
	}
	_, err := s.c.requestEndpoint(ctx, "PATCH", "v2", "clients/"+mac, s.site, nil, payload)
	return err
}

// ConnectedClients lists the clients currently connected to the site.
func (s *SiteClient) ConnectedClients(ctx context.Context) ([]Client, error) {
	version, err := s.SiteVersion(ctx)
	if err != nil {
		return nil, err
	}
	if version.Compare(ParseVersionMust("6.2.0.0")) >= 0 {
		body := map[string]any{
			"filters":               map[string]any{"active": true},
			"sorts":                 map[string]any{},
			"hideHealthUnsupported": true,
			"scope":                 1,
		}
		entries, err := s.c.pagesOpenAPI(ctx, "v2", "clients", s.site, body)
		if err != nil {
			return nil, err
		}
		clients := make([]Client, 0, len(entries))
		for _, entry := range entries {
			clients = append(clients, NewClient(entry))
		}
		return clients, nil
	}
	entries, err := s.c.pagesV2(ctx, "clients", s.site, url.Values{"filters.active": {"false"}})
	if err != nil {
		return nil, err
	}
	clients := make([]Client, 0, len(entries))
	for _, entry := range entries {
		c := NewClient(entry)
		if c.IsActive() {
			clients = append(clients, c)
		}
	}
	return clients, nil
}

// Networks lists the LAN networks of the site.
func (s *SiteClient) Networks(ctx context.Context) ([]LanNetwork, error) {
	body := map[string]any{"type": "lan"}
	entries, err := s.c.pagesOpenAPI(ctx, "v2", "lan-networks", s.site, body)
	if err != nil {
		return nil, err
	}
	networks := make([]LanNetwork, 0, len(entries))
	for _, entry := range entries {
		networks = append(networks, NewLanNetwork(entry))
	}
	return networks, nil
}

// VLANs lists the VLAN-only networks of the site.
func (s *SiteClient) VLANs(ctx context.Context) ([]LanNetwork, error) {
	networks, err := s.Networks(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]LanNetwork, 0, len(networks))
	for _, n := range networks {
		if n.Purpose() == "vlan" {
			out = append(out, n)
		}
	}
	return out, nil
}

// CreateVLAN creates a VLAN-only network with the given name and VLAN ID.
// It rejects names and IDs that conflict with the site's existing networks.
func (s *SiteClient) CreateVLAN(ctx context.Context, name string, vlanID int) (LanNetwork, error) {
	create, err := NewVlanCreate(name, vlanID)
	if err != nil {
		return LanNetwork{}, err
	}
	networks, err := s.Networks(ctx)
	if err != nil {
		return LanNetwork{}, err
	}
	if err := ValidateVlanConflicts(create.Payload(), networks, nil); err != nil {
		return LanNetwork{}, err
	}
	m, err := s.c.openapiRequest(ctx, "POST", "v2", "lan-networks", s.site, nil, create.Payload())
	if err != nil {
		return LanNetwork{}, err
	}
	return NewLanNetwork(m), nil
}

// UpdateVLAN changes the name and/or VLAN ID of a VLAN-only network while
// preserving its existing writable settings.
func (s *SiteClient) UpdateVLAN(ctx context.Context, network LanNetwork, update VlanUpdate) (LanNetwork, error) {
	if network.ID() == "" {
		return LanNetwork{}, &ValidationError{Msg: "network ID must not be empty"}
	}
	payload, err := update.Payload(network)
	if err != nil {
		return LanNetwork{}, err
	}
	m, err := s.c.openapiRequest(ctx, "PATCH", "v2", "lan-networks/"+network.ID(), s.site, nil, payload)
	if err != nil {
		return LanNetwork{}, err
	}
	return NewLanNetwork(m), nil
}

// KnownClients lists the clients known to the site.
func (s *SiteClient) KnownClients(ctx context.Context) ([]Client, error) {
	entries, err := s.c.pagesV2(ctx, "insight/clients", s.site, url.Values{})
	if err != nil {
		return nil, err
	}
	clients := make([]Client, 0, len(entries))
	for _, entry := range entries {
		clients = append(clients, NewClient(entry))
	}
	return clients, nil
}
