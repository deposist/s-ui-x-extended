package service

import (
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// AWGSettings contains the validated settings used by the managed endpoint.
type AWGSettings struct {
	Enabled              bool
	EndpointTag          string
	PublicEndpoint       string
	Subnet               netip.Prefix
	DNS                  []netip.Addr
	DefaultDeviceLimit   int
	ReconcileIntervalSec int
	StatsIntervalSec     int
	MTU                  uint32
}

func (s *SettingService) GetAWGSettings() (AWGSettings, error) {
	snapshot, err := s.getSettingsSnapshot(
		"awgEnabled", "awgEndpointTag", "awgPublicEndpoint", "awgSubnet", "awgDNS",
		"awgDefaultDeviceLimit", "awgReconcileIntervalSec", "awgStatsIntervalSec", "awgMTU",
	)
	if err != nil {
		return AWGSettings{}, err
	}
	return parseAWGSettings(snapshot)
}

func parseAWGSettings(values map[string]string) (AWGSettings, error) {
	var result AWGSettings
	var err error
	if result.Enabled, err = strconv.ParseBool(values["awgEnabled"]); err != nil {
		return result, fmt.Errorf("invalid awgEnabled")
	}
	result.EndpointTag = strings.TrimSpace(values["awgEndpointTag"])
	if result.Subnet, err = netip.ParsePrefix(strings.TrimSpace(values["awgSubnet"])); err != nil || !result.Subnet.Addr().Is4() {
		return result, fmt.Errorf("invalid awgSubnet")
	}
	result.Subnet = result.Subnet.Masked()
	result.PublicEndpoint = strings.TrimSpace(values["awgPublicEndpoint"])
	if result.PublicEndpoint != "" {
		host, port, splitErr := net.SplitHostPort(result.PublicEndpoint)
		if splitErr != nil || strings.TrimSpace(host) == "" {
			return result, fmt.Errorf("invalid awgPublicEndpoint")
		}
		parsedPort, portErr := strconv.ParseUint(port, 10, 16)
		if portErr != nil || parsedPort == 0 {
			return result, fmt.Errorf("invalid awgPublicEndpoint")
		}
	}
	for _, item := range strings.Split(values["awgDNS"], ",") {
		if strings.TrimSpace(item) == "" {
			continue
		}
		address, parseErr := netip.ParseAddr(strings.TrimSpace(item))
		if parseErr != nil {
			return result, fmt.Errorf("invalid awgDNS")
		}
		result.DNS = append(result.DNS, address)
	}
	if len(result.DNS) == 0 {
		return result, fmt.Errorf("invalid awgDNS")
	}
	if result.DefaultDeviceLimit, err = parseBoundedAWGInt(values, "awgDefaultDeviceLimit", 0, 100); err != nil {
		return result, err
	}
	if result.ReconcileIntervalSec, err = parseBoundedAWGInt(values, "awgReconcileIntervalSec", 5, 3600); err != nil {
		return result, err
	}
	if result.StatsIntervalSec, err = parseBoundedAWGInt(values, "awgStatsIntervalSec", 10, 3600); err != nil {
		return result, err
	}
	mtu, err := parseBoundedAWGInt(values, "awgMTU", 0, 65535)
	if err != nil {
		return result, err
	}
	result.MTU = uint32(mtu) // #nosec G115 -- parseBoundedAWGInt limits mtu to uint16 range.
	if result.Enabled && (result.EndpointTag == "" || result.PublicEndpoint == "") {
		return result, fmt.Errorf("enabled AWG requires endpoint tag and public endpoint")
	}
	return result, nil
}

func parseBoundedAWGInt(values map[string]string, key string, minimum, maximum int) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(values[key]))
	if err != nil || value < minimum || value > maximum {
		return 0, fmt.Errorf("invalid %s", key)
	}
	return value, nil
}

func validateAWGSettingInput(key, value string) error {
	values := map[string]string{
		"awgEnabled":              defaultValueMap["awgEnabled"],
		"awgEndpointTag":          defaultValueMap["awgEndpointTag"],
		"awgPublicEndpoint":       defaultValueMap["awgPublicEndpoint"],
		"awgSubnet":               defaultValueMap["awgSubnet"],
		"awgDNS":                  defaultValueMap["awgDNS"],
		"awgDefaultDeviceLimit":   defaultValueMap["awgDefaultDeviceLimit"],
		"awgReconcileIntervalSec": defaultValueMap["awgReconcileIntervalSec"],
		"awgStatsIntervalSec":     defaultValueMap["awgStatsIntervalSec"],
		"awgMTU":                  defaultValueMap["awgMTU"],
	}
	if _, ok := values[key]; !ok {
		return nil
	}
	values[key] = value
	_, err := parseAWGSettings(values)
	return err
}

// ValidateAWGManagedEndpoint checks that the configured endpoint is suitable
// for exclusive server-side peer management. Errors never include endpoint
// options or key material.
func ValidateAWGManagedEndpoint(db *gorm.DB, settings AWGSettings) error {
	if !settings.Enabled {
		return nil
	}
	_, options, err := loadAWGManagedEndpoint(db, settings)
	if err != nil {
		return err
	}
	if options.ListenPort == 0 {
		return fmt.Errorf("managed AWG endpoint requires listen_port")
	}
	if options.Amnezia == nil || options.Amnezia.JC <= 0 || options.Amnezia.JMin <= 0 || options.Amnezia.JMax <= 0 ||
		options.Amnezia.S1 <= 0 || options.Amnezia.S2 <= 0 || options.Amnezia.S3 <= 0 || options.Amnezia.S4 <= 0 ||
		options.Amnezia.H1 == nil || options.Amnezia.H2 == nil || options.Amnezia.H3 == nil || options.Amnezia.H4 == nil {
		return fmt.Errorf("managed AWG endpoint requires an AWG 2.0 profile")
	}
	if options.Amnezia.J1 != "" || options.Amnezia.J2 != "" || options.Amnezia.J3 != "" || options.Amnezia.ITime != 0 {
		return fmt.Errorf("managed AWG endpoint uses parameters unsupported by the current backend")
	}
	for _, peer := range options.Peers {
		if peer.Address != "" || peer.Port != 0 {
			return fmt.Errorf("managed AWG endpoint must not be configured as a client endpoint")
		}
	}
	return nil
}
