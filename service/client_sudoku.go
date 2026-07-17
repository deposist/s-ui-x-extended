package service

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"filippo.io/edwards25519"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/util/common"
	"gorm.io/gorm"
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

func parseSudokuMasterKey(value string) (*edwards25519.Scalar, error) {
	decoded, err := hex.DecodeString(strings.TrimSpace(value))
	if err != nil || len(decoded) != 32 {
		return nil, common.NewError("Sudoku inbound key must be a canonical 32-byte master private key")
	}
	master, err := edwards25519.NewScalar().SetCanonicalBytes(decoded)
	if err != nil {
		return nil, common.NewError("Sudoku inbound key must be a canonical 32-byte master private key")
	}
	return master, nil
}

func sudokuPublicKey(master *edwards25519.Scalar) string {
	return hex.EncodeToString(edwards25519.NewIdentityPoint().ScalarBaseMult(master).Bytes())
}

func splitSudokuKeyMatchesMaster(value string, master *edwards25519.Scalar) bool {
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != 64 {
		return false
	}
	left, err := edwards25519.NewScalar().SetCanonicalBytes(decoded[:32])
	if err != nil {
		return false
	}
	right, err := edwards25519.NewScalar().SetCanonicalBytes(decoded[32:])
	if err != nil {
		return false
	}
	recovered := edwards25519.NewScalar().Add(left, right)
	return recovered.Equal(master) == 1
}

func generateSudokuMasterKey() (string, error) {
	var seed [64]byte
	if _, err := rand.Read(seed[:]); err != nil {
		return "", fmt.Errorf("generate Sudoku master key: %w", err)
	}
	master, err := edwards25519.NewScalar().SetUniformBytes(seed[:])
	if err != nil {
		return "", fmt.Errorf("generate Sudoku master key: %w", err)
	}
	return hex.EncodeToString(master.Bytes()), nil
}

func splitSudokuMasterKey(master *edwards25519.Scalar) (string, error) {
	var seed [64]byte
	if _, err := rand.Read(seed[:]); err != nil {
		return "", fmt.Errorf("generate Sudoku client split key: %w", err)
	}
	left, err := edwards25519.NewScalar().SetUniformBytes(seed[:])
	if err != nil {
		return "", fmt.Errorf("generate Sudoku client split key: %w", err)
	}
	right := edwards25519.NewScalar().Subtract(master, left)
	return hex.EncodeToString(append(left.Bytes(), right.Bytes()...)), nil
}

func ensureClientSudokuKeys(tx *gorm.DB, client *model.Client) error {
	var inboundIDs []uint
	if err := json.Unmarshal(client.Inbounds, &inboundIDs); err != nil {
		return err
	}
	if len(inboundIDs) == 0 {
		return normalizeClientSudokuConfig(client)
	}

	var inbounds []model.Inbound
	if err := tx.Model(model.Inbound{}).Select("id", "type", "options").Where("id IN ? AND type = ?", inboundIDs, "sudoku").Find(&inbounds).Error; err != nil {
		return err
	}
	if len(inbounds) == 0 {
		return normalizeClientSudokuConfig(client)
	}

	var config map[string]json.RawMessage
	if len(bytes.TrimSpace(client.Config)) > 0 {
		if err := json.Unmarshal(client.Config, &config); err != nil {
			return err
		}
	}
	if config == nil {
		config = make(map[string]json.RawMessage)
	}
	var sudoku struct {
		Key  string            `json:"key,omitempty"`
		Keys map[string]string `json:"keys,omitempty"`
	}
	if raw := config["sudoku"]; len(bytes.TrimSpace(raw)) > 0 && !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		if err := json.Unmarshal(raw, &sudoku); err != nil {
			return common.NewError(invalidSudokuSplitKeyMessage)
		}
	}
	legacyKey, err := normalizeSudokuSplitKey(sudoku.Key)
	if err != nil {
		return err
	}
	if sudoku.Keys == nil {
		sudoku.Keys = make(map[string]string, len(inbounds))
	}
	assigned := make(map[string]struct{}, len(inbounds))
	for _, inbound := range inbounds {
		id := strconv.FormatUint(uint64(inbound.Id), 10)
		assigned[id] = struct{}{}
		var options struct {
			Key       string `json:"key"`
			MasterKey string `json:"master_key"`
		}
		if err := json.Unmarshal(inbound.Options, &options); err != nil {
			return err
		}
		masterValue := options.MasterKey
		if masterValue == "" {
			// Compatibility with builds that briefly stored generated master
			// scalars directly in key before master_key was introduced.
			masterValue = options.Key
		}
		master, masterErr := parseSudokuMasterKey(masterValue)
		if existing := strings.TrimSpace(sudoku.Keys[id]); existing != "" {
			normalized, err := normalizeSudokuSplitKey(existing)
			if err != nil {
				return err
			}
			if masterErr != nil || splitSudokuKeyMatchesMaster(normalized, master) {
				sudoku.Keys[id] = normalized
				continue
			}
		}
		if legacyKey != "" && (masterErr != nil || splitSudokuKeyMatchesMaster(legacyKey, master)) {
			sudoku.Keys[id] = legacyKey
			continue
		}
		if masterErr != nil {
			// Existing PSK/public-key inbounds remain usable through the legacy
			// shared-key fallback; only canonical master scalars support splitting.
			continue
		}
		sudoku.Keys[id], err = splitSudokuMasterKey(master)
		if err != nil {
			return err
		}
	}
	for id := range sudoku.Keys {
		if _, ok := assigned[id]; !ok {
			delete(sudoku.Keys, id)
		}
	}
	sudoku.Key = ""
	raw, err := json.Marshal(sudoku)
	if err != nil {
		return err
	}
	config["sudoku"] = raw
	client.Config, err = json.Marshal(config)
	return err
}

func ensureSudokuInboundMasterKey(inbound *model.Inbound, generate bool) error {
	if inbound.Type != "sudoku" {
		return nil
	}
	var options map[string]json.RawMessage
	if err := json.Unmarshal(inbound.Options, &options); err != nil {
		return err
	}
	var key, masterKey string
	if raw := options["key"]; len(raw) > 0 {
		_ = json.Unmarshal(raw, &key)
	}
	if raw := options["master_key"]; len(raw) > 0 {
		_ = json.Unmarshal(raw, &masterKey)
	}
	if strings.TrimSpace(masterKey) != "" || strings.TrimSpace(key) != "" || !generate {
		return nil
	}
	masterKey, err := generateSudokuMasterKey()
	if err != nil {
		return err
	}
	master, err := parseSudokuMasterKey(masterKey)
	if err != nil {
		return err
	}
	options["master_key"], _ = json.Marshal(masterKey)
	options["key"], _ = json.Marshal(sudokuPublicKey(master))
	inbound.Options, err = json.Marshal(options)
	return err
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
	keys := func(raw json.RawMessage) any {
		var config map[string]any
		if json.Unmarshal(raw, &config) != nil {
			return nil
		}
		sudoku, _ := config["sudoku"].(map[string]any)
		return sudoku
	}
	return jsonValuesEqual(keys(a), keys(b))
}

// clientCoreConfigEqual ignores Sudoku delivery keys because they change
// client artifacts, not the server inbound. Every other client change keeps
// the existing core reload behavior.
func clientCoreConfigEqual(a, b json.RawMessage) bool {
	strip := func(raw json.RawMessage) any {
		var config map[string]any
		if json.Unmarshal(raw, &config) != nil {
			return string(raw)
		}
		if sudoku, ok := config["sudoku"].(map[string]any); ok {
			delete(sudoku, "key")
			delete(sudoku, "keys")
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
