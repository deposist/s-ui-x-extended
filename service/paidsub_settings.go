package service

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// NewPaidSubHTTPClient builds the HTTP client the paid-subscriptions bot uses,
// honoring its OWN transport config (independent from the admin notifier):
// either its own proxy (paidSubProxy*) or a sing-box outbound (paidSubOutboundTag).
// The longer timeout accommodates getUpdates long-polling.
func NewPaidSubHTTPClient(timeout time.Duration) (*http.Client, error) {
	s := &SettingService{}
	mode, _ := s.GetPaidSubTransportMode()
	if mode == "outbound" {
		tag, _ := s.GetPaidSubOutboundTag()
		return newCoreOutboundHTTPClient(tag, timeout)
	}
	cfg, err := s.paidSubProxyConfig()
	if err != nil {
		return nil, err
	}
	client, err := newTelegramHTTPClient(cfg)
	if err != nil {
		return nil, err
	}
	if timeout > 0 {
		client.Timeout = timeout
	}
	return client, nil
}

// Exported accessors for the experimental "Paid Subscriptions" module (paidsub
// package). They live in the service package so they can reuse the unexported,
// decryption-aware getString/getBool/getInt helpers; the encrypted token keys
// are decrypted transparently here and must never be logged.

func (s *SettingService) GetPaidSubEnabled() (bool, error) {
	return s.getBool("paidSubEnabled")
}

func (s *SettingService) GetPaidSubBotToken() (string, error) {
	return s.getString("paidSubBotToken")
}

func (s *SettingService) GetPaidSubBotPollSeconds() (int, error) {
	v, err := s.getInt("paidSubBotPollSeconds")
	if err != nil {
		return 25, err
	}
	if v < 1 {
		v = 1
	}
	if v > 50 {
		v = 50
	}
	return v, nil
}

func (s *SettingService) GetPaidSubUpdateOffset() (int64, error) {
	str, err := s.getString("paidSubUpdateOffset")
	if err != nil {
		return 0, err
	}
	if str == "" {
		return 0, nil
	}
	return strconv.ParseInt(str, 10, 64)
}

func (s *SettingService) SetPaidSubUpdateOffset(offset int64) error {
	return s.setString("paidSubUpdateOffset", strconv.FormatInt(offset, 10))
}

func (s *SettingService) GetPaidSubAutoRegister() (bool, error) {
	return s.getBool("paidSubAutoRegister")
}

// GetPaidSubAutoInbounds returns the admin-selected inbound ids that newly
// auto-registered clients are assigned to. Invalid JSON yields an empty list
// (auto-registration then has nothing to assign and is effectively disabled).
func (s *SettingService) GetPaidSubAutoInbounds() ([]uint, error) {
	str, err := s.getString("paidSubAutoInbounds")
	if err != nil {
		return nil, err
	}
	if str == "" {
		return []uint{}, nil
	}
	var ids []uint
	if err := json.Unmarshal([]byte(str), &ids); err != nil {
		return []uint{}, nil
	}
	return ids, nil
}

func (s *SettingService) GetPaidSubTrialDays() (int, error) {
	return s.getInt("paidSubTrialDays")
}

func (s *SettingService) GetPaidSubTrialVolumeGB() (int, error) {
	return s.getInt("paidSubTrialVolumeGB")
}

func (s *SettingService) GetPaidSubMaxClients() (int, error) {
	return s.getInt("paidSubMaxClients")
}

func (s *SettingService) GetPaidSubStartRateLimitPerMin() (int, error) {
	return s.getInt("paidSubStartRateLimitPerMin")
}

func (s *SettingService) GetPaidSubCurrency() (string, error) {
	return s.getString("paidSubCurrency")
}

func (s *SettingService) GetPaidSubStarsEnabled() (bool, error) {
	return s.getBool("paidSubStarsEnabled")
}

func (s *SettingService) GetPaidSubYooKassaEnabled() (bool, error) {
	return s.getBool("paidSubYooKassaEnabled")
}

func (s *SettingService) GetPaidSubYooKassaToken() (string, error) {
	return s.getString("paidSubYooKassaToken")
}

func (s *SettingService) GetPaidSubStripeEnabled() (bool, error) {
	return s.getBool("paidSubStripeEnabled")
}

func (s *SettingService) GetPaidSubStripeToken() (string, error) {
	return s.getString("paidSubStripeToken")
}

func (s *SettingService) GetPaidSubPayMasterEnabled() (bool, error) {
	return s.getBool("paidSubPayMasterEnabled")
}

func (s *SettingService) GetPaidSubPayMasterToken() (string, error) {
	return s.getString("paidSubPayMasterToken")
}

func (s *SettingService) GetPaidSubCryptoBotEnabled() (bool, error) {
	return s.getBool("paidSubCryptoBotEnabled")
}

func (s *SettingService) GetPaidSubCryptoBotToken() (string, error) {
	return s.getString("paidSubCryptoBotToken")
}

func (s *SettingService) GetPaidSubExternalEnabled() (bool, error) {
	return s.getBool("paidSubExternalEnabled")
}

func (s *SettingService) GetPaidSubExternalUrlTemplate() (string, error) {
	return s.getString("paidSubExternalUrlTemplate")
}

func (s *SettingService) GetPaidSubOrderTTLMinutes() (int, error) {
	return s.getInt("paidSubOrderTTLMinutes")
}

func (s *SettingService) GetPaidSubGreeting() (string, error) {
	return s.getString("paidSubGreeting")
}

// GetPaidSubRefundRevoke reports the admin policy for the bot's user-initiated
// Stars auto-refund: when true (default), a successful refund also rolls back
// the days/traffic that order granted (anti-abuse: buy → refund → keep using).
// The user never chooses this; the panel refund button has its own per-refund
// toggle.
func (s *SettingService) GetPaidSubRefundRevoke() (bool, error) {
	return s.getBool("paidSubRefundRevoke")
}

func (s *SettingService) GetPaidSubTransportMode() (string, error) {
	return s.getString("paidSubTransportMode")
}

func (s *SettingService) GetPaidSubOutboundTag() (string, error) {
	return s.getString("paidSubOutboundTag")
}

func (s *SettingService) paidSubProxyConfig() (telegramProxyConfig, error) {
	proxyURL, err := s.getString("paidSubProxyURL")
	if err != nil {
		return telegramProxyConfig{}, err
	}
	username, err := s.getString("paidSubProxyUsername")
	if err != nil {
		return telegramProxyConfig{}, err
	}
	password, err := s.getString("paidSubProxyPassword")
	if err != nil {
		return telegramProxyConfig{}, err
	}
	return telegramProxyConfig{URL: proxyURL, Username: username, Password: password}, nil
}
