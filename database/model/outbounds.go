package model

import "encoding/json"

// No outbound option struct declares these legacy inbound rule fields;
// option/outbound.go DialerOptions (lines 72-95) does not include them.
var legacyOutboundOptionFields = map[string]struct{}{
	"proxy_protocol":                  {},
	"proxy_protocol_accept_no_header": {},
	"udp_disable_domain_unmapping":    {},
}

type Outbound struct {
	Id      uint            `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Type    string          `json:"type" form:"type"`
	Tag     string          `json:"tag" form:"tag" gorm:"unique"`
	Options json.RawMessage `json:"-" form:"-"`
}

func (o *Outbound) UnmarshalJSON(data []byte) error {
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

	// Remaining fields
	o.Options, err = json.MarshalIndent(raw, "", "  ")
	return err
}

// MarshalJSON customizes marshalling
func (o Outbound) MarshalJSON() ([]byte, error) {
	// Combine fixed fields and dynamic fields into one map
	combined := make(map[string]interface{})
	combined["type"] = o.Type
	combined["tag"] = o.Tag

	if o.Options != nil {
		var restFields map[string]json.RawMessage
		if err := json.Unmarshal(o.Options, &restFields); err != nil {
			return nil, err
		}

		for k, v := range restFields {
			if _, legacy := legacyOutboundOptionFields[k]; legacy {
				continue
			}
			combined[k] = v
		}
	}

	return json.Marshal(combined)
}
