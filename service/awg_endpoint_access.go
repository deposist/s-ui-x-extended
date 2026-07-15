package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/database/model"
	"gorm.io/gorm"
)

const maxAWGEndpointDeviceLimit = 100

var ErrAWGEndpointAccessDenied = errors.New("AWG endpoint access denied")

type AWGEndpointAccessInfo struct {
	EndpointID         uint     `json:"endpointId"`
	Tag                string   `json:"tag"`
	PublicEndpoint     string   `json:"publicEndpoint"`
	Subnet             string   `json:"subnet"`
	DNS                []string `json:"dns"`
	DeviceLimit        int      `json:"deviceLimit"`
	DefaultDeviceLimit int      `json:"defaultDeviceLimit"`
}

func parseAWGEndpointMetadata(endpoint model.Endpoint) (model.AWGEndpointMetadata, error) {
	var metadata model.AWGEndpointMetadata
	if len(endpoint.Ext) == 0 || string(endpoint.Ext) == "null" {
		return metadata, nil
	}
	if err := json.Unmarshal(endpoint.Ext, &metadata); err != nil {
		return metadata, errors.New("invalid AWG endpoint metadata")
	}
	return metadata, nil
}

func validateAWGEndpointMetadata(endpoint model.Endpoint) (model.AWGEndpointMetadata, error) {
	metadata, err := parseAWGEndpointMetadata(endpoint)
	if err != nil || !metadata.Managed {
		return metadata, err
	}
	if endpoint.Type != "wireguard" {
		return metadata, errors.New("managed AWG endpoint must use WireGuard")
	}
	if metadata.DefaultDeviceLimit < 1 || metadata.DefaultDeviceLimit > maxAWGEndpointDeviceLimit {
		return metadata, fmt.Errorf("AWG default device limit must be between 1 and %d", maxAWGEndpointDeviceLimit)
	}
	host, port, err := net.SplitHostPort(strings.TrimSpace(metadata.PublicEndpoint))
	if err != nil || strings.TrimSpace(host) == "" {
		return metadata, errors.New("invalid AWG public endpoint")
	}
	parsedPort, err := strconv.ParseUint(port, 10, 16)
	if err != nil || parsedPort == 0 {
		return metadata, errors.New("invalid AWG public endpoint")
	}
	if len(metadata.DNS) == 0 {
		return metadata, errors.New("managed AWG endpoint requires DNS")
	}
	for _, value := range metadata.DNS {
		if _, err := netip.ParseAddr(strings.TrimSpace(value)); err != nil {
			return metadata, errors.New("invalid AWG endpoint DNS")
		}
	}
	return metadata, nil
}

func AWGSettingsForEndpoint(endpoint model.Endpoint) (AWGSettings, error) {
	metadata, err := validateAWGEndpointMetadata(endpoint)
	if err != nil {
		return AWGSettings{}, err
	}
	// Every caller (device manager, endpoint access, peer injection) expects a
	// managed endpoint here. Building settings for an unmanaged endpoint made
	// ReplaceClientAWGEndpointAccess silently accept assignments to plain
	// WireGuard endpoints that no device manager will ever reconcile.
	if !metadata.Managed {
		return AWGSettings{}, errors.New("endpoint is not a managed AWG endpoint")
	}
	var options awgManagedEndpointOptions
	if err := json.Unmarshal(endpoint.Options, &options); err != nil || len(options.Address) == 0 {
		return AWGSettings{}, errors.New("managed AWG endpoint options are invalid")
	}
	prefix, err := netip.ParsePrefix(options.Address[0])
	if err != nil || !prefix.Addr().Is4() {
		return AWGSettings{}, errors.New("managed AWG endpoint requires an IPv4 subnet")
	}
	dns := make([]netip.Addr, 0, len(metadata.DNS))
	for _, value := range metadata.DNS {
		address, _ := netip.ParseAddr(strings.TrimSpace(value))
		dns = append(dns, address)
	}
	return AWGSettings{
		Enabled: true, EndpointTag: endpoint.Tag, PublicEndpoint: metadata.PublicEndpoint,
		Subnet: prefix.Masked(), DNS: dns, DefaultDeviceLimit: metadata.DefaultDeviceLimit,
		ReconcileIntervalSec: 30, StatsIntervalSec: 60, MTU: options.MTU,
	}, nil
}

func LoadAWGEndpointByID(db *gorm.DB, endpointID uint) (model.Endpoint, AWGSettings, error) {
	var endpoint model.Endpoint
	if db == nil || endpointID == 0 {
		return endpoint, AWGSettings{}, ErrAWGEndpointAccessDenied
	}
	if err := db.First(&endpoint, endpointID).Error; err != nil {
		return endpoint, AWGSettings{}, err
	}
	settings, err := AWGSettingsForEndpoint(endpoint)
	return endpoint, settings, err
}

