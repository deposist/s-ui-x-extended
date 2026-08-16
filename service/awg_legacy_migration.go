package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/deposist/s-ui-x-extended/database/model"

	"gorm.io/gorm"
)

// MigrateLegacyAWGSettings converts the removed settings-based managed-AWG
// mode into endpoint Ext metadata so the endpoint-scoped device manager owns
// the endpoint: it stamps managed metadata onto the tagged endpoint, re-points
// legacy (endpoint_id = 0) devices at it, grants their clients access rows,
// and switches the legacy settings off. It runs from app start, is
// idempotent, and never touches an endpoint that is already ext-managed.
func MigrateLegacyAWGSettings(db *gorm.DB) (bool, error) {
	if db == nil {
		return false, errors.New("AWG migration database is unavailable")
	}
	if strings.TrimSpace(readLegacyAWGSetting(db, "awgEnabled")) != "true" {
		return false, nil
	}
	tag := strings.TrimSpace(readLegacyAWGSetting(db, "awgEndpointTag"))
	if tag == "" {
		return false, nil
	}
	publicEndpoint := strings.TrimSpace(readLegacyAWGSetting(db, "awgPublicEndpoint"))
	limit, limitErr := strconv.Atoi(strings.TrimSpace(readLegacyAWGSetting(db, "awgDefaultDeviceLimit")))
	if limitErr != nil || limit < 1 || limit > maxAWGEndpointDeviceLimit {
		limit = 3
	}
	dns := make([]string, 0, 4)
	for _, item := range strings.Split(readLegacyAWGSetting(db, "awgDNS"), ",") {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, err := netip.ParseAddr(trimmed); err != nil {
			return false, fmt.Errorf("legacy awgDNS entry %q is invalid", trimmed)
		}
		dns = append(dns, trimmed)
	}

	migrated := false
	err := db.Transaction(func(tx *gorm.DB) error {
		var endpoint model.Endpoint
		if err := tx.Where("tag = ?", tag).First(&endpoint).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("legacy managed AWG endpoint %q not found; create or rename a WireGuard endpoint with this tag and restart the panel", tag)
			}
			return err
		}
		metadata, err := parseAWGEndpointMetadata(endpoint)
		if err != nil {
			return err
		}
		if !metadata.Managed {
			merged := map[string]json.RawMessage{}
			if len(endpoint.Ext) > 0 && string(endpoint.Ext) != "null" {
				if err := json.Unmarshal(endpoint.Ext, &merged); err != nil {
					return errors.New("legacy AWG endpoint has invalid Ext metadata")
				}
			}
			// Ext is the flat AWGEndpointMetadata document; merge field by
			// field so unknown future keys survive the conversion.
			fields := map[string]any{
				"managed":            true,
				"publicEndpoint":     publicEndpoint,
				"dns":                dns,
				"defaultDeviceLimit": limit,
			}
			for name, value := range fields {
				encoded, err := json.Marshal(value)
				if err != nil {
					return err
				}
				merged[name] = encoded
			}
			encoded, err := json.Marshal(merged)
			if err != nil {
				return err
			}
			candidate := endpoint
			candidate.Ext = encoded
			converted, err := AWGSettingsForEndpoint(candidate)
			if err != nil {
				return fmt.Errorf("legacy managed AWG endpoint %q cannot become endpoint-managed: %w", tag, err)
			}
			if err := checkLegacyDevicesFitSubnet(tx, converted.Subnet); err != nil {
				return err
			}
			if err := tx.Model(&model.Endpoint{}).Where("id = ?", endpoint.Id).Update("ext", encoded).Error; err != nil {
				return err
			}
			migrated = true
		}
		// The legacy mode managed exactly one endpoint, so every legacy
		// device (endpoint_id = 0) belongs to it.
		if err := tx.Model(&model.AWGDevice{}).Where("endpoint_id = ?", 0).Update("endpoint_id", endpoint.Id).Error; err != nil {
			return err
		}
		now := time.Now().Unix()
		var clientIDs []uint
		if err := tx.Model(&model.AWGDevice{}).Where("endpoint_id = ?", endpoint.Id).Distinct().Pluck("client_id", &clientIDs).Error; err != nil {
			return err
		}
		for _, clientID := range clientIDs {
			access := model.ClientEndpointAccess{ClientId: clientID, EndpointId: endpoint.Id, Source: "migration", CreatedAt: now, UpdatedAt: now}
			if err := tx.Where(map[string]any{"client_id": clientID, "endpoint_id": endpoint.Id}).FirstOrCreate(&access).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&model.Setting{}).Where("key = ?", "awgEnabled").Update("value", "false").Error; err != nil {
			return err
		}
		return nil
	})
	return migrated, err
}

// checkLegacyDevicesFitSubnet guards the subtle legacy conversion case: legacy
// addresses were allocated from the awgSubnet setting, while the endpoint
// manager derives the subnet from the endpoint address prefix. A device
// outside that prefix could never be re-added to the live interface.
func checkLegacyDevicesFitSubnet(tx *gorm.DB, prefix netip.Prefix) error {
	var devices []model.AWGDevice
	if err := tx.Where("endpoint_id = ?", 0).Find(&devices).Error; err != nil {
		return err
	}
	for _, device := range devices {
		address, err := netip.ParseAddr(device.IPv4Address)
		if err != nil || !prefix.Contains(address) {
			return fmt.Errorf("legacy AWG device %q has address %s outside the endpoint subnet %s; widen the endpoint address prefix and restart the panel", device.Name, device.IPv4Address, prefix)
		}
	}
	return nil
}

func readLegacyAWGSetting(db *gorm.DB, key string) string {
	var row model.Setting
	if err := db.Where("key = ?", key).First(&row).Error; err != nil {
		return defaultValueMap[key]
	}
	return row.Value
}
