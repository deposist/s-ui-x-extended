package importxui

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/deposist/s-ui-x/database/model"
	"github.com/gofrs/uuid/v5"
)

type clientAggregate struct {
	Email        string
	Enable       bool
	Inbounds     map[uint]struct{}
	Config       map[string]map[string]any
	SubSecret    string
	Desc         string
	Group        string
	LimitIP      int
	Volume       int64
	Expiry       int64
	Up           int64
	Down         int64
	LastOnline   int64
	SeenMetadata bool
}

func newClientAggregate(email string) *clientAggregate {
	return &clientAggregate{
		Email:    email,
		Enable:   true,
		Inbounds: map[uint]struct{}{},
		Config:   deterministicClientConfig(email),
		Group:    "imported",
	}
}

func collectClientAggregates(src *sourceDB, refs []ClientRef, inboundIDBySrc map[int64]uint) (map[string]*clientAggregate, error) {
	aggs := map[string]*clientAggregate{}
	get := func(email string) *clientAggregate {
		email = strings.TrimSpace(email)
		if email == "" {
			return nil
		}
		agg, ok := aggs[email]
		if !ok {
			agg = newClientAggregate(email)
			aggs[email] = agg
		}
		return agg
	}

	if err := src.eachClientTraffic(func(traffic xuiClientTraffic) error {
		agg := get(traffic.Email)
		if agg == nil {
			return nil
		}
		agg.Enable = agg.Enable && traffic.Enable
		agg.Up += traffic.Up
		agg.Down += traffic.Down
		agg.Volume = maxInt64(agg.Volume, traffic.Total)
		agg.Expiry = maxInt64(agg.Expiry, millisToSeconds(traffic.ExpiryTime))
		agg.LastOnline = maxInt64(agg.LastOnline, millisToSeconds(traffic.LastOnline))
		if dstID, ok := inboundIDBySrc[traffic.InboundID]; ok && dstID > 0 {
			agg.Inbounds[dstID] = struct{}{}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	for _, ref := range refs {
		agg := get(ref.Email)
		if agg == nil {
			continue
		}
		agg.SeenMetadata = true
		if ref.HasEnable {
			agg.Enable = agg.Enable && ref.Enable
		}
		if ref.DstInboundID > 0 {
			agg.Inbounds[ref.DstInboundID] = struct{}{}
		}
		agg.Volume = maxInt64(agg.Volume, ref.TotalGB)
		agg.Expiry = maxInt64(agg.Expiry, millisToSeconds(ref.ExpiryTime))
		agg.LimitIP = maxInt(agg.LimitIP, ref.LimitIP)
		if agg.Desc == "" && strings.TrimSpace(ref.Comment) != "" {
			agg.Desc = strings.TrimSpace(ref.Comment)
		}
		if agg.SubSecret == "" && ref.SubID != "" {
			agg.SubSecret = ref.SubID
		}
		if ref.TgID != "" {
			agg.Group = ref.TgID
		}
		applyProtocolConfig(agg.Config, ref)
	}
	return aggs, nil
}

func applyProtocolConfig(config map[string]map[string]any, ref ClientRef) {
	switch ref.Protocol {
	case "vless":
		if ref.UUID != "" {
			config["vless"]["uuid"] = ref.UUID
		}
		if ref.Flow != "" {
			config["vless"]["flow"] = ref.Flow
		}
	case "vmess":
		if ref.UUID != "" {
			config["vmess"]["uuid"] = ref.UUID
		}
		config["vmess"]["alterId"] = 0
	case "trojan":
		if ref.Password != "" {
			config["trojan"]["password"] = ref.Password
		}
	case "shadowsocks":
		if ref.Password != "" {
			config["shadowsocks"]["password"] = ref.Password
		}
	case "http":
		if ref.Password != "" {
			config["http"]["password"] = ref.Password
		}
	case "socks":
		if ref.Password != "" {
			config["socks"]["password"] = ref.Password
		}
	}
}

func (agg *clientAggregate) toModel() (model.Client, error) {
	inbounds := make([]uint, 0, len(agg.Inbounds))
	for id := range agg.Inbounds {
		inbounds = append(inbounds, id)
	}
	sort.Slice(inbounds, func(i, j int) bool { return inbounds[i] < inbounds[j] })
	inboundsJSON, err := marshalJSON(inbounds)
	if err != nil {
		return model.Client{}, err
	}
	configJSON, err := marshalJSON(agg.Config)
	if err != nil {
		return model.Client{}, err
	}
	subSecret := agg.SubSecret
	if subSecret == "" {
		generated, err := uuid.NewV4()
		if err != nil {
			return model.Client{}, err
		}
		subSecret = generated.String()
	}
	return model.Client{
		Enable:      agg.Enable,
		Name:        agg.Email,
		SubSecret:   subSecret,
		Config:      configJSON,
		Inbounds:    inboundsJSON,
		Links:       nil,
		Volume:      agg.Volume,
		Expiry:      agg.Expiry,
		Down:        agg.Down,
		Up:          agg.Up,
		Desc:        agg.Desc,
		Group:       agg.Group,
		LimitIP:     agg.LimitIP,
		IPLimitMode: "monitor",
		LastOnline:  agg.LastOnline,
		ResetDays:   0,
		TotalUp:     0,
		TotalDown:   0,
	}, nil
}

func mergeInboundJSON(existing json.RawMessage, add map[uint]struct{}) (json.RawMessage, error) {
	values := map[uint]struct{}{}
	if len(existing) > 0 {
		var ids []uint
		if err := json.Unmarshal(existing, &ids); err != nil {
			return nil, err
		}
		for _, id := range ids {
			values[id] = struct{}{}
		}
	}
	for id := range add {
		values[id] = struct{}{}
	}
	ids := make([]uint, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return marshalJSON(ids)
}

func millisToSeconds(value int64) int64 {
	if value <= 0 {
		return 0
	}
	if value > 1_000_000_000_000 {
		return value / 1000
	}
	return value
}

func maxInt64(a, b int64) int64 {
	if b > a {
		return b
	}
	return a
}

func maxInt(a, b int) int {
	if b > a {
		return b
	}
	return a
}

func summaryDetails(summary Summary) map[string]any {
	return map[string]any{
		"inbounds": map[string]any{
			"total":     summary.Inbounds.Total,
			"imported":  summary.Inbounds.Imported,
			"skipped":   summary.Inbounds.Skipped,
			"conflicts": summary.Inbounds.Conflicts,
		},
		"endpoints": map[string]any{
			"imported": summary.Endpoints.Imported,
			"skipped":  summary.Endpoints.Skipped,
		},
		"outbounds": map[string]any{
			"imported": summary.Outbounds.Imported,
			"skipped":  summary.Outbounds.Skipped,
		},
		"tls": map[string]any{
			"created": summary.TLS.Created,
			"reused":  summary.TLS.Reused,
		},
		"clients": map[string]any{
			"unique_emails": summary.Clients.UniqueEmails,
			"merged":        summary.Clients.Merged,
			"created":       summary.Clients.Created,
		},
		"historical": map[string]any{
			"total":    summary.Historical.Total,
			"imported": summary.Historical.Imported,
			"skipped":  summary.Historical.Skipped,
		},
		"routing": map[string]any{
			"total":    summary.Routing.Total,
			"imported": summary.Routing.Imported,
			"skipped":  summary.Routing.Skipped,
		},
	}
}

func auditDetails(summary Summary) ([]byte, error) {
	return json.Marshal(summaryDetails(summary))
}
