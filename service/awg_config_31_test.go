package service

import (
	"context"
	"encoding/json"
	"net/netip"
	"strings"
	"testing"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/database/model"
)
func testAWG31ServerKey(t *testing.T) string {
	t.Helper()
	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	return key.String()
}

func TestRenderOwnedConfigAWG31Flags(t *testing.T) {
	render := func(amnezia map[string]any) string {
		manager, _ := newAWGCreateTestManager(t, &fakeAWGProvisioner{
			snapshot: func(context.Context) (AWGPeerSnapshot, error) { return AWGPeerSnapshot{}, nil },
			add:      func(context.Context, AWGPeerSpec) error { return nil },
			remove:   func(context.Context, string) error { return nil },
		})
		options := map[string]any{
			"address":     []string{"10.77.0.1/29"},
			"private_key": testAWG31ServerKey(t),
			"listen_port": 51820,
			"mtu":         1380,
			"amnezia":     amnezia,
		}
		encoded, err := json.Marshal(options)
		if err != nil {
			t.Fatal(err)
		}
		if err := database.GetDB().Create(&model.Endpoint{Type: "wireguard", Tag: "awg31", Options: encoded}).Error; err != nil {
			t.Fatal(err)
		}
		manager.deps.LoadSettings = func() (AWGSettings, error) {
			return AWGSettings{Enabled: true, EndpointTag: "awg31", PublicEndpoint: "vpn.example.com:51820", Subnet: netip.MustParsePrefix("10.77.0.0/29")}, nil
		}
		client := createAWGEligibleClient(t)
		device, err := manager.CreateDevice(context.Background(), client.Id, "config31", "Phone", 1, 0)
		if err != nil {
			t.Fatal(err)
		}
		cfg, err := manager.RenderOwnedConfig(context.Background(), device.ID, client.Id)
		if err != nil {
			t.Fatal(err)
		}
		return string(cfg)
	}

	t.Run("flags on are rendered", func(t *testing.T) {
		cfg := render(map[string]any{"jc": 4, "random_trailers": true, "disable_cookies": true})
		if !strings.Contains(cfg, "RandomTrailers = true\n") {
			t.Fatalf("config lacks RandomTrailers:\n%s", cfg)
		}
		if !strings.Contains(cfg, "DisableCookies = true\n") {
			t.Fatalf("config lacks DisableCookies:\n%s", cfg)
		}
	})

	t.Run("flags off stay absent", func(t *testing.T) {
		cfg := render(map[string]any{"jc": 4})
		if strings.Contains(cfg, "RandomTrailers") {
			t.Fatalf("config must not mention RandomTrailers:\n%s", cfg)
		}
		if strings.Contains(cfg, "DisableCookies") {
			t.Fatalf("config must not mention DisableCookies:\n%s", cfg)
		}
	})
}
