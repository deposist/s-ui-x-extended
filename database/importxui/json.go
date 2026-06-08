package importxui

import (
	"encoding/json"
	"fmt"
)

func marshalJSON(v any) (json.RawMessage, error) {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return json.RawMessage(raw), nil
}

func decodeJSON(raw json.RawMessage, dst any) error {
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}
	return nil
}
