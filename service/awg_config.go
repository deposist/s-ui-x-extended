package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"

	"github.com/deposist/s-ui-x-extended/database/model"

	qrcode "github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

var (
	ErrAWGConfigUnavailable = errors.New("AWG configuration is unavailable")
	ErrAWGConfigTooLargeQR  = errors.New("AWG configuration is too large for QR")
)

// RenderOwnedConfig decrypts and renders the complete current AWG 2.0 config
// only for a provisioned device belonging to clientID.
func (m *AWGManager) RenderOwnedConfig(ctx context.Context, deviceID, clientID uint) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if m == nil || m.deps.DB == nil {
		return nil, ErrAWGConfigUnavailable
	}
	result := m.submit(ctx, func(context.Context) awgCommandResult {
		config, err := m.renderOwnedConfigInWorker(deviceID, clientID)
		return awgCommandResult{value: config, err: err}
	})
	if result.err != nil {
		return nil, result.err
	}
	config, ok := result.value.([]byte)
	if !ok {
		return nil, ErrAWGConfigUnavailable
	}
	return config, nil
}

func (m *AWGManager) renderOwnedConfigInWorker(deviceID, clientID uint) ([]byte, error) {
	var device model.AWGDevice
	if err := m.deps.DB.Where("id = ? AND client_id = ? AND endpoint_id = ?", deviceID, clientID, m.deps.EndpointID).First(&device).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAWGDeviceNotFound
		}
		return nil, err
	}
	if !device.DesiredEnabled || !device.Provisioned || device.SyncState != "in_sync" {
		return nil, ErrAWGConfigUnavailable
	}
	settings, err := m.deps.LoadSettings()
	if err != nil || !settings.Enabled {
		return nil, ErrAWGConfigUnavailable
	}
	managed, options, err := loadAWGManagedEndpoint(m.deps.DB, settings)
	if err != nil || options.Amnezia == nil {
		return nil, ErrAWGConfigUnavailable
	}
	cipher, err := m.awgCipher()
	if err != nil {
		return nil, ErrAWGConfigUnavailable
	}
	privateKey, err := cipher.Decrypt(clientID, device.CryptoContext, device.PrivateKeyEnc)
	if err != nil {
		return nil, ErrAWGConfigUnavailable
	}
	defer clear(privateKey)
	psk, err := cipher.Decrypt(clientID, device.CryptoContext, device.PSKEnc)
	if err != nil {
		return nil, ErrAWGConfigUnavailable
	}
	defer clear(psk)
	if len(privateKey) != 32 || len(psk) != 32 {
		return nil, ErrAWGConfigUnavailable
	}
	mtu := settings.MTU
	if mtu == 0 {
		mtu = options.MTU
	}
	if mtu == 0 {
		mtu = 1420
	}
	dns := make([]string, len(settings.DNS))
	for i := range settings.DNS {
		dns[i] = settings.DNS[i].String()
	}
	a := options.Amnezia
	var config strings.Builder
	config.WriteString("[Interface]\n")
	config.WriteString("PrivateKey = " + base64.StdEncoding.EncodeToString(privateKey) + "\n")
	config.WriteString("Address = " + device.IPv4Address + "/32\n")
	config.WriteString("DNS = " + strings.Join(dns, ", ") + "\n")
	config.WriteString("MTU = " + strconv.FormatUint(uint64(mtu), 10) + "\n")
	writeAWGConfigInt(&config, "Jc", a.JC)
	writeAWGConfigInt(&config, "Jmin", a.JMin)
	writeAWGConfigInt(&config, "Jmax", a.JMax)
	writeAWGConfigInt(&config, "S1", a.S1)
	writeAWGConfigInt(&config, "S2", a.S2)
	writeAWGConfigInt(&config, "S3", a.S3)
	writeAWGConfigInt(&config, "S4", a.S4)
	writeAWGConfigValue(&config, "H1", a.H1)
	writeAWGConfigValue(&config, "H2", a.H2)
	writeAWGConfigValue(&config, "H3", a.H3)
	writeAWGConfigValue(&config, "H4", a.H4)
	writeAWGConfigString(&config, "I1", a.I1)
	writeAWGConfigString(&config, "I2", a.I2)
	writeAWGConfigString(&config, "I3", a.I3)
	writeAWGConfigString(&config, "I4", a.I4)
	writeAWGConfigString(&config, "I5", a.I5)
	writeAWGConfigString(&config, "J1", a.J1)
	writeAWGConfigString(&config, "J2", a.J2)
	writeAWGConfigString(&config, "J3", a.J3)
	writeAWGConfigInt64(&config, "Itime", a.ITime)
	config.WriteString("\n[Peer]\n")
	config.WriteString("PublicKey = " + managed.ServerPublicKey + "\n")
	config.WriteString("PresharedKey = " + base64.StdEncoding.EncodeToString(psk) + "\n")
	config.WriteString("Endpoint = " + settings.PublicEndpoint + "\n")
	config.WriteString("AllowedIPs = " + awgClientAllowedIPs(settings.ClientAllowedIPs) + "\n")
	config.WriteString("PersistentKeepalive = " + strconv.Itoa(awgClientKeepalive(settings.ClientKeepalive)) + "\n")
	return []byte(config.String()), nil
}

// awgClientAllowedIPs renders the AllowedIPs override. Values were already
// validated as CIDR prefixes (validateAWGEndpointMetadata), but this is text
// entering an INI config, so anything that fails to re-parse or contains a
// line break is dropped defensively rather than written out.
func awgClientAllowedIPs(overrides []string) string {
	safe := make([]string, 0, len(overrides))
	for _, value := range overrides {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || strings.ContainsAny(trimmed, "\r\n") {
			continue
		}
		if _, err := netip.ParsePrefix(trimmed); err != nil {
			continue
		}
		safe = append(safe, trimmed)
	}
	if len(safe) == 0 {
		return "0.0.0.0/0, ::/0"
	}
	return strings.Join(safe, ", ")
}

func awgClientKeepalive(override int) int {
	if override <= 0 || override > 3600 {
		return 25
	}
	return override
}

// RenderAWGConfigQR encodes the rendered AWG config as a PNG QR image. The
// upper size bound is not a hand-picked constant: it is whatever the QR codec
// can actually fit at the chosen error-correction level. AWG 2.0 configs carry
// long junk/init-packet fields (I1-I5, J1-J3), so Low ECC is used for the
// widest byte capacity (~2953 bytes at version 40). When the payload still
// exceeds that, qrcode.Encode reports "content too long to encode"; that case
// is surfaced as ErrAWGConfigTooLargeQR so callers can fall back to the .conf
// download instead of shipping a broken image.
func RenderAWGConfigQR(config []byte) ([]byte, error) {
	if len(config) == 0 {
		return nil, ErrAWGConfigUnavailable
	}
	png, err := qrcode.Encode(string(config), qrcode.Low, 512)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAWGConfigTooLargeQR, err)
	}
	return png, nil
}

func writeAWGConfigInt(builder *strings.Builder, key string, value int) {
	if value > 0 {
		builder.WriteString(key + " = " + strconv.Itoa(value) + "\n")
	}
}

func writeAWGConfigInt64(builder *strings.Builder, key string, value int64) {
	if value > 0 {
		builder.WriteString(key + " = " + strconv.FormatInt(value, 10) + "\n")
	}
}

func writeAWGConfigValue(builder *strings.Builder, key string, value any) {
	if value == nil {
		return
	}
	writeAWGConfigString(builder, key, fmt.Sprint(value))
}

func writeAWGConfigString(builder *strings.Builder, key, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		builder.WriteString(key + " = " + value + "\n")
	}
}
