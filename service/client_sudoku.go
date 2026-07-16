package service

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/util/common"
)

const invalidSudokuSplitKeyMessage = "invalid Sudoku split private key"

// Ed25519 subgroup order L in canonical little-endian scalar encoding.
var ed25519ScalarOrder = [32]byte{
	0xed, 0xd3, 0xf5, 0x5c, 0x1a, 0x63, 0x12, 0x58,
	0xd6, 0x9c, 0xf7, 0xa2, 0xde, 0xf9, 0xde, 0x14,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10,
}

func canonicalEd25519Scalar(encoded []byte) bool {
	if len(encoded) != 32 {
		return false
	}
	// Scalars are little-endian. Canonical encodings are integers strictly below L.
	for i := len(encoded) - 1; i >= 0; i-- {
		if encoded[i] < ed25519ScalarOrder[i] {
			return true
		}
		if encoded[i] > ed25519ScalarOrder[i] {
			return false
		}
	}
	return false
}

func normalizeSudokuSplitKey(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if len(value) != 128 {
		return "", common.NewError(invalidSudokuSplitKeyMessage)
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != 64 || !canonicalEd25519Scalar(decoded[:32]) || !canonicalEd25519Scalar(decoded[32:]) {
		return "", common.NewError(invalidSudokuSplitKeyMessage)
	}
	return strings.ToLower(value), nil
}

// normalizeClientSudokuConfig validates the API-provided client config and
// normalizes its optional delivery-only split key. Errors never contain the key.
func normalizeClientSudokuConfig(client *model.Client) error {
	if len(bytes.TrimSpace(client.Config)) == 0 {
		return nil
	}
	var config map[string]json.RawMessage
	if err := json.Unmarshal(client.Config, &config); err != nil {
		return err
	}
	raw, exists := config["sudoku"]
	if !exists || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil
	}
	var sudoku map[string]json.RawMessage
	if err := json.Unmarshal(raw, &sudoku); err != nil {
		return common.NewError(invalidSudokuSplitKeyMessage)
	}
	keyRaw, exists := sudoku["key"]
	if !exists {
		return nil
	}
	var key string
	if err := json.Unmarshal(keyRaw, &key); err != nil {
		return common.NewError(invalidSudokuSplitKeyMessage)
	}
	normalized, err := normalizeSudokuSplitKey(key)
	if err != nil {
		return err
	}
	sudoku["key"], _ = json.Marshal(normalized)
	config["sudoku"], _ = json.Marshal(sudoku)
	client.Config, err = json.Marshal(config)
	return err
}

func clientSudokuDeliveryEqual(a, b json.RawMessage) bool {
	key := func(raw json.RawMessage) string {
		var config map[string]map[string]any
		if json.Unmarshal(raw, &config) != nil {
			return ""
		}
		value, _ := config["sudoku"]["key"].(string)
		return strings.TrimSpace(value)
	}
	return key(a) == key(b)
}

// clientCoreConfigEqual ignores only the Sudoku split key because it changes
// delivery artifacts, not the server inbound. Every other client change keeps
// the existing core reload behavior.
func clientCoreConfigEqual(a, b json.RawMessage) bool {
	strip := func(raw json.RawMessage) any {
		var config map[string]any
		if json.Unmarshal(raw, &config) != nil {
			return string(raw)
		}
		if sudoku, ok := config["sudoku"].(map[string]any); ok {
			delete(sudoku, "key")
			if len(sudoku) == 0 {
				delete(config, "sudoku")
			}
		}
		return config
	}
	return jsonValuesEqual(strip(a), strip(b))
}

func jsonValuesEqual(a, b any) bool {
	aa, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return bytes.Equal(aa, bb)
}
