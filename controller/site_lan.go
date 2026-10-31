package controller

import (
	"context"
	"fmt"
	models "github.com/RonaldPhilipsen/omada/controller/models"
	"net/url"
	"strconv"
)

// LanNetworks lists the LAN networks of the site.
func (s *SiteClient) LanNetworks(ctx context.Context) ([]models.LanNetwork, error) {
	entries, err := s.c.PagesV2(ctx, "lanNetworks", s.site, url.Values{"type": {"lan"}})
	if err != nil {
		return nil, err
	}
	networks := make([]models.LanNetwork, 0, len(entries))
	for _, entry := range entries {
		networks = append(networks, models.NewLanNetwork(entry))
	}
	return networks, nil
}

// LanNetwork returns the LAN network with the given ID.
func (s *SiteClient) LanNetwork(ctx context.Context, id string) (models.LanNetwork, error) {
	if id == "" {
		return models.LanNetwork{}, &models.ValidationError{Msg: "network ID must not be empty"}
	}
	m, err := s.c.RequestEndpoint(ctx, "GET", "v2", "lanNetworks/"+id, s.site, nil, nil)
	if err != nil {
		return models.LanNetwork{}, err
	}
	if len(m) == 0 {
		return models.LanNetwork{}, &models.ValidationError{Msg: "network not found"}
	}
	return models.NewLanNetwork(m), nil
}

// LanNetworkCreate is the create request for a LAN network.
type LanNetworkCreate struct {
	Name        string
	VLAN        int
	Subnet      string
	Mask        string
	Gateway     string
	DHCPEnabled *bool
	StartIP     string
	EndIP       string
	DNSServers  []string
}

// NewLanNetworkCreate builds a validated LAN network create request.
func NewLanNetworkCreate(name string, vlan int) (LanNetworkCreate, error) {
	if name == "" {
		return LanNetworkCreate{}, &models.ValidationError{Msg: "LAN network name must not be empty"}
	}
	if vlan < 1 || vlan > 4090 {
		return LanNetworkCreate{}, &models.ValidationError{Msg: "VLAN ID must be an integer from 1 to 4090"}
	}
	if vlan == 1 {
		return LanNetworkCreate{}, &models.ValidationError{Msg: "VLAN 1 is protected"}
	}
	return LanNetworkCreate{Name: name, VLAN: vlan}, nil
}

// CreateLanNetwork creates a LAN network.
func (s *SiteClient) CreateLanNetwork(ctx context.Context, create LanNetworkCreate) (models.LanNetwork, error) {
	if _, err := NewLanNetworkCreate(create.Name, create.VLAN); err != nil {
		return models.LanNetwork{}, err
	}
	body := map[string]any{
		"name": create.Name,
		"vlan": create.VLAN,
	}
	m, err := s.c.RequestEndpoint(ctx, "POST", "v2", "lanNetworks", s.site, nil, body)
	if err != nil {
		return models.LanNetwork{}, err
	}
	if len(m) == 0 {
		return models.LanNetwork{}, &models.ValidationError{Msg: "network not found"}
	}
	return models.NewLanNetwork(m), nil
}

// LanNetworkUpdate changes a LAN network partially; only the set fields apply.
type LanNetworkUpdate struct {
	NetworkID   string
	Name        *string
	VLAN        *int
	Subnet      *string
	Mask        *string
	Gateway     *string
	DHCPEnabled *bool
	StartIP     *string
	EndIP       *string
	DNSServers  []string
}

// ID returns the updated LAN network identity.
func (u LanNetworkUpdate) ID() string { return u.NetworkID }

// UpdateLanNetwork updates a LAN network partially.
func (s *SiteClient) UpdateLanNetwork(ctx context.Context, update LanNetworkUpdate) (models.LanNetwork, error) {
	if update.NetworkID == "" {
		return models.LanNetwork{}, &models.ValidationError{Msg: "network ID must not be empty"}
	}
	if update.Name == nil && update.VLAN == nil && update.Subnet == nil && update.Mask == nil &&
		update.Gateway == nil && update.DHCPEnabled == nil && update.StartIP == nil && update.EndIP == nil &&
		update.DNSServers == nil {
		return models.LanNetwork{}, &models.ValidationError{Msg: "at least one LAN network change is required"}
	}
	body := map[string]any{}
	if update.Name != nil {
		body["name"] = *update.Name
	}
	if update.VLAN != nil {
		body["vlan"] = *update.VLAN
	}
	if update.Subnet != nil {
		body["subnet"] = *update.Subnet
	}
	if update.Mask != nil {
		body["mask"] = *update.Mask
	}
	if update.Gateway != nil {
		body["gateway"] = *update.Gateway
	}
	if update.DHCPEnabled != nil {
		body["dhcpEnabled"] = *update.DHCPEnabled
	}
	if update.StartIP != nil {
		body["startIP"] = *update.StartIP
	}
	if update.EndIP != nil {
		body["endIP"] = *update.EndIP
	}
	if update.DNSServers != nil {
		body["dnsServers"] = update.DNSServers
	}
	m, err := s.c.RequestEndpoint(ctx, "PUT", "v2", "lanNetworks/"+update.NetworkID, s.site, nil, body)
	if err != nil {
		return models.LanNetwork{}, err
	}
	if len(m) == 0 {
		return models.LanNetwork{}, &models.ValidationError{Msg: "network not found"}
	}
	return models.NewLanNetwork(m), nil
}

