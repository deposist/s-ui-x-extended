package model

import "encoding/json"

type Provider struct {
	Id      uint            `json:"id" form:"id" gorm:"primaryKey;autoIncrement"`
	Type    string          `json:"type" form:"type"`
	Tag     string          `json:"tag" form:"tag" gorm:"unique"`
	Options json.RawMessage `json:"-" form:"-"`
}

func (p *Provider) UnmarshalJSON(data []byte) error {
	var err error
	var raw map[string]interface{}
	if err = json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Extract fixed fields and store the rest in Options
	if val, exists := raw["id"].(float64); exists {
		p.Id = uint(val)
	}
	delete(raw, "id")
	p.Type, _ = raw["type"].(string)
	delete(raw, "type")
	p.Tag, _ = raw["tag"].(string)
	delete(raw, "tag")

	// Remaining fields
	p.Options, err = json.MarshalIndent(raw, "", "  ")
	return err
}

// MarshalJSON customizes marshalling
func (p Provider) MarshalJSON() ([]byte, error) {
	// Combine fixed fields and dynamic fields into one map
	combined := make(map[string]interface{})
	combined["type"] = p.Type
	combined["tag"] = p.Tag

	if p.Options != nil {
		var restFields map[string]json.RawMessage
		if err := json.Unmarshal(p.Options, &restFields); err != nil {
			return nil, err
		}

		for k, v := range restFields {
			combined[k] = v
		}
	}

	return json.Marshal(combined)
}
