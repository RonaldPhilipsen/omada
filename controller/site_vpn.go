package controller

import (
	"context"
	"net/url"
)

// vpnCategoryOrder lists the VPN policy tabs in stable order.
var vpnCategoryOrder = []VpnCategory{VpnServer, VpnClient, VpnSiteToSite}

// VpnPolicies lists all VPN policies of the site across its tabs. Tabs
// whose list endpoint is unavailable are skipped.
func (s *SiteClient) VpnPolicies(ctx context.Context) ([]VpnPolicy, error) {
	var policies []VpnPolicy
	for _, category := range vpnCategoryOrder {
		entries, err := s.c.pagesOpenAPIGet(ctx, "v2", "vpn/"+category.ListEndpoint(), s.site, url.Values{})
		if err != nil {
			continue
		}
		for _, entry := range entries {
			policies = append(policies, NewVpnPolicy(category, entry))
		}
	}
	return policies, nil
}

// VpnPolicy returns the VPN policy with the given controller ID.
func (s *SiteClient) VpnPolicy(ctx context.Context, id string) (VpnPolicy, error) {
	if id == "" {
		return VpnPolicy{}, &ValidationError{Msg: "VPN policy ID must not be empty"}
	}
	policies, err := s.VpnPolicies(ctx)
	if err != nil {
		return VpnPolicy{}, err
	}
	for _, policy := range policies {
		if policy.PolicyID() == id {
			return policy, nil
		}
	}
	return VpnPolicy{}, &ValidationError{Msg: "VPN policy not found"}
}

// SetVpnPolicyEnabled enables or disables a VPN policy by controller ID and
// returns the updated policy.
func (s *SiteClient) SetVpnPolicyEnabled(ctx context.Context, id string, enabled bool) (VpnPolicy, error) {
	if id == "" {
		return VpnPolicy{}, &ValidationError{Msg: "VPN policy ID must not be empty"}
	}
	m, err := s.c.openapiRequest(ctx, "PATCH", "v1", "vpn/"+id+"/status", s.site, nil, map[string]any{"status": enabled})
	if err != nil {
		return VpnPolicy{}, err
	}
	policies, err := s.VpnPolicies(ctx)
	if err != nil {
		return VpnPolicy{}, err
	}
	for _, policy := range policies {
		if policy.PolicyID() == id {
			return policy, nil
		}
	}
	return VpnPolicy{data: m, category: VpnServer}, nil
}