// VlanDelete identifies a VLAN-only network to delete.
type VlanDelete struct {
	VLAN int
}

// PreviewVlanCreate previews a VLAN-only network creation.
func (s *SiteClient) PreviewVlanCreate(ctx context.Context, create models.VlanCreate) (map[string]any, error) {
	m, err := s.c.OpenapiRequest(ctx, "POST", "v2", "vlan/preview", s.site, nil, create.Payload())
	if err != nil {
		return nil, err
	}
	return m, nil
}

// VlanCreate creates a VLAN-only network; it does not configure routing or DHCP.
func (s *SiteClient) VlanCreate(ctx context.Context, create models.VlanCreate) (map[string]any, error) {
	m, err := s.c.OpenapiRequest(ctx, "POST", "v2", "vlans", s.site, nil, create.Payload())
	if err != nil {
		return nil, err
	}
	if err := s.verifyVlan(ctx, create.VLANID()); err != nil {
		return nil, err
	}
	return m, nil
}

// PreviewVlanUpdate previews a VLAN-only network update.
func (s *SiteClient) PreviewVlanUpdate(ctx context.Context, update models.VlanUpdate) (map[string]any, error) {
	body := map[string]any{}
	if update.Name() != nil {
		body["name"] = *update.Name()
	}
	if update.VLANID() != nil {
		body["vlan"] = *update.VLANID()
	}
	m, err := s.c.OpenapiRequest(ctx, "POST", "v2", "vlan/preview", s.site, nil, body)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// VlanUpdate changes a VLAN-only network; it preserves writable settings.
func (s *SiteClient) VlanUpdate(ctx context.Context, update models.VlanUpdate) (map[string]any, error) {
	body := map[string]any{}
	if update.Name() != nil {
		body["name"] = *update.Name()
	}
	if update.VLANID() != nil {
		body["vlan"] = *update.VLANID()
	}
	m, err := s.c.OpenapiRequest(ctx, "POST", "v2", "vlans", s.site, nil, body)
	if err != nil {
		return nil, err
	}
	vlan := 0
	if update.VLANID() != nil {
		vlan = *update.VLANID()
	}
	if err := s.verifyVlan(ctx, vlan); err != nil {
		return nil, err
	}
	return m, nil
}

// PreviewVlanDelete previews a VLAN-only network deletion.
func (s *SiteClient) PreviewVlanDelete(ctx context.Context, del VlanDelete) (map[string]any, error) {
	body := map[string]any{"vlan": del.VLAN}
	m, err := s.c.OpenapiRequest(ctx, "POST", "v2", "vlan/preview", s.site, nil, body)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// VlanDelete deletes a VLAN-only network.
func (s *SiteClient) VlanDelete(ctx context.Context, del VlanDelete) (map[string]any, error) {
	m, err := s.c.OpenapiRequest(ctx, "DELETE", "v2", "vlans/"+strconv.Itoa(del.VLAN), s.site, nil, nil)
	if err != nil {
		return nil, err
	}
	if err := s.verifyVlan(ctx, del.VLAN); err != nil {
		return nil, err
	}
	return m, nil
}

// verifyVlan reads back a VLAN write outcome after a successful mutation.
func (s *SiteClient) verifyVlan(ctx context.Context, vlan int) error {
	m, err := s.c.OpenapiRequest(ctx, "GET", "v2", "vlans", s.site, url.Values{"vlan": {strconv.Itoa(vlan)}}, nil)
	if err != nil {
		return fmt.Errorf("write may have succeeded; inspect controller state before retrying: %w", err)
	}
	if len(m) == 0 {
		return fmt.Errorf("write may have succeeded; inspect controller state before retrying: vlan %d missing", vlan)
	}
	return nil
}