func EffectiveAWGEndpointAccess(db *gorm.DB, clientID, endpointID uint) (model.ClientEndpointAccess, AWGSettings, int, error) {
	var access model.ClientEndpointAccess
	if db == nil {
		return access, AWGSettings{}, 0, ErrAWGEndpointAccessDenied
	}
	if err := db.Where("client_id = ? AND endpoint_id = ?", clientID, endpointID).First(&access).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return access, AWGSettings{}, 0, ErrAWGEndpointAccessDenied
		}
		return access, AWGSettings{}, 0, err
	}
	_, settings, err := LoadAWGEndpointByID(db, endpointID)
	if err != nil {
		return access, AWGSettings{}, 0, err
	}
	limit := access.DeviceLimit
	if limit == 0 {
		limit = settings.DefaultDeviceLimit
	}
	return access, settings, limit, nil
}

func ListClientAWGEndpointAccess(db *gorm.DB, clientID uint) ([]AWGEndpointAccessInfo, error) {
	var accesses []model.ClientEndpointAccess
	if err := db.Where("client_id = ?", clientID).Order("endpoint_id").Find(&accesses).Error; err != nil {
		return nil, err
	}
	result := make([]AWGEndpointAccessInfo, 0, len(accesses))
	for _, access := range accesses {
		endpoint, settings, err := LoadAWGEndpointByID(db, access.EndpointId)
		if err != nil {
			continue
		}
		limit := access.DeviceLimit
		if limit == 0 {
			limit = settings.DefaultDeviceLimit
		}
		dns := make([]string, len(settings.DNS))
		for i := range settings.DNS {
			dns[i] = settings.DNS[i].String()
		}
		result = append(result, AWGEndpointAccessInfo{EndpointID: endpoint.Id, Tag: endpoint.Tag,
			PublicEndpoint: settings.PublicEndpoint, Subnet: settings.Subnet.String(), DNS: dns,
			DeviceLimit: limit, DefaultDeviceLimit: settings.DefaultDeviceLimit})
	}
	return result, nil
}

// InjectManagedAWGEndpointPeers injects device peers into every ext-managed
// endpoint's core config. Failures are per-endpoint and joined, so one broken
// endpoint cannot drop peers for the others.
func InjectManagedAWGEndpointPeers(db *gorm.DB, endpoints []json.RawMessage) ([]json.RawMessage, error) {
	endpointIDs, err := ListManagedAWGEndpoints(db)
	if err != nil {
		return endpoints, err
	}
	var joined error
	for _, endpointID := range endpointIDs {
		_, settings, loadErr := LoadAWGEndpointByID(db, endpointID)
		if loadErr != nil {
			joined = errors.Join(joined, loadErr)
			continue
		}
		endpoints, loadErr = injectAWGManagedEndpointPeersForEndpoint(db, settings, endpointID, endpoints)
		if loadErr != nil {
			joined = errors.Join(joined, loadErr)
		}
	}
	return endpoints, joined
}

func ReplaceClientAWGEndpointAccess(tx *gorm.DB, clientID uint, endpointIDs []uint) error {
	if tx == nil || clientID == 0 {
		return ErrAWGEndpointAccessDenied
	}
	wanted := make(map[uint]struct{}, len(endpointIDs))
	for _, endpointID := range endpointIDs {
		if endpointID == 0 {
			return errors.New("invalid AWG endpoint id")
		}
		if _, exists := wanted[endpointID]; exists {
			continue
		}
		if _, _, err := LoadAWGEndpointByID(tx, endpointID); err != nil {
			return err
		}
		wanted[endpointID] = struct{}{}
	}
	var current []model.ClientEndpointAccess
	if err := tx.Where("client_id = ?", clientID).Find(&current).Error; err != nil {
		return err
	}
	now := time.Now().Unix()
	for _, access := range current {
		if _, keep := wanted[access.EndpointId]; keep {
			delete(wanted, access.EndpointId)
			continue
		}
		if err := tx.Model(&model.AWGDevice{}).Where("client_id = ? AND endpoint_id = ? AND desired_enabled = ?", clientID, access.EndpointId, true).
			Updates(map[string]any{"desired_enabled": false, "sync_state": "pending_remove", "updated_at": now}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&access).Error; err != nil {
			return err
		}
	}
	for endpointID := range wanted {
		access := model.ClientEndpointAccess{ClientId: clientID, EndpointId: endpointID, Source: "manual", CreatedAt: now, UpdatedAt: now}
		if err := tx.Create(&access).Error; err != nil {
			return err
		}
	}
	return nil
}
