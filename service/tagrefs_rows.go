package service

import (
	"encoding/json"
	"fmt"

	"github.com/deposist/s-ui-x-extended/database/model"
)

// ssmServersOf extracts the servers map (mount path → managed shadowsocks
// inbound tag) from an ssm-api service row. Malformed options yield no
// entries: a broken row must never block an unrelated save.
func ssmServersOf(row model.Service) map[string]string {
	if row.Type != "ssm-api" || len(row.Options) == 0 {
		return nil
	}
	var opts struct {
		Servers map[string]string `json:"servers"`
	}
	if err := json.Unmarshal(row.Options, &opts); err != nil {
		return nil
	}
	return opts.Servers
}

func scanServiceRowsForInboundTag(rows []model.Service, tag string) []TagReference {
	var refs []TagReference
	for _, row := range rows {
		for path, inboundTag := range ssmServersOf(row) {
			if inboundTag == tag {
				refs = append(refs, TagReference{
					Kind:    "ssm-api service",
					Locator: fmt.Sprintf("ssm-api service %q (servers[%q])", row.Tag, path),
				})
			}
		}
	}
	return refs
}

func ssmServiceIdsReferencingInbound(rows []model.Service, tag string) []uint {
	var ids []uint
	for _, row := range rows {
		for _, inboundTag := range ssmServersOf(row) {
			if inboundTag == tag {
				ids = append(ids, row.Id)
				break
			}
		}
	}
	return ids
}

// optionsMapOf decodes an entity Options blob for reference scanning. A
// malformed row yields nil: a broken row must never block an unrelated save.
func optionsMapOf(options json.RawMessage) map[string]any {
	if len(options) == 0 {
		return nil
	}
	var opts map[string]any
	if err := json.Unmarshal(options, &opts); err != nil {
		return nil
	}
	return opts
}

// scanOutboundRowsForTag finds outbound rows whose dial detour or group
// membership points at the tag. Group members and detour targets are captured
// at adapter construction, so every hit is eager.
func scanOutboundRowsForTag(rows []model.Outbound, tag string, excludeId uint) []TagReference {
	var refs []TagReference
	for _, row := range rows {
		if row.Id == excludeId {
			continue
		}
		opts := optionsMapOf(row.Options)
		if opts == nil {
			continue
		}
		if detour, _ := opts["detour"].(string); detour == tag {
			refs = append(refs, TagReference{
				Kind:    "outbound detour",
				Locator: fmt.Sprintf("outbound %q (detour)", row.Tag),
			})
		}
		if row.Type != "selector" && row.Type != "urltest" && row.Type != FailoverType {
			continue
		}
		if members, _ := opts["outbounds"].([]any); containsTag(members, tag) {
			refs = append(refs, TagReference{
				Kind:    "group member",
				Locator: fmt.Sprintf("%s %q (outbounds list)", row.Type, row.Tag),
			})
		}
		if def, _ := opts["default"].(string); def == tag {
			refs = append(refs, TagReference{
				Kind:    "group default",
				Locator: fmt.Sprintf("%s %q (default)", row.Type, row.Tag),
			})
		}
	}
	return refs
}

func containsTag(members []any, tag string) bool {
	for _, member := range members {
		if s, ok := member.(string); ok && s == tag {
			return true
		}
	}
	return false
}

// scanEndpointRowsForTag finds endpoints (e.g. wireguard) dialing through the
// tag via detour - captured eagerly at endpoint construction.
func scanEndpointRowsForTag(rows []model.Endpoint, tag string, excludeId uint) []TagReference {
	var refs []TagReference
	for _, row := range rows {
		if row.Id == excludeId {
			continue
		}
		opts := optionsMapOf(row.Options)
		if detour, _ := opts["detour"].(string); detour == tag {
			refs = append(refs, TagReference{
				Kind:    "endpoint detour",
				Locator: fmt.Sprintf("endpoint %q (detour)", row.Tag),
			})
		}
	}
	return refs
}

// scanServiceRowsForOutboundDetour finds services (e.g. ccm/ocm) dialing
// through the tag via detour - their dialer is built at service construction.
func scanServiceRowsForOutboundDetour(rows []model.Service, tag string) []TagReference {
	var refs []TagReference
	for _, row := range rows {
		opts := optionsMapOf(row.Options)
		if detour, _ := opts["detour"].(string); detour == tag {
			refs = append(refs, TagReference{
				Kind:    "service detour",
				Locator: fmt.Sprintf("%s service %q (detour)", row.Type, row.Tag),
			})
		}
	}
	return refs
}
