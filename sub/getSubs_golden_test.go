package sub

import (
	"encoding/base64"
	"testing"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/service"
)

// goldenSubLinks is a fixed client.Links fixture exercising the two offline link
// kinds GetSubs assembles from the database (a "local" URI and a verbatim
// "external" URI). "sub" links are intentionally omitted because they fetch over
// the network and would make the output non-deterministic.
const goldenSubLinks = `[` +
	`{"type":"local","remark":"local-1","uri":"vless://11111111-1111-4111-8111-111111111111@example.com:443?type=ws#node-local-1"},` +
	`{"type":"external","remark":"ext-1","uri":"trojan://pass@example.org:8443#node-ext-1"}` +
	`]`

// goldenSubBody is the exact, byte-for-byte body GetSubs must produce for
// goldenSubLinks with subShowInfo=false / subNameInRemark=false (clientInfo
// empty, so the local URI is passed through unchanged) joined by "\n".
const goldenSubBody = "vless://11111111-1111-4111-8111-111111111111@example.com:443?type=ws#node-local-1\n" +
	"trojan://pass@example.org:8443#node-ext-1"

func seedGoldenSubClient(t *testing.T, secret string) {
	t.Helper()
	settingService := &service.SettingService{}
	if _, err := settingService.GetAllSetting(); err != nil {
		t.Fatal(err)
	}
	client := model.Client{
		Enable:    true,
		Name:      "golden-client",
		SubSecret: secret,
		Inbounds:  []byte("[]"),
		Links:     []byte(goldenSubLinks),
	}
	if err := database.GetDB().Create(&client).Error; err != nil {
		t.Fatal(err)
	}
}

func setSubSetting(t *testing.T, key, value string) {
	t.Helper()
	if err := database.GetDB().Model(model.Setting{}).Where("key = ?", key).Update("value", value).Error; err != nil {
		t.Fatal(err)
	}
	// Display settings are served from a TTL snapshot; drop it so the change is
	// observed by the next GetSubs call within this test.
	resetSubDisplaySettingsCacheForTest()
}

// TestGetSubsGoldenPlaintextByteExact pins the database-driven GetSubs assembly
// path (client lookup + display settings + link assembly + join) to an exact
// byte string, and proves the same input yields identical output on repeat. The
// existing determinism tests cover only the pure LinkGenerator / ConvertToClashMeta
// functions, not this DB-backed path.
func TestGetSubsGoldenPlaintextByteExact(t *testing.T) {
	initSubTestDB(t)
	const secret = "golden-secret-0001"
	seedGoldenSubClient(t, secret)
	setSubSetting(t, "subEncode", "false")

	body, _, err := (&SubService{}).GetSubs(secret)
	if err != nil {
		t.Fatal(err)
	}
	if body == nil {
		t.Fatal("GetSubs returned nil body")
	}
	if *body != goldenSubBody {
		t.Fatalf("GetSubs body mismatch:\n got: %q\nwant: %q", *body, goldenSubBody)
	}

	// Determinism: a second call on identical state must be byte-identical.
	body2, _, err := (&SubService{}).GetSubs(secret)
	if err != nil {
		t.Fatal(err)
	}
	if body2 == nil || *body2 != *body {
		t.Fatalf("GetSubs is non-deterministic: first=%q second=%v", *body, body2)
	}
}

// TestGetSubsGoldenBase64Encoding pins the default encoded path: with
// subEncode=true the body must be exactly the standard-base64 encoding of the
// plaintext golden body.
func TestGetSubsGoldenBase64Encoding(t *testing.T) {
	initSubTestDB(t)
	const secret = "golden-secret-0002"
	seedGoldenSubClient(t, secret)
	setSubSetting(t, "subEncode", "true")

	body, _, err := (&SubService{}).GetSubs(secret)
	if err != nil {
		t.Fatal(err)
	}
	if body == nil {
		t.Fatal("GetSubs returned nil body")
	}
	want := base64.StdEncoding.EncodeToString([]byte(goldenSubBody))
	if *body != want {
		t.Fatalf("GetSubs base64 body mismatch:\n got: %q\nwant: %q", *body, want)
	}
	decoded, err := base64.StdEncoding.DecodeString(*body)
	if err != nil {
		t.Fatalf("encoded body is not valid base64: %v", err)
	}
	if string(decoded) != goldenSubBody {
		t.Fatalf("decoded body mismatch:\n got: %q\nwant: %q", string(decoded), goldenSubBody)
	}
}
