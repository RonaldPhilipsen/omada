package controller

import (
	"context"
	"net/url"
	"strings"
)

// DhcpReservations lists the DHCP reservations of the site.
func (s *SiteClient) DhcpReservations(ctx context.Context) ([]DhcpReservation, error) {
	version, err := s.SiteVersion(ctx)
	if err != nil {
		return nil, err
	}
	var (
		entries []map[string]any
		listErr error
	)
	if version.Compare(ParseVersionMust("6.2.0.0")) >= 0 {
		entries, listErr = s.c.pagesOpenAPIGet(ctx, "v1", "setting/service/dhcp", s.site, url.Values{})
	} else {
		entries, listErr = s.c.pagesV2(ctx, "setting/service/dhcp", s.site, url.Values{})
	}
	if listErr != nil {
		return nil, listErr
	}
	reservations := make([]DhcpReservation, 0, len(entries))
	for _, entry := range entries {
		reservations = append(reservations, NewDhcpReservation(entry))
	}
	return reservations, nil
}

// DhcpReservation returns the DHCP reservation for the given IP, if any.
func (s *SiteClient) DhcpReservation(ctx context.Context, ip string) (DhcpReservation, error) {
	if ip == "" {
		return DhcpReservation{}, &ValidationError{Msg: "DHCP reservation IP must not be empty"}
	}
	reservations, err := s.DhcpReservations(ctx)
	if err != nil {
		return DhcpReservation{}, err
	}
	for _, reservation := range reservations {
		if reservation.IP() == ip {
			return reservation, nil
		}
	}
	return DhcpReservation{}, &ValidationError{Msg: "DHCP reservation not found"}
}

// CreateDhcpReservation creates a DHCP reservation for the host, IP, and MAC.
// The host, IP, and MAC must not be empty.
func (s *SiteClient) CreateDhcpReservation(ctx context.Context, host, ip, mac string) (DhcpReservation, error) {
	if host == "" || ip == "" || mac == "" {
		return DhcpReservation{}, &ValidationError{Msg: "DHCP reservation host, IP, and MAC must not be empty"}
	}
	body := map[string]any{
		"hostName": host,
		"ip":       ip,
		"mac":      mac,
	}
	m, err := s.c.openapiRequest(ctx, "POST", "v1", "setting/service/dhcp", s.site, nil, body)
	if err != nil {
		return DhcpReservation{}, err
	}
	return NewDhcpReservation(m), nil
}

// DhcpReservationUpdate describes the changes to apply to a DHCP
// reservation. At least one field must be set.
type DhcpReservationUpdate struct {
	IP   string
	Host *string
	MAC  *string
}

// UpdateDhcpReservation updates a DHCP reservation partially; only the set
// fields apply. An empty update is a validation error.
func (s *SiteClient) UpdateDhcpReservation(ctx context.Context, update DhcpReservationUpdate) (DhcpReservation, error) {
	if update.IP == "" && update.Host == nil && update.MAC == nil {
		return DhcpReservation{}, &ValidationError{Msg: "DHCP reservation update requires at least one field"}
	}
	var mac string
	if update.MAC != nil {
		mac = strings.ToLower(*update.MAC)
	} else {
		existing, err := s.DhcpReservation(ctx, update.IP)
		if err != nil {
			return DhcpReservation{}, err
		}
		mac = existing.MAC()
	}
	body := map[string]any{}
	if update.IP != "" {
		body["ip"] = update.IP
	}
	if update.Host != nil {
		body["description"] = *update.Host
	}
	m, err := s.c.openapiRequest(ctx, "PATCH", "v1", "setting/service/dhcp/"+mac, s.site, nil, body)
	if err != nil {
		return DhcpReservation{}, err
	}
	return NewDhcpReservation(m), nil
}

// DeleteDhcpReservation deletes the DHCP reservation for the given IP.
func (s *SiteClient) DeleteDhcpReservation(ctx context.Context, ip string) error {
	if ip == "" {
		return &ValidationError{Msg: "DHCP reservation IP must not be empty"}
	}
	existing, err := s.DhcpReservation(ctx, ip)
	if err != nil {
		return err
	}
	_, err = s.c.openapiRequest(ctx, "DELETE", "v1", "setting/service/dhcp/"+strings.ToLower(existing.MAC()), s.site, nil, nil)
	return err
}
