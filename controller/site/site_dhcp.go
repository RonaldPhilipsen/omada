package site

import (
	"context"
	models "github.com/RonaldPhilipsen/omada/controller/models"
	"net/url"
	"strings"
)

// DhcpReservations lists the DHCP reservations of the site.
func (s *SiteClient) DhcpReservations(ctx context.Context) ([]models.DhcpReservation, error) {
	version, err := s.SiteVersion(ctx)
	if err != nil {
		return nil, err
	}
	var (
		entries []map[string]any
		listErr error
	)
	if version.Compare(models.ParseVersionMust("6.2.0.0")) >= 0 {
		entries, listErr = s.c.PagesOpenAPIGet(ctx, "v1", "setting/service/dhcp", s.site, url.Values{})
	} else {
		entries, listErr = s.c.PagesV2(ctx, "setting/service/dhcp", s.site, url.Values{})
	}
	if listErr != nil {
		return nil, listErr
	}
	reservations := make([]models.DhcpReservation, 0, len(entries))
	for _, entry := range entries {
		reservations = append(reservations, models.NewDhcpReservation(entry))
	}
	return reservations, nil
}

// models.DhcpReservation returns the DHCP reservation for the given IP, if any.
func (s *SiteClient) DhcpReservation(ctx context.Context, ip string) (models.DhcpReservation, error) {
	if ip == "" {
		return models.DhcpReservation{}, &models.ValidationError{Msg: "DHCP reservation IP must not be empty"}
	}
	reservations, err := s.DhcpReservations(ctx)
	if err != nil {
		return models.DhcpReservation{}, err
	}
	for _, reservation := range reservations {
		if reservation.IP() == ip {
			return reservation, nil
		}
	}
	return models.DhcpReservation{}, &models.ValidationError{Msg: "DHCP reservation not found"}
}

// CreateDhcpReservation creates a DHCP reservation for the host, IP, and MAC.
// The host, IP, and MAC must not be empty.
func (s *SiteClient) CreateDhcpReservation(ctx context.Context, host, ip, mac string) (models.DhcpReservation, error) {
	if host == "" || ip == "" || mac == "" {
		return models.DhcpReservation{}, &models.ValidationError{Msg: "DHCP reservation host, IP, and MAC must not be empty"}
	}
	body := map[string]any{
		"hostName": host,
		"ip":       ip,
		"mac":      mac,
	}
	m, err := s.c.OpenapiRequest(ctx, "POST", "v1", "setting/service/dhcp", s.site, nil, body)
	if err != nil {
		return models.DhcpReservation{}, err
	}
	return models.NewDhcpReservation(m), nil
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
func (s *SiteClient) UpdateDhcpReservation(ctx context.Context, update DhcpReservationUpdate) (models.DhcpReservation, error) {
	if update.IP == "" && update.Host == nil && update.MAC == nil {
		return models.DhcpReservation{}, &models.ValidationError{Msg: "DHCP reservation update requires at least one field"}
	}
	var mac string
	if update.MAC != nil {
		mac = strings.ToLower(*update.MAC)
	} else {
		existing, err := s.DhcpReservation(ctx, update.IP)
		if err != nil {
			return models.DhcpReservation{}, err
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
	m, err := s.c.OpenapiRequest(ctx, "PATCH", "v1", "setting/service/dhcp/"+mac, s.site, nil, body)
	if err != nil {
		return models.DhcpReservation{}, err
	}
	return models.NewDhcpReservation(m), nil
}

// DeleteDhcpReservation deletes the DHCP reservation for the given IP.
func (s *SiteClient) DeleteDhcpReservation(ctx context.Context, ip string) error {
	if ip == "" {
		return &models.ValidationError{Msg: "DHCP reservation IP must not be empty"}
	}
	existing, err := s.DhcpReservation(ctx, ip)
	if err != nil {
		return err
	}
	_, err = s.c.OpenapiRequest(ctx, "DELETE", "v1", "setting/service/dhcp/"+strings.ToLower(existing.MAC()), s.site, nil, nil)
	return err
}
