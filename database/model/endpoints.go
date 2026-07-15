package model

import (
	"encoding/json"
)

type Endpoint struct {
	Id      uint            `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Type    string          `json:"type" form:"type"`
	Tag     string          `json:"tag" form:"tag" gorm:"unique"`
	Options json.RawMessage `json:"-" form:"-"`
	Ext     json.RawMessage `json:"ext" form:"ext"`
}

func (o *Endpoint) UnmarshalJSON(data []byte) error {
	var err error
	var raw map[string]interface{}
	if err = json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Extract fixed fields and store the rest in Options
	if val, exists := raw["id"].(float64); exists {
		o.Id = uint(val)
	}
	delete(raw, "id")
	o.Type, _ = raw["type"].(string)
	delete(raw, "type")
	o.Tag, _ = raw["tag"].(string)
	delete(raw, "tag")
	o.Ext, _ = json.MarshalIndent(raw["ext"], "", "  ")
	delete(raw, "ext")
	// awgManaged is a panel-only marker added by EndpointService.GetAll; the
	// frontend echoes it back on save. It must never reach Options, because
	// Options is written verbatim into the sing-box config and sing-box
	// rejects unknown fields (core fails to start).
	delete(raw, "awgManaged")

	if o.Type == "warp" {
		normalizeWarpWireGuardOptions(raw)
	}

	// Remaining fields
	o.Options, err = json.MarshalIndent(raw, "", "  ")
	return err
}

// MarshalJSON customizes marshalling
func (o Endpoint) MarshalJSON() ([]byte, error) {
	// Combine fixed fields and dynamic fields into one map
	combined := make(map[string]interface{})
	switch o.Type {
	case "warp":
		combined["type"] = "wireguard"
	default:
		combined["type"] = o.Type
	}
	combined["tag"] = o.Tag

	if o.Options != nil {
		var restFields map[string]json.RawMessage
		if err := json.Unmarshal(o.Options, &restFields); err != nil {
			return nil, err
		}
		if o.Type == "warp" || o.Type == "wireguard" {
			normalizeWarpWireGuardRawOptions(restFields)
		}
		// Heal rows poisoned before UnmarshalJSON started stripping the
		// panel-only awgManaged marker; sing-box rejects unknown fields.
		delete(restFields, "awgManaged")

		for k, v := range restFields {
			combined[k] = v
		}
	}

	return json.Marshal(combined)
}

func normalizeWarpWireGuardOptions(raw map[string]interface{}) {
	delete(raw, "reserved")
	peers, ok := raw["peers"].([]interface{})
	if !ok {
		return
	}
	for _, item := range peers {
		peer, ok := item.(map[string]interface{})
		if ok {
			delete(peer, "reserved")
		}
	}
}

func normalizeWarpWireGuardRawOptions(raw map[string]json.RawMessage) {
	delete(raw, "reserved")
	peersRaw, ok := raw["peers"]
	if !ok {
		return
	}
	var peers []map[string]json.RawMessage
	if err := json.Unmarshal(peersRaw, &peers); err != nil {
		return
	}
	changed := false
	for _, peer := range peers {
		if _, ok := peer["reserved"]; ok {
			delete(peer, "reserved")
			changed = true
		}
	}
	if !changed {
		return
	}
	encoded, err := json.Marshal(peers)
	if err != nil {
		return
	}
	raw["peers"] = encoded
}
